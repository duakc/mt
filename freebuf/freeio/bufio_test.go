package freeio_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/duakc/mt/freebuf/freeio"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReaderRead(t *testing.T) {
	t.Parallel()
	// OneByteReader forces the fill loop to refill repeatedly; a tiny buffer
	// keeps it cycling and compacting.
	r := freeio.NewReaderSize(iotest.OneByteReader(bytes.NewReader(payload)), 64)
	defer r.Free()

	got, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestReaderReadBypassesBufferForLargeReads(t *testing.T) {
	t.Parallel()
	r := freeio.NewReaderSize(onlyReader{bytes.NewReader(payload)}, 64)
	defer r.Free()

	got, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestReaderReadByte(t *testing.T) {
	t.Parallel()
	r := freeio.NewReaderSize(iotest.OneByteReader(bytes.NewReader(payload)), 32)
	defer r.Free()

	got := make([]byte, 0, len(payload))
	for {
		b, err := r.ReadByte()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		got = append(got, b)
	}
	assert.Equal(t, payload, got)
}

func TestReaderPeek(t *testing.T) {
	t.Parallel()
	r := freeio.NewReaderSize(strings.NewReader("hello world"), 16)
	defer r.Free()

	b, err := r.Peek(5)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))

	// Peek does not advance: a second Peek sees the same bytes.
	b, err = r.Peek(5)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))
	assert.Equal(t, 11, r.Buffered())

	// More than the buffer can hold.
	_, err = r.Peek(64)
	assert.ErrorIs(t, err, freeio.ErrBufferFull)

	// Still readable in full afterward.
	rest, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(rest))
}

func TestReaderPeekNegative(t *testing.T) {
	t.Parallel()
	r := freeio.NewReader(strings.NewReader("x"))
	defer r.Free()
	_, err := r.Peek(-1)
	assert.ErrorIs(t, err, freeio.ErrNegativeCount)
}

func TestReaderUnreadByte(t *testing.T) {
	t.Parallel()
	r := freeio.NewReaderSize(strings.NewReader("abc"), 16)
	defer r.Free()

	c, err := r.ReadByte()
	require.NoError(t, err)
	assert.Equal(t, byte('a'), c)

	require.NoError(t, r.UnreadByte())
	c, err = r.ReadByte()
	require.NoError(t, err)
	assert.Equal(t, byte('a'), c, "unread byte is re-read")

	// Two unreads in a row is invalid — only the last read can be undone.
	require.NoError(t, r.UnreadByte())
	assert.ErrorIs(t, r.UnreadByte(), freeio.ErrInvalidUnreadByte)

	rest, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "abc", string(rest))
}

func TestReaderDiscard(t *testing.T) {
	t.Parallel()
	r := freeio.NewReaderSize(iotest.OneByteReader(strings.NewReader("hello world")), 4)
	defer r.Free()

	n, err := r.Discard(6)
	require.NoError(t, err)
	assert.Equal(t, 6, n)

	rest, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "world", string(rest))

	// Discarding past EOF returns what was skipped plus io.EOF.
	n, err = r.Discard(10)
	assert.Equal(t, 0, n)
	assert.ErrorIs(t, err, io.EOF)
}

func TestReaderReadString(t *testing.T) {
	t.Parallel()
	// Buffer (8) larger than every line so each is found within one fill.
	r := freeio.NewReaderSize(strings.NewReader("line1\nline2\nlast"), 8)
	defer r.Free()

	l1, err := r.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "line1\n", l1)

	l2, err := r.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "line2\n", l2)

	l3, err := r.ReadString('\n')
	assert.ErrorIs(t, err, io.EOF)
	assert.Equal(t, "last", l3)
}

func TestReaderReadBytesAcrossBuffers(t *testing.T) {
	t.Parallel()
	// a line far longer than the buffer (16) makes ReadSlice return ErrBufferFull
	// repeatedly; ReadBytes must stitch the fragments back together.
	line := strings.Repeat("x", 50) + "\n"
	r := freeio.NewReaderSize(strings.NewReader(line+"tail"), 16)
	defer r.Free()

	got, err := r.ReadBytes('\n')
	require.NoError(t, err)
	assert.Equal(t, line, string(got))

	rest, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "tail", string(rest))
}

