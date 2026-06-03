package freebuf

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Data large enough to span many parts must round-trip cleanly through the
// chunked storage.
func TestMultiPartBuffer_WriteRead_MultiPart(t *testing.T) {
	mp := NewMultiPart()
	defer mp.FreeMe()

	data := make([]byte, PartMinimalSize*5+13)
	for i := range data {
		data[i] = byte(i % 256)
	}
	n, err := mp.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)

	out := make([]byte, len(data))
	n, err = mp.Read(out)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, out)
	assert.Equal(t, 0, mp.Len())
}

// Regression
func TestMultiPartBuffer_WriteByte_FullTailDoesNotLoseBytes(t *testing.T) {
	mp := NewMultiPart()
	defer mp.FreeMe()

	mp.Write(bytes.Repeat([]byte{'A'}, PartMinimalSize))
	assert.NoError(t, mp.WriteByte('Z'))
	assert.Equal(t, PartMinimalSize+1, mp.Len())

	out := make([]byte, PartMinimalSize+1)
	n, err := mp.Read(out)
	assert.NoError(t, err)
	assert.Equal(t, PartMinimalSize+1, n)
	assert.Equal(t, byte('Z'), out[PartMinimalSize])
}

func TestMultiPartBuffer_WriteByte_NoStrandedParts(t *testing.T) {
	mp := NewMultiPart()
	defer mp.FreeMe()

	for range PartMinimalSize {
		assert.NoError(t, mp.WriteByte('x'))
	}
	assert.Equal(t, 1, len(mp.parts))
}

// Regression
func TestMultiPartBuffer_ReadFrom_PropagatesError(t *testing.T) {
	mp := NewMultiPart()
	defer mp.FreeMe()

	n, err := mp.ReadFrom(&dummyReader{size: 0, err: io.ErrUnexpectedEOF})
	assert.Equal(t, int64(0), n)
	assert.Equal(t, io.ErrUnexpectedEOF, err)
}

func TestMultiPartBuffer_HeadCompactsOverTime(t *testing.T) {
	mp := NewMultiPart()
	defer mp.FreeMe()

	chunk := bytes.Repeat([]byte("y"), PartMinimalSize)
	out := make([]byte, PartMinimalSize)
	for range 1000 {
		mp.Write(chunk)
		mp.Read(out)
	}
	assert.Equal(t, 0, mp.Len())
	assert.LessOrEqual(t, len(mp.parts), 16)
}

func TestMultiPartBuffer_ReadFrom(t *testing.T) {
	mp := NewMultiPart()
	defer mp.FreeMe()

	data := make([]byte, PartIncSize*2+100)
	for i := range data {
		data[i] = byte(i % 256)
	}
	n, err := mp.ReadFrom(bytes.NewReader(data))
	assert.NoError(t, err)
	assert.Equal(t, int64(len(data)), n)

	out := make([]byte, len(data))
	nn, _ := mp.Read(out)
	assert.Equal(t, len(data), nn)
	assert.Equal(t, data, out)
}

func TestMultiPartBuffer_WriteTo_Empty(t *testing.T) {
	mp := NewMultiPart()
	defer mp.FreeMe()

	var out bytes.Buffer
	n, err := mp.WriteTo(&out)
	assert.Equal(t, int64(0), n)
	assert.Equal(t, io.EOF, err)
}

func TestMultiPartBuffer_Reset(t *testing.T) {
	mp := NewMultiPart()
	defer mp.FreeMe()

	mp.Write(bytes.Repeat([]byte{1}, PartMinimalSize*3))
	assert.Greater(t, mp.PartCount(), 0)

	mp.Reset()
	assert.Equal(t, 0, mp.Len())
	assert.Equal(t, 0, mp.PartCount())

	// Slice header should still be reusable for subsequent writes.
	mp.Write([]byte("hi"))
	assert.Equal(t, 2, mp.Len())
	assert.Equal(t, 1, mp.PartCount())
}

func TestMultiPartBuffer_Chunks_ZeroCopyTraversal(t *testing.T) {
	mp := NewMultiPart()
	defer mp.FreeMe()

	payload := bytes.Repeat([]byte{0xAB}, PartMinimalSize*2+100)
	mp.Write(payload)

	var total int
	var pieces int
	for chunk := range mp.Chunks() {
		total += len(chunk)
		pieces++
	}
	assert.Equal(t, len(payload), total)
	assert.GreaterOrEqual(t, pieces, 2)

	out := make([]byte, len(payload))
	n, _ := mp.Read(out)
	assert.Equal(t, len(payload), n)
	assert.Equal(t, payload, out)
}

func TestMultiPartBuffer_Chunks_EarlyBreak(t *testing.T) {
	mp := NewMultiPart()
	defer mp.FreeMe()

	mp.Write(bytes.Repeat([]byte{1}, PartMinimalSize*4))

	seen := 0
	for range mp.Chunks() {
		seen++
		break
	}
	assert.Equal(t, 1, seen)
}
