# web-speedtest 1.0.0 Beta 12 — 21 Apr 2026

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
