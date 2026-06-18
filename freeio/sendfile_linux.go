//go:build linux

package freeio

import (
	"fmt"
	"syscall"
)

// sendfileConn copies a regular-file source to a socket destination with
// sendfile(2): the file data never enters user space. It counts per syscall.
// Reports handled == false (with n == 0) when sendfile does not apply to these
// descriptors, so the caller can fall back. Used as the splice fallback on
// kernels too old for splice(2).
func sendfileConn(srcConn, dstConn syscall.Conn, writeCounters, readCounters []CounterFunc) (n int64, handled bool, err error) {
	srcRaw, e := srcConn.SyscallConn()
	if e != nil {
		return 0, false, nil
	}
	dstRaw, e := dstConn.SyscallConn()
	if e != nil {
		return 0, false, nil
	}
	handled = true

	// The file fd stays valid for the whole transfer inside srcRaw.Read's
	// callback (a regular file is always "ready", so it runs once); the inner
	// dstRaw.Write drives socket writability and EAGAIN.
	readErr := srcRaw.Read(func(inFd uintptr) (done bool) {
		for {
			var sent int
			var sErr error
			writeErr := dstRaw.Write(func(outFd uintptr) (ready bool) {
				sent, sErr = syscall.Sendfile(int(outFd), int(inFd), nil, maxZeroCopyChunk)
				return sErr != syscall.EAGAIN
			})

			if writeErr != nil {
				err = writeErr
				return true
			}

			if sErr != nil {
				if (sErr == syscall.EINVAL || sErr == syscall.ENOSYS) && n == 0 {
					handled = false
					return true
				}
				err = fmt.Errorf("freeio: sendfile: %w", sErr)
				return true
			}

			if sent == 0 {
				return true // EOF
			}

			n += int64(sent)
			for _, c := range readCounters {
				c(int64(sent))
			}
			for _, c := range writeCounters {
				c(int64(sent))
			}
		}
	})
	if readErr != nil && err == nil {
		err = readErr
	}
	return n, handled, err
}
