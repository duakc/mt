//go:build linux && amd64

package freeio

// __NR_copy_file_range for x86-64, from the Linux source
// arch/x86/entry/syscalls/syscall_64.tbl. Not exported by package syscall, so
// it is wired up per-GOARCH here.
const sysCopyFileRange uintptr = 326
