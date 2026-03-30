// ---------------------------------------------------------------------------------------
//
//	client.go
//	---------
//
//	CLI speed test client. Connects to a web-speedtest server and measures ping,
//	download speed, and upload speed.
//
//	(c) 2026 WaterJuice — Released under the Unlicense; see LICENSE.
//
//	Version History
//	---------------
//	Mar 2026 - Created (Python)
//	Mar 2026 - Rewritten in Go
//
// ---------------------------------------------------------------------------------------
package internal

// ---------------------------------------------------------------------------------------
//
//	Imports
//
// ---------------------------------------------------------------------------------------

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------------------
//
//	Constants
//
// ---------------------------------------------------------------------------------------

const (
	// Number of ping samples
	pingCount = 10

	// Download test size: 25 MB
	downloadSize = 25 * 1024 * 1024

	// Upload test size: 10 MB
	uploadSize = 10 * 1024 * 1024

	// Read chunk size: 64 KB
	clientChunkSize = 65536

	// Number of parallel streams for download/upload
	parallelStreams = 6

	// Connection timeout in seconds
	clientTimeout = 30 * time.Second
)

// ---------------------------------------------------------------------------------------
//
//	Helpers
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// formatSpeed formats a speed value in bits per second into a human-readable string,
// automatically selecting the appropriate unit (bps, Kbps, Mbps, or Gbps).
func formatSpeed(bitsPerSecond float64) string {
	if bitsPerSecond >= 1_000_000_000 {
		return fmt.Sprintf("%.2f Gbps", bitsPerSecond/1_000_000_000)
	}
	if bitsPerSecond >= 1_000_000 {
		return fmt.Sprintf("%.2f Mbps", bitsPerSecond/1_000_000)
	}
	if bitsPerSecond >= 1_000 {
		return fmt.Sprintf("%.2f Kbps", bitsPerSecond/1_000)
	}
	return fmt.Sprintf("%.0f bps", bitsPerSecond)
}

// ---------------------------------------------------------------------------------------
// formatLatency formats a latency value in milliseconds into a human-readable string.
// Values under 10 ms show two decimals; values under 100 ms show one decimal.
func formatLatency(ms float64) string {
	if ms < 10.0 {
		return fmt.Sprintf("%.2f ms", ms)
	}
	if ms < 100.0 {
		return fmt.Sprintf("%.1f ms", ms)
	}
	return fmt.Sprintf("%.0f ms", ms)
}

// ---------------------------------------------------------------------------------------
// progressBar renders a Unicode block progress bar of the given width. Fraction is
// clamped to [0, 1]. Filled positions use █ and empty positions use ░.
func progressBar(fraction float64, width int) string {
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	filled := int(fraction * float64(width))
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

// ---------------------------------------------------------------------------------------
//
//	Speed Test Functions
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// testPing measures round-trip latency by sending pingCount HTTP GET requests to
// /api/ping and averaging the results. Returns the average latency in milliseconds
// and true on success, or (0, false) if all samples failed.
func testPing(client *http.Client, baseURL string, isTTY bool, quiet bool) (float64, bool) {
	url := baseURL + "/api/ping"
	var latencies []float64

	if !quiet {
		fmt.Print(colorDim(fmt.Sprintf("  Testing ping (%d samples)...", pingCount), isTTY))
	}

	for range pingCount {
		start := time.Now()
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		io.ReadAll(resp.Body)
		resp.Body.Close()
		elapsed := float64(time.Since(start).Microseconds()) / 1000
		latencies = append(latencies, elapsed)
	}

	if !quiet {
		fmt.Print("\r")
	}

	if len(latencies) == 0 {
		return 0, false
	}

	avg := 0.0
	for _, l := range latencies {
		avg += l
	}
	avg /= float64(len(latencies))
	return avg, true
}

// ---------------------------------------------------------------------------------------
// testDownload measures download throughput using parallelStreams concurrent HTTP
// GET requests to /api/download. Each stream downloads downloadSize/parallelStreams
// bytes. A live progress bar with current speed is shown unless quiet is true.
// Returns bits per second and true on success, or (0, false) on failure.
func testDownload(client *http.Client, baseURL string, isTTY bool, quiet bool) (float64, bool) {
	streamSize := downloadSize / parallelStreams
	totalTarget := streamSize * parallelStreams
	var totalReceived atomic.Int64
	var mu sync.Mutex

	if !quiet {
		fmt.Print(colorDim("  Testing download...", isTTY))
	}

	start := time.Now()

	var wg sync.WaitGroup
	var downloadErr error

	for range parallelStreams {
		wg.Add(1)
		go func() {
			defer wg.Done()
			url := fmt.Sprintf("%s/api/download?size=%d", baseURL, streamSize)
			resp, err := client.Get(url)
			if err != nil {
				mu.Lock()
				downloadErr = err
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			buf := make([]byte, clientChunkSize)
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					totalReceived.Add(int64(n))
					if !quiet {
						received := totalReceived.Load()
						fraction := float64(received) / float64(totalTarget)
						elapsed := time.Since(start).Seconds()
						speed := float64(received*8) / max(elapsed, 0.001)
						fmt.Printf("\r  Testing download... %s %s   ",
							progressBar(fraction, 30), formatSpeed(speed))
					}
				}
				if err != nil {
					break
				}
			}
		}()
	}

	wg.Wait()

	if !quiet {
		fmt.Print("\r")
	}

	if downloadErr != nil {
		if !quiet {
			fmt.Printf("\r  Download failed: %s%40s\n", downloadErr, "")
		}
		return 0, false
	}

	elapsed := time.Since(start).Seconds()
	received := totalReceived.Load()

	if elapsed <= 0 || received == 0 {
		return 0, false
	}

	bps := float64(received*8) / elapsed
	return bps, true
}

