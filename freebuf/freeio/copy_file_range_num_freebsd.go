//go:build freebsd

package freeio

// SYS_copy_file_range from FreeBSD sys/kern/syscalls.master (available since
// FreeBSD 13). FreeBSD syscall numbers are arch-independent.
const sysCopyFileRange uintptr = 569
