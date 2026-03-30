// ---------------------------------------------------------------------------------------
//
//	server.go
//	---------
//
//	HTTP server for speed testing, built on net/http with hand-rolled WebSocket
//	support (RFC 6455). Serves a web UI for browser-based testing and provides API
//	endpoints for the CLI client. The browser uses WebSocket for low-overhead ping
//	measurement while download/upload use standard HTTP.
//
//	Endpoints:
//	  GET  /              — web UI (HTML page with embedded CSS/JS)
//	  GET  /api/ping      — small JSON payload for latency (CLI client)
//	  GET  /api/download  — streams random bytes for download speed measurement
//	  POST /api/upload    — accepts a body of data for upload speed measurement
//	  GET  /api/info      — server metadata (version, etc.)
//	  WS   /ws            — WebSocket endpoint for browser ping measurement
//
//	(c) 2026 WaterJuice — Released under the Unlicense; see LICENSE.
//
//	Version History
//	---------------
//	Mar 2026 - Created (Python asyncio)
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
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ---------------------------------------------------------------------------------------
//
//	Embedded Static Files
//
// ---------------------------------------------------------------------------------------

//go:embed static/index.html
var indexHTML string

// ---------------------------------------------------------------------------------------
//
//	Constants
//
// ---------------------------------------------------------------------------------------

const (
	// Default test duration for download/upload phases (seconds)
	defaultTestDuration = 8

	// Maximum download size per request: 100 MB
	maxDownloadSize = 100 * 1024 * 1024

	// Chunk size for streaming: 64 KB
	serverChunkSize = 65536
)

// Pre-generated random chunk for download (avoids generating randomness per-request)
var randomChunk [serverChunkSize]byte

// init pre-generates a random 64 KB chunk at startup. This chunk is reused for
// every download response, avoiding per-request entropy generation costs.
func init() {
	rand.Read(randomChunk[:])
}

// ---------------------------------------------------------------------------------------
//
//	Server
//
// ---------------------------------------------------------------------------------------

// speedTestServer is the core HTTP handler. It holds server configuration and
// implements http.Handler to route requests to the appropriate endpoint.
type speedTestServer struct {
	name         string
	testDuration int
	verbose      bool
	version      string
	isTTY        bool
}

// ---------------------------------------------------------------------------------------
// log prints a timestamped message to stdout when verbose mode is enabled.
// Output is dimmed with ANSI codes when connected to a TTY.
func (s *speedTestServer) log(format string, args ...any) {
	if !s.verbose {
		return
	}
	ts := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)
	if s.isTTY {
		fmt.Printf("  \033[2m[%s] %s\033[0m\n", ts, msg)
	} else {
		fmt.Printf("  [%s] %s\n", ts, msg)
	}
}

// ---------------------------------------------------------------------------------------
//
//	HTTP Handler
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// ServeHTTP is the main HTTP request router. It normalises the path, logs the
// request, and dispatches to the appropriate handler based on method and path.
// WebSocket upgrade requests to /ws are handled before normal HTTP routing.
func (s *speedTestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimRight(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}

	s.log("%s %s", r.Method, path)

	// WebSocket upgrade
	if path == "/ws" && strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		s.handleWebSocket(w, r)
		return
	}

	// CORS preflight
	if r.Method == "OPTIONS" {
		s.handleCORSPreflight(w)
		return
	}

	switch {
	case r.Method == "GET" && path == "/":
		s.handleIndex(w)
	case r.Method == "GET" && path == "/api/ping":
		s.handlePing(w)
	case r.Method == "GET" && path == "/api/download":
		s.handleDownload(w, r)
	case r.Method == "GET" && path == "/api/info":
		s.handleInfo(w)
	case r.Method == "POST" && path == "/api/upload":
		s.handleUpload(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Not found"}`))
	}
}

// ---------------------------------------------------------------------------------------
//
//	Route Handlers
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// handleIndex serves the web UI at GET /. The embedded HTML template has its
// {{SERVER_NAME}} and {{VERSION}} placeholders replaced with the configured values.
func (s *speedTestServer) handleIndex(w http.ResponseWriter) {
	content := strings.ReplaceAll(indexHTML, "{{SERVER_NAME}}", html.EscapeString(s.name))
	content = strings.ReplaceAll(content, "{{VERSION}}", html.EscapeString(s.version))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(content))
}

// ---------------------------------------------------------------------------------------
// handlePing returns a small JSON payload at GET /api/ping for latency measurement.
// Includes a Server-Timing header so the CLI client can subtract server processing
// time from the round-trip to get a more accurate network latency figure.
func (s *speedTestServer) handlePing(w http.ResponseWriter) {
	t0 := time.Now()
	data := map[string]any{"timestamp": float64(time.Now().UnixMilli()) / 1000}
	body, _ := json.Marshal(data)
	processingMs := float64(time.Since(t0).Microseconds()) / 1000

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Server-Timing")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Server-Timing", fmt.Sprintf("processing;dur=%.2f", processingMs))
	w.Write(body)
}

