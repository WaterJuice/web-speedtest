# web-speedtest 1.0.0 Beta 10 — 30 Mar 2026

- Rewritten from Python to Go for single-binary distribution
- Distributed as platform-specific wheels via PyPI (using bin2whl)
- Zero runtime dependencies — single statically-linked binary
- Cross-compiled for macOS (arm64/amd64), Linux (arm64/amd64), Windows (arm64/amd64)
- HTTP server built on Go's net/http with hand-rolled WebSocket (RFC 6455)
- Web UI embedded in binary via go:embed
- All existing functionality preserved: server, client, web UI, API endpoints
- Identical CLI interface and web UI appearance

# web-speedtest 1.0.0 Beta 9 — 16 Mar 2026

- Initial release
- Server mode with web UI for browser-based speed testing
- Client mode for CLI-based speed testing
- Ping, download, and upload measurements
- Zero external dependencies
- Asyncio-based server (replaces http.server/ThreadingHTTPServer)
- WebSocket ping measurement for low-overhead latency (with HTTP fallback)
- Hand-rolled WebSocket implementation (RFC 6455) — stdlib only
- Server-Timing header on HTTP ping endpoint for CLI client accuracy
- Warmup exclusion (first 2s discarded) for download and upload accuracy
- Median-based ping reporting for outlier robustness
- Larger gauge dials in web UI
