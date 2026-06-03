# freebuf benchmarks

Compares `bytes.Buffer`, `freebuf.SerialBuffer`, `freebuf.MultiPartBuffer` and
the limited variant `freebuf.NewSerialLimited` across four workloads, plus a
ladder sweep that fixes the `NewExcept` threshold. Each op uses a fresh buffer
so the numbers reflect both processing time and allocation cost.

Hardware: AMD Ryzen 9 9900X, linux/amd64. Run via:

```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -c -o freebuf_bench ./freebuf
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -tags=freebuf_low_mem -c -o freebuf_bench_low ./freebuf
./freebuf_bench     -test.bench=. -test.benchmem -test.run=^$
./freebuf_bench_low -test.bench=. -test.benchmem -test.run=^$
```

## Default build

`PartMinimalSize = 1024`, `PartReadIncSize = 16384`.

```text
goos: linux
goarch: amd64
pkg: github.com/duakc/mt/freebuf
cpu: AMD Ryzen 9 9900X 12-Core Processor
BenchmarkBufferWrite_64KB/BytesBuffer-24                  377460              3239 ns/op        20232.44 MB/s      65536 B/op          1 allocs/op
BenchmarkBufferWrite_64KB/SerialBuffer-24                1542337               777.1 ns/op      84334.90 MB/s          0 B/op          0 allocs/op
BenchmarkBufferWrite_64KB/MultiPartBuffer-24              842928              1405 ns/op        46636.47 MB/s        248 B/op          5 allocs/op
BenchmarkBufferWrite_64KB/SerialBufferLimited-24         1491178               800.6 ns/op      81856.04 MB/s         48 B/op          1 allocs/op
BenchmarkBufferWriteByte_4K/BytesBuffer-24                261784              5008 ns/op         817.91 MB/s        8128 B/op          7 allocs/op
BenchmarkBufferWriteByte_4K/SerialBuffer-24               169432              7323 ns/op         559.33 MB/s           0 B/op          0 allocs/op
BenchmarkBufferWriteByte_4K/MultiPartBuffer-24            151758              8135 ns/op         503.48 MB/s           8 B/op          1 allocs/op
BenchmarkBufferReadFrom_256KB/BytesBuffer-24               15175             76382 ns/op        3432.00 MB/s     1048116 B/op         12 allocs/op
BenchmarkBufferReadFrom_256KB/SerialBuffer-24              16785             71473 ns/op        3667.71 MB/s      956133 B/op          6 allocs/op
BenchmarkBufferReadFrom_256KB/MultiPartBuffer-24          321338              3647 ns/op        71872.60 MB/s        555 B/op          7 allocs/op
BenchmarkBufferRoundTrip_64KB/BytesBuffer-24              225360              5281 ns/op        12410.02 MB/s      65536 B/op          1 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBuffer-24             881952              1305 ns/op        50229.20 MB/s          0 B/op          0 allocs/op
BenchmarkBufferRoundTrip_64KB/MultiPartBuffer-24          621304              1931 ns/op        33930.29 MB/s        248 B/op          5 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBufferLimited-24      916826              1313 ns/op        49897.56 MB/s         48 B/op          1 allocs/op
```

## Low-memory build

`PartMinimalSize = 64`, `PartReadIncSize = 1024`.

