package net

import (
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// References:
// https://github.com/zhangpeihao/gowebsocket/blob/master/conn.go#L78-L123

const (
	readBufferSize  = 1024
	writeBufferSize = 1024
)

// Conn implement the net.Conn interface.
// All data are transfered in binary stream.
type Conn struct {
	ws *websocket.Conn
	r  io.Reader
}

// Create a server side connection.
func NewConn(w http.ResponseWriter, r *http.Request, responseHeader http.Header,
	readBufSize, writeBufSize int) (conn *Conn, err error) {
	var ws *websocket.Conn
	if ws, err = websocket.Upgrade(w, r, responseHeader, readBufSize,
		writeBufSize); err != nil {
		return
	}
	conn = &Conn{
		ws: ws,
	}
	return
}

// Read reads data from the connection.
// Read can be made to time out and return a Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (conn *Conn) Read(b []byte) (n int, err error) {
	var opCode int
	if conn.r == nil {
		// New message
		var r io.Reader
		for {
			if opCode, r, err = conn.ws.NextReader(); err != nil {
				return
			}
			if opCode != websocket.BinaryMessage && opCode != websocket.TextMessage {
				continue
			}
			conn.r = r
			break
		}
	}
	n, err = conn.r.Read(b)
	if err != nil {
		if err == io.EOF {
			// Message finished
			conn.r = nil
			err = nil
		}
	}
	return
}

// Write writes data to the connection.
// Write can be made to time out and return a Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (conn *Conn) Write(b []byte) (n int, err error) {
	var w io.WriteCloser
	if w, err = conn.ws.NextWriter(websocket.BinaryMessage); err != nil {
		return
	}
	if n, err = w.Write(b); err != nil {
		return
	}
	err = w.Close()
	return
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (conn *Conn) Close() error {
	return conn.ws.Close()
}

// SetReadDeadline sets the deadline for future Read calls.
// A zero value for t means Read will not time out.
func (conn *Conn) SetReadDeadline(t time.Time) error {
	return conn.ws.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (conn *Conn) SetWriteDeadline(t time.Time) error {
	return conn.ws.SetWriteDeadline(t)
}

// GetWebsocketConn get the underlying websocket connection
func (conn *Conn) GetWebsocketConn() *websocket.Conn {
	return conn.ws
}
