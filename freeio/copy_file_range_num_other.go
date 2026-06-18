//go:build linux && !amd64 && !arm64

package freeio

// 0 disables copy_file_range on Linux GOARCHs whose __NR_copy_file_range is not
// wired up above; copyFileRangeConn then reports handled=false and the caller
// falls back to splice. Add the arch's number (from its syscall.tbl /
// asm-generic/unistd.h) to enable it.
const sysCopyFileRange uintptr = 0
