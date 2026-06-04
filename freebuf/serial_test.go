package freebuf

import (
	"bytes"
	"io"
	"testing"

	"github.com/duakc/mt/freebuf/internal"

	"github.com/stretchr/testify/assert"
)

func TestSerialBuffer_GrowsAcrossPoolBoundaries(t *testing.T) {
	b := NewSerial()
	defer b.FreeMe()

	want := make([]byte, internal.MaxAllocatableSize*2+777)
	for i := range want {
		want[i] = byte(i)
	}
	n, err := b.Write(want)
	assert.NoError(t, err)
	assert.Equal(t, len(want), n)

	got := make([]byte, len(want))
	n, _ = b.Read(got)
	assert.Equal(t, len(want), n)
	assert.Equal(t, want, got)
}

func TestSerialBuffer_CompactsWhenDrained(t *testing.T) {
	b := NewSerial()
	defer b.FreeMe()

	b.Write([]byte("first"))
	b.Read(make([]byte, 5))

	capBefore := cap(b.data)
	b.Write([]byte("second"))
	assert.Equal(t, capBefore, cap(b.data))
	assert.Equal(t, "second", string(b.Bytes()))
}

func TestSerialBuffer_CompactsBeforeGrowing(t *testing.T) {
	b := NewSerial()
	defer b.FreeMe()

	b.Write(bytes.Repeat([]byte("A"), PartMinimalSize))
	assert.Equal(t, PartMinimalSize, cap(b.data))

	b.Read(make([]byte, PartMinimalSize/2))
	b.Write(bytes.Repeat([]byte("B"), PartMinimalSize/2))

	assert.Equal(t, PartMinimalSize, cap(b.data))
	assert.Equal(t, PartMinimalSize, b.Len())
}

func TestSerialBuffer_FreeBytes_Grow_Truncated(t *testing.T) {
	b := NewSerial()
	defer b.FreeMe()

	b.Grow(64)
	assert.GreaterOrEqual(t, len(b.FreeBytes()), 64)

	copy(b.FreeBytes(), []byte{10, 20, 30})
	b.Truncated(3)
	assert.Equal(t, []byte{10, 20, 30}, b.Bytes())

	b.Truncated(1)
	assert.Equal(t, []byte{10}, b.Bytes())
}

func TestSerialBuffer_ReadFrom(t *testing.T) {
	b := NewSerial()
	defer b.FreeMe()

	data := make([]byte, PartIncSize*3+13)
	for i := range data {
		data[i] = byte(i ^ 0x5a)
	}
	n, err := b.ReadFrom(bytes.NewReader(data))
	assert.NoError(t, err)
	assert.Equal(t, int64(len(data)), n)

	out := make([]byte, len(data))
	nn, _ := b.Read(out)
	assert.Equal(t, len(data), nn)
	assert.Equal(t, data, out)
}

func TestSerialBuffer_ReadFrom_PropagatesError(t *testing.T) {
	b := NewSerial()
	defer b.FreeMe()

	n, err := b.ReadFrom(&dummyReader{size: 0, err: io.ErrUnexpectedEOF})
	assert.Equal(t, int64(0), n)
	assert.Equal(t, io.ErrUnexpectedEOF, err)
}

func TestSerialBuffer_WriteTo_PropagatesError(t *testing.T) {
	b := NewSerial()
	defer b.FreeMe()

	b.Write([]byte("hello"))
	n, err := b.WriteTo(&dummyWriter{accepted: 0, err: io.ErrClosedPipe})
	assert.Equal(t, int64(0), n)
	assert.Equal(t, io.ErrClosedPipe, err)
}

func TestSerialBuffer_WriteTo_Empty(t *testing.T) {
	b := NewSerial()
	defer b.FreeMe()

	var out bytes.Buffer
	n, err := b.WriteTo(&out)
	assert.Equal(t, int64(0), n)
	assert.Equal(t, io.EOF, err)
}

