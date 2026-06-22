# freeio benchmarks

Each benchmark pairs a `freeio` API against its standard-library counterpart
(sub-benchmarks `freeio` / `std`, plus `freeio-counted` where the counter path
differs). Payload is 1.3 MB; file benchmarks run on a tmpfs-like temp dir.

```
go test -bench=. -benchmem -run=^$ ./freebuf/freeio
```

**System:** AMD Ryzen 9 9900X (12C/24T), Linux 6.12, Go 1.26.2, linux/amd64,
`performance` governor. `CopyConn` churns TCP per iteration; its figures are
warm-state (first runs discarded — the CPU's boost clock ramps over the first
~1 s run, inflating it ~20% regardless of which copy runs first). Loopback only.

```text
BenchmarkCopyInMem/freeio                44286 ns/op   30059 MB/s        0 B/op    0 allocs/op
BenchmarkCopyInMem/std                   44278 ns/op   30064 MB/s        0 B/op    0 allocs/op
BenchmarkCopyInMem/freeio-counted        29486 ns/op   45147 MB/s        0 B/op    0 allocs/op
BenchmarkCopyGeneric/freeio              29405 ns/op   45271 MB/s        0 B/op    0 allocs/op
BenchmarkCopyGeneric/std                 32059 ns/op   41524 MB/s    32768 B/op    1 allocs/op
BenchmarkCopyBuffer/freeio               20081 ns/op                    32 B/op    2 allocs/op
BenchmarkCopyBuffer/std                  19959 ns/op                    32 B/op    2 allocs/op
BenchmarkCopyFile/freeio                122935 ns/op                   824 B/op   11 allocs/op
BenchmarkCopyFile/std                   122454 ns/op                   312 B/op    7 allocs/op
BenchmarkReadFile/freeio                 45296 ns/op                  2644 B/op   13 allocs/op
BenchmarkReadFile/std                    84328 ns/op               1335658 B/op    5 allocs/op
BenchmarkCopyFS/freeio                  110994 ns/op                  4757 B/op   89 allocs/op
BenchmarkCopyFS/std                     110406 ns/op                  4970 B/op   88 allocs/op
BenchmarkCopyConn/freeio                151047 ns/op    8810 MB/s     2140 B/op   50 allocs/op
BenchmarkCopyConn/freeio-counted        151823 ns/op    8768 MB/s     2265 B/op   57 allocs/op
BenchmarkCopyConn/std                   150905 ns/op    8821 MB/s     2135 B/op   50 allocs/op
```

Buffered Reader / Writer vs `bufio` at the same buffer size (`WriteByte`/`Write`
target `io.Discard`; `ReadString`'s 393 KB / 8192 allocs is the returned-string
cost, equal on both sides):

```text
BenchmarkBufWriterWriteByte/freeio        3734 ns/op    1097 MB/s        0 B/op    0 allocs/op
BenchmarkBufWriterWriteByte/std           4504 ns/op     909 MB/s        0 B/op    0 allocs/op
BenchmarkBufWriterWrite/freeio             257 ns/op   63724 MB/s        0 B/op    0 allocs/op
BenchmarkBufWriterWrite/std                265 ns/op   61915 MB/s        0 B/op    0 allocs/op
BenchmarkBufReaderReadByte/freeio      1229804 ns/op    1082 MB/s        0 B/op    0 allocs/op
BenchmarkBufReaderReadByte/std         1453277 ns/op     916 MB/s        0 B/op    0 allocs/op
BenchmarkBufReaderReadString/freeio     137049 ns/op    2630 MB/s   393216 B/op 8192 allocs/op
BenchmarkBufReaderReadString/std        155799 ns/op    2314 MB/s   393216 B/op 8192 allocs/op
```

## Summary

- **Copy** matches or beats `io.Copy`; the userspace fallback (`CopyGeneric`) is
  **0-alloc** — its staging slice comes from the pool — vs the standard
  library's 32 KB scratch per call.
- **Conn copies tie `io.Copy`:** with no counter `freeio.Copy` delegates to the
  same `(*net.TCPConn).WriteTo` splice the standard library uses, so warm it
  matches `std` (~151 µs). The counter path drives a hand-rolled splice that also
  ties, keeping per-chunk byte counting essentially free.
- **ReadFile** returns a pooled `Buffer` instead of a fresh slice — ~1.9× faster
  than `os.ReadFile` at **2.6 KB/op vs 1.3 MB/op**, since the bytes live in
  recycled ≤64 KB pool chunks.
- **Buffered Reader/Writer beat `bufio`:** a raw pooled `[]byte` with direct
  slice hot paths gives ~1.2× on `WriteByte` and `ReadByte`, ~1.15× on
  `ReadString`, and ties `Write` — all 0-alloc, with the backing array recycled
  (which bufio never does).
- **File ops** (`CopyFile`, `CopyFS`) are on par with the standard library (the
  file→file path uses `copy_file_range`). The few extra `CopyFile` allocations
  come from the `Stat`-for-mode and same-file guard.

Numbers are a point-in-time snapshot; re-run with `-count` for stable figures.