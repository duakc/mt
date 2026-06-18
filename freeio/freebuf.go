package freeio

import (
	"io"

	"github.com/duakc/mt/freebuf"
)

func ReadAll(r io.Reader) (freebuf.Buffer, error) {
	return freebuf.ReadAll(r)
}

func ReadN(r io.Reader, n int) (freebuf.Buffer, error) {
	return freebuf.ReadN(r, n)
}

func ReadFull(r io.Reader, dst freebuf.Buffer, n int) (read int64, err error) {
	return freebuf.ReadFull(r, dst, n)
}
