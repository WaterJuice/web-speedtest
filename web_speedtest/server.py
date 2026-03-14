# ----------------------------------------------------------------------------------------
#   server.py
#   ---------
#
#   HTTP server for speed testing. Serves a web UI for browser-based testing and
#   provides API endpoints for the CLI client. Endpoints:
#
#     GET  /              — web UI (HTML page with embedded CSS/JS)
#     GET  /api/ping      — returns a small JSON payload for latency measurement
#     GET  /api/download  — streams random bytes for download speed measurement
#     POST /api/upload    — accepts a body of data for upload speed measurement
#     GET  /api/info      — returns server metadata (version, etc.)
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

import html as html_mod
import json
import os
import time
from functools import partial
from http.server import BaseHTTPRequestHandler
from http.server import ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs
from urllib.parse import urlparse
from .colour import bold
from .colour import cyan
from .colour import dim
from .colour import green
from .version import VERSION_STR

# ----------------------------------------------------------------------------------------
#   Constants
# ----------------------------------------------------------------------------------------

# Default test duration for download/upload phases (seconds)
_DEFAULT_TEST_DURATION = 8

# Maximum download size per request: 100 MB (client loops until time is up)
_MAX_DOWNLOAD_SIZE = 100 * 1024 * 1024

# Chunk size for streaming: 64 KB
_CHUNK_SIZE = 65536

# Pre-generated random chunk for download (avoids generating randomness per-request)
_RANDOM_CHUNK: bytes = os.urandom(_CHUNK_SIZE)

# Static file directory
_STATIC_DIR = Path(__file__).parent / "static"

