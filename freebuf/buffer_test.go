package freebuf

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFreeBuf_New(t *testing.T) {
	buf := New(1024)
	assert.NotNil(t, buf)
	assert.NotNil(t, buf.part)
	assert.Equal(t, 1024, cap(buf.part.data))
	assert.Equal(t, 0, buf.Len())
	buf.FreeMe()
}

func TestFreeBuf_ReadWrite(t *testing.T) {
	buf := New(16)
	defer buf.FreeMe()

	n, err := buf.Write([]byte("hello"))
	assert.Equal(t, 5, n)
	assert.NoError(t, err)
	assert.Equal(t, 5, buf.Len())

	data := make([]byte, 5)
	n, err = buf.Read(data)
	assert.Equal(t, 5, n)
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(data))
	assert.Equal(t, 0, buf.Len())

	n, err = buf.Read(make([]byte, 1))
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)
}

func TestFreeBuf_WriteByte(t *testing.T) {
	buf := New(3)
	defer buf.FreeMe()

	assert.NoError(t, buf.WriteByte('a'))
	assert.NoError(t, buf.WriteByte('b'))
	assert.NoError(t, buf.WriteByte('c'))
	assert.Equal(t, 3, buf.Len())

	err := buf.WriteByte('d')
	assert.Equal(t, io.ErrShortBuffer, err)
}

func TestFreeBuf_ReadByte(t *testing.T) {
	buf := New(3)
	defer buf.FreeMe()

	buf.Write([]byte("ab"))
	b, err := buf.ReadByte()
	assert.NoError(t, err)
	assert.Equal(t, byte('a'), b)

	b, err = buf.ReadByte()
	assert.NoError(t, err)
	assert.Equal(t, byte('b'), b)

	_, err = buf.ReadByte()
	assert.Equal(t, io.EOF, err)
}

func TestFreeBuf_WriteString(t *testing.T) {
	buf := New(8)
	defer buf.FreeMe()

	n, err := buf.WriteString("free")
	assert.Equal(t, 4, n)
	assert.NoError(t, err)
	assert.Equal(t, 4, buf.Len())
	buf.WriteString("buf!")
	n, err = buf.WriteString("overflow")
	assert.Equal(t, 0, n)
	assert.Equal(t, io.ErrShortBuffer, err)
}

func TestFreeBuf_ReadFrom(t *testing.T) {
	buf := New(10)
	defer buf.FreeMe()

	r := strings.NewReader("1234567890")
	n, err := buf.ReadFrom(r)
	assert.Equal(t, int64(10), n)
	assert.NoError(t, err)
	assert.Equal(t, 10, buf.Len())
	r2 := strings.NewReader("extra")
	n, err = buf.ReadFrom(r2)
	assert.Equal(t, int64(0), n)
	assert.Equal(t, io.ErrShortBuffer, err)
}

func TestFreeBuf_WriteTo(t *testing.T) {
	buf := New(10)
	defer buf.FreeMe()

	buf.Write([]byte("0123456789"))
	var out bytes.Buffer
	n, err := buf.WriteTo(&out)
	assert.Equal(t, int64(10), n)
	assert.NoError(t, err)
	assert.Equal(t, "0123456789", out.String())
	assert.Equal(t, 0, buf.Len())

	n, err = buf.WriteTo(&out)
	assert.Equal(t, int64(0), n)
	assert.Equal(t, io.EOF, err)
}

func TestFreeBuf_Len(t *testing.T) {
	buf := New(64)
	defer buf.FreeMe()

	assert.Equal(t, 0, buf.Len())
	buf.Write([]byte("data"))
	assert.Equal(t, 4, buf.Len())
	buf.Read(make([]byte, 2))
	assert.Equal(t, 2, buf.Len())
}

func TestFreeBuf_FreeMe(t *testing.T) {
	buf := New(10)
	buf.FreeMe()
	assert.Nil(t, buf.part)
}

func TestFreeBuf_FreeBytes(t *testing.T) {
	buf := New(10)
	defer buf.FreeMe()
	assert.Equal(t, 10, len(buf.FreeBytes()))
	nn, err := buf.Write([]byte{0, 0, 0})
	assert.Equal(t, 3, nn)
	assert.NoError(t, err)
	assert.Equal(t, 7, len(buf.FreeBytes()))
	_, err = buf.ReadByte()
	assert.NoError(t, err)
	assert.Equal(t, 7, len(buf.FreeBytes()))
}

func TestFreeBuf_Bytes(t *testing.T) {
	buf := New(10)
	defer buf.FreeMe()
	nn, err := buf.Write([]byte{0, 1, 2})
	assert.Equal(t, 3, nn)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0, 1, 2}, buf.Bytes())
	_, err = buf.ReadByte()
	assert.NoError(t, err)
	assert.Equal(t, []byte{1, 2}, buf.Bytes())
}

func TestFreeBuf_Truncated(t *testing.T) {
	buf := New(10)
	defer buf.FreeMe()
	n := copy(buf.FreeBytes(), []byte{0, 1, 2})
	assert.Equal(t, 3, n)
	buf.Truncated(3)
	assert.Equal(t, []byte{0, 1, 2}, buf.Bytes())
}

func TestFreeBuf_Truncated2(t *testing.T) {
	buf := New(10)
	defer buf.FreeMe()
	nn, err := buf.Write([]byte{0, 1, 2})
	assert.Equal(t, 3, nn)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0, 1, 2}, buf.Bytes())
	buf.Truncated(2)
	assert.Equal(t, []byte{0, 1}, buf.Bytes())
	_, err = buf.ReadByte()
	assert.NoError(t, err)
	// overflowed size
	buf.Truncated(100)
	assert.Equal(t, []byte{1, 2, 0, 0, 0, 0, 0, 0, 0}, buf.Bytes())
}
