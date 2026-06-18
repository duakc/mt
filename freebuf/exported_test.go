package freebuf_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/duakc/mt/freebuf"

	"github.com/stretchr/testify/assert"
)

type factory struct {
	name string
	new  func() freebuf.Buffer
}

func factories() []factory {
	return []factory{
		{"Serial", func() freebuf.Buffer { return freebuf.NewSerial() }},
		{"MultiPart", func() freebuf.Buffer { return freebuf.NewMultiPart() }},
	}
}

type bufferCase struct {
	name string
	fn   func(t *testing.T, buf freebuf.Buffer)
}

func bufferCases() []bufferCase {
	return []bufferCase{
		{"WriteRead", testWriteRead},
		{"WriteByteReadByte", testWriteByteReadByte},
		{"WriteString", testWriteString},
		{"ReadFrom", testReadFrom},
		{"ReadFromPropagatesError", testReadFromPropagatesError},
		{"WriteTo", testWriteTo},
		{"WriteToEmpty", testWriteToEmpty},
		{"WriteToPropagatesError", testWriteToPropagatesError},
		{"CopyIsDeepCopy", testCopyIsDeepCopy},
		{"GrowHandlesReservedWrite", testGrowHandlesReservedWrite},
		{"ReadFromOnce", testReadFromOnce},
		{"WriteToOnce", testWriteToOnce},
	}
}

func TestBuffer(t *testing.T) {
	for _, c := range bufferCases() {
		for _, f := range factories() {
			t.Run(strings.Join(
				[]string{c.name, f.name}, "/"), func(t *testing.T) {
				buf := f.new()
				defer buf.FreeMe()
				c.fn(t, buf)
			})
		}
	}
}

type chunkReader struct {
	chunks [][]byte
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.chunks[0])
	if n < len(r.chunks[0]) {
		r.chunks[0] = r.chunks[0][n:]
	} else {
		r.chunks = r.chunks[1:]
	}
	return n, nil
}

type errReader struct{ err error }

func (r *errReader) Read(p []byte) (int, error) { return 0, r.err }

type dummyWriter struct {
	accepted int
	err      error
}

func (w *dummyWriter) Write(p []byte) (int, error) {
	if w.accepted == 0 {
		return 0, w.err
	}

	sub := min(w.accepted, len(p))
	w.accepted -= sub
	return sub, nil
}

func testWriteRead(t *testing.T, buf freebuf.Buffer) {
	payload := make([]byte, 32*1024+17)
	for i := range payload {
		payload[i] = byte(i)
	}
	n, err := buf.Write(payload)
	assert.NoError(t, err)
	assert.Equal(t, len(payload), n)
	assert.Equal(t, len(payload), buf.Len())

	out := make([]byte, len(payload))
	n, err = buf.Read(out)
	assert.NoError(t, err)
	assert.Equal(t, len(payload), n)
	assert.Equal(t, payload, out)
	assert.Equal(t, 0, buf.Len())
}

func testWriteByteReadByte(t *testing.T, buf freebuf.Buffer) {
	const count = 8192
	for i := range count {
		assert.NoError(t, buf.WriteByte(byte(i)))
	}
	assert.Equal(t, count, buf.Len())
	for i := range count {
		b, err := buf.ReadByte()
		assert.NoError(t, err)
		assert.Equalf(t, byte(i), b, "i=%d", i)
	}
	_, err := buf.ReadByte()
	assert.Equal(t, io.EOF, err)
}

func testWriteString(t *testing.T, buf freebuf.Buffer) {
	s := strings.Repeat("abc", 4096)
	n, err := buf.WriteString(s)
	assert.NoError(t, err)
	assert.Equal(t, len(s), n)

	out := make([]byte, len(s))
	buf.Read(out)
	assert.Equal(t, s, string(out))
}

func testReadFrom(t *testing.T, buf freebuf.Buffer) {
	payload := make([]byte, 50000)
	for i := range payload {
		payload[i] = byte(i ^ 0x5a)
	}
	n, err := buf.ReadFrom(bytes.NewReader(payload))
	assert.NoError(t, err)
	assert.Equal(t, int64(len(payload)), n)

	out := make([]byte, len(payload))
	nn, _ := buf.Read(out)
	assert.Equal(t, len(payload), nn)
	assert.Equal(t, payload, out)
}

func testReadFromPropagatesError(t *testing.T, buf freebuf.Buffer) {
	n, err := buf.ReadFrom(&errReader{err: io.ErrUnexpectedEOF})
	assert.Equal(t, int64(0), n)
	assert.Equal(t, io.ErrUnexpectedEOF, err)
}

func testWriteTo(t *testing.T, buf freebuf.Buffer) {
	payload := bytes.Repeat([]byte("xy"), 8192)
	buf.Write(payload)

	var out bytes.Buffer
	n, err := buf.WriteTo(&out)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(payload)), n)
	assert.Equal(t, payload, out.Bytes())
	assert.Equal(t, 0, buf.Len())
}

func testWriteToEmpty(t *testing.T, buf freebuf.Buffer) {
	var out bytes.Buffer
	n, err := buf.WriteTo(&out)
	assert.Equal(t, int64(0), n)
	assert.Equal(t, io.EOF, err)
}

func testWriteToPropagatesError(t *testing.T, buf freebuf.Buffer) {
	buf.Write([]byte("hello"))
	n, err := buf.WriteTo(&dummyWriter{accepted: 0, err: io.ErrClosedPipe})
	assert.Equal(t, int64(0), n)
	assert.Equal(t, io.ErrClosedPipe, err)
}

func testCopyIsDeepCopy(t *testing.T, buf freebuf.Buffer) {
	buf.Write([]byte("hello"))

	cp := buf.Copy()
	defer cp.FreeMe()
	assert.Equal(t, 5, cp.Len())

	buf.Write([]byte("!!!"))
	cp.Write([]byte("???"))

	srcOut := make([]byte, 8)
	cpOut := make([]byte, 8)
	buf.Read(srcOut)
	cp.Read(cpOut)
	assert.Equal(t, "hello!!!", string(srcOut))
	assert.Equal(t, "hello???", string(cpOut))
}

func testGrowHandlesReservedWrite(t *testing.T, buf freebuf.Buffer) {
	buf.Grow(32 * 1024)

	payload := bytes.Repeat([]byte{0xAB}, 32*1024)
	n, err := buf.Write(payload)
	assert.NoError(t, err)
	assert.Equal(t, len(payload), n)

	out := make([]byte, len(payload))
	nn, _ := buf.Read(out)
	assert.Equal(t, len(payload), nn)
	assert.Equal(t, payload, out)
}

func testReadFromOnce(t *testing.T, buf freebuf.Buffer) {
	src := &chunkReader{chunks: [][]byte{
		[]byte("hello"),
		[]byte("world"),
	}}
	n, err := buf.ReadFromOnce(src)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, 5, buf.Len())

	n, err = buf.ReadFromOnce(src)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, 10, buf.Len())

	_, err = buf.ReadFromOnce(src)
	assert.Equal(t, io.EOF, err)
}

func testWriteToOnce(t *testing.T, buf freebuf.Buffer) {
	buf.Write([]byte("payload"))

	w := &dummyWriter{accepted: 3}
	n, err := buf.WriteToOnce(w)
	assert.Equal(t, 3, n)
	assert.Equal(t, io.ErrShortWrite, err)
	assert.Equal(t, 4, buf.Len())

	w.accepted = 100
	n, err = buf.WriteToOnce(w)
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, 0, buf.Len())

	n, err = buf.WriteToOnce(w)
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)
}
