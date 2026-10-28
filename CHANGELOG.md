# web-speedtest 1.1.0 — 28 Oct 2026

- Built with Go 1.27.3 to pick up the fix for the eMudura TLS issue.
- `go.mod` now requires Go 1.27.3 (was 1.25.0), so builds from source can't fall back to an older toolchain without the fix. This also brings in Go 1.27's default runtime (`GODEBUG`) settings, including its current TLS defaults.

# web-speedtest 1.0.1 — 30 May 2026

Moved to new GitHub location: https://github.com/WaterJuice/web-speedtest

# web-speedtest 1.0.0 — 23 Apr 2026

- Initial release
- Server mode — HTTP server with embedded web UI for browser-based speed testing
- Client mode — CLI tool that measures ping, download, and upload from any web-speedtest server
- Web UI with animated gauges, dark theme, and live progress
- Copy test results from the web UI to the clipboard as text or as a PNG image
- WebSocket ping for low-overhead latency measurement, with HTTP fallback
- Server-Timing header on the HTTP ping endpoint so the CLI client can subtract server processing time
- Warmup exclusion (first 2 seconds discarded) for download/upload accuracy
- Median-based ping reporting for outlier robustness
- Parallel streams (6) for both download and upload tests
- Hand-rolled WebSocket implementation (RFC 6455) — stdlib only
- API endpoints: `/api/ping`, `/api/download`, `/api/upload`, `/api/info`, `/ws`
- Beta ribbon shown automatically on pre-release builds, hidden on full `X.Y.Z` releases
- Single statically-linked Go binary with zero runtime dependencies
- Distributed as platform-specific Python wheels via PyPI (using bin2whl)
- Cross-platform: macOS, Linux, and Windows on both arm64 and amd64
