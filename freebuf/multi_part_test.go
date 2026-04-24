package freebuf

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiPartBuffer_New(t *testing.T) {
	mp := NewMultiPart()
	assert.NotNil(t, mp)
	assert.Equal(t, 0, len(mp.parts))
}

func TestMultiPartBuffer_WriteRead(t *testing.T) {
	mp := NewMultiPart()

	data := make([]byte, PartReadIncSize*3)
	for i := range data {
		data[i] = byte(i % 256)
	}

	n, err := mp.Write(data)
	assert.Equal(t, len(data), n)
	assert.NoError(t, err)

	out := make([]byte, len(data))
	n, err = mp.Read(out)
	assert.Equal(t, len(data), n)
	assert.NoError(t, err)
	assert.Equal(t, data, out)
}

func TestMultiPartBuffer_WriteByte(t *testing.T) {
	mp := NewMultiPart()

	for i := 0; i < 10000; i++ {
		assert.NoError(t, mp.WriteByte(byte(i)))
	}

	for i := 0; i < 10000; i++ {
		b, err := mp.ReadByte()
		assert.NoError(t, err)
		assert.Equal(t, byte(i), b)
	}

	_, err := mp.ReadByte()
	assert.Equal(t, io.EOF, err)
}

func TestMultiPartBuffer_WriteString(t *testing.T) {
	mp := NewMultiPart()

	largeStr := strings.Repeat("data", 5000)
	n, err := mp.WriteString(largeStr)
	assert.Equal(t, len(largeStr), n)
	assert.NoError(t, err)

	out := make([]byte, len(largeStr))
	n, err = mp.Read(out)
	assert.Equal(t, len(largeStr), n)
	assert.NoError(t, err)
	assert.Equal(t, largeStr, string(out))
}

func TestMultiPartBuffer_ReadFrom(t *testing.T) {
	mp := NewMultiPart()

	data := make([]byte, PartReadIncSize*2+100)
	for i := range data {
		data[i] = byte(i % 256)
	}
	r := bytes.NewReader(data)

	n, err := mp.ReadFrom(r)
	assert.Equal(t, int64(len(data)), n)
	assert.NoError(t, err)

	out := make([]byte, len(data))
	nn, _ := mp.Read(out)
	assert.Equal(t, len(data), nn)
	assert.Equal(t, data, out)
}

func TestMultiPartBuffer_WriteTo(t *testing.T) {
	mp := NewMultiPart()

	data := make([]byte, PartReadIncSize*2+100)
	for i := range data {
		data[i] = byte(i % 256)
	}
	mp.Write(data)

	var out bytes.Buffer
	n, err := mp.WriteTo(&out)
	assert.Equal(t, int64(len(data)), n)
	assert.NoError(t, err)
	assert.Equal(t, data, out.Bytes())
}

func TestMultiPartBuffer_EmptyRead(t *testing.T) {
	mp := NewMultiPart()
	buf := make([]byte, 10)
	n, err := mp.Read(buf)
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)
}

func TestMultiPartBuffer_Interface(t *testing.T) {
	mp := NewMultiPart()

	mp.Write([]byte("small"))
	out := make([]byte, 5)
	mp.Read(out)
	assert.Equal(t, "small", string(out))

	mp.WriteByte('A')
	mp.WriteString("BC")
	out2 := make([]byte, 3)
	mp.Read(out2)
	assert.Equal(t, "ABC", string(out2))
}