# ----------------------------------------------------------------------------------------
#   Request Handler
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
class SpeedTestHandler(BaseHTTPRequestHandler):
    """HTTP request handler for speed test server."""

    server_version = f"web-speedtest/{VERSION_STR}"

    # ------------------------------------------------------------------------------------
    def log_message(self, format: str, *args: object) -> None:  # noqa: A002  # pyright: ignore[reportUnusedParameter]
        """Override to use our own logging format."""
        server: SpeedTestServer = self.server  # pyright: ignore[reportAssignmentType]
        if server.verbose:
            timestamp = time.strftime("%H:%M:%S")
            print(dim(f"  [{timestamp}] {args[0]}"))

    # ------------------------------------------------------------------------------------
    def _send_json(self, data: dict[str, object], status: int = 200) -> None:
        """Send a JSON response."""
        body = json.dumps(data).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(body)

    # ------------------------------------------------------------------------------------
    def _send_cors_headers(self) -> None:
        """Send CORS headers for preflight requests."""
        self.send_response(204)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.send_header("Access-Control-Expose-Headers", "Server-Timing")
        self.send_header("Access-Control-Max-Age", "86400")
        self.end_headers()

    # ------------------------------------------------------------------------------------
    def do_OPTIONS(self) -> None:
        """Handle CORS preflight requests."""
        self._send_cors_headers()

    # ------------------------------------------------------------------------------------
    def do_GET(self) -> None:
        """Route GET requests."""
        parsed = urlparse(self.path)
        path = parsed.path.rstrip("/") or "/"

        if path == "/":
            self._handle_index()
        elif path == "/api/ping":
            self._handle_ping()
        elif path == "/api/download":
            self._handle_download(parsed.query)
        elif path == "/api/info":
            self._handle_info()
        else:
            self._handle_static(path)

    # ------------------------------------------------------------------------------------
    def do_POST(self) -> None:
        """Route POST requests."""
        parsed = urlparse(self.path)
        path = parsed.path.rstrip("/")

        if path == "/api/upload":
            self._handle_upload()
        else:
            self._send_json({"error": "Not found"}, status=404)

    # ------------------------------------------------------------------------------------
    def _handle_index(self) -> None:
        """Serve the web UI, injecting the server name."""
        index_path = _STATIC_DIR / "index.html"
        if not index_path.exists():
            self._send_json({"error": "Web UI not found"}, status=500)
            return

        server: SpeedTestServer = self.server  # pyright: ignore[reportAssignmentType]
        html = index_path.read_text(encoding="utf-8")
        # Inject the server name and version into the page
        html = html.replace(
            "{{SERVER_NAME}}", html_mod.escape(server.server_name_label)
        )
        html = html.replace("{{VERSION}}", html_mod.escape(VERSION_STR))
        body = html.encode("utf-8")

        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(body)

    # ------------------------------------------------------------------------------------
    def _handle_static(self, path: str) -> None:
        """Serve static files from the static directory."""
        # Sanitise path to prevent directory traversal
        clean = Path(path.lstrip("/"))
        if ".." in clean.parts:
            self._send_json({"error": "Forbidden"}, status=403)
            return

        file_path = _STATIC_DIR / clean
        if not file_path.is_file():
            self._send_json({"error": "Not found"}, status=404)
            return

        content_types: dict[str, str] = {
            ".html": "text/html; charset=utf-8",
            ".css": "text/css; charset=utf-8",
            ".js": "application/javascript; charset=utf-8",
            ".png": "image/png",
            ".svg": "image/svg+xml",
            ".ico": "image/x-icon",
        }
        ext = file_path.suffix.lower()
        content_type = content_types.get(ext, "application/octet-stream")

        body = file_path.read_bytes()
        self.send_response(200)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    # ------------------------------------------------------------------------------------
    def _handle_ping(self) -> None:
        """Return a small JSON payload for latency measurement.

        Includes a Server-Timing header so the client can subtract server
        processing time from the round-trip to get a more accurate network
        latency figure.
        """
        t0 = time.monotonic()
        data = json.dumps({"timestamp": time.time()}).encode("utf-8")
        processing_ms = (time.monotonic() - t0) * 1000

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Expose-Headers", "Server-Timing")
        self.send_header("Cache-Control", "no-store")
        self.send_header("Server-Timing", f"processing;dur={processing_ms:.2f}")
        self.end_headers()
        self.wfile.write(data)

    # ------------------------------------------------------------------------------------
    def _handle_download(self, query: str) -> None:
        """Stream random data for download speed measurement."""
        params = parse_qs(query)
        size_str = params.get("size", [str(_MAX_DOWNLOAD_SIZE)])[0]
        try:
            size = int(size_str)
        except ValueError:
            size = _MAX_DOWNLOAD_SIZE

        size = max(1, min(size, _MAX_DOWNLOAD_SIZE))

        self.send_response(200)
        self.send_header("Content-Type", "application/octet-stream")
        self.send_header("Content-Length", str(size))
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Cache-Control", "no-store")
        self.end_headers()

        remaining = size
        while remaining > 0:
            chunk_len = min(_CHUNK_SIZE, remaining)
            try:
                if chunk_len == _CHUNK_SIZE:
                    self.wfile.write(_RANDOM_CHUNK)
                else:
                    self.wfile.write(_RANDOM_CHUNK[:chunk_len])
                remaining -= chunk_len
            except (BrokenPipeError, ConnectionResetError):
                return

    # ------------------------------------------------------------------------------------
    def _handle_upload(self) -> None:
        """Accept uploaded data and return the size received."""
        content_length_str = self.headers.get("Content-Length", "0")
        try:
            content_length = int(content_length_str)
        except ValueError:
            content_length = 0

        start = time.time()
        received = 0
        try:
            while received < content_length:
                chunk_len = min(_CHUNK_SIZE, content_length - received)
                data = self.rfile.read(chunk_len)
                if not data:
                    break
                received += len(data)
        except (BrokenPipeError, ConnectionResetError):
            return
        elapsed = time.time() - start

        try:
            self._send_json(
                {
                    "size": received,
                    "duration": elapsed,
                    "timestamp": time.time(),
                }
            )
        except (BrokenPipeError, ConnectionResetError):
            return

    # ------------------------------------------------------------------------------------
    def _handle_info(self) -> None:
        """Return server metadata."""
        server: SpeedTestServer = self.server  # pyright: ignore[reportAssignmentType]
        self._send_json(
            {
                "version": VERSION_STR,
                "server": "web-speedtest",
                "name": server.server_name_label,
                "test_duration": server.test_duration,
            }
        )


# ----------------------------------------------------------------------------------------
#   Server
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
class SpeedTestServer(ThreadingHTTPServer):
    """Threaded HTTP server with verbose flag and optional name."""

    daemon_threads = True

    verbose: bool = True
    server_name_label: str = "Speed Test"
    test_duration: int = _DEFAULT_TEST_DURATION


# ----------------------------------------------------------------------------------------
def run_server(
    host: str,
    port: int,
    verbose: bool,
    name: str = "Speed Test",
    test_duration: int = _DEFAULT_TEST_DURATION,
) -> int:
    """Start the speed test server."""
    server = SpeedTestServer((host, port), partial(SpeedTestHandler))  # pyright: ignore[reportArgumentType]
    server.verbose = verbose
    server.server_name_label = name
    server.test_duration = test_duration

    print(bold("web-speedtest server"))
    print(dim(f"  version:  {VERSION_STR}"))
    print(dim(f"  name:     {name}"))
    print(dim(f"  address:  {host}:{port}"))
    print(dim(f"  duration: {test_duration}s per phase"))
    print()
    if host in ("0.0.0.0", "::"):  # noqa: S104
        print(
            green("  Listening on port ") + cyan(str(port)) + green(" (all interfaces)")
        )
    else:
        print(green("  Listening on ") + cyan(f"http://{host}:{port}/"))
    print(dim("  Press Ctrl+C to stop"))
    print()

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print()
        print(dim("Shutting down..."))
        server.shutdown()

    return 0
