# freebuf benchmarks

Compares `bytes.Buffer`, `freebuf.SerialBuffer`, `freebuf.MultiPartBuffer` and
the limited variant `freebuf.NewSerialLimited` across four workloads, plus a
ladder sweep that fixes the `NewExcept` threshold. Each op uses a fresh buffer
so the numbers reflect both processing time and allocation cost.

Run on two machines to show the architecture spread:

- **linux/amd64** — AMD Ryzen 9 9900X
- **darwin/arm64** — Apple M5

```
# linux (build elsewhere, copy the binary over — the box needs no Go toolchain):
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -c -o freebuf_bench ./freebuf
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -tags=freebuf_low_mem -c -o freebuf_bench_low ./freebuf
./freebuf_bench     -test.bench=. -test.benchmem -test.run=^$
./freebuf_bench_low -test.bench=. -test.benchmem -test.run=^$
# host:
go test -bench=. -benchmem -run=^$ ./freebuf
```

The default build sets `PartMinimalSize = 4096`, `PartIncSize = 16384`; the
low-memory build (`-tags=freebuf_low_mem`) sets `1024`/`256`. Past the 64KB pool
ceiling a `SerialBuffer` grows geometrically — 1.5× (default) / 1.25× (low-mem),
see `serialGrowShift`.

## linux/amd64 — AMD Ryzen 9 9900X

### Default build

```text
BenchmarkBufferWrite_64KB/BytesBuffer-24                3306 ns/op    19822 MB/s    65536 B/op     1 allocs/op
BenchmarkBufferWrite_64KB/SerialBuffer-24                786.5 ns/op   83323 MB/s        0 B/op     0 allocs/op
BenchmarkBufferWrite_64KB/MultiPartBuffer-24            1123 ns/op    58370 MB/s      248 B/op     5 allocs/op
BenchmarkBufferWrite_64KB/SerialBufferLimited-24         800.4 ns/op   81876 MB/s       48 B/op     1 allocs/op
BenchmarkBufferWriteByte_4K/BytesBuffer-24              5333 ns/op      768 MB/s     8128 B/op     7 allocs/op
BenchmarkBufferWriteByte_4K/SerialBuffer-24             6758 ns/op      606 MB/s        0 B/op     0 allocs/op
BenchmarkBufferWriteByte_4K/MultiPartBuffer-24          7687 ns/op      532 MB/s        8 B/op     1 allocs/op
BenchmarkBufferReadFrom_256KB/BytesBuffer-24           71355 ns/op     3673 MB/s  1048117 B/op    12 allocs/op
BenchmarkBufferReadFrom_256KB/SerialBuffer-24          62970 ns/op     4163 MB/s   842178 B/op     7 allocs/op
BenchmarkBufferReadFrom_256KB/MultiPartBuffer-24        2501 ns/op   104813 MB/s      554 B/op     7 allocs/op
BenchmarkBufferRoundTrip_64KB/BytesBuffer-24            4811 ns/op    13620 MB/s    65536 B/op     1 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBuffer-24           1272 ns/op    51523 MB/s        0 B/op     0 allocs/op
BenchmarkBufferRoundTrip_64KB/MultiPartBuffer-24        1668 ns/op    39296 MB/s      249 B/op     5 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBufferLimited-24    1311 ns/op    49999 MB/s       48 B/op     1 allocs/op
```

### Low-memory build

