package freeio_test

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/duakc/mt/freebuf"
	"github.com/duakc/mt/freeio"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTemp(t *testing.T, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "src")
	require.NoError(t, os.WriteFile(p, data, 0o644))
	return p
}

func TestCopyFile(t *testing.T) {
	t.Parallel()

	src := writeTemp(t, payload)
	dst := filepath.Join(t.TempDir(), "dst")

	var read, wrote int64
	n, err := freeio.CopyFileWithCounter(dst, src,
		[]freeio.CounterFunc{adder(&wrote)}, []freeio.CounterFunc{adder(&read)})

	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), n)
	assert.Equal(t, int64(len(payload)), read, "read counter")
	assert.Equal(t, int64(len(payload)), wrote, "write counter")

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestCopyFileSameFile(t *testing.T) {
	t.Parallel()

	p := writeTemp(t, payload)
	_, err := freeio.CopyFile(p, p)
	require.Error(t, err)

	got, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, payload, got, "source must be untouched")
}

func TestCopyFS(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.bin"), payload[:1000], 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "sub", "b.bin"), payload[:2000], 0o644))

	dstDir := filepath.Join(t.TempDir(), "out")
	n, files, err := freeio.CopyFS(dstDir, os.DirFS(srcDir))
	require.NoError(t, err)
	assert.Equal(t, int64(3000), n)
	assert.Equal(t, 2, files)

	a, err := os.ReadFile(filepath.Join(dstDir, "a.bin"))
	require.NoError(t, err)
	assert.Equal(t, payload[:1000], a)
	bb, err := os.ReadFile(filepath.Join(dstDir, "sub", "b.bin"))
	require.NoError(t, err)
	assert.Equal(t, payload[:2000], bb)
}

func TestCopyLimitedFileToFile(t *testing.T) {
	t.Parallel()

	srcF, err := os.Open(writeTemp(t, payload))
	require.NoError(t, err)
	defer srcF.Close()

	dstPath := filepath.Join(t.TempDir(), "dst")
	dstF, err := os.Create(dstPath)
	require.NoError(t, err)
	defer dstF.Close()

	var read int64
	n, err := freeio.CopyWithCounter(dstF, io.LimitReader(srcF, 1000), nil, []freeio.CounterFunc{adder(&read)})
	require.NoError(t, err)
	assert.Equal(t, int64(1000), n)
	assert.Equal(t, int64(1000), read)
	require.NoError(t, dstF.Sync())

	got, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, payload[:1000], got)
}

func TestReadFile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		data []byte
	}{
		{"small", payload[:1024]}, // SerialBuffer, pre-sized
		{"large", payload},        // MultiPartBuffer, grown
		{"empty", []byte{}},       // zero-length file
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var read int64
			buf, err := freeio.ReadFileWithCounter(writeTemp(t, tc.data), []freeio.CounterFunc{adder(&read)})
			require.NoError(t, err)
			defer buf.FreeMe()

			assert.Equal(t, len(tc.data), buf.Len())
			assert.Equal(t, int64(len(tc.data)), read)

			// Round-trip the contents back out to verify they survived.
			out := filepath.Join(t.TempDir(), "out")
			require.NoError(t, freeio.WriteFile(out, buf))
			got, err := os.ReadFile(out)
			require.NoError(t, err)
			assert.Equal(t, tc.data, got)
		})
	}
}

func TestWriteFileEmpty(t *testing.T) {
	t.Parallel()

	buf := freebuf.NewSerial()
	defer buf.FreeMe()

	out := filepath.Join(t.TempDir(), "empty")
	require.NoError(t, freeio.WriteFile(out, buf))

	info, err := os.Stat(out)
	require.NoError(t, err)
	assert.Equal(t, int64(0), info.Size())
}

func TestCopyFileToConn(t *testing.T) {
	t.Parallel()

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

	f, err := os.Open(writeTemp(t, payload))
	require.NoError(t, err)
	defer f.Close()

	conn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)

	var read, wrote int64
	n, err := freeio.CopyWithCounter(conn, f,
		[]freeio.CounterFunc{adder(&wrote)}, []freeio.CounterFunc{adder(&read)})
	require.NoError(t, err)
	require.NoError(t, conn.(*net.TCPConn).CloseWrite())

	assert.Equal(t, int64(len(payload)), n)
	assert.Equal(t, int64(len(payload)), read, "read counter")
	assert.Equal(t, int64(len(payload)), wrote, "write counter")

	got := <-recv
	_ = conn.Close()
	assert.Equal(t, payload, got)
}
