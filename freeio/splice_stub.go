//go:build !linux

package freeio

import "syscall"

// splice(2) is Linux-only.
func spliceConn(_, _ syscall.Conn, _, _ []CounterFunc) (n int64, handled bool, err error) {
	return 0, false, nil
}
