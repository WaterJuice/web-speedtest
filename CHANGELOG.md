# web-speedtest 1.0.0 Beta 6 — 14 Mar 2026

- Initial release
- Server mode with web UI for browser-based speed testing
- Client mode for CLI-based speed testing
- Ping, download, and upload measurements
- Zero external dependencies
- Server-Timing header on ping endpoint for more accurate latency measurement
- Warmup exclusion (first 2s discarded) for download and upload accuracy
- Larger upload chunks (10 MB) to reduce per-request overhead
- Median-based ping reporting for outlier robustness
- Larger gauge dials in web UI
