# Unreleased

- Switch download and upload tests from fixed-size to time-based (default 8 seconds per phase)
- Add `--duration` / `-d` server option to configure test phase duration
- Server advertises `test_duration` via `/api/info`; web UI reads and respects it
- Web UI now shows three persistent gauges (ping, download, upload) that reveal progressively

# web-speedtest 1.0.0 Beta 5 — 11 Mar 2026

- Initial release
- Server mode with web UI for browser-based speed testing
- Client mode for CLI-based speed testing
- Ping, download, and upload measurements
- Zero external dependencies
