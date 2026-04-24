package freebuf

import (
	"io"

	"github.com/duakc/mt/freebuf/internal"
)

type Buffer interface {
	io.ReadWriter
	io.StringWriter
	io.ByteWriter
	io.ByteReader
	io.ReaderFrom
	io.WriterTo

	Len() int
	FreeMe()
}

type CommonBuffer struct {
	parts []*bytePart
}

func (c *CommonBuffer) shrinkFront() {
	for len(c.parts) > 0 {
		p := c.parts[0]
		if p.len() != 0 {
			break
		}
		c.parts[0] = nil
		free(p)
		c.parts = c.parts[1:]
	}
}

func (c *CommonBuffer) inc(n int) {
	if n == 0 {
		return
	}
	bp := alloc(n)
	c.parts = append(c.parts, bp)
}

func (c *CommonBuffer) writeK(k int) []*bytePart {
	boot := len(c.parts) - 1
	for k > 0 {
		if len(c.parts) == 0 {
			c.inc(k)
		}
		bp := c.parts[len(c.parts)-1]
		k -= bp.freeSpace()
		c.inc(k)
	}

	if boot == -1 {
		boot = 0
	}
	return c.parts[boot:]
}

func (c *CommonBuffer) readK(k int) []*bytePart {
	i := 0
	for ; i < len(c.parts) && k > 0; i++ {
		bp := c.parts[i]
		k -= bp.len()
	}

	return c.parts[:i]
}

func (c *CommonBuffer) Read(p []byte) (n int, err error) {
	bps := c.readK(len(p))
	defer c.shrinkFront()
	for i := 0; i < len(bps); i++ {
		bp := bps[i]
		nn, readErr := bp.read(p[n:])
		n += mustRead("Read", nn, readErr)
	}
	if len(p) != n {
		err = io.EOF
	}
	return
}

func (c *CommonBuffer) Write(p []byte) (n int, err error) {
	bps := c.writeK(len(p))
	for i := 0; i < len(bps); i++ {
		nn, err := bps[i].write(p[n:])
		n += mustWrite("Write", nn, err)
	}
	return n, nil
}

func (c *CommonBuffer) WriteByte(b byte) error {
	bps := c.writeK(1)
	mustWrite("WriteByte", 0, bps[0].writeByte(b))
	return nil
}

func (c *CommonBuffer) WriteString(s string) (n int, err error) {
	bps := c.writeK(len(s))
	for i := 0; i < len(bps); i++ {
		nn, err := bps[i].writeString(s[n:])
		n += mustWrite("WriteString", nn, err)
	}
	return n, nil
}

func (c *CommonBuffer) ReadByte() (byte, error) {
	bps := c.readK(1)
	defer c.shrinkFront()
	if len(bps) == 0 {
		return 0, io.EOF
	}
	b, err := bps[0].readByte()
	mustRead("ReadByte", 0, err)
	return b, nil
}

func (c *CommonBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	readBuffer := internal.Get(maxSize)
	defer internal.Put(readBuffer)
	for {
		var nn int
		nn, err = ReadUntil(r, readBuffer)
		n += int64(nn)
		if err != nil {
			break
		}
		written, err := WriteUntil(c, readBuffer[:nn])
		mustRead("ReadFrom", written, err)
	}
	if err == io.EOF {
		err = nil
	}
	return n, err
}

func (c *CommonBuffer) WriteTo(w io.Writer) (n int64, err error) {
	writeBuffer := internal.Get(maxSize)
	defer internal.Put(writeBuffer)
	defer c.shrinkFront()
	for len(c.parts) == 0 {
		bps := c.readK(len(writeBuffer))
		if len(bps) == 0 {
			break
		}
		read := 0
		for i := 0; i < len(bps); i++ {
			bp := bps[i]
			nn, readErr := bp.read(writeBuffer[read:])
			read += mustRead("WriteTo", nn, readErr)
		}
		written := 0
		written, err = WriteUntil(w, writeBuffer)
		n += int64(written)
		if err != nil {
			break
		}

		c.shrinkFront()
	}
	return n, err
}

func (c *CommonBuffer) Len() int {
	v := 0
	for i := 0; i < len(c.parts); i++ {
		v += c.parts[i].len()
	}
	return v
}

func (c *CommonBuffer) FreeMe() {
	freeMulti(c.parts)
	for i := 0; i < len(c.parts); i++ {
		c.parts[i] = nil
	}
}
