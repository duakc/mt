//go:build linux

package freeio

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSendfileConn exercises the hand-rolled sendfile(2) path directly — the
// copyCounted dispatch prefers splice when the kernel supports it, so sendfile
// (the splice fallback) is otherwise not reached on a modern kernel.
func TestSendfileConn(t *testing.T) {
	data := make([]byte, 300*1024)
	for i := range data {
		data[i] = byte(i*7 + 3)
	}
	p := filepath.Join(t.TempDir(), "f")
	require.NoError(t, os.WriteFile(p, data, 0o644))
	f, err := os.Open(p)
	require.NoError(t, err)
	defer f.Close()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	recv := make(chan []byte, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			recv <- nil
			return
		}
		defer c.Close()
		b, _ := io.ReadAll(c)
		recv <- b
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)

	var read, wrote int64
	n, handled, err := sendfileConn(f, conn.(syscall.Conn),
		[]CounterFunc{func(x int64) { wrote += x }}, []CounterFunc{func(x int64) { read += x }})
	require.NoError(t, err)
	require.True(t, handled)
	require.NoError(t, conn.(*net.TCPConn).CloseWrite())

	assert.Equal(t, int64(len(data)), n)
	assert.Equal(t, int64(len(data)), read, "read counter")
	assert.Equal(t, int64(len(data)), wrote, "write counter")

	got := <-recv
	_ = conn.Close()
	assert.Equal(t, data, got)
}