```text
goos: linux
goarch: amd64
pkg: github.com/duakc/mt/freebuf
cpu: AMD Ryzen 9 9900X 12-Core Processor
BenchmarkBufferWrite_64KB/BytesBuffer-24                  363013              3278 ns/op        19992.78 MB/s      65536 B/op          1 allocs/op
BenchmarkBufferWrite_64KB/SerialBuffer-24                1506864               793.7 ns/op      82566.80 MB/s          0 B/op          0 allocs/op
BenchmarkBufferWrite_64KB/MultiPartBuffer-24              399037              2975 ns/op        22032.47 MB/s       1021 B/op          7 allocs/op
BenchmarkBufferWrite_64KB/SerialBufferLimited-24         1503007               801.7 ns/op      81749.24 MB/s         48 B/op          1 allocs/op
BenchmarkBufferWriteByte_4K/BytesBuffer-24                254162              4871 ns/op         840.87 MB/s        8128 B/op          7 allocs/op
BenchmarkBufferWriteByte_4K/SerialBuffer-24               174316              7111 ns/op         576.00 MB/s           0 B/op          0 allocs/op
BenchmarkBufferWriteByte_4K/MultiPartBuffer-24            151142              8164 ns/op         501.72 MB/s          56 B/op          3 allocs/op
BenchmarkBufferReadFrom_256KB/BytesBuffer-24               15645             79283 ns/op        3306.43 MB/s     1048116 B/op         12 allocs/op
BenchmarkBufferReadFrom_256KB/SerialBuffer-24              13290             86152 ns/op        3042.82 MB/s      966955 B/op         10 allocs/op
BenchmarkBufferReadFrom_256KB/MultiPartBuffer-24           96802             12389 ns/op        21159.81 MB/s       4573 B/op         10 allocs/op
BenchmarkBufferRoundTrip_64KB/BytesBuffer-24              227481              5840 ns/op        11221.44 MB/s      65536 B/op          1 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBuffer-24             886222              1283 ns/op        51068.14 MB/s          0 B/op          0 allocs/op
BenchmarkBufferRoundTrip_64KB/MultiPartBuffer-24          322227              3670 ns/op        17856.86 MB/s       1021 B/op          7 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBufferLimited-24      907279              1318 ns/op        49737.50 MB/s         48 B/op          1 allocs/op
```

## Crossover sweep (BenchmarkBufferAcrossSizes)

Write+Read roundtrip across a payload ladder, fresh buffer per op. This is the
data that fixes the `NewExcept` threshold.

### Default build

```text
BenchmarkBufferAcrossSizes/16KB/Serial-24                5648442               211.9 ns/op     77301.43 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/16KB/MultiPart-24             2966636               407.2 ns/op     40238.88 MB/s       56 B/op    3 allocs/op
BenchmarkBufferAcrossSizes/32KB/Serial-24                2102662               573.4 ns/op     57145.01 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/32KB/MultiPart-24             1284088               937.9 ns/op     34936.68 MB/s      120 B/op    4 allocs/op
BenchmarkBufferAcrossSizes/48KB/Serial-24                1000000              1018   ns/op     48271.58 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/48KB/MultiPart-24              821943              1476   ns/op     33296.04 MB/s      248 B/op    5 allocs/op
BenchmarkBufferAcrossSizes/64KB/Serial-24                 934390              1278   ns/op     51263.05 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/64KB/MultiPart-24              606984              1936   ns/op     33858.81 MB/s      249 B/op    5 allocs/op
BenchmarkBufferAcrossSizes/80KB/Serial-24                 173506              6868   ns/op     11927.36 MB/s    81920 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/80KB/MultiPart-24              469434              2501   ns/op     32753.11 MB/s      506 B/op    6 allocs/op
BenchmarkBufferAcrossSizes/96KB/Serial-24                 236096              5066   ns/op     19403.09 MB/s    98304 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/96KB/MultiPart-24              393810              2979   ns/op     33000.67 MB/s      506 B/op    6 allocs/op
BenchmarkBufferAcrossSizes/128KB/Serial-24                175207              7096   ns/op     18471.99 MB/s   131073 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/128KB/MultiPart-24             306888              3920   ns/op     33438.22 MB/s      506 B/op    6 allocs/op
BenchmarkBufferAcrossSizes/192KB/Serial-24                 85614             12580   ns/op     15628.89 MB/s   196610 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/192KB/MultiPart-24             214906              5607   ns/op     35065.66 MB/s     1023 B/op    7 allocs/op
BenchmarkBufferAcrossSizes/256KB/Serial-24                 64030             19542   ns/op     13414.25 MB/s   262145 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/256KB/MultiPart-24             144144              8183   ns/op     32034.95 MB/s     1024 B/op    7 allocs/op
```

### Low-memory build

