# ----------------------------------------------------------------------------------------
#   websocket.py
#   ------------
#
#   Minimal WebSocket implementation (RFC 6455) using only the standard library.
#   Provides handshake computation, frame encoding (server-to-client), and frame
#   decoding (client-to-server with masking).
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

import asyncio
import base64
import hashlib
import struct

# ----------------------------------------------------------------------------------------
#   Constants
# ----------------------------------------------------------------------------------------

# Magic GUID from RFC 6455 section 4.2.2
_WS_GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

# WebSocket opcodes
OP_TEXT = 0x1
OP_BINARY = 0x2
OP_CLOSE = 0x8
OP_PING = 0x9
OP_PONG = 0xA

# ----------------------------------------------------------------------------------------
#   Handshake
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
def compute_accept_key(client_key: str) -> str:
    """Compute the Sec-WebSocket-Accept value for a given client key."""
    digest = hashlib.sha1((client_key + _WS_GUID).encode()).digest()  # noqa: S324
    return base64.b64encode(digest).decode()


# ----------------------------------------------------------------------------------------
def build_upgrade_response(client_key: str) -> bytes:
    """Build the HTTP 101 response to complete the WebSocket handshake."""
    accept = compute_accept_key(client_key)
    return (
        "HTTP/1.1 101 Switching Protocols\r\n"
        "Upgrade: websocket\r\n"
        "Connection: Upgrade\r\n"
        f"Sec-WebSocket-Accept: {accept}\r\n"
        "\r\n"
    ).encode()


# ----------------------------------------------------------------------------------------
#   Frame Encoding / Decoding
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
def encode_frame(opcode: int, payload: bytes) -> bytes:
    """Encode a WebSocket frame for sending from server to client (unmasked)."""
    header = bytes([0x80 | opcode])  # FIN bit set + opcode
    length = len(payload)
    if length < 126:
        header += bytes([length])
    elif length < 65536:
        header += bytes([126]) + struct.pack("!H", length)
    else:
        header += bytes([127]) + struct.pack("!Q", length)
    return header + payload


# ----------------------------------------------------------------------------------------
async def read_frame(reader: asyncio.StreamReader) -> tuple[int, bytes]:
    """Read and decode a WebSocket frame from a client (masked).

    Returns (opcode, payload). Raises asyncio.IncompleteReadError if the
    connection closes mid-frame, or ConnectionError for protocol violations.
    """
    data = await reader.readexactly(2)
    opcode = data[0] & 0x0F
    masked = bool(data[1] & 0x80)
    length = data[1] & 0x7F

    if length == 126:
        data = await reader.readexactly(2)
        length = struct.unpack("!H", data)[0]
    elif length == 127:
        data = await reader.readexactly(8)
        length = struct.unpack("!Q", data)[0]

    if masked:
        mask_key = await reader.readexactly(4)
        raw = await reader.readexactly(length)
        payload = bytes(b ^ mask_key[i % 4] for i, b in enumerate(raw))
    else:
        payload = await reader.readexactly(length)

    return opcode, payload