func TestSerialBuffer_Limited_Overflow(t *testing.T) {
	b := NewSerialLimited(8)
	defer b.FreeMe()

	assert.Equal(t, 8, len(b.FreeBytes()))

	n, err := b.Write([]byte("0123456"))
	assert.NoError(t, err)
	assert.Equal(t, 7, n)

	n, err = b.Write([]byte("XYZ"))
	assert.Equal(t, io.ErrShortBuffer, err)
	assert.Equal(t, 1, n) // only one byte fits

	out := make([]byte, 8)
	nn, _ := b.Read(out)
	assert.Equal(t, 8, nn)
	assert.Equal(t, "0123456X", string(out))
}

func TestSerialBuffer_Limited_RecyclesAfterRead(t *testing.T) {
	b := NewSerialLimited(4)
	defer b.FreeMe()

	n, _ := b.Write([]byte("abcd"))
	assert.Equal(t, 4, n)

	_, err := b.Write([]byte("e"))
	assert.Equal(t, io.ErrShortBuffer, err)

	out := make([]byte, 2)
	b.Read(out)
	assert.Equal(t, "ab", string(out))

	n, err = b.Write([]byte("ef"))
	assert.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestSerialBuffer_Limited_WriteByteFull(t *testing.T) {
	b := NewSerialLimited(2)
	defer b.FreeMe()

	assert.NoError(t, b.WriteByte('a'))
	assert.NoError(t, b.WriteByte('b'))
	assert.Equal(t, io.ErrShortBuffer, b.WriteByte('c'))
}

func TestSerialBuffer_Limited_ReadFromFull(t *testing.T) {
	b := NewSerialLimited(4)
	defer b.FreeMe()

	b.Write([]byte("xxxx"))
	n, err := b.ReadFrom(bytes.NewReader([]byte("more")))
	assert.Equal(t, int64(0), n)
	assert.Equal(t, io.ErrShortBuffer, err)
}

func TestSerialBuffer_Reset_KeepsBacking(t *testing.T) {
	b := NewSerial()
	defer b.FreeMe()

	b.Write(bytes.Repeat([]byte{'a'}, PartMinimalSize))
	capBefore := cap(b.data)
	assert.Greater(t, capBefore, 0)

	b.Reset()
	assert.Equal(t, 0, b.Len())
	assert.Equal(t, capBefore, cap(b.data))

	// Subsequent write fits without growing.
	b.Write([]byte("x"))
	assert.Equal(t, capBefore, cap(b.data))
}

func TestSerialBuffer_Next_ZeroCopy(t *testing.T) {
	b := NewSerial()
	defer b.FreeMe()

	b.Write([]byte("hello world"))
	first := b.Next(5)
	assert.Equal(t, []byte("hello"), first)
	assert.Equal(t, 6, b.Len())

	// Over-asking returns whatever remains, not a short read error.
	rest := b.Next(100)
	assert.Equal(t, []byte(" world"), rest)
	assert.Equal(t, 0, b.Len())

	// Empty buffer returns nil.
	assert.Nil(t, b.Next(5))
}

// Copy on a populated buffer must return an independent buffer holding the
// same unread bytes; mutating either side must not affect the other.
func TestSerialBuffer_Copy_Independent(t *testing.T) {
	src := NewSerial()
	defer src.FreeMe()
	src.Write([]byte("hello"))

	cp := src.Copy()
	defer cp.FreeMe()
	assert.Equal(t, 5, cp.Len())

	src.Write([]byte("!!!"))
	cp.Write([]byte("???"))
	assert.Equal(t, 8, src.Len())
	assert.Equal(t, 8, cp.Len())

	srcOut := make([]byte, 8)
	cpOut := make([]byte, 8)
	src.Read(srcOut)
	cp.Read(cpOut)
	assert.Equal(t, "hello!!!", string(srcOut))
	assert.Equal(t, "hello???", string(cpOut))
}

// Copy on a limited buffer must preserve the limit so the copy enforces the
// same ceiling on subsequent writes.
func TestSerialBuffer_Copy_PreservesLimit(t *testing.T) {
	src := NewSerialLimited(8)
	defer src.FreeMe()
	src.Write([]byte("12345"))

	cp := src.Copy()
	defer cp.FreeMe()
	assert.Equal(t, 5, cp.Len())

	// 3 more bytes fit, 4th overflows the limit.
	n, err := cp.Write([]byte("xyzW"))
	assert.Equal(t, 3, n)
	assert.Equal(t, io.ErrShortBuffer, err)
}