```text
BenchmarkBufferWrite_64KB/BytesBuffer-24                3336 ns/op    19642 MB/s    65536 B/op     1 allocs/op
BenchmarkBufferWrite_64KB/SerialBuffer-24                788.1 ns/op   83156 MB/s        0 B/op     0 allocs/op
BenchmarkBufferWrite_64KB/MultiPartBuffer-24            2757 ns/op    23772 MB/s     1021 B/op     7 allocs/op
BenchmarkBufferWrite_64KB/SerialBufferLimited-24         801.2 ns/op   81793 MB/s       48 B/op     1 allocs/op
BenchmarkBufferWriteByte_4K/BytesBuffer-24              5475 ns/op      748 MB/s     8128 B/op     7 allocs/op
BenchmarkBufferWriteByte_4K/SerialBuffer-24             6945 ns/op      589 MB/s        0 B/op     0 allocs/op
BenchmarkBufferWriteByte_4K/MultiPartBuffer-24          7897 ns/op      518 MB/s       56 B/op     3 allocs/op
BenchmarkBufferReadFrom_256KB/BytesBuffer-24           91766 ns/op     2856 MB/s  1048117 B/op    12 allocs/op
BenchmarkBufferReadFrom_256KB/SerialBuffer-24         116308 ns/op     2253 MB/s  1338199 B/op    17 allocs/op
BenchmarkBufferReadFrom_256KB/MultiPartBuffer-24       11386 ns/op    23023 MB/s     4574 B/op    10 allocs/op
BenchmarkBufferRoundTrip_64KB/BytesBuffer-24            5565 ns/op    11777 MB/s    65536 B/op     1 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBuffer-24           1305 ns/op    50219 MB/s        0 B/op     0 allocs/op
BenchmarkBufferRoundTrip_64KB/MultiPartBuffer-24        3339 ns/op    19626 MB/s     1021 B/op     7 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBufferLimited-24    1319 ns/op    49693 MB/s       48 B/op     1 allocs/op
```

## darwin/arm64 — Apple M5

### Default build

```text
BenchmarkBufferWrite_64KB/BytesBuffer-10                4335 ns/op    15116 MB/s    65536 B/op     1 allocs/op
BenchmarkBufferWrite_64KB/SerialBuffer-10                967.3 ns/op   67752 MB/s        0 B/op     0 allocs/op
BenchmarkBufferWrite_64KB/MultiPartBuffer-10            1158 ns/op    56607 MB/s      248 B/op     5 allocs/op
BenchmarkBufferWrite_64KB/SerialBufferLimited-10         928.2 ns/op   70605 MB/s       48 B/op     1 allocs/op
BenchmarkBufferWriteByte_4K/BytesBuffer-10              7743 ns/op      528 MB/s     8128 B/op     7 allocs/op
BenchmarkBufferWriteByte_4K/SerialBuffer-10             7954 ns/op      514 MB/s        0 B/op     0 allocs/op
BenchmarkBufferWriteByte_4K/MultiPartBuffer-10          7798 ns/op      525 MB/s        8 B/op     1 allocs/op
BenchmarkBufferReadFrom_256KB/BytesBuffer-10           50476 ns/op     5193 MB/s  1048113 B/op    12 allocs/op
BenchmarkBufferReadFrom_256KB/SerialBuffer-10          72398 ns/op     3620 MB/s   828582 B/op     7 allocs/op
BenchmarkBufferReadFrom_256KB/MultiPartBuffer-10        4994 ns/op    52496 MB/s      553 B/op     7 allocs/op
BenchmarkBufferRoundTrip_64KB/BytesBuffer-10            4947 ns/op    13247 MB/s    65536 B/op     1 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBuffer-10           1914 ns/op    34234 MB/s        0 B/op     0 allocs/op
BenchmarkBufferRoundTrip_64KB/MultiPartBuffer-10        2567 ns/op    25527 MB/s      248 B/op     5 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBufferLimited-10    2130 ns/op    30765 MB/s       48 B/op     1 allocs/op
```

### Low-memory build

