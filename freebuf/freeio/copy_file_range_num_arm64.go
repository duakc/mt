//go:build linux && arm64

package freeio

// __NR_copy_file_range from Linux include/uapi/asm-generic/unistd.h (the table
// arm64/riscv64/loong64/... share). Not exported by package syscall.
const sysCopyFileRange uintptr = 285
