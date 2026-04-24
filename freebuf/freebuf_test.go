package freebuf

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type dummyReader struct {
	size int
	err  error
}

func (r *dummyReader) Read(p []byte) (n int, err error) {
	return r.size, r.err
}

type dummyWriter struct {
	accepted int
	err      error
}

func (w *dummyWriter) Write(p []byte) (n int, err error) {
	if w.accepted == 0 {
		return 0, w.err
	}

	sub := min(w.accepted, len(p))
	w.accepted -= sub
	return sub, nil
}

func TestReadUntil(t *testing.T) {
	type Case struct {
		Input   io.Reader
		Buf     []byte
		OutputN int
		Err     error
	}
	cases := []Case{
		{Input: &dummyReader{size: 0}, Buf: make([]byte, 1), Err: io.ErrNoProgress},
		{Input: &dummyReader{size: 1}, Buf: make([]byte, 1), OutputN: 1},
		{Input: &dummyReader{size: 1}, OutputN: 0},
		{Input: &dummyReader{size: 0, err: io.EOF}, Buf: make([]byte, 1), OutputN: 0, Err: io.EOF},
		{
			Input: &dummyReader{size: 1, err: io.ErrUnexpectedEOF},
			Buf:   make([]byte, 1), OutputN: 1, Err: io.ErrUnexpectedEOF,
		},
	}

	for i := 0; i < len(cases); i++ {
		cc := cases[i]
		nn, err := ReadUntil(cc.Input, cc.Buf)
		assert.Equalf(t, cc.OutputN, nn, "testCase.Index=%d", i)
		if cc.Err == nil {
			assert.NoErrorf(t, err, "testCase.Index=%d", i)
		} else {
			assert.Equalf(t, cc.Err, err, "testCase.Index=%d", i)
		}
	}
}

func TestWriteUntil(t *testing.T) {
	type Case struct {
		Input io.Writer
		Buf   []byte

		OutputN int
		Err     error
	}
	cases := []Case{
		{Input: &dummyWriter{accepted: 100}, Buf: make([]byte, 1), OutputN: 1},
		{Input: &dummyWriter{accepted: 1}, Buf: make([]byte, 1), OutputN: 1},
		{
			Input: &dummyWriter{accepted: 1}, Buf: make([]byte, 10),
			OutputN: 1, Err: io.ErrShortWrite,
		},
		{
			Input: &dummyWriter{accepted: 0, err: io.ErrShortBuffer}, Buf: make([]byte, 1),
			OutputN: 0, Err: io.ErrShortBuffer,
		},
	}
	for i := 0; i < len(cases); i++ {
		cc := cases[i]
		nn, err := WriteUntil(cc.Input, cc.Buf)
		assert.Equalf(t, cc.OutputN, nn, "testCase.Index=%d", i)
		if cc.Err == nil {
			assert.NoErrorf(t, err, "testCase.Index=%d", i)
		} else {
			assert.Equalf(t, cc.Err, err, "testCase.Index=%d", i)
		}
	}
}
