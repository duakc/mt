package freeio_test

import (
	"bytes"
	"io"
	"net"
	"testing"

	"github.com/duakc/mt/freebuf"
	"github.com/duakc/mt/freebuf/freeio"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// payload is a deterministic, multi-chunk-sized blob (crosses both the 256KB
// readFromChunk and the 1MB splice window more than once).
var payload = func() []byte {
	b := make([]byte, 1300*1024)
	for i := range b {
		b[i] = byte(i*31 + 7)
	}
	return b
}()

// adder returns a CounterFunc that sums into total. Copy invokes read/write counters
// from a single goroutine, so a plain accumulator is race-free here.
func adder(total *int64) freeio.CounterFunc {
	return func(n int64) { *total += n }
}

// onlyReader / onlyWriter hide WriterTo / ReaderFrom so a copy is forced onto
// the generic / buffer-staging path.
type (
	onlyReader struct{ io.Reader }
	onlyWriter struct{ io.Writer }
)

func TestCopy(t *testing.T) {
	t.Parallel()

	var dst bytes.Buffer
	n, err := freeio.Copy(&dst, bytes.NewReader(payload))
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), n)
	assert.Equal(t, payload, dst.Bytes())
}

func TestCopyWithCounter(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		src      func() io.Reader
		dst      func() io.Writer
		wantN    int64
		wantData []byte
	}{
		{
			name:     "buffered generic",
			src:      func() io.Reader { return onlyReader{bytes.NewReader(payload)} },
			dst:      func() io.Writer { return onlyWriter{&bytes.Buffer{}} },
			wantN:    int64(len(payload)),
			wantData: payload,
		},
		{
			name:     "reader-from dst",
			src:      func() io.Reader { return onlyReader{bytes.NewReader(payload)} },
			dst:      func() io.Writer { return &bytes.Buffer{} },
			wantN:    int64(len(payload)),
			wantData: payload,
		},
		{
			name:     "limited reader",
			src:      func() io.Reader { return io.LimitReader(bytes.NewReader(payload), 100) },
			dst:      func() io.Writer { return &bytes.Buffer{} },
			wantN:    100,
			wantData: payload[:100],
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var read, wrote int64
			dst := tc.dst()
			n, err := freeio.CopyWithCounter(dst, tc.src(),
				[]freeio.CounterFunc{adder(&wrote)}, []freeio.CounterFunc{adder(&read)})
			require.NoError(t, err)
			assert.Equal(t, tc.wantN, n)
			assert.Equal(t, tc.wantN, read, "read counter")
			assert.Equal(t, tc.wantN, wrote, "write counter")

			// Recover the written bytes regardless of the onlyWriter wrapper.
			switch w := dst.(type) {
			case onlyWriter:
				assert.Equal(t, tc.wantData, w.Writer.(*bytes.Buffer).Bytes())
			case *bytes.Buffer:
				assert.Equal(t, tc.wantData, w.Bytes())
			}
		})
	}
}

func TestCopyBuffer(t *testing.T) {
	t.Parallel()

	scratch := freebuf.NewSerial()
	defer scratch.FreeMe()

	var dst bytes.Buffer
	// Hide WriterTo/ReaderFrom so the staging loop (not a fast path) runs.
	n, err := freeio.CopyBuffer(onlyWriter{&dst}, onlyReader{bytes.NewReader(payload)}, scratch)
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), n)
	assert.Equal(t, payload, dst.Bytes())
}

func TestCounterReaderWriter(t *testing.T) {
	t.Parallel()

	var read int64
	cr := freeio.NewCounterReader(bytes.NewReader(payload), adder(&read))
	got, err := io.ReadAll(cr)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
	assert.Equal(t, int64(len(payload)), read)

	var wrote int64
	var dst bytes.Buffer
	cw := freeio.NewCounterWriter(&dst, adder(&wrote))
	m, err := cw.Write(payload)
	require.NoError(t, err)
	assert.Equal(t, len(payload), m)
	assert.Equal(t, int64(len(payload)), wrote)
	assert.Equal(t, payload, dst.Bytes())
}

// TestCopyConn exercises the conn<->conn path (splice on Linux) through a small
// TCP proxy: client -> proxy -> upstream.
func TestCopyConn(t *testing.T) {
	t.Parallel()

	upLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer upLn.Close()

	upRecv := make(chan []byte, 1)
	go func() {
		c, err := upLn.Accept()
		if err != nil {
			upRecv <- nil
			return
		}
		defer c.Close()
		b, _ := io.ReadAll(c)
		upRecv <- b
	}()

	proxyLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer proxyLn.Close()

	var read, wrote int64
	copyErr := make(chan error, 1)
	go func() {
		clientConn, err := proxyLn.Accept()
		if err != nil {
			copyErr <- err
			return
		}
		defer clientConn.Close()
		upConn, err := net.Dial("tcp", upLn.Addr().String())
		if err != nil {
			copyErr <- err
			return
		}
		_, err = freeio.CopyWithCounter(upConn, clientConn,
			[]freeio.CounterFunc{adder(&wrote)}, []freeio.CounterFunc{adder(&read)})
		_ = upConn.(*net.TCPConn).CloseWrite()
		copyErr <- err
	}()

	client, err := net.Dial("tcp", proxyLn.Addr().String())
	require.NoError(t, err)
	_, err = client.Write(payload)
	require.NoError(t, err)
	require.NoError(t, client.(*net.TCPConn).CloseWrite())

	require.NoError(t, <-copyErr)
	got := <-upRecv
	_ = client.Close()

	assert.Equal(t, payload, got)
	assert.Equal(t, int64(len(payload)), read, "read counter")
	assert.Equal(t, int64(len(payload)), wrote, "write counter")
}