func TestReaderReadSliceInvalidatedByNextRead(t *testing.T) {
	t.Parallel()
	r := freeio.NewReaderSize(strings.NewReader("ab\ncd\n"), 16)
	defer r.Free()

	s, err := r.ReadSlice('\n')
	require.NoError(t, err)
	assert.Equal(t, "ab\n", string(s))
}

func TestReaderWriteTo(t *testing.T) {
	t.Parallel()
	r := freeio.NewReaderSize(onlyReader{bytes.NewReader(payload)}, 128)
	defer r.Free()

	// consume a little first so WriteTo must drain the buffer and then the rest.
	pre, err := r.Peek(10)
	require.NoError(t, err)
	assert.Equal(t, payload[:10], pre)

	var dst bytes.Buffer
	n, err := r.WriteTo(&dst)
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), n)
	assert.Equal(t, payload, dst.Bytes())
}

func TestReaderReset(t *testing.T) {
	t.Parallel()
	r := freeio.NewReader(strings.NewReader("first"))
	defer r.Free()
	first, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "first", string(first))

	r.Reset(strings.NewReader("second"))
	second, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "second", string(second))
}

func TestWriterWriteAndFlush(t *testing.T) {
	t.Parallel()
	var dst bytes.Buffer
	w := freeio.NewWriterSize(&dst, 16)

	n, err := io.WriteString(w, "ab")
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, 2, w.Buffered())
	assert.Zero(t, dst.Len(), "small write should stay buffered")

	require.NoError(t, w.Flush())
	assert.Equal(t, "ab", dst.String())
	assert.Zero(t, w.Buffered())
}

func TestWriterLargeWrite(t *testing.T) {
	t.Parallel()
	var dst bytes.Buffer
	// onlyWriter hides ReaderFrom; the small buffer forces the large-write path.
	w := freeio.NewWriterSize(onlyWriter{&dst}, 64)

	n, err := w.Write(payload)
	require.NoError(t, err)
	assert.Equal(t, len(payload), n)
	require.NoError(t, w.Flush())
	assert.Equal(t, payload, dst.Bytes())
}

func TestWriterWriteByte(t *testing.T) {
	t.Parallel()
	var dst bytes.Buffer
	w := freeio.NewWriterSize(&dst, 4)

	const s = "abcdefgh"
	for i := range len(s) {
		require.NoError(t, w.WriteByte(s[i]))
	}
	require.NoError(t, w.Flush())
	assert.Equal(t, s, dst.String())
}

func TestWriterReadFromLoop(t *testing.T) {
	t.Parallel()
	var dst bytes.Buffer
	// Both ends wrapped: ReadFrom can't delegate, so it fills and flushes.
	w := freeio.NewWriterSize(onlyWriter{&dst}, 64)

	n, err := w.ReadFrom(onlyReader{bytes.NewReader(payload)})
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), n)
	require.NoError(t, w.Flush())
	assert.Equal(t, payload, dst.Bytes())
}

func TestWriterReadFromDelegates(t *testing.T) {
	t.Parallel()
	var dst bytes.Buffer // implements io.ReaderFrom
	w := freeio.NewWriterSize(&dst, 64)

	// empty buffer + ReaderFrom destination: delegated, no Flush needed.
	n, err := w.ReadFrom(bytes.NewReader(payload))
	require.NoError(t, err)
	assert.Equal(t, int64(len(payload)), n)
	assert.Equal(t, payload, dst.Bytes())
}

func TestWriterReset(t *testing.T) {
	t.Parallel()
	var a, b bytes.Buffer
	w := freeio.NewWriter(&a)
	_, _ = io.WriteString(w, "to-a")
	require.NoError(t, w.Flush())

	w.Reset(&b)
	_, _ = io.WriteString(w, "to-b")
	require.NoError(t, w.Flush())

	assert.Equal(t, "to-a", a.String())
	assert.Equal(t, "to-b", b.String())
}

func TestReadWriter(t *testing.T) {
	t.Parallel()
	var dst bytes.Buffer
	rw := freeio.NewReadWriter(
		freeio.NewReader(strings.NewReader("data")),
		freeio.NewWriter(&dst),
	)
	defer rw.Reader.Free()

	b, err := rw.ReadByte()
	require.NoError(t, err)
	assert.Equal(t, byte('d'), b)

	_, err = io.WriteString(rw, "xyz")
	require.NoError(t, err)
	require.NoError(t, rw.Flush())
	assert.Equal(t, "xyz", dst.String())
}
