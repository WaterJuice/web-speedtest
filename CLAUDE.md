# CLAUDE.md

This file provides guidance for AI agents working on this project.

## Project Overview

web-speedtest is a network speed test tool with two modes:

- **Server mode** — an HTTP server that provides a beautiful web UI for browser-based speed testing (ping, download, upload) and API endpoints for programmatic access
- **Client mode** — a CLI tool that connects to a web-speedtest server and measures ping, download, and upload speeds from the terminal

The project is written in Go and distributed as platform-specific Python wheels via PyPI (using bin2whl). This means users install it with `pip install web-speedtest` or `uvx web-speedtest`, but the binary is a statically-linked Go executable — no Python runtime required at execution time.

## Language and Spelling

Use **Australian English** throughout:
- colour (not color)
- initialise (not initialize)
- sanitise (not sanitize)
- organisation (not organization)

## Code Style

### Go Files

Every Go file should have:
1. A file header block with description and version history
2. Section headers separating major sections (Imports, Constants, Functions, etc.)
3. Horizontal separators (87 dashes after `// `, 90 chars total) above each function definition

Example structure:
```go
// ---------------------------------------------------------------------------------------
//
//	filename.go
//	-----------
//
//	Brief description of what this module does.
//
//	(c) 2026 WaterJuice — Released under the Unlicense; see LICENSE.
//
//	Version History
//	---------------
//	Mar 2026 - Created
//
// ---------------------------------------------------------------------------------------
package internal

// ---------------------------------------------------------------------------------------
//
//	Imports
//
// ---------------------------------------------------------------------------------------

import (
	"fmt"
)

// ---------------------------------------------------------------------------------------
//
//	Functions
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// MyFunction does something.
func MyFunction() {
}
```

### General

- Go 1.25+
- Zero external dependencies — stdlib only
- Use `gofmt` for formatting, `go vet` for linting
- Run `make format` to auto-fix formatting
- Run `make check` to verify formatting and lint
- CLI uses manual argument parsing (no flag package, no external CLI libs)
- TTY-aware ANSI colours for terminal output

## Common Commands

```bash
make help       # Show all available targets
make check      # Run gofmt check + go vet
make format     # Auto-format Go source with gofmt
make go-build   # Cross-compile for all 6 platforms
make build      # Full build: check, go-build, docs, platform wheels
make docs       # Build HTML documentation into html/
make clean      # Remove build artefacts
make dev        # Build for current platform + symlink into .venv/bin/
make run ARGS="server --port 3000"  # Build and run with arguments
```

## Project Structure

```
main.go                 # Entry point, calls internal.Run(Version)
go.mod                  # Go module definition
internal/
├── cli.go              # CLI argument parsing, help text, colour helpers
├── server.go           # HTTP server with embedded web UI and WebSocket
├── client.go           # CLI speed test client
├── websocket.go        # WebSocket protocol (RFC 6455) — handshake and framing
└── static/
    └── index.html      # Web UI (single-page app with embedded CSS/JS)
wheel.json              # bin2whl configuration for platform wheels
pyproject.toml          # Minimal — just for uv dev dependencies
Makefile                # Build orchestration
```

## Architecture

### Server Mode (`web-speedtest server`)

- HTTP server built on `net/http` — zero dependencies
- Hand-rolled WebSocket support via `http.Hijacker` (RFC 6455)
- Web UI embedded at compile time using `//go:embed`
- Serves a web UI at `/` with a beautiful dark-themed single-page app
- API endpoints:
  - `GET /api/ping` — small JSON response for latency measurement (CLI client)
  - `GET /api/download?size=N` — streams random bytes for download speed
  - `POST /api/upload` — accepts data for upload speed measurement
  - `GET /api/info` — server metadata (version, etc.)
  - `WS /ws` — WebSocket endpoint for low-overhead browser ping measurement
- Pre-generates a random chunk to avoid per-request entropy costs
- Supports CORS for cross-origin browser requests

### Client Mode (`web-speedtest client`)

- Uses `net/http` — zero dependencies
- Connects to any web-speedtest server
- Measures:
  - Ping: average of 10 round-trip samples to `/api/ping`
  - Download: parallel streams from `/api/download` measuring throughput
  - Upload: parallel streams to `/api/upload` measuring throughput
- Displays live progress with a progress bar (download)
- Coloured terminal output with summary

### Web UI

- Single HTML file with embedded CSS and JavaScript
- Dark theme with animated SVG gauges
- Real-time progress during each test phase
- Uses WebSocket for ping with HTTP fallback
- Embedded in the Go binary via `//go:embed`

## Key Design Decisions

1. **Go binary, PyPI distribution** — statically-linked Go binary wrapped in platform wheels via bin2whl
2. **Zero dependencies** — Go stdlib only
3. **net/http server** — idiomatic Go HTTP server with WebSocket via Hijack
4. **Single HTML file** — web UI is one self-contained file with embedded CSS/JS
5. **go:embed** — HTML compiled into the binary, no external files needed at runtime
6. **WebSocket ping** — persistent connection for accurate latency, with HTTP fallback
7. **Streaming download** — random data streamed in chunks, not generated all at once
8. **Pre-generated random chunk** — single 64 KB chunk reused for download speed
9. **Manual CLI parsing** — no flag package, consistent with tls-switch project conventions
10. **Dual mode** — same binary provides both server and client

## Build & Distribution

- Cross-compiled for 6 platforms: macOS (arm64/amd64), Linux (arm64/amd64), Windows (arm64/amd64)
- All binaries are statically linked (`CGO_ENABLED=0`)
- Version injected at build time via `-ldflags -X main.Version=...`
- Platform wheels built using `bin2whl` from `wheel.json` config
- Published to PyPI via `cal-publish-python`

### Platform Wheel Tags

| Platform       | Wheel Tag                       |
|---------------|---------------------------------|
| macOS arm64   | `macosx_11_0_arm64`           |
| macOS amd64   | `macosx_10_12_x86_64`         |
| Linux amd64   | `manylinux_2_17_x86_64`       |
| Linux arm64   | `manylinux_2_17_aarch64`      |
| Windows amd64 | `win_amd64`                    |
| Windows arm64 | `win_arm64`                    |

## Testing Changes

After making changes:
1. Run `make check` to verify formatting and vet pass
2. Run `make go-build` to verify cross-compilation works
3. Test server: `make run ARGS="server"`
4. Test client: `make run ARGS="client localhost:8080"`

## Versioning

- Version is derived from git tags via `git describe --tags --always`
- Create a tag like `1.0.0` before running `make build` for a release (no `v` prefix)
- Version is injected at build time via `-ldflags`
- Falls back to "dev" if no tags exist

## Commits

When committing:
- Use clear, descriptive commit messages
- Include `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>` in commits made with AI assistance
- **Never rewrite git history** unless explicitly asked to

## Licence

Released under the [Unlicense](https://unlicense.org/) — public domain.