// ---------------------------------------------------------------------------------------
// handleDownload streams random data at GET /api/download for download speed measurement.
// The ?size=N query parameter controls the number of bytes (clamped to 1..100 MB).
// Data is written in 64 KB chunks from a pre-generated random buffer, with periodic
// flushes to avoid excessive server-side buffering.
func (s *speedTestServer) handleDownload(w http.ResponseWriter, r *http.Request) {
	sizeStr := r.URL.Query().Get("size")
	size := maxDownloadSize
	if sizeStr != "" {
		if n, err := strconv.Atoi(sizeStr); err == nil {
			size = n
		}
	}
	if size < 1 {
		size = 1
	}
	if size > maxDownloadSize {
		size = maxDownloadSize
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(size))
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-store")

	flusher, canFlush := w.(http.Flusher)
	remaining := size
	for remaining > 0 {
		n := serverChunkSize
		if n > remaining {
			n = remaining
		}
		_, err := w.Write(randomChunk[:n])
		if err != nil {
			return
		}
		remaining -= n
		if canFlush && remaining%(16*serverChunkSize) == 0 {
			flusher.Flush()
		}
	}
}

// ---------------------------------------------------------------------------------------
// handleUpload accepts uploaded data at POST /api/upload and returns the number of
// bytes received and the elapsed time as JSON. Used by both the web UI and CLI client
// to measure upload throughput.
func (s *speedTestServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	received := 0
	buf := make([]byte, serverChunkSize)
	for {
		n, err := r.Body.Read(buf)
		received += n
		if err != nil {
			break
		}
	}
	elapsed := time.Since(start).Seconds()

	s.sendJSON(w, http.StatusOK, map[string]any{
		"size":      received,
		"duration":  elapsed,
		"timestamp": float64(time.Now().UnixMilli()) / 1000,
	})
}

// ---------------------------------------------------------------------------------------
// handleInfo returns server metadata as JSON at GET /api/info, including the version
// string, server name, and configured test duration. The web UI fetches this on
// startup to learn the server's test duration.
func (s *speedTestServer) handleInfo(w http.ResponseWriter) {
	s.sendJSON(w, http.StatusOK, map[string]any{
		"version":       s.version,
		"server":        "web-speedtest",
		"name":          s.name,
		"test_duration": s.testDuration,
	})
}

// ---------------------------------------------------------------------------------------
// handleCORSPreflight responds to OPTIONS requests with permissive CORS headers,
// allowing the web UI to make cross-origin API requests from any domain.
func (s *speedTestServer) handleCORSPreflight(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Expose-Headers", "Server-Timing")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------------------
//
//	WebSocket Handler
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// handleWebSocket upgrades the connection at /ws to WebSocket and handles the
// low-latency ping/pong loop. The browser sends "ping" text frames and the server
// immediately responds with "pong", avoiding per-request HTTP overhead for more
// accurate latency measurement.
func (s *speedTestServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	bufrw, conn, err := wsUpgrade(w, r)
	if err != nil {
		return
	}
	defer conn.Close()

	s.log("WebSocket /ws connected")
	defer s.log("WebSocket /ws disconnected")

	for {
		opcode, payload, err := wsReadFrame(bufrw)
		if err != nil {
			return
		}

		switch opcode {
		case opClose:
			conn.Write(wsEncodeFrame(opClose, nil))
			return
		case opPing:
			conn.Write(wsEncodeFrame(opPong, payload))
		case opText:
			if string(payload) == "ping" {
				conn.Write(wsEncodeFrame(opText, []byte("pong")))
			}
		}
	}
}

// ---------------------------------------------------------------------------------------
//
//	Helpers
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// sendJSON marshals data to JSON and writes it as an HTTP response with standard
// CORS and cache-control headers.
func (s *speedTestServer) sendJSON(w http.ResponseWriter, status int, data map[string]any) {
	body, _ := json.Marshal(data)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	w.Write(body)
}

// ---------------------------------------------------------------------------------------
//
//	Server Entry Point
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// startServer creates and starts the speed test HTTP server. It prints a startup
// banner, binds to the given host:port, and blocks until a SIGINT/SIGTERM is
// received. Returns 0 on clean shutdown, 1 on error.
func startServer(version string, host string, port int, name string, testDuration int, verbose bool) int {
	isTTY := isStdoutTerminal()

	srv := &speedTestServer{
		name:         name,
		testDuration: testDuration,
		verbose:      verbose,
		version:      version,
		isTTY:        isTTY,
	}

	// Print startup banner
	fmt.Println(colorBold("web-speedtest server", isTTY))
	fmt.Println(colorDim(fmt.Sprintf("  version:  %s", version), isTTY))
	fmt.Println(colorDim(fmt.Sprintf("  name:     %s", name), isTTY))
	fmt.Println(colorDim(fmt.Sprintf("  address:  %s:%d", host, port), isTTY))
	fmt.Println(colorDim(fmt.Sprintf("  duration: %ds per phase", testDuration), isTTY))
	fmt.Println()
	if host == "0.0.0.0" || host == "::" {
		fmt.Println(colorGreen("  Listening on port ", isTTY) +
			colorCyan(strconv.Itoa(port), isTTY) +
			colorGreen(" (all interfaces)", isTTY))
	} else {
		fmt.Println(colorGreen("  Listening on ", isTTY) +
			colorCyan(fmt.Sprintf("http://%s:%d/", host, port), isTTY))
	}
	fmt.Println(colorDim("  Press Ctrl+C to stop", isTTY))
	fmt.Println()

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	httpServer := &http.Server{
		Addr:    addr,
		Handler: srv,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println()
		fmt.Println(colorDim("Shutting down...", isTTY))
		httpServer.Close()
	}()

	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Server error: %s\n", err)
		return 1
	}

	return 0
}
