package freeio

import (
	"io"
	"sync/atomic"
)

// CounterFunc receives byte counts as a copy progresses. A func, not an
// interface, so the per-chunk hot path stays a direct call; pass a slice to
// attach several at once. Never called with n <= 0.
type CounterFunc func(n int64)

// CounterReader is a reader that carries read counters and can be unwrapped to
// the reader beneath it. Copy unwraps it so the kernel fast paths still see the
// concrete fd-backed stream while the counters fire.
type CounterReader interface {
	io.Reader
	UnwrapReadCounter() (io.Reader, []CounterFunc)
}

// CounterWriter is the write-side counterpart of CounterReader.
type CounterWriter interface {
	io.Writer
	UnwrapWriteCounter() (io.Writer, []CounterFunc)
}

func UnwrapReadCounter(r io.Reader) (io.Reader, []CounterFunc) {
	var counters []CounterFunc
	for {
		cr, ok := r.(CounterReader)
		if !ok {
			return r, counters
		}
		var cs []CounterFunc
		r, cs = cr.UnwrapReadCounter()
		counters = append(counters, cs...)
	}
}

func UnwrapWriteCounter(w io.Writer) (io.Writer, []CounterFunc) {
	var counters []CounterFunc
	for {
		cw, ok := w.(CounterWriter)
		if !ok {
			return w, counters
		}
		var cs []CounterFunc
		w, cs = cw.UnwrapWriteCounter()
		counters = append(counters, cs...)
	}
}

func NewCounterReader(r io.Reader, counters ...CounterFunc) CounterReader {
	return &funcCounterReader{r: r, counters: counters}
}

func NewCounterWriter(w io.Writer, counters ...CounterFunc) CounterWriter {
	return &funcCounterWriter{w: w, counters: counters}
}

type funcCounterReader struct {
	r        io.Reader
	counters []CounterFunc
}

func (r *funcCounterReader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if n > 0 {
		for _, c := range r.counters {
			c(int64(n))
		}
	}
	return
}

func (r *funcCounterReader) UnwrapReadCounter() (io.Reader, []CounterFunc) {
	return r.r, r.counters
}

type funcCounterWriter struct {
	w        io.Writer
	counters []CounterFunc
}

func (w *funcCounterWriter) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	if n > 0 {
		for _, c := range w.counters {
			c(int64(n))
		}
	}
	return
}

func (w *funcCounterWriter) UnwrapWriteCounter() (io.Writer, []CounterFunc) {
	return w.w, w.counters
}

type CountedReader struct {
	R io.Reader
	N atomic.Int64
}

func (r *CountedReader) Read(p []byte) (n int, err error) {
	n, err = r.R.Read(p)
	if n > 0 {
		r.N.Add(int64(n))
	}
	return
}

func (r *CountedReader) UnwrapReadCounter() (io.Reader, []CounterFunc) {
	return r.R, []CounterFunc{r.add}
}

func (r *CountedReader) add(n int64) { r.N.Add(n) }

type CountedWriter struct {
	W io.Writer
	N atomic.Int64
}

func (w *CountedWriter) Write(p []byte) (n int, err error) {
	n, err = w.W.Write(p)
	if n > 0 {
		w.N.Add(int64(n))
	}
	return
}

func (w *CountedWriter) UnwrapWriteCounter() (io.Writer, []CounterFunc) {
	return w.W, []CounterFunc{w.add}
}

func (w *CountedWriter) add(n int64) { w.N.Add(n) }

type CountedReadWriter struct {
	*CountedWriter
	*CountedReader
}
