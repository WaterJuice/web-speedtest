# web-speedtest

A network speed test with a web UI and CLI client — measure ping, download, and upload speeds.

## Why?

Running a quick network speed test usually means visiting a third-party website full of ads, or installing a heavy CLI tool with many dependencies. web-speedtest gives you a lightweight, self-hosted alternative — start the server on any machine, then test from a browser or the command line.

## Features

- **Server mode** — HTTP server with a web UI for browser-based speed testing
- **Client mode** — CLI tool for terminal-based speed measurements
- **Ping test** — measure round-trip latency
- **Download test** — measure download throughput (configurable size)
- **Upload test** — measure upload throughput
- **Web UI** — dark-themed single-page app with animated gauge and live progress
- **API endpoints** — programmatic access for custom integrations
- **Zero dependencies** — stdlib only
- **Cross-platform** — works on macOS, Linux, and Windows

## Requirements

- Python 3.12+

## Quick Start

### Install

```bash
pip install web-speedtest
```

Or run directly with uv:

```bash
uvx web-speedtest
```

### Start the server

```bash
web-speedtest server
```

Then open `http://localhost:8080` in your browser.

### Run a CLI speed test

```bash
web-speedtest client localhost:8080
```

See the [Usage](usage.md) page for full details on all commands and API endpoints.

## How It Works

The server uses Python's built-in `http.server` module to serve both a web UI and API endpoints. The web UI is a single-page app with embedded CSS and JavaScript — no build step or frontend dependencies needed.

Speed tests work by:

1. **Ping** — sending small requests and measuring round-trip time
2. **Download** — streaming random data from the server and measuring throughput
3. **Upload** — sending random data to the server and measuring throughput

The CLI client uses `urllib.request` to hit the same API endpoints, providing a terminal-based alternative to the browser UI.