```text
BenchmarkBufferAcrossSizes/16KB/Serial-24                5670381               210.3 ns/op     77902.99 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/16KB/MultiPart-24             1307472               916.9 ns/op     17868.99 MB/s      248 B/op    5 allocs/op
BenchmarkBufferAcrossSizes/32KB/Serial-24                2536392               475.9 ns/op     68849.97 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/32KB/MultiPart-24              634234              1852   ns/op     17697.80 MB/s      506 B/op    6 allocs/op
BenchmarkBufferAcrossSizes/48KB/Serial-24                1000000              1037   ns/op     47380.49 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/48KB/MultiPart-24              425382              2808   ns/op     17504.58 MB/s     1021 B/op    7 allocs/op
BenchmarkBufferAcrossSizes/64KB/Serial-24                 938408              1289   ns/op     50840.35 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/64KB/MultiPart-24              337256              3646   ns/op     17976.88 MB/s     1021 B/op    7 allocs/op
BenchmarkBufferAcrossSizes/80KB/Serial-24                 147362              8128   ns/op     10079.21 MB/s    81920 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/80KB/MultiPart-24              254065              4688   ns/op     17475.36 MB/s     2185 B/op    8 allocs/op
BenchmarkBufferAcrossSizes/96KB/Serial-24                 216561              9925   ns/op      9904.31 MB/s    98304 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/96KB/MultiPart-24              217563              5564   ns/op     17667.84 MB/s     2186 B/op    8 allocs/op
BenchmarkBufferAcrossSizes/128KB/Serial-24                162542             12733   ns/op     10293.93 MB/s   131073 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/128KB/MultiPart-24             168996              7247   ns/op     18085.54 MB/s     2187 B/op    8 allocs/op
BenchmarkBufferAcrossSizes/192KB/Serial-24                 69606             17597   ns/op     11172.57 MB/s   196610 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/192KB/MultiPart-24             108810             11099   ns/op     17713.56 MB/s     4536 B/op    9 allocs/op
BenchmarkBufferAcrossSizes/256KB/Serial-24                 49222             24757   ns/op     10588.77 MB/s   262146 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/256KB/MultiPart-24              81219             15032   ns/op     17438.63 MB/s     4542 B/op    9 allocs/op
```

### The crossover

Both build modes flip at the same place: between **64KB and 80KB**.

The mechanism shows up in the allocation column — at 80KB the Serial row jumps
from `0 B/op` to `81920 B/op`, because the backing slice has outgrown
`internal.MaxAllocatableSize` (65536) and now comes from `make()` instead of
the pool. From there on every realloc doubles the backing buffer and memcopies
the existing content, while MultiPartBuffer just appends another pooled chunk
and never moves bytes.

This is what `NewExcept(except int)` reads:

- `except <= 65536` → `SerialBuffer` (contiguous, pool-backed, no copies)
- `except >  65536` → `MultiPartBuffer` (chunked, no realloc)

The threshold lives in `factory.go` as
`serialMultiPartCrossover = internal.MaxAllocatableSize`.

## Notes

- **SerialBuffer / SerialBufferLimited** are insensitive to the build flag —
  the pool returns the same backing slice once a size class is touched.
  Limited carries one extra 48-byte allocation (the SerialBuffer struct on
  the heap, forced to escape by the eager `getSerialBuffer` call in
  `NewSerialLimited`).
- **MultiPartBuffer** is highly sensitive to the flag: low-mem mode shrinks
  each chunk from 1024 → 64 bytes, so a 64KB Write goes from ~64 parts to
  ~1024 parts and the per-chunk bookkeeping cost dominates. The crossover
  point against SerialBuffer stays at ~64KB though, because that's where the
  contiguous side falls off the pool.
- For payloads that outrun the 64KB pool ceiling, MultiPartBuffer wins by
  ~20× in default mode at 256KB (no realloc / memcpy). In low-mem mode the
  win shrinks to ~1.6× at 256KB (more chunks means more bookkeeping) but the
  direction never reverses within the sizes measured.
- **WriteByte**: `bytes.Buffer` beats `SerialBuffer` by ~30–45% in both
  build modes (5008/4871 ns vs 7323/7111 ns) despite paying a fresh 8KB +
  7-allocs cost per op. The reason is documented in
  [serial.go](serial.go) at `(*SerialBuffer).WriteByte`. TL;DR:
  `bytes.Buffer.WriteByte` is inlined by the compiler down to a 3-instruction
  hot path; ours calls `ensureFree`, which is too large to inline, so every
  per-byte write pays a function-call prologue/epilogue. This was a deliberate
  trade — keeping the bookkeeping in one helper makes the multi-byte paths
  (`Write`, `WriteString`, `ReadFrom`) simpler and equally fast. Anyone whose
  hot path is dominated by `WriteByte` and who cannot tolerate the gap should
  reach for `bytes.Buffer` directly or fork in a hand-inlined fast path.