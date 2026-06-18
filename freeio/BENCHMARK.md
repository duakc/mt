# freeio benchmarks

Each benchmark pairs a `freeio` implementation against its standard-library
counterpart (sub-benchmarks `freeio` / `std`, plus `freeio-counted` where the
counter path differs). Payload is 1.3 MB. Build and run:

```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -c -o freeio_bench ./freeio
./freeio_bench -test.run=^$ -test.bench=. -test.benchmem -test.count=2
# or, on the host:
go test -run=^$ -bench=. -benchmem -count=2 ./freeio
```

`CopyConn` churns TCP connections per iteration, so use `-count` to average it
(loopback only — real links are network-bound). File/`ReadFile`/`CopyFS`
benchmarks here ran on tmpfs, which isolates the syscall cost from disk.

## linux/amd64 — AMD Ryzen 9 9900X

```text
BenchmarkCopyInMem/freeio-24             44275 ns/op   30067 MB/s        0 B/op    0 allocs/op
BenchmarkCopyInMem/std-24                44279 ns/op   30064 MB/s        0 B/op    0 allocs/op
BenchmarkCopyInMem/freeio-counted-24     44391 ns/op   29988 MB/s       64 B/op    2 allocs/op
BenchmarkCopyGeneric/freeio-24           30420 ns/op   43760 MB/s       49 B/op    1 allocs/op
BenchmarkCopyGeneric/std-24              32408 ns/op   41076 MB/s    32768 B/op    1 allocs/op
BenchmarkCopyBuffer/freeio-24            19142 ns/op                    32 B/op    2 allocs/op
BenchmarkCopyBuffer/std-24               19035 ns/op                    32 B/op    2 allocs/op
BenchmarkCopyFile/freeio-24             121304 ns/op                   776 B/op   10 allocs/op
BenchmarkCopyFile/std-24                119522 ns/op                   312 B/op    7 allocs/op
BenchmarkReadFile/freeio-24              39000 ns/op                  2690 B/op   13 allocs/op
BenchmarkReadFile/std-24                 69470 ns/op               1335664 B/op    5 allocs/op
BenchmarkCopyFS/freeio-24               109613 ns/op                  4751 B/op   88 allocs/op
BenchmarkCopyFS/std-24                  109843 ns/op                  4892 B/op   87 allocs/op
BenchmarkCopyConn/freeio-24             160005 ns/op    8320 MB/s     2147 B/op   50 allocs/op
BenchmarkCopyConn/freeio-counted-24     149094 ns/op    8929 MB/s     2268 B/op   57 allocs/op
BenchmarkCopyConn/std-24                151537 ns/op    8785 MB/s     2138 B/op   50 allocs/op
```

## darwin/arm64 — Apple M5

```text
BenchmarkCopyInMem/freeio-10             20331 ns/op   65477 MB/s        0 B/op    0 allocs/op
BenchmarkCopyInMem/std-10                16356 ns/op   81391 MB/s        0 B/op    0 allocs/op
BenchmarkCopyInMem/freeio-counted-10     18337 ns/op   72597 MB/s       64 B/op    2 allocs/op
BenchmarkCopyGeneric/freeio-10           30509 ns/op   43621 MB/s       49 B/op    1 allocs/op
BenchmarkCopyGeneric/std-10              31343 ns/op   42472 MB/s    32768 B/op    1 allocs/op
BenchmarkCopyBuffer/freeio-10            34488 ns/op                    32 B/op    2 allocs/op
BenchmarkCopyBuffer/std-10               34971 ns/op                    32 B/op    2 allocs/op
BenchmarkCopyFile/freeio-10             334763 ns/op                 33680 B/op   10 allocs/op
BenchmarkCopyFile/std-10                340450 ns/op                 33168 B/op    7 allocs/op
BenchmarkReadFile/freeio-10              63844 ns/op                  2653 B/op   13 allocs/op
BenchmarkReadFile/std-10                 73807 ns/op               1335707 B/op    5 allocs/op
BenchmarkCopyFS/freeio-10               436015 ns/op                138288 B/op   90 allocs/op
BenchmarkCopyFS/std-10                  465951 ns/op                138320 B/op   89 allocs/op
BenchmarkCopyConn/freeio-10             336474 ns/op    3956 MB/s    35228 B/op   56 allocs/op
BenchmarkCopyConn/freeio-counted-10     333549 ns/op    3991 MB/s     2322 B/op   51 allocs/op
BenchmarkCopyConn/std-10                339256 ns/op    3924 MB/s    35204 B/op   56 allocs/op
```

## Reading the results

- **In-memory fast path** (`CopyInMem`): `freeio.Copy` hands a `bytes.Reader`
  straight to its `WriteTo`, so it matches `io.Copy` with zero allocations.
  Counters cost one small wrapper allocation (64 B) and no measurable time.
- **Generic / buffered fallback** (`CopyGeneric`): same speed as `io.Copy` but
  the staging buffer is pooled — **49 B/op vs the standard library's 32 KB/op**.
- **ReadFile**: returns a pooled `Buffer` instead of a fresh slice — on the
  Ryzen ~1.8× faster and **2.7 KB/op vs 1.3 MB/op**.
- **CopyFile / CopyFS**: on par with the standard library (on Linux both reach
  `copy_file_range`; the per-file overhead is a few extra allocations from the
  dispatch).
- **CopyConn (Linux)**: with no counter `freeio.Copy` delegates to the conn's
  own `WriteTo`, i.e. the same `splice` path `io.Copy` takes, so it ties `std`.
  The hand-rolled splice (`freeio-counted`) keeps per-syscall counting at the
  same throughput. On darwin there is no `splice`, so the no-counter path falls
  back like `io.Copy` (32 KB scratch), while the counted path uses the pooled
  buffer — note the **2.3 KB/op vs 35 KB/op** there.

Numbers are a point-in-time snapshot; re-run with `-count` for stable figures.