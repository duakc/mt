//go:build darwin || freebsd || dragonfly

package freeio

import (
	"fmt"
	"syscall"
)

// sendfileConn copies a regular-file source to a socket destination with
// sendfile(2) on the BSD family. Unlike Linux, BSD sendfile takes an explicit
// file offset (it does not touch the fd offset) and can report a partial count
// together with EAGAIN, so we track the offset and count per call. Because the
// fd offset is never advanced, any error before the first byte is a safe
// fall-through (handled == false): a later buffered copy re-reads from the
// start.
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

	var offset int64
	readErr := srcRaw.Read(func(inFd uintptr) (done bool) {
		for {
			var lastSent int
			var lastErr error
			writeErr := dstRaw.Write(func(outFd uintptr) (ready bool) {
				off := offset
				lastSent, lastErr = syscall.Sendfile(int(outFd), int(inFd), &off, maxZeroCopyChunk)
				if lastSent > 0 { // BSD can report bytes sent alongside EAGAIN
					offset += int64(lastSent)
					n += int64(lastSent)
					for _, c := range readCounters {
						c(int64(lastSent))
					}
					for _, c := range writeCounters {
						c(int64(lastSent))
					}
				}
				return lastErr != syscall.EAGAIN
			})
			if writeErr != nil {
				err = writeErr
				return true
			}

			if lastErr != nil {
				if n == 0 { // nothing sent and fd offset untouched: safe to fall back
					handled = false
					return true
				}

				err = fmt.Errorf("freeio: sendfile: %w", lastErr)
				return true
			}
			if lastSent == 0 {
				return true // EOF
			}
		}
	})
	if readErr != nil && err == nil {
		err = readErr
	}
	return n, handled, err
}
