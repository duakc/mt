package freebuf

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

// Grow reserves at least n bytes of free space so subsequent Writes within
// that budget don't trigger further allocations. Verifies the impl-specific
// invariant (PartCount unchanged); the generic Grow contract is covered by
// the black-box tests.
func TestMultiPartBuffer_Grow_ReservesSpace(t *testing.T) {
	mp := NewMultiPart()
	defer mp.FreeMe()

	mp.Grow(PartMinimalSize * 5)
	partsBefore := mp.PartCount()

	mp.Write(bytes.Repeat([]byte{1}, PartMinimalSize*3))
	assert.Equal(t, partsBefore, mp.PartCount())
}

