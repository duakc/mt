//go:build !linux && !freebsd

package freeio

import "syscall"

func copyFileRangeConn(_, _ syscall.Conn, _, _ []CounterFunc) (n int64, handled bool, err error) {
	return 0, false, nil
}