// ---------------------------------------------------------------------------------------
// testUpload measures upload throughput using parallelStreams concurrent HTTP POST
// requests to /api/upload. Each stream uploads uploadSize/parallelStreams bytes of
// random data. Returns bits per second and true on success, or (0, false) on failure.
func testUpload(client *http.Client, baseURL string, isTTY bool, quiet bool) (float64, bool) {
	streamSize := uploadSize / parallelStreams
	totalSize := streamSize * parallelStreams

	if !quiet {
		fmt.Print(colorDim("  Testing upload...", isTTY))
	}

	// Generate random upload payload
	payload := make([]byte, streamSize)
	rand.Read(payload)

	start := time.Now()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var uploadErr error

	for range parallelStreams {
		wg.Add(1)
		go func() {
			defer wg.Done()
			url := baseURL + "/api/upload"
			resp, err := client.Post(url, "application/octet-stream",
				bytes.NewReader(payload))
			if err != nil {
				mu.Lock()
				uploadErr = err
				mu.Unlock()
				return
			}
			io.ReadAll(resp.Body)
			resp.Body.Close()
		}()
	}

	wg.Wait()

	if !quiet {
		fmt.Print("\r")
	}

	if uploadErr != nil {
		if !quiet {
			fmt.Printf("\r  Upload failed: %s%40s\n", uploadErr, "")
		}
		return 0, false
	}

	elapsed := time.Since(start).Seconds()

	if elapsed <= 0 {
		return 0, false
	}

	bps := float64(totalSize*8) / elapsed
	return bps, true
}

// ---------------------------------------------------------------------------------------
// checkServer verifies that a web-speedtest server is reachable by fetching /api/info
// and confirming the response contains "server": "web-speedtest". Prints the server
// version unless quiet is true. Returns true if the server is valid and reachable.
func checkServer(client *http.Client, baseURL string, isTTY bool, quiet bool) bool {
	url := baseURL + "/api/info"
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	var data map[string]any
	if json.Unmarshal(body, &data) != nil {
		return false
	}

	if data["server"] != "web-speedtest" {
		return false
	}

	if !quiet {
		version, _ := data["version"].(string)
		if version == "" {
			version = "unknown"
		}
		fmt.Println(colorDim(fmt.Sprintf("  Server version: %s", version), isTTY))
	}
	return true
}

// ---------------------------------------------------------------------------------------
//
//	URL Normalisation
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// normaliseURL ensures the server URL has an http:// or https:// scheme and strips
// any trailing slashes. If no scheme is present, http:// is prepended.
func normaliseURL(serverURL string) string {
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		serverURL = "http://" + serverURL
	}
	return strings.TrimRight(serverURL, "/")
}

