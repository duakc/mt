package freebuf

import (
	"io"

	"github.com/duakc/mt/freebuf/internal"
)

var _ Buffer = (*MultiPartBuffer)(nil)

type MultiPartBuffer struct {
	parts []*bytePart
}

func NewMultiPart() *MultiPartBuffer {
	return new(MultiPartBuffer)
}

func (c *MultiPartBuffer) shrinkFront() {
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

func (c *MultiPartBuffer) inc(n int) {
	if n == 0 {
		return
	}
	bp := alloc(n)
	c.parts = append(c.parts, bp)
}

func (c *MultiPartBuffer) writeK(k int) []*bytePart {
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

func (c *MultiPartBuffer) readK(k int) []*bytePart {
	i := 0
	for ; i < len(c.parts) && k > 0; i++ {
		bp := c.parts[i]
		k -= bp.len()
	}

	return c.parts[:i]
}

func (c *MultiPartBuffer) Read(p []byte) (n int, err error) {
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

func (c *MultiPartBuffer) Write(p []byte) (n int, err error) {
	bps := c.writeK(len(p))
	for i := 0; i < len(bps); i++ {
		nn, err := bps[i].write(p[n:])
		n += mustWrite("Write", nn, err)
	}
	return n, nil
}

func (c *MultiPartBuffer) WriteByte(b byte) error {
	bps := c.writeK(1)
	mustWrite("WriteByte", 0, bps[0].writeByte(b))
	return nil
}

func (c *MultiPartBuffer) WriteString(s string) (n int, err error) {
	bps := c.writeK(len(s))
	for i := 0; i < len(bps); i++ {
		nn, err := bps[i].writeString(s[n:])
		n += mustWrite("WriteString", nn, err)
	}
	return n, nil
}

func (c *MultiPartBuffer) ReadByte() (byte, error) {
	bps := c.readK(1)
	defer c.shrinkFront()
	if len(bps) == 0 {
		return 0, io.EOF
	}
	b, err := bps[0].readByte()
	mustRead("ReadByte", 0, err)
	return b, nil
}

func (c *MultiPartBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	for {
		if len(c.parts) == 0 {
			c.inc(PartReadIncSize)
		}
		bp := c.parts[len(c.parts)-1]
		once, err := bp.readFromOnce(r)
		n += int64(once)
		if err == errBytePartReadFromOnceFull {
			c.inc(PartReadIncSize)
			err = nil
		}
		if err != nil {
			break
		}
	}
	if err == io.EOF {
		err = nil
	}
	return n, err
}

func (c *MultiPartBuffer) WriteTo(w io.Writer) (n int64, err error) {
	writeBuffer := internal.Get(PartReadIncSize)
	defer internal.Put(writeBuffer)
	for ; len(c.parts) != 0; c.shrinkFront() {
		nn, readErr := c.Read(writeBuffer)
		if readErr != nil && readErr != io.EOF {
			err = readErr
			break
		}
		written := 0
		written, err = WriteUntil(w, writeBuffer[:nn])
		n += int64(written)
		if err != nil || readErr == io.EOF {
			break
		}
	}
	return n, err
}

func (c *MultiPartBuffer) Len() int {
	v := 0
	for i := 0; i < len(c.parts); i++ {
		v += c.parts[i].len()
	}
	return v
}

func (c *MultiPartBuffer) FreeMe() {
	freeMulti(c.parts)
	for i := 0; i < len(c.parts); i++ {
		c.parts[i] = nil
	}
}
