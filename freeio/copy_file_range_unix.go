//go:build linux || freebsd

package freeio

import (
	"fmt"
	"syscall"
)

// copyFileRangeConn copies regular file -> regular file with copy_file_range(2),
// entirely in the kernel (and reflink-capable on supporting filesystems),
// counting per syscall. Both fds are regular files, so no readiness polling is
// needed. Reports handled == false (with n == 0) when the call is unsupported
// (old kernel, cross-filesystem, or unknown syscall number for this GOARCH), so
// the caller can fall back to splice.
func copyFileRangeConn(srcConn, dstConn syscall.Conn, writeCounters, readCounters []CounterFunc) (n int64, handled bool, err error) {
	if sysCopyFileRange == 0 {
		return 0, false, nil
	}
	srcRaw, e := srcConn.SyscallConn()
	if e != nil {
		return 0, false, nil
	}
	dstRaw, e := dstConn.SyscallConn()
	if e != nil {
		return 0, false, nil
	}
	handled = true

	ctrlErr := srcRaw.Control(func(inFd uintptr) {
		if e := dstRaw.Control(func(outFd uintptr) {
			for {
				// copy_file_range(2):
				//   ssize_t copy_file_range(int fd_in, off_t *off_in,
				//       int fd_out, off_t *off_out, size_t len, unsigned int flags)
				// The two off_* pointers are NULL (0) so the kernel uses and
				// advances each fd's own offset; flags is 0 (none defined). Go's
				// syscall package has no wrapper, so it is invoked via Syscall6
				// with the per-GOARCH number sysCopyFileRange (see the
				// copy_file_range_num_*.go files for sources).
				r1, _, errno := syscall.Syscall6(sysCopyFileRange,
					inFd, 0, outFd, 0, uintptr(maxZeroCopyChunk), 0)

				if errno == syscall.EINTR {
					continue
				}

				if errno != 0 {
					if n == 0 && (errno == syscall.EINVAL || errno == syscall.ENOSYS ||
						errno == syscall.EXDEV || errno == syscall.EOPNOTSUPP) {
						handled = false
					} else {
						err = fmt.Errorf("freeio: copy_file_range: %w", errno)
					}
					return
				}

				copied := int64(r1)
				if copied == 0 {
					return // EOF
				}

				n += copied
				for _, c := range readCounters {
					c(copied)
				}
				for _, c := range writeCounters {
					c(copied)
				}
			}
		}); e != nil && err == nil {
			err = e
		}
	})
	if ctrlErr != nil && err == nil {
		err = ctrlErr
	}
	return n, handled, err
}
