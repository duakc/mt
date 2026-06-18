//go:build !linux && !darwin && !freebsd && !dragonfly

package freeio

import "syscall"

func sendfileConn(_, _ syscall.Conn, _, _ []CounterFunc) (n int64, handled bool, err error) {
	return 0, false, nil
}
