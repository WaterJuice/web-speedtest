# web-speedtest

A network speed test with a web UI and CLI client — measure ping, download, and upload speeds.

web-speedtest provides two modes: a server that hosts a beautiful browser-based speed test, and a CLI client for quick terminal-based measurements. Both use the same API, and the entire tool is a single statically-linked binary — zero runtime dependencies.

## Features

- **Server mode** — HTTP server with a web UI for browser-based speed testing
- **Client mode** — CLI tool for terminal-based speed measurements
- **Ping test** — measure round-trip latency
- **Download test** — time-based download throughput measurement (configurable duration)
- **Upload test** — time-based upload throughput measurement
- **Web UI** — dark-themed single-page app with animated gauges and live progress
- **API endpoints** — programmatic access for custom integrations
- **Zero dependencies** — single statically-linked Go binary
- **Coloured output** — TTY-aware ANSI colours for CLI results
- **Cross-platform** — works on macOS, Linux, and Windows

## Installation

```bash
pip install web-speedtest
```

Or run directly with uv:

```bash
uvx web-speedtest
```

## Quick Start

### Start the server

```bash
# Start on default port 8080
web-speedtest server

# Start on a custom port
web-speedtest server --port 3000

# Bind to a specific address
web-speedtest server --host 127.0.0.1 --port 8080

# Set a custom test duration (default is 8 seconds per phase)
web-speedtest server --duration 15
```

Then open `http://localhost:8080` in your browser to run a speed test.

### Run a CLI speed test

```bash
# Test against a local server
web-speedtest client localhost:8080

# Test against a remote server
web-speedtest client http://speedtest.example.com:8080
```

## API Endpoints

The server provides these endpoints for programmatic access:

| Endpoint | Method | Description |
|---|---|---|
| `/api/ping` | GET | Returns a small JSON payload for latency measurement |
| `/api/download?size=N` | GET | Streams N bytes of random data (max 100 MB per request; client loops until time is up) |
| `/api/upload` | POST | Accepts a body of data and returns size/duration |
| `/api/info` | GET | Returns server metadata including `test_duration` (seconds) |

## Development

```bash
# Set up development environment
make dev

# Run linting and format checking
make check

# Auto-format code
make format

# Build wheel and docs
make build
```

## Licence

Released under the [Unlicense](https://unlicense.org/) — public domain.
