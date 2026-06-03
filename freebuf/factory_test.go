package freebuf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// NewExcept picks SerialBuffer up to and including the pool ceiling, and
// MultiPartBuffer once we know the payload will outgrow it. The decision is
// made without allocating any backing storage.
func TestNewExcept_PicksByThreshold(t *testing.T) {
	cases := []struct {
		except int
		want   Buffer
	}{
		{0, &SerialBuffer{}},
		{1024, &SerialBuffer{}},
		{serialMultiPartCrossover, &SerialBuffer{}},
		{serialMultiPartCrossover + 1, &MultiPartBuffer{}},
		{4 * serialMultiPartCrossover, &MultiPartBuffer{}},
	}
	for _, c := range cases {
		got := NewExcept(c.except)
		assert.IsTypef(t, c.want, got, "except=%d", c.except)
		got.FreeMe()
	}
}

// NewExcept does not pre-allocate; the returned buffer must still grow on
// demand if the actual payload differs from the hint.
func TestNewExcept_NoEagerAllocation(t *testing.T) {
	buf := NewExcept(serialMultiPartCrossover - 1).(*SerialBuffer)
	defer buf.FreeMe()
	assert.Nil(t, buf.data)

	mp := NewExcept(serialMultiPartCrossover + 1).(*MultiPartBuffer)
	defer mp.FreeMe()
	assert.Equal(t, 0, len(mp.parts))
}
