// ---------------------------------------------------------------------------------------
//
//	websocket.go
//	------------
//
//	Minimal WebSocket implementation (RFC 6455) using only the standard library.
//	Provides handshake computation, frame encoding (server-to-client), and frame
//	decoding (client-to-server with masking).
//
//	(c) 2026 WaterJuice — Released under the Unlicense; see LICENSE.
//
//	Version History
//	---------------
//	Mar 2026 - Created (Python)
//	Mar 2026 - Rewritten in Go
//
// ---------------------------------------------------------------------------------------
package internal

// ---------------------------------------------------------------------------------------
//
//	Imports
//
// ---------------------------------------------------------------------------------------

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
)

// ---------------------------------------------------------------------------------------
//
//	Constants
//
// ---------------------------------------------------------------------------------------

// Magic GUID from RFC 6455 section 4.2.2
const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// WebSocket opcodes
const (
	opText   = 0x1
	opBinary = 0x2
	opClose  = 0x8
	opPing   = 0x9
	opPong   = 0xA
)

// Maximum allowed WebSocket frame payload size (16 MB)
const wsMaxFrameSize = 16 * 1024 * 1024

// ---------------------------------------------------------------------------------------
//
//	Handshake
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// wsAcceptKey computes the Sec-WebSocket-Accept value for a given client key.
func wsAcceptKey(clientKey string) string {
	h := sha1.New() //nolint:gosec
	h.Write([]byte(clientKey + wsGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// ---------------------------------------------------------------------------------------
// wsUpgrade hijacks an HTTP connection and completes the WebSocket handshake.
// Returns the buffered reader/writer (for reading frames) and the underlying
// connection (for writing frames and closing).
func wsUpgrade(w http.ResponseWriter, r *http.Request) (*bufio.ReadWriter, net.Conn, error) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "WebSocket not supported", http.StatusInternalServerError)
		return nil, nil, fmt.Errorf("hijack not supported")
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return nil, nil, err
	}

	accept := wsAcceptKey(r.Header.Get("Sec-WebSocket-Key"))
	bufrw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	bufrw.WriteString("Upgrade: websocket\r\n")
	bufrw.WriteString("Connection: Upgrade\r\n")
	bufrw.WriteString("Sec-WebSocket-Accept: " + accept + "\r\n")
	bufrw.WriteString("\r\n")
	bufrw.Flush()

	return bufrw, conn, nil
}

// ---------------------------------------------------------------------------------------
//
//	Frame Encoding / Decoding
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// wsReadFrame reads and decodes a WebSocket frame from a client (masked).
// Returns (opcode, payload, error).
func wsReadFrame(r io.Reader) (int, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, nil, err
	}

	opcode := int(header[0] & 0x0F)
	masked := header[1]&0x80 != 0
	length := int(header[1] & 0x7F)

	if length == 126 {
		ext := make([]byte, 2)
		if _, err := io.ReadFull(r, ext); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint16(ext))
	} else if length == 127 {
		ext := make([]byte, 8)
		if _, err := io.ReadFull(r, ext); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint64(ext))
	}

	if length > wsMaxFrameSize {
		return 0, nil, fmt.Errorf("frame too large: %d bytes", length)
	}

	if masked {
		maskKey := make([]byte, 4)
		if _, err := io.ReadFull(r, maskKey); err != nil {
			return 0, nil, err
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(r, payload); err != nil {
			return 0, nil, err
		}
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
		return opcode, payload, nil
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, err
	}
	return opcode, payload, nil
}

// ---------------------------------------------------------------------------------------
// wsEncodeFrame encodes a WebSocket frame for sending from server to client (unmasked).
func wsEncodeFrame(opcode int, payload []byte) []byte {
	var frame []byte
	frame = append(frame, byte(0x80|opcode))

	length := len(payload)
	if length < 126 {
		frame = append(frame, byte(length))
	} else if length < 65536 {
		frame = append(frame, 126)
		frame = append(frame, byte(length>>8), byte(length))
	} else {
		frame = append(frame, 127)
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(length))
		frame = append(frame, b...)
	}

	frame = append(frame, payload...)
	return frame
}