```text
BenchmarkBufferWrite_64KB/BytesBuffer-10                3670 ns/op    17857 MB/s    65536 B/op     1 allocs/op
BenchmarkBufferWrite_64KB/SerialBuffer-10                919.4 ns/op   71281 MB/s        0 B/op     0 allocs/op
BenchmarkBufferWrite_64KB/MultiPartBuffer-10            2468 ns/op    26551 MB/s     1019 B/op     7 allocs/op
BenchmarkBufferWrite_64KB/SerialBufferLimited-10         837.7 ns/op   78231 MB/s       48 B/op     1 allocs/op
BenchmarkBufferWriteByte_4K/BytesBuffer-10              7805 ns/op      524 MB/s     8128 B/op     7 allocs/op
BenchmarkBufferWriteByte_4K/SerialBuffer-10             8041 ns/op      509 MB/s        0 B/op     0 allocs/op
BenchmarkBufferWriteByte_4K/MultiPartBuffer-10          7947 ns/op      515 MB/s       56 B/op     3 allocs/op
BenchmarkBufferReadFrom_256KB/BytesBuffer-10           52555 ns/op     4987 MB/s  1048114 B/op    12 allocs/op
BenchmarkBufferReadFrom_256KB/SerialBuffer-10          83243 ns/op     3149 MB/s  1307652 B/op    15 allocs/op
BenchmarkBufferReadFrom_256KB/MultiPartBuffer-10       11826 ns/op    22166 MB/s     4563 B/op    10 allocs/op
BenchmarkBufferRoundTrip_64KB/BytesBuffer-10            4981 ns/op    13156 MB/s    65536 B/op     1 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBuffer-10           1972 ns/op    33239 MB/s        0 B/op     0 allocs/op
BenchmarkBufferRoundTrip_64KB/MultiPartBuffer-10        3839 ns/op    17070 MB/s     1019 B/op     7 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBufferLimited-10    2125 ns/op    30837 MB/s       48 B/op     1 allocs/op
```

## Crossover sweep (BenchmarkBufferAcrossSizes)

Write+Read roundtrip across a payload ladder, fresh buffer per op — the data
that fixes the `NewExcept` threshold. Shown on the Ryzen; the M5 flips at the
same place.

### Default build

```text
BenchmarkBufferAcrossSizes/16KB/Serial-24       207.4 ns/op    78985 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/16KB/MultiPart-24    335.1 ns/op    48888 MB/s       56 B/op    3 allocs/op
BenchmarkBufferAcrossSizes/32KB/Serial-24       558.4 ns/op    58685 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/32KB/MultiPart-24    812.7 ns/op    40318 MB/s      120 B/op    4 allocs/op
BenchmarkBufferAcrossSizes/48KB/Serial-24      1020   ns/op    48166 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/48KB/MultiPart-24   1277   ns/op    38495 MB/s      248 B/op    5 allocs/op
BenchmarkBufferAcrossSizes/64KB/Serial-24      1291   ns/op    50767 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/64KB/MultiPart-24   1690   ns/op    38777 MB/s      249 B/op    5 allocs/op
BenchmarkBufferAcrossSizes/80KB/Serial-24      6563   ns/op    12481 MB/s    81920 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/80KB/MultiPart-24   2174   ns/op    37682 MB/s      506 B/op    6 allocs/op
BenchmarkBufferAcrossSizes/96KB/Serial-24      4997   ns/op    19671 MB/s    98304 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/96KB/MultiPart-24   2606   ns/op    37724 MB/s      506 B/op    6 allocs/op
BenchmarkBufferAcrossSizes/128KB/Serial-24     6602   ns/op    19854 MB/s   131073 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/128KB/MultiPart-24  3377   ns/op    38809 MB/s      506 B/op    6 allocs/op
BenchmarkBufferAcrossSizes/192KB/Serial-24    12152   ns/op    16179 MB/s   196610 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/192KB/MultiPart-24  4664   ns/op    42153 MB/s     1023 B/op    7 allocs/op
BenchmarkBufferAcrossSizes/256KB/Serial-24    18638   ns/op    14065 MB/s   262146 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/256KB/MultiPart-24  6504   ns/op    40303 MB/s     1024 B/op    7 allocs/op
```

### Low-memory build

