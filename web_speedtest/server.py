# ----------------------------------------------------------------------------------------
#   server.py
#   ---------
#
#   Async HTTP server for speed testing, built on asyncio with hand-rolled WebSocket
#   support (RFC 6455). Serves a web UI for browser-based testing and provides API
#   endpoints for the CLI client. The browser uses WebSocket for low-overhead ping
#   measurement while download/upload use standard HTTP.
#
#   Endpoints:
#     GET  /              — web UI (HTML page with embedded CSS/JS)
#     GET  /api/ping      — small JSON payload for latency (CLI client)
#     GET  /api/download  — streams random bytes for download speed measurement
#     POST /api/upload    — accepts a body of data for upload speed measurement
#     GET  /api/info      — server metadata (version, etc.)
#     WS   /ws            — WebSocket endpoint for browser ping measurement
#
#   (c) 2026 WaterJuice — Released under the Unlicense; see LICENSE.
#
#   Version History
#   ---------------
#   Mar 2026 - Created
#   Mar 2026 - Rewritten from http.server to asyncio with WebSocket support
# ----------------------------------------------------------------------------------------

# ----------------------------------------------------------------------------------------
#   Imports
# ----------------------------------------------------------------------------------------

import asyncio
import html as html_mod
import json
import os
import time
from dataclasses import dataclass
from pathlib import Path
from urllib.parse import parse_qs
from urllib.parse import urlparse
from .colour import bold
from .colour import cyan
from .colour import dim
from .colour import green
from .version import VERSION_STR
from .websocket import OP_CLOSE
from .websocket import OP_PING
from .websocket import OP_PONG
from .websocket import OP_TEXT
from .websocket import build_upgrade_response
from .websocket import encode_frame
from .websocket import read_frame

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

# HTTP status reason phrases
_STATUS_PHRASES: dict[int, str] = {
    200: "OK",
    204: "No Content",
    403: "Forbidden",
    404: "Not Found",
    500: "Internal Server Error",
}

# Content type map for static files
_CONTENT_TYPES: dict[str, str] = {
    ".html": "text/html; charset=utf-8",
    ".css": "text/css; charset=utf-8",
    ".js": "application/javascript; charset=utf-8",
    ".png": "image/png",
    ".svg": "image/svg+xml",
    ".ico": "image/x-icon",
}

# ----------------------------------------------------------------------------------------
#   Server Configuration
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
@dataclass
class ServerConfig:
    """Configuration passed to all request handlers."""

    name: str = "Speed Test"
    test_duration: int = _DEFAULT_TEST_DURATION
    verbose: bool = True


# ----------------------------------------------------------------------------------------
#   HTTP Request / Response Helpers
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
@dataclass
class HTTPRequest:
    """Parsed HTTP request."""

    method: str
    path: str
    query: str
    headers: dict[str, str]
    version: str


# ----------------------------------------------------------------------------------------
async def _read_request(reader: asyncio.StreamReader) -> HTTPRequest | None:
    """Read and parse an HTTP request. Returns None if the connection closed."""
    try:
        request_line = await reader.readline()
    except (ConnectionError, asyncio.IncompleteReadError):
        return None

    if not request_line or request_line == b"\r\n":
        return None

    try:
        line = request_line.decode("utf-8", errors="replace").strip()
        parts = line.split(" ", 2)
        if len(parts) < 3:
            return None
        method, raw_path, version = parts
    except (ValueError, UnicodeDecodeError):
        return None

    parsed = urlparse(raw_path)
    path = parsed.path.rstrip("/") or "/"
    query = parsed.query

    # Read headers
    headers: dict[str, str] = {}
    while True:
        try:
            header_line = await reader.readline()
        except (ConnectionError, asyncio.IncompleteReadError):
            return None
        if header_line in (b"\r\n", b"\n", b""):
            break
        try:
            decoded = header_line.decode("utf-8", errors="replace").strip()
            if ":" in decoded:
                key, value = decoded.split(":", 1)
                headers[key.strip().lower()] = value.strip()
        except (ValueError, UnicodeDecodeError):
            continue

    return HTTPRequest(
        method=method, path=path, query=query, headers=headers, version=version
    )


# ----------------------------------------------------------------------------------------
async def _send_response(
    writer: asyncio.StreamWriter,
    status: int,
    headers: dict[str, str],
    body: bytes = b"",
) -> None:
    """Write an HTTP response."""
    phrase = _STATUS_PHRASES.get(status, "OK")
    writer.write(f"HTTP/1.1 {status} {phrase}\r\n".encode())
    for key, value in headers.items():
        writer.write(f"{key}: {value}\r\n".encode())
    writer.write(b"\r\n")
    if body:
        writer.write(body)
    await writer.drain()


# ----------------------------------------------------------------------------------------
async def _send_json(
    writer: asyncio.StreamWriter,
    data: dict[str, object],
    status: int = 200,
    extra_headers: dict[str, str] | None = None,
) -> None:
    """Send a JSON response with standard CORS headers."""
    body = json.dumps(data).encode("utf-8")
    headers: dict[str, str] = {
        "Content-Type": "application/json",
        "Content-Length": str(len(body)),
        "Access-Control-Allow-Origin": "*",
        "Cache-Control": "no-store",
    }
    if extra_headers:
        headers.update(extra_headers)
    await _send_response(writer, status, headers, body)


# ----------------------------------------------------------------------------------------
def _log(config: ServerConfig, message: str) -> None:
    """Log a request if verbose mode is enabled."""
    if config.verbose:
        timestamp = time.strftime("%H:%M:%S")
        print(dim(f"  [{timestamp}] {message}"))


# ----------------------------------------------------------------------------------------
#   Route Handlers
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
async def _handle_index(writer: asyncio.StreamWriter, config: ServerConfig) -> None:
    """Serve the web UI, injecting the server name."""
    index_path = _STATIC_DIR / "index.html"
    if not index_path.exists():
        await _send_json(writer, {"error": "Web UI not found"}, status=500)
        return

    html = index_path.read_text(encoding="utf-8")
    html = html.replace("{{SERVER_NAME}}", html_mod.escape(config.name))
    html = html.replace("{{VERSION}}", html_mod.escape(VERSION_STR))
    body = html.encode("utf-8")

    await _send_response(
        writer,
        200,
        {
            "Content-Type": "text/html; charset=utf-8",
            "Content-Length": str(len(body)),
            "Cache-Control": "no-store",
        },
        body,
    )


# ----------------------------------------------------------------------------------------
async def _handle_static(writer: asyncio.StreamWriter, path: str) -> None:
    """Serve static files from the static directory."""
    clean = Path(path.lstrip("/"))
    if ".." in clean.parts:
        await _send_json(writer, {"error": "Forbidden"}, status=403)
        return

    file_path = _STATIC_DIR / clean
    if not file_path.is_file():
        await _send_json(writer, {"error": "Not found"}, status=404)
        return

    ext = file_path.suffix.lower()
    content_type = _CONTENT_TYPES.get(ext, "application/octet-stream")
    body = file_path.read_bytes()

    await _send_response(
        writer,
        200,
        {
            "Content-Type": content_type,
            "Content-Length": str(len(body)),
        },
        body,
    )


# ----------------------------------------------------------------------------------------
async def _handle_ping(writer: asyncio.StreamWriter) -> None:
    """Return a small JSON payload for latency measurement (used by CLI client).

    Includes a Server-Timing header so the client can subtract server processing
    time from the round-trip to get a more accurate network latency figure.
    """
    t0 = time.monotonic()
    data = json.dumps({"timestamp": time.time()}).encode("utf-8")
    processing_ms = (time.monotonic() - t0) * 1000

    await _send_response(
        writer,
        200,
        {
            "Content-Type": "application/json",
            "Content-Length": str(len(data)),
            "Access-Control-Allow-Origin": "*",
            "Access-Control-Expose-Headers": "Server-Timing",
            "Cache-Control": "no-store",
            "Server-Timing": f"processing;dur={processing_ms:.2f}",
        },
        data,
    )


# ----------------------------------------------------------------------------------------
async def _handle_download(
    writer: asyncio.StreamWriter,
    query: str,
) -> None:
    """Stream random data for download speed measurement."""
    params = parse_qs(query)
    size_str = params.get("size", [str(_MAX_DOWNLOAD_SIZE)])[0]
    try:
        size = int(size_str)
    except ValueError:
        size = _MAX_DOWNLOAD_SIZE

    size = max(1, min(size, _MAX_DOWNLOAD_SIZE))

    writer.write(
        f"HTTP/1.1 200 OK\r\n"
        f"Content-Type: application/octet-stream\r\n"
        f"Content-Length: {size}\r\n"
        f"Access-Control-Allow-Origin: *\r\n"
        f"Cache-Control: no-store\r\n"
        f"\r\n".encode()
    )
    await writer.drain()

    remaining = size
    while remaining > 0:
        chunk_len = min(_CHUNK_SIZE, remaining)
        try:
            if chunk_len == _CHUNK_SIZE:
                writer.write(_RANDOM_CHUNK)
            else:
                writer.write(_RANDOM_CHUNK[:chunk_len])
            remaining -= chunk_len
            # Drain periodically to avoid buffering the entire response
            if remaining % (16 * _CHUNK_SIZE) == 0 or remaining == 0:
                await writer.drain()
        except (ConnectionError, BrokenPipeError, OSError):
            return


# ----------------------------------------------------------------------------------------
async def _handle_upload(
    reader: asyncio.StreamReader,
    writer: asyncio.StreamWriter,
    headers: dict[str, str],
) -> None:
    """Accept uploaded data and return the size received."""
    content_length_str = headers.get("content-length", "0")
    try:
        content_length = int(content_length_str)
    except ValueError:
        content_length = 0

    start = time.time()
    received = 0
    try:
        while received < content_length:
            chunk_len = min(_CHUNK_SIZE, content_length - received)
            data = await reader.read(chunk_len)
            if not data:
                break
            received += len(data)
    except (ConnectionError, OSError):
        return
    elapsed = time.time() - start

    try:
        await _send_json(
            writer,
            {
                "size": received,
                "duration": elapsed,
                "timestamp": time.time(),
            },
        )
    except (ConnectionError, OSError):
        return


# ----------------------------------------------------------------------------------------
async def _handle_info(
    writer: asyncio.StreamWriter,
    config: ServerConfig,
) -> None:
    """Return server metadata."""
    await _send_json(
        writer,
        {
            "version": VERSION_STR,
            "server": "web-speedtest",
            "name": config.name,
            "test_duration": config.test_duration,
        },
    )


# ----------------------------------------------------------------------------------------
async def _handle_cors_preflight(writer: asyncio.StreamWriter) -> None:
    """Handle CORS preflight (OPTIONS) requests."""
    await _send_response(
        writer,
        204,
        {
            "Access-Control-Allow-Origin": "*",
            "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
            "Access-Control-Allow-Headers": "Content-Type",
            "Access-Control-Expose-Headers": "Server-Timing",
            "Access-Control-Max-Age": "86400",
        },
    )


# ----------------------------------------------------------------------------------------
#   WebSocket Handler
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
async def _handle_websocket(
    reader: asyncio.StreamReader,
    writer: asyncio.StreamWriter,
    request: HTTPRequest,
    config: ServerConfig,
) -> None:
    """Handle a WebSocket connection for low-latency ping measurement.

    The browser sends text frames containing "ping"; the server immediately
    responds with "pong". This avoids per-request HTTP overhead and gives
    a much more accurate network latency measurement.
    """
    client_key = request.headers.get("sec-websocket-key", "")
    writer.write(build_upgrade_response(client_key))
    await writer.drain()

    _log(config, "WebSocket /ws connected")

    try:
        while True:
            opcode, payload = await read_frame(reader)

            if opcode == OP_CLOSE:
                writer.write(encode_frame(OP_CLOSE, b""))
                await writer.drain()
                break
            elif opcode == OP_PING:
                writer.write(encode_frame(OP_PONG, payload))
                await writer.drain()
            elif opcode == OP_TEXT:
                msg = payload.decode("utf-8", errors="replace")
                if msg == "ping":
                    writer.write(encode_frame(OP_TEXT, b"pong"))
                    await writer.drain()
    except (asyncio.IncompleteReadError, ConnectionError, OSError):
        pass

    _log(config, "WebSocket /ws disconnected")


# ----------------------------------------------------------------------------------------
#   Connection Handler
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
async def _handle_connection(
    reader: asyncio.StreamReader,
    writer: asyncio.StreamWriter,
    config: ServerConfig,
) -> None:
    """Handle a single TCP connection. Supports HTTP keep-alive and WebSocket upgrade."""
    try:
        while True:
            request = await _read_request(reader)
            if request is None:
                break

            method = request.method.upper()
            path = request.path

            _log(config, f"{method} {path}")

            # WebSocket upgrade
            if (
                path == "/ws"
                and request.headers.get("upgrade", "").lower() == "websocket"
            ):
                await _handle_websocket(reader, writer, request, config)
                break  # WebSocket takes over the connection

            # CORS preflight
            if method == "OPTIONS":
                await _handle_cors_preflight(writer)
            # GET routes
            elif method == "GET":
                if path == "/":
                    await _handle_index(writer, config)
                elif path == "/api/ping":
                    await _handle_ping(writer)
                elif path == "/api/download":
                    await _handle_download(writer, request.query)
                elif path == "/api/info":
                    await _handle_info(writer, config)
                else:
                    await _handle_static(writer, path)
            # POST routes
            elif method == "POST":
                if path == "/api/upload":
                    await _handle_upload(reader, writer, request.headers)
                else:
                    await _send_json(writer, {"error": "Not found"}, status=404)
            else:
                await _send_json(writer, {"error": "Not found"}, status=404)

            # Check Connection header for keep-alive
            conn = request.headers.get("connection", "").lower()
            if conn == "close":
                break
            # HTTP/1.0 defaults to close unless keep-alive is explicit
            if request.version == "HTTP/1.0" and conn != "keep-alive":
                break
    except (ConnectionError, BrokenPipeError, OSError):
        pass
    finally:
        try:
            writer.close()
            await writer.wait_closed()
        except (ConnectionError, OSError):
            pass


# ----------------------------------------------------------------------------------------
#   Server Entry Point
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
def run_server(
    host: str,
    port: int,
    verbose: bool,
    name: str = "Speed Test",
    test_duration: int = _DEFAULT_TEST_DURATION,
) -> int:
    """Start the speed test server."""
    config = ServerConfig(name=name, test_duration=test_duration, verbose=verbose)

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

    async def _serve() -> None:
        server = await asyncio.start_server(
            lambda r, w: _handle_connection(r, w, config),
            host,
            port,
        )
        async with server:
            await server.serve_forever()

    try:
        asyncio.run(_serve())
    except KeyboardInterrupt:
        print()
        print(dim("Shutting down..."))

    return 0
