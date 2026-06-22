//go:build linux

package freeio

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

// Uses only the stdlib syscall package (no golang.org/x/sys dependency).
// syscall.Splice returns int on some GOARCHs and int64 on others, so every
// result is funnelled through int(...) immediately rather than a typed variable.

const (
	// maxSpliceSize is the most data a single splice(2) asks the kernel to move.
	// splice routes through a pipe, and 1MB is the default ceiling for the pipe
	// buffer (fs.pipe-max-size), so the pipe is resized to match below.
	maxSpliceSize = 1 << 20

	// Values from the Linux uapi headers (not exported by package syscall):
	spliceFMove     = 0x1   // SPLICE_F_MOVE      <linux/splice.h>
	spliceFNonblock = 0x2   // SPLICE_F_NONBLOCK  <linux/splice.h>: pipe ops nonblocking
	fSetPipeSz      = 0x407 // F_SETPIPE_SZ       <linux/fcntl.h> (1031)
)

// spliceConn moves bytes from src to dst entirely in the kernel via a pooled
// pipe and splice(2), counting per syscall so accounting stays real-time. It
// reports handled=false (with n==0) when splice cannot be used for these
// descriptors, so the caller can fall back to a generic copy.
func spliceConn(srcConn, dstConn syscall.Conn, writeCounters, readCounters []CounterFunc) (n int64, handled bool, err error) {
	srcRaw, e := srcConn.SyscallConn()
	if e != nil {
		return 0, false, nil
	}
	dstRaw, e := dstConn.SyscallConn()
	if e != nil {
		return 0, false, nil
	}

	pipe := getPipe()
	if pipe == nil {
		return 0, false, nil
	}
	defer putPipe(pipe)

	var (
		readN       int
		readErr     error
		writeErr    error
		writeRemain int
	)

	// src -> pipe. One splice per ready event; EAGAIN parks until readable.
	readFunc := func(fd uintptr) (done bool) {
		nn, se := syscall.Splice(int(fd), nil, pipe.wfd, nil, maxSpliceSize, spliceFNonblock)
		readN = int(nn)
		readErr = se
		return readErr != syscall.EAGAIN
	}
	// pipe -> dst. Drain the whole window; EAGAIN parks until writable.
	writeFunc := func(fd uintptr) (done bool) {
		for writeRemain > 0 {
			nn, se := syscall.Splice(pipe.rfd, nil, int(fd), nil, writeRemain, spliceFNonblock|spliceFMove)
			if se != nil {
				writeErr = se
				return writeErr != syscall.EAGAIN
			}
			writeRemain -= int(nn)
			pipe.data -= int(nn)
		}
		return true
	}

	for {

		// Read phase.
		if err = srcRaw.Read(readFunc); err != nil {
			return n, true, err
		}

		if readErr != nil {
			// EINVAL/ENOSYS mean splice is not applicable to these descriptors
			// (wrong fd type, old kernel). Only retreat to the generic path if
			// nothing has been moved yet — once bytes are in flight we own them.
			if (readErr == syscall.EINVAL || readErr == syscall.ENOSYS) && n == 0 {
				return 0, false, nil
			}
			return n, true, fmt.Errorf("freeio: splice read: %w", readErr)
		}

		if readN == 0 {
			return n, true, nil // src EOF
		}

		// Write phase. The pipe now holds readN bytes; track them so a pipe
		// abandoned mid-drain (on error) is destroyed instead of pooled dirty.
		pipe.data = readN
		writeRemain = readN
		if err = dstRaw.Write(writeFunc); err != nil {
			return n, true, err
		}
		if writeErr != nil {
			return n, true, fmt.Errorf("freeio: splice write: %w", writeErr)
		}

		// read == write for a splice copy: feed both counters the same delta.
		for _, c := range readCounters {
			c(int64(readN))
		}
		for _, c := range writeCounters {
			c(int64(readN))
		}

		n += int64(readN)
	}
}

type splicePipeFields struct {
	rfd  int
	wfd  int
	data int // unconsumed bytes currently buffered in the pipe
}

type splicePipe struct {
	splicePipeFields

	// Pad so the struct is large enough to skip the runtime tiny allocator,
	// which would otherwise prevent the finalizer from running.
	_ [24 - unsafe.Sizeof(splicePipeFields{})%24]byte
}

// splicePipePool recycles pipes to avoid repeated pipe() syscalls. Pool entries
// reclaimed by the GC have their fds closed by a finalizer set at creation.
var splicePipePool = sync.Pool{New: newPoolPipe}

func newPoolPipe() any {
	p := newPipe()
	if p == nil {
		return nil
	}
	runtime.SetFinalizer(p, destroyPipe)
	return p
}

func getPipe() *splicePipe {
	v := splicePipePool.Get()
	if v == nil {
		return nil
	}
	return v.(*splicePipe)
}

func putPipe(p *splicePipe) {
	// A pipe with leftover data cannot be safely reused; drop the finalizer and
	// destroy it now rather than returning a dirty pipe to the pool.
	if p.data != 0 {
		runtime.SetFinalizer(p, nil)
		destroyPipe(p)
		return
	}
	splicePipePool.Put(p)
}

func newPipe() *splicePipe {
	var fds [2]int
	if err := syscall.Pipe2(fds[:], syscall.O_CLOEXEC|syscall.O_NONBLOCK); err != nil {
		return nil
	}
	// Best-effort: grow the pipe to maxSpliceSize so each splice can move more
	// per syscall. A smaller pipe still works, just with more syscalls.
	_, _, _ = syscall.Syscall(syscall.SYS_FCNTL, uintptr(fds[0]), uintptr(fSetPipeSz), uintptr(maxSpliceSize))
	return &splicePipe{splicePipeFields: splicePipeFields{rfd: fds[0], wfd: fds[1]}}
}

func destroyPipe(p *splicePipe) {
	_ = syscall.Close(p.rfd)
	_ = syscall.Close(p.wfd)
}