```text
BenchmarkBufferAcrossSizes/16KB/Serial-24       210.2 ns/op    77931 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/16KB/MultiPart-24    854.4 ns/op    19175 MB/s      248 B/op    5 allocs/op
BenchmarkBufferAcrossSizes/32KB/Serial-24       579.3 ns/op    56564 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/32KB/MultiPart-24   1718   ns/op    19073 MB/s      505 B/op    6 allocs/op
BenchmarkBufferAcrossSizes/48KB/Serial-24      1025   ns/op    47949 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/48KB/MultiPart-24   2609   ns/op    18839 MB/s     1021 B/op    7 allocs/op
BenchmarkBufferAcrossSizes/64KB/Serial-24      1284   ns/op    51028 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/64KB/MultiPart-24   3325   ns/op    19707 MB/s     1021 B/op    7 allocs/op
BenchmarkBufferAcrossSizes/80KB/Serial-24      7752   ns/op    10567 MB/s    81920 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/80KB/MultiPart-24   4335   ns/op    18898 MB/s     2185 B/op    8 allocs/op
BenchmarkBufferAcrossSizes/96KB/Serial-24      5212   ns/op    18859 MB/s    98304 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/96KB/MultiPart-24   5101   ns/op    19270 MB/s     2186 B/op    8 allocs/op
BenchmarkBufferAcrossSizes/128KB/Serial-24     6938   ns/op    18891 MB/s   131073 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/128KB/MultiPart-24  6592   ns/op    19883 MB/s     2188 B/op    8 allocs/op
BenchmarkBufferAcrossSizes/192KB/Serial-24    12781   ns/op    15383 MB/s   196609 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/192KB/MultiPart-24 10212   ns/op    19251 MB/s     4538 B/op    9 allocs/op
BenchmarkBufferAcrossSizes/256KB/Serial-24    19105   ns/op    13721 MB/s   262145 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/256KB/MultiPart-24 13564   ns/op    19327 MB/s     4549 B/op    9 allocs/op
```

### The crossover

Both build modes flip between **64KB and 80KB**.

The mechanism is in the allocation column — at 80KB the Serial row jumps from
`0 B/op` to `81920 B/op`, because the backing slice has outgrown
`internal.MaxAllocatableSize` (65536) and now comes from `make()` instead of
the pool. From there each grow reallocs the backing buffer (1.5× default /
1.25× low-mem) and memcopies the existing content, while MultiPartBuffer just
appends another pooled chunk and never moves bytes.

This is what `NewExcept(except int)` reads:

- `except <= 65536` → `SerialBuffer` (contiguous, pool-backed, no copies)
- `except >  65536` → `MultiPartBuffer` (chunked, no realloc)

The threshold lives in `factory.go` as
`serialMultiPartCrossover = internal.MaxAllocatableSize`.

## Notes

- **SerialBuffer / SerialBufferLimited** are insensitive to the build flag —
  the pool returns the same backing slice once a size class is touched. Limited
  carries one extra 48-byte allocation (the `SerialBuffer` struct escaping via
  the eager `getSerialBuffer` in `NewSerialLimited`).
- **MultiPartBuffer** is sensitive to the flag: low-mem mode shrinks each chunk
  from 1024 → 64 bytes, so a 64KB Write goes from ~64 parts to ~1024 parts and
  per-chunk bookkeeping dominates. The crossover against SerialBuffer stays at
  ~64KB though — that's where the contiguous side falls off the pool.
- **SerialBuffer.ReadFrom past the pool ceiling** grows geometrically
  (`serialGrowShift`: 1.5× default, 1.25× low-mem) so a large read stays O(1)
  amortized. The low-mem 256KB row still costs more (`116µs`/17 allocs vs the
  default `63µs`/7) because 1.25× growth from a smaller start takes more
  reallocs — but it is no longer O(n²). For large unknown-size reads prefer
  `MultiPartBuffer` (`ReadAll` already picks it): ~25× faster at 256KB, no
  realloc/memcpy.
- **WriteByte**: `bytes.Buffer` beats `SerialBuffer` by ~25% (5333 ns vs
  6758 ns on the Ryzen) despite a fresh 8KB + 7-allocs cost, because
  `bytes.Buffer.WriteByte` inlines down to a 3-instruction hot path while ours
  calls `ensureFree`, which is too large to inline. Deliberate trade — see the
  comment at `(*SerialBuffer).WriteByte` in [serial.go](serial.go). If your hot
  path is dominated by `WriteByte`, reach for `bytes.Buffer`.
- **Architecture**: the M5 and Ryzen agree on the shape (Serial 0-alloc writes,
  MultiPart winning large ReadFrom, the 64-80KB crossover); absolute throughput
  differs but the ranking between buffer types is consistent across both.