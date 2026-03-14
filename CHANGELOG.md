# web-speedtest 1.0.0 Beta 8 — 14 Mar 2026

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
- Larger upload chunks (10 MB) to reduce per-request overhead
- Median-based ping reporting for outlier robustness
- Larger gauge dials in web UI
