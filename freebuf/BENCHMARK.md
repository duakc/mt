# freebuf benchmarks

`bytes.Buffer` vs `freebuf.SerialBuffer` / `MultiPartBuffer` / the limited
variant `NewSerialLimited`, across four workloads plus a size sweep. Every op
starts from a fresh buffer, so the numbers fold in allocation cost.

```
go test -bench=. -benchmem -run=^$ ./freebuf
```

**System:** AMD Ryzen 9 9900X (12C/24T), Linux 6.12, Go 1.26.2, linux/amd64,
`performance` governor. Default build (`PartMinimalSize=4096`,
`PartIncSize=16384`); a low-memory build (`-tags freebuf_low_mem`) shrinks those
to 1024/256 and grows past the pool ceiling at 1.25× instead of 1.5×, trading
throughput for footprint.

```text
BenchmarkBufferWrite_64KB/BytesBuffer            3276 ns/op   20006 MB/s    65536 B/op    1 allocs/op
BenchmarkBufferWrite_64KB/SerialBuffer            777 ns/op   84322 MB/s        0 B/op    0 allocs/op
BenchmarkBufferWrite_64KB/MultiPartBuffer        1136 ns/op   57672 MB/s      248 B/op    5 allocs/op
BenchmarkBufferWrite_64KB/SerialBufferLimited     799 ns/op   81928 MB/s       48 B/op    1 allocs/op
BenchmarkBufferWriteByte_4K/BytesBuffer          5643 ns/op     726 MB/s     8128 B/op    7 allocs/op
BenchmarkBufferWriteByte_4K/SerialBuffer         3726 ns/op    1100 MB/s        0 B/op    0 allocs/op
BenchmarkBufferWriteByte_4K/MultiPartBuffer      8100 ns/op     506 MB/s        8 B/op    1 allocs/op
BenchmarkBufferReadFrom_256KB/BytesBuffer       77712 ns/op    3373 MB/s  1048116 B/op   12 allocs/op
BenchmarkBufferReadFrom_256KB/SerialBuffer      70131 ns/op    3738 MB/s   840412 B/op    7 allocs/op
BenchmarkBufferReadFrom_256KB/MultiPartBuffer    2515 ns/op  104230 MB/s      554 B/op    7 allocs/op
BenchmarkBufferRoundTrip_64KB/BytesBuffer        5531 ns/op   11850 MB/s    65536 B/op    1 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBuffer       1300 ns/op   50415 MB/s        0 B/op    0 allocs/op
BenchmarkBufferRoundTrip_64KB/MultiPartBuffer    1692 ns/op   38710 MB/s      248 B/op    5 allocs/op
BenchmarkBufferRoundTrip_64KB/SerialBufferLimited 1315 ns/op  49865 MB/s       48 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/16KB/Serial            212 ns/op   77472 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/16KB/MultiPart         338 ns/op   48410 MB/s       56 B/op    3 allocs/op
BenchmarkBufferAcrossSizes/64KB/Serial           1294 ns/op   50661 MB/s        0 B/op    0 allocs/op
BenchmarkBufferAcrossSizes/64KB/MultiPart        1687 ns/op   38839 MB/s      248 B/op    5 allocs/op
BenchmarkBufferAcrossSizes/256KB/Serial         21916 ns/op   11961 MB/s   262146 B/op    1 allocs/op
BenchmarkBufferAcrossSizes/256KB/MultiPart       6754 ns/op   38813 MB/s     1024 B/op    7 allocs/op
```

## Summary

- **SerialBuffer** is the all-rounder up to the 64 KB pool ceiling: contiguous,
  pool-backed, **0 allocs**, ~4× faster than `bytes.Buffer` on Write/RoundTrip
  and never reallocating.
- **WriteByte** beats `bytes.Buffer` ~1.5× at 0 allocs — the common path is an
  inlinable bounds-check-and-store, `ensureFree` only on a full buffer.
- **MultiPartBuffer** wins large or unknown-size reads: a 256 KB `ReadFrom` is
  ~30× faster than `bytes.Buffer` at ~550 B/op, appending pooled chunks instead
  of doubling one slice. `ReadAll` picks it for this reason.
- **Crossover at 64 KB:** at/below the ceiling prefer SerialBuffer (0-alloc); the
  size sweep has them close up to 64 KB, then at 256 KB the contiguous backing
  falls off the pool into one big heap allocation and MultiPartBuffer pulls ~3×
  ahead. `NewExcept` makes that choice from the expected size.
- **Limited variant** matches SerialBuffer's speed with a single up-front
  allocation and a hard capacity ceiling.

Numbers are a point-in-time snapshot; re-run with `-count` for stable figures.