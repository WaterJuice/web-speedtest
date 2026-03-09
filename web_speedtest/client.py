# ----------------------------------------------------------------------------------------
#   client.py
#   ---------
#
#   CLI speed test client. Connects to a web-speedtest server and measures ping,
#   download speed, and upload speed.
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

import json
import os
import sys
import time
import urllib.error
import urllib.request
from typing import Literal
from .colour import bold
from .colour import cyan
from .colour import dim
from .colour import green
from .colour import red
from .colour import yellow

# ----------------------------------------------------------------------------------------
#   Types
# ----------------------------------------------------------------------------------------

OutputMode = Literal["normal", "compact", "json"]

# ----------------------------------------------------------------------------------------
#   Constants
# ----------------------------------------------------------------------------------------

# Number of ping samples
_PING_COUNT = 10

# Download test size: 25 MB
_DOWNLOAD_SIZE = 25 * 1024 * 1024

# Upload test size: 10 MB
_UPLOAD_SIZE = 10 * 1024 * 1024

# Read chunk size: 64 KB
_CHUNK_SIZE = 65536

# Connection timeout in seconds
_TIMEOUT = 30

# ----------------------------------------------------------------------------------------
#   Helpers
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
def _format_speed(bits_per_second: float) -> str:
    """Format a speed value in human-readable form."""
    if bits_per_second >= 1_000_000_000:
        return f"{bits_per_second / 1_000_000_000:.2f} Gbps"
    if bits_per_second >= 1_000_000:
        return f"{bits_per_second / 1_000_000:.2f} Mbps"
    if bits_per_second >= 1_000:
        return f"{bits_per_second / 1_000:.2f} Kbps"
    return f"{bits_per_second:.0f} bps"


# ----------------------------------------------------------------------------------------
def _format_latency(ms: float) -> str:
    """Format a latency value."""
    if ms < 1.0:
        return f"{ms * 1000:.0f} µs"
    if ms < 100.0:
        return f"{ms:.1f} ms"
    return f"{ms:.0f} ms"


# ----------------------------------------------------------------------------------------
def _progress_bar(fraction: float, width: int = 30) -> str:
    """Render a simple progress bar."""
    filled = int(fraction * width)
    bar = "█" * filled + "░" * (width - filled)
    return f"[{bar}]"


# ----------------------------------------------------------------------------------------
#   Speed Test Functions
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
def _test_ping(base_url: str, quiet: bool = False) -> float | None:
    """Measure ping latency. Returns average latency in milliseconds or None on error."""
    url = f"{base_url}/api/ping"
    latencies: list[float] = []

    if not quiet:
        print(dim(f"  Testing ping ({_PING_COUNT} samples)..."), end="", flush=True)

    for _ in range(_PING_COUNT):
        try:
            start = time.monotonic()
            req = urllib.request.Request(url)
            with urllib.request.urlopen(req, timeout=_TIMEOUT) as resp:
                resp.read()
            elapsed = (time.monotonic() - start) * 1000
            latencies.append(elapsed)
        except (urllib.error.URLError, OSError, TimeoutError):
            pass

    if not quiet:
        print("\r", end="")

    if not latencies:
        return None

    avg = sum(latencies) / len(latencies)
    return avg


# ----------------------------------------------------------------------------------------
def _test_download(base_url: str, quiet: bool = False) -> float | None:
    """Measure download speed. Returns bits per second or None on error."""
    url = f"{base_url}/api/download?size={_DOWNLOAD_SIZE}"

    if not quiet:
        print(dim("  Testing download..."), end="", flush=True)

    try:
        req = urllib.request.Request(url)
        start = time.monotonic()
        received = 0
        with urllib.request.urlopen(req, timeout=_TIMEOUT) as resp:
            while True:
                chunk = resp.read(_CHUNK_SIZE)
                if not chunk:
                    break
                received += len(chunk)
                if not quiet:
                    fraction = received / _DOWNLOAD_SIZE
                    speed_so_far = (received * 8) / max(time.monotonic() - start, 0.001)
                    print(
                        f"\r  Testing download... {_progress_bar(fraction)} {_format_speed(speed_so_far)}   ",
                        end="",
                        flush=True,
                    )
        elapsed = time.monotonic() - start
    except (urllib.error.URLError, OSError, TimeoutError) as e:
        if not quiet:
            print(f"\r  Download failed: {e}{'':40}")
        return None

    if not quiet:
        print("\r", end="")

    if elapsed <= 0 or received == 0:
        return None

    bits_per_second = (received * 8) / elapsed
    return bits_per_second


# ----------------------------------------------------------------------------------------
def _test_upload(base_url: str, quiet: bool = False) -> float | None:
    """Measure upload speed. Returns bits per second or None on error."""
    url = f"{base_url}/api/upload"

    if not quiet:
        print(dim("  Testing upload..."), end="", flush=True)

    # Generate random upload payload
    payload = os.urandom(_UPLOAD_SIZE)

    try:
        req = urllib.request.Request(
            url,
            data=payload,
            method="POST",
        )
        req.add_header("Content-Type", "application/octet-stream")
        req.add_header("Content-Length", str(len(payload)))

        start = time.monotonic()
        with urllib.request.urlopen(req, timeout=_TIMEOUT) as resp:
            resp.read()
        elapsed = time.monotonic() - start
    except (urllib.error.URLError, OSError, TimeoutError) as e:
        if not quiet:
            print(f"\r  Upload failed: {e}{'':40}")
        return None

    if not quiet:
        print("\r", end="")

    if elapsed <= 0:
        return None

    bits_per_second = (_UPLOAD_SIZE * 8) / elapsed
    return bits_per_second


# ----------------------------------------------------------------------------------------
def _check_server(base_url: str, quiet: bool = False) -> bool:
    """Check if the server is reachable."""
    url = f"{base_url}/api/info"
    try:
        req = urllib.request.Request(url)
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            if data.get("server") == "web-speedtest":
                if not quiet:
                    print(dim(f"  Server version: {data.get('version', 'unknown')}"))
                return True
    except (urllib.error.URLError, OSError, TimeoutError, json.JSONDecodeError):
        pass
    return False


# ----------------------------------------------------------------------------------------
#   Main Client Function
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
def _normalise_url(server_url: str) -> str:
    """Normalise a server URL, adding http:// if missing."""
    if not server_url.startswith(("http://", "https://")):
        server_url = f"http://{server_url}"
    return server_url.rstrip("/")


# ----------------------------------------------------------------------------------------
def _run_normal(server_url: str) -> int:
    """Run speed test with full interactive output."""
    print(bold("web-speedtest client"))
    print(dim(f"  server:   {server_url}"))
    print()

    # Check server
    print(dim("  Connecting..."), end="", flush=True)
    if not _check_server(server_url):
        print(f"\r{red('  Failed to connect to server.')}{'':40}")
        print(red(f"  Could not reach {server_url}"), file=sys.stderr)
        return 1
    print(f"\r{green('  Connected.')}{'':40}")
    print()

    # --- Ping ---
    ping_ms = _test_ping(server_url)
    if ping_ms is not None:
        print(f"  {'Ping:':<14}{cyan(_format_latency(ping_ms))}{'':40}")
    else:
        print(f"  {'Ping:':<14}{yellow('failed')}{'':40}")
    print()

    # --- Download ---
    download_bps = _test_download(server_url)
    if download_bps is not None:
        print(f"  {'Download:':<14}{cyan(_format_speed(download_bps))}{'':40}")
    else:
        print(f"  {'Download:':<14}{yellow('failed')}{'':40}")
    print()

    # --- Upload ---
    upload_bps = _test_upload(server_url)
    if upload_bps is not None:
        print(f"  {'Upload:':<14}{cyan(_format_speed(upload_bps))}{'':40}")
    else:
        print(f"  {'Upload:':<14}{yellow('failed')}{'':40}")
    print()

    # --- Summary ---
    print(dim("  " + "-" * 40))
    if ping_ms is not None and download_bps is not None and upload_bps is not None:
        print(bold("  Results"))
        print(f"    Ping:      {green(_format_latency(ping_ms))}")
        print(f"    Download:  {green(_format_speed(download_bps))}")
        print(f"    Upload:    {green(_format_speed(upload_bps))}")
    else:
        print(yellow("  Some tests failed — results may be incomplete."))
    print()

    return 0


# ----------------------------------------------------------------------------------------
def _run_compact(server_url: str) -> int:
    """Run speed test with compact one-line output."""
    if not _check_server(server_url, quiet=True):
        print(red(f"error: could not reach {server_url}"), file=sys.stderr)
        return 1

    ping_ms = _test_ping(server_url, quiet=True)
    download_bps = _test_download(server_url, quiet=True)
    upload_bps = _test_upload(server_url, quiet=True)

    ping_str = _format_latency(ping_ms) if ping_ms is not None else "failed"
    down_str = _format_speed(download_bps) if download_bps is not None else "failed"
    up_str = _format_speed(upload_bps) if upload_bps is not None else "failed"

    print(f"ping {ping_str} / down {down_str} / up {up_str}")
    return 0


# ----------------------------------------------------------------------------------------
def _run_json(server_url: str) -> int:
    """Run speed test with JSON output."""
    if not _check_server(server_url, quiet=True):
        result = {"error": f"could not reach {server_url}"}
        print(json.dumps(result))
        return 1

    ping_ms = _test_ping(server_url, quiet=True)
    download_bps = _test_download(server_url, quiet=True)
    upload_bps = _test_upload(server_url, quiet=True)

    result: dict[str, object] = {
        "server": server_url,
        "ping_ms": round(ping_ms) if ping_ms is not None else None,
        "ping": _format_latency(ping_ms) if ping_ms is not None else None,
        "download_bps": round(download_bps) if download_bps is not None else None,
        "download": _format_speed(download_bps) if download_bps is not None else None,
        "upload_bps": round(upload_bps) if upload_bps is not None else None,
        "upload": _format_speed(upload_bps) if upload_bps is not None else None,
    }
    print(json.dumps(result, indent=2))
    return 0


# ----------------------------------------------------------------------------------------
def run_client(server_url: str, output: OutputMode = "normal") -> int:
    """Run a speed test against the given server."""
    server_url = _normalise_url(server_url)

    if output == "json":
        return _run_json(server_url)
    if output == "compact":
        return _run_compact(server_url)
    return _run_normal(server_url)