// ---------------------------------------------------------------------------------------
//
//	Client Entry Points
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// runClientNormal runs the full interactive speed test with coloured output, progress
// bars, and a summary panel. Returns 0 on success, 1 if the server is unreachable.
func runClientNormal(serverURL string) int {
	isTTY := isStdoutTerminal()
	client := &http.Client{Timeout: clientTimeout}

	fmt.Println(colorBold("web-speedtest client", isTTY))
	fmt.Println(colorDim(fmt.Sprintf("  server:   %s", serverURL), isTTY))
	fmt.Println()

	// Check server
	fmt.Print(colorDim("  Connecting...", isTTY))
	if !checkServer(client, serverURL, isTTY, false) {
		fmt.Printf("\r%s%40s\n", colorRed("  Failed to connect to server.", isTTY), "")
		fmt.Fprintln(os.Stderr, colorRed(fmt.Sprintf("  Could not reach %s", serverURL), isTTY))
		return 1
	}
	fmt.Printf("\r%s%40s\n", colorGreen("  Connected.", isTTY), "")
	fmt.Println()

	// Ping
	pingMs, pingOk := testPing(client, serverURL, isTTY, false)
	if pingOk {
		fmt.Printf("  %-14s%s%40s\n", "Ping:", colorCyan(formatLatency(pingMs), isTTY), "")
	} else {
		fmt.Printf("  %-14s%s%40s\n", "Ping:", colorYellow("failed", isTTY), "")
	}
	fmt.Println()

	// Download
	downloadBps, downloadOk := testDownload(client, serverURL, isTTY, false)
	if downloadOk {
		fmt.Printf("  %-14s%s%40s\n", "Download:", colorCyan(formatSpeed(downloadBps), isTTY), "")
	} else {
		fmt.Printf("  %-14s%s%40s\n", "Download:", colorYellow("failed", isTTY), "")
	}
	fmt.Println()

	// Upload
	uploadBps, uploadOk := testUpload(client, serverURL, isTTY, false)
	if uploadOk {
		fmt.Printf("  %-14s%s%40s\n", "Upload:", colorCyan(formatSpeed(uploadBps), isTTY), "")
	} else {
		fmt.Printf("  %-14s%s%40s\n", "Upload:", colorYellow("failed", isTTY), "")
	}
	fmt.Println()

	// Summary
	fmt.Println(colorDim("  "+strings.Repeat("-", 40), isTTY))
	if pingOk && downloadOk && uploadOk {
		fmt.Println(colorBold("  Results", isTTY))
		fmt.Printf("    Ping:      %s\n", colorGreen(formatLatency(pingMs), isTTY))
		fmt.Printf("    Download:  %s\n", colorGreen(formatSpeed(downloadBps), isTTY))
		fmt.Printf("    Upload:    %s\n", colorGreen(formatSpeed(uploadBps), isTTY))
	} else {
		fmt.Println(colorYellow("  Some tests failed — results may be incomplete.", isTTY))
	}
	fmt.Println()

	return 0
}

// ---------------------------------------------------------------------------------------
// runClientCompact runs the speed test and prints a single summary line suitable for
// scripting: "ping X / down Y / up Z". Returns 0 on success, 1 on connection failure.
func runClientCompact(serverURL string) int {
	client := &http.Client{Timeout: clientTimeout}
	if !checkServer(client, serverURL, false, true) {
		fmt.Fprintf(os.Stderr, "error: could not reach %s\n", serverURL)
		return 1
	}

	pingMs, pingOk := testPing(client, serverURL, false, true)
	downloadBps, downloadOk := testDownload(client, serverURL, false, true)
	uploadBps, uploadOk := testUpload(client, serverURL, false, true)

	pingStr := "failed"
	if pingOk {
		pingStr = formatLatency(pingMs)
	}
	downStr := "failed"
	if downloadOk {
		downStr = formatSpeed(downloadBps)
	}
	upStr := "failed"
	if uploadOk {
		upStr = formatSpeed(uploadBps)
	}

	fmt.Printf("ping %s / down %s / up %s\n", pingStr, downStr, upStr)
	return 0
}

// ---------------------------------------------------------------------------------------
// runClientJSON runs the speed test and prints results as pretty-printed JSON with
// both raw numeric values (ping_ms, download_bps, upload_bps) and human-readable
// strings. Failed tests produce null values. Returns 0 on success, 1 on connection failure.
func runClientJSON(serverURL string) int {
	client := &http.Client{Timeout: clientTimeout}
	if !checkServer(client, serverURL, false, true) {
		result := map[string]any{"error": fmt.Sprintf("could not reach %s", serverURL)}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return 1
	}

	pingMs, pingOk := testPing(client, serverURL, false, true)
	downloadBps, downloadOk := testDownload(client, serverURL, false, true)
	uploadBps, uploadOk := testUpload(client, serverURL, false, true)

	type jsonResult struct {
		Server      string  `json:"server"`
		PingMs      *int    `json:"ping_ms"`
		Ping        *string `json:"ping"`
		DownloadBps *int    `json:"download_bps"`
		Download    *string `json:"download"`
		UploadBps   *int    `json:"upload_bps"`
		Upload      *string `json:"upload"`
	}

	result := jsonResult{Server: serverURL}

	if pingOk {
		ms := int(pingMs + 0.5)
		s := formatLatency(pingMs)
		result.PingMs = &ms
		result.Ping = &s
	}
	if downloadOk {
		bps := int(downloadBps + 0.5)
		s := formatSpeed(downloadBps)
		result.DownloadBps = &bps
		result.Download = &s
	}
	if uploadOk {
		bps := int(uploadBps + 0.5)
		s := formatSpeed(uploadBps)
		result.UploadBps = &bps
		result.Upload = &s
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
	return 0
}

// ---------------------------------------------------------------------------------------
// runClient is the top-level entry point for the client subcommand. It normalises the
// server URL and dispatches to the appropriate output mode (normal, compact, or JSON).
func runClient(serverURL string, compact bool, jsonOutput bool) int {
	serverURL = normaliseURL(serverURL)

	if jsonOutput {
		return runClientJSON(serverURL)
	}
	if compact {
		return runClientCompact(serverURL)
	}
	return runClientNormal(serverURL)
}
