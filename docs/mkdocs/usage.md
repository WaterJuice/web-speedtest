# Usage

web-speedtest provides two subcommands: `server` and `client`.

## `server` — Start the speed test server

```bash
web-speedtest server
web-speedtest server --port 3000
web-speedtest server --host 127.0.0.1 --port 8080
web-speedtest server --name "Office Network"
web-speedtest server --quiet
```

Starts an HTTP server that serves the web UI and API endpoints. Open the displayed URL in a browser to run a speed test, or use the CLI client to test from the terminal.

| Option          | Description                                          |
|-----------------|------------------------------------------------------|
| `--host`, `-H`  | Address to bind to (default: `0.0.0.0`)             |
| `--port`, `-p`  | Port to listen on (default: `8080`)                 |
| `--name`, `-n`  | Server display name shown in the web UI (default: `Speed Test`) |
| `--duration`, `-d` | Duration of each test phase in seconds (default: `8`) |
| `--quiet`, `-q` | Suppress request logging                             |

### Web UI

The server serves a browser-based speed test at `/`. The UI features:

- A central animated gauge showing real-time measurements
- Automatic progression through ping, download, and upload tests
- Results displayed in a clean summary panel
- Dark theme optimised for readability

### API Endpoints

The server provides these endpoints for programmatic access:

#### `GET /api/ping`

Returns a small JSON payload for latency measurement.

```json
{"timestamp": 1709912345.678}
```

#### `GET /api/download?size=N`

Streams random bytes for download speed measurement. The `size` parameter specifies the number of bytes to send (default: 100 MB, maximum: 100 MB).

#### `POST /api/upload`

Accepts a body of data and returns the size received and duration.

```json
{"size": 10485760, "duration": 0.523, "timestamp": 1709912345.678}
```

#### `GET /api/info`

Returns server metadata.

```json
{"version": "1.0.0", "server": "web-speedtest", "name": "Speed Test", "test_duration": 8}
```

All API endpoints include `Access-Control-Allow-Origin: *` for cross-origin browser requests.

## `client` — Run a CLI speed test

```bash
web-speedtest client localhost:8080
web-speedtest client http://speedtest.example.com:8080
web-speedtest client localhost:8080 --compact
web-speedtest client localhost:8080 --json
```

Connects to a web-speedtest server and measures ping, download, and upload speeds from the terminal. Results are displayed with coloured output and a summary.

The `client` command is the default — you can omit the subcommand name:

```bash
web-speedtest localhost:8080
```

The client runs three tests in sequence:

1. **Ping** — sends 10 small requests and reports the average round-trip latency
2. **Download** — streams 25 MB from the server using 6 parallel streams and reports throughput
3. **Upload** — sends 10 MB to the server using 6 parallel streams and reports throughput

Progress is displayed in real-time with a progress bar during the download test.

| Argument / Option   | Description                                                            |
|---------------------|------------------------------------------------------------------------|
| `SERVER`            | Server URL or host:port (e.g. `localhost:8080` or `http://host:8080`) |
| `--compact`, `-c`   | Compact one-line output                                               |
| `--json`, `-j`      | Output results as JSON                                                |

If the URL scheme is omitted, `http://` is assumed.

### Output Modes

**Normal** (default) — full interactive output with progress bar and coloured summary.

**Compact** (`--compact`) — single-line output suitable for scripting:

```
ping 4.2 ms / down 523.46 Mbps / up 312.35 Mbps
```

**JSON** (`--json`) — machine-readable output with both raw values and human-readable strings:

```json
{
  "server": "http://localhost:8080",
  "ping_ms": 4,
  "ping": "4.2 ms",
  "download_bps": 548798259,
  "download": "548.80 Mbps",
  "upload_bps": 327516160,
  "upload": "327.52 Mbps"
}
```

## Global Options

| Option       | Description                         |
|--------------|-------------------------------------|
| `--version`  | Show version and exit               |
| `--license`  | Show licence information and exit   |
| `--help`     | Show help and exit                  |
