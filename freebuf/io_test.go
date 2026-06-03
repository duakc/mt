package freebuf

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadAll_DrainsToMultiPart(t *testing.T) {
	src := bytes.Repeat([]byte{0x42}, PartIncSize*2+33)
	buf, err := ReadAll(bytes.NewReader(src))
	defer buf.FreeMe()

	assert.NoError(t, err)
	assert.Equal(t, len(src), buf.Len())

	out := make([]byte, len(src))
	n, _ := buf.Read(out)
	assert.Equal(t, len(src), n)
	assert.Equal(t, src, out)
}

func TestReadAll_PropagatesError(t *testing.T) {
	buf, err := ReadAll(&dummyReader{size: 0, err: io.ErrClosedPipe})
	defer buf.FreeMe()

	assert.Equal(t, io.ErrClosedPipe, err)
	assert.Equal(t, 0, buf.Len())
}

func TestReadN_Exact_SerialUnderThreshold(t *testing.T) {
	src := strings.NewReader("hello world!!")
	buf, err := ReadN(src, 5)
	defer buf.FreeMe()

	assert.NoError(t, err)
	assert.IsType(t, &SerialBuffer{}, buf)
	assert.Equal(t, 5, buf.Len())

	out := make([]byte, 5)
	buf.Read(out)
	assert.Equal(t, "hello", string(out))
}

func TestReadN_MultiPartAboveThreshold(t *testing.T) {
	src := bytes.Repeat([]byte{1}, serialMultiPartCrossover*2)
	buf, err := ReadN(bytes.NewReader(src), serialMultiPartCrossover+1)
	defer buf.FreeMe()

	assert.NoError(t, err)
	assert.IsType(t, &MultiPartBuffer{}, buf)
	assert.Equal(t, serialMultiPartCrossover+1, buf.Len())
}

func TestReadN_ShortSource(t *testing.T) {
	buf, err := ReadN(strings.NewReader("abc"), 10)
	defer buf.FreeMe()

	assert.Equal(t, io.ErrUnexpectedEOF, err)
	assert.Equal(t, 3, buf.Len())
}

// ReadN with an empty source returns io.EOF and an empty buffer.
func TestReadN_EmptySource(t *testing.T) {
	buf, err := ReadN(strings.NewReader(""), 4)
	defer buf.FreeMe()

	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 0, buf.Len())
}

func TestReadFull_AppendsToExisting(t *testing.T) {
	dst := NewSerial()
	defer dst.FreeMe()
	dst.Write([]byte("prefix:"))

	n, err := ReadFull(strings.NewReader("payload!!"), dst, 7)
	assert.NoError(t, err)
	assert.Equal(t, int64(7), n)
	assert.Equal(t, "prefix:payload", string(dst.Bytes()))
}

func TestReadFull_ShortRead(t *testing.T) {
	dst := NewSerial()
	defer dst.FreeMe()

	n, err := ReadFull(strings.NewReader("ab"), dst, 5)
	assert.Equal(t, io.ErrUnexpectedEOF, err)
	assert.Equal(t, int64(2), n)
	assert.Equal(t, "ab", string(dst.Bytes()))
}

func TestReadFull_LimitedDstOverflow(t *testing.T) {
	dst := NewSerialLimited(4)
	defer dst.FreeMe()

	n, err := ReadFull(strings.NewReader("abcdefgh"), dst, 8)
	assert.Equal(t, io.ErrShortBuffer, err)
	assert.Equal(t, int64(4), n)
}
