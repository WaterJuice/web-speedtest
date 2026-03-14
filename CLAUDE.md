# CLAUDE.md

This file provides guidance for AI agents working on this project.

## Project Overview

web-speedtest is a network speed test tool with two modes:

- **Server mode** — an HTTP server that provides a beautiful web UI for browser-based speed testing (ping, download, upload) and API endpoints for programmatic access
- **Client mode** — a CLI tool that connects to a web-speedtest server and measures ping, download, and upload speeds from the terminal

## Language and Spelling

Use **Australian English** throughout:
- colour (not color)
- initialise (not initialize)
- sanitise (not sanitize)
- organisation (not organization)

## Code Style

### Python Files

Every Python file should have:
1. A file header block with description and version history
2. Section headers separating major sections (Imports, Constants, Functions, etc.)
3. Horizontal separators (92 chars of `-`) above each function definition

Example structure:
```python
# ----------------------------------------------------------------------------------------
#   filename.py
#   -----------
#
#   Brief description of what this module does.
#
#   (c) 2026 WaterJuice — Released under the Unlicense; see LICENSE.
#
#   Version History
#   ---------------
#   Mar 2026 - Created
# ----------------------------------------------------------------------------------------

# ----------------------------------------------------------------------------------------
#   Imports
# ----------------------------------------------------------------------------------------

import sys

# ----------------------------------------------------------------------------------------
#   Functions
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
def my_function() -> None:
    """Docstring here."""
    pass
```

### General

- Python 3.12+ (do **not** use `from __future__ import annotations`)
- Use type hints throughout
- Prefer pathlib.Path over os.path
- Single-line imports, no blank lines between import groups (configured in pyproject.toml)
- Run `make format` to auto-fix import ordering
- Zero external dependencies — stdlib only
- CLI uses argbuilder.py (custom argparse wrapper), not click or argparse directly

## Common Commands

```bash
make help       # Show all available targets
make check      # Run ruff + pyright
make format     # Auto-fix and format code
make build      # Build wheel + docs into output/
make docs       # Build HTML documentation into html/
make clean      # Remove build artefacts
make dev        # Just create dev (.venv) setup
```

## Project Structure

```
web_speedtest/
├── __init__.py       # Package init, exports __version__
├── __main__.py       # Entry point for python -m web_speedtest
├── version.py        # Version string handling
├── argbuilder.py     # Custom argparse wrapper (from cal-publish-python)
├── colour.py         # ANSI colour output (TTY-aware)
├── cli.py            # CLI commands and argument parsing
├── server.py         # Async HTTP server (asyncio) with WebSocket support
├── websocket.py      # WebSocket protocol (RFC 6455) — handshake and framing
├── client.py         # CLI speed test client
└── static/
    └── index.html    # Web UI (single-page app with embedded CSS/JS)
```

## Architecture

### Server Mode (`web-speedtest server`)

- Async HTTP server built on `asyncio.start_server` — zero dependencies
- Hand-rolled HTTP request parsing and WebSocket support (RFC 6455)
- Serves a web UI at `/` with a beautiful dark-themed single-page app
- API endpoints:
  - `GET /api/ping` — small JSON response for latency measurement (CLI client)
  - `GET /api/download?size=N` — streams random bytes for download speed
  - `POST /api/upload` — accepts data for upload speed measurement
  - `GET /api/info` — server metadata (version, etc.)
  - `WS /ws` — WebSocket endpoint for low-overhead browser ping measurement
- Pre-generates a random chunk to avoid per-request entropy costs
- Supports HTTP keep-alive and CORS for cross-origin browser requests

### Client Mode (`web-speedtest client`)

- Uses `urllib.request` — zero dependencies
- Connects to any web-speedtest server
- Measures:
  - Ping: average of 10 round-trip samples to `/api/ping`
  - Download: streams 25 MB from `/api/download` and measures throughput
  - Upload: sends 10 MB to `/api/upload` and measures throughput
- Displays live progress with a progress bar
- Coloured terminal output with summary

### Web UI

- Single HTML file with embedded CSS and JavaScript
- Dark theme with animated gauge
- Real-time progress during each test phase
- Uses `ReadableStream` for live download progress
- WebSocket ping with HTTP fallback for environments without WebSocket

## Key Design Decisions

1. **Zero dependencies** — stdlib only (`asyncio`, `urllib.request`, `json`, etc.)
2. **Async server** — `asyncio.start_server` with hand-rolled HTTP parsing and WebSocket
3. **Single HTML file** — web UI is one self-contained file with embedded CSS/JS
4. **WebSocket ping** — persistent connection for accurate latency, with HTTP fallback
5. **Streaming download** — random data streamed in chunks, not generated all at once
6. **Pre-generated random chunk** — single 64 KB chunk reused for download speed
7. **Warmup exclusion** — first 2s of download/upload discarded for steady-state accuracy
8. **argbuilder for CLI** — custom argparse wrapper, consistent with other WaterJuice projects
9. **Dual mode** — same package provides both server and client

## Testing Changes

After making changes:
1. Run `make check` to verify linting and types pass
2. Run `make build` to verify the full build works
3. Test server: `uv run web-speedtest server`
4. Test client: `uv run web-speedtest client localhost:8080`

## Versioning

- Version is derived from git tags via uv-dynamic-versioning
- Create a tag like `1.0.0` before running `make build` for a release (no `v` prefix)
- The build generates `_version.py` at build time, which is not committed
- If no tags exist, version falls back to "dev"

## Commits

When committing:
- Use clear, descriptive commit messages
- Include `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>` in commits made with AI assistance
- **Never rewrite git history** unless explicitly asked to

## Licence

Released under the [Unlicense](https://unlicense.org/) — public domain.
