package freebuf

import (
	"fmt"
	"io"
	"sync"

	"github.com/duakc/mt/freebuf/internal"
)

var bytePartPool = sync.Pool{
	New: func() any {
		return new(bytePart)
	},
}

const (
	maxSize = internal.MaxAllocatableSize
)

func alloc(size int) *bytePart {
	bp := bytePartPool.Get().(*bytePart)
	if size <= maxSize {
		buffer := internal.Get(size)
		buffer = buffer[:cap(buffer)]
		bp.managed = true
		bp.data = buffer
	} else {
		bp.managed = false
		bp.data = make([]byte, size)
	}
	return bp
}

func free(b *bytePart) {
	if b == nil || !b.managed || b.data == nil || len(b.data) > maxSize {
		return
	}

	b.reset()
	internal.Put(b.data)
	b.data = nil
	bytePartPool.Put(b)
}

func freeMulti(bs []*bytePart) {
	for i := 0; i < len(bs); i++ {
		free(bs[i])
	}
}

type bytePart struct {
	data []byte

	r, w    int
	managed bool
}

func (h *bytePart) reset() {
	clear(h.data)
	h.r = 0
	h.w = 0
}

func (h *bytePart) write(b []byte) (int, error) {
	if h.freeSpace() == 0 {
		return 0, io.ErrShortBuffer
	}

	to := min(h.w+len(b), len(h.data))
	nn := copy(h.data[h.w:to], b)
	h.w += nn

	return nn, nil
}

func (h *bytePart) writeByte(b byte) error {
	if h.freeSpace() == 0 {
		return io.ErrShortBuffer
	}
	h.w++
	h.data[h.w] = b
	return nil
}

func (h *bytePart) readByte() (byte, error) {
	if h.len() == 0 {
		return 0, io.EOF
	}
	b := h.data[h.r]
	h.r++
	return b, nil
}

func (h *bytePart) writeString(s string) (int, error) {
	if h.freeSpace() == 0 {
		return 0, io.ErrShortBuffer
	}

	to := min(h.w+len(s), len(h.data))
	nn := copy(h.data[h.w:to], s)
	h.w += nn

	return nn, nil
}

func (h *bytePart) read(b []byte) (int, error) {
	to := min(h.r+len(b), h.w)
	nn := copy(b[:h.w-h.r], h.data[h.r:to])
	h.r += nn
	var err error
	if nn != len(b) {
		err = io.EOF
	}
	return nn, err
}

func (h *bytePart) len() int {
	return h.w - h.r
}

func (h *bytePart) freeSpace() int {
	return len(h.data) - h.w
}

func mustWrite(what string, n int, err error) int {
	if err != nil {
		panic(fmt.Sprintf("%s: written=%d: %s", what, n, err.Error()))
	}
	return n
}

func mustRead(what string, n int, err error) int {
	if err != nil {
		panic(fmt.Sprintf("%s: read=%d: %s", what, n, err.Error()))
	}
	return n
}
