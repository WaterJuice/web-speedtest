# Usage

web-speedtest provides two subcommands: `server` and `client`.

## `server` — Start the speed test server

```bash
web-speedtest server
web-speedtest server --port 3000
web-speedtest server --host 127.0.0.1 --port 8080
web-speedtest server --quiet
```

Starts an HTTP server that serves the web UI and API endpoints. Open the displayed URL in a browser to run a speed test, or use the CLI client to test from the terminal.

| Option          | Description                                     |
|-----------------|-------------------------------------------------|
| `--host`, `-H`  | Address to bind to (default: `0.0.0.0`)        |
| `--port`, `-p`  | Port to listen on (default: `8080`)            |
| `--quiet`, `-q` | Suppress request logging                        |

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

Streams random bytes for download speed measurement. The `size` parameter specifies the number of bytes to send (default: 25 MB, maximum: 100 MB).

#### `POST /api/upload`

Accepts a body of data and returns the size received and duration.

```json
{"size": 10485760, "duration": 0.523, "timestamp": 1709912345.678}
```

#### `GET /api/info`

Returns server metadata.

```json
{"version": "1.0.0", "server": "web-speedtest"}
```

All API endpoints include `Access-Control-Allow-Origin: *` for cross-origin browser requests.

## `client` — Run a CLI speed test

```bash
web-speedtest client localhost:8080
web-speedtest client http://speedtest.example.com:8080
```

Connects to a web-speedtest server and measures ping, download, and upload speeds from the terminal. Results are displayed with coloured output and a summary.

The client runs three tests in sequence:

1. **Ping** — sends 10 small requests and reports the average round-trip latency
2. **Download** — streams 25 MB from the server and reports throughput in Mbps
3. **Upload** — sends 10 MB to the server and reports throughput in Mbps

Progress is displayed in real-time with a progress bar during the download test.

| Argument  | Description                                                            |
|-----------|------------------------------------------------------------------------|
| `SERVER`  | Server URL or host:port (e.g. `localhost:8080` or `http://host:8080`) |

If the URL scheme is omitted, `http://` is assumed.

## Global Options

| Option       | Description                         |
|--------------|-------------------------------------|
| `--version`  | Show version and exit               |
| `--license`  | Show licence information and exit   |
| `--help`     | Show help and exit                  |
