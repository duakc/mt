package freeio_test

import (
	"bytes"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/duakc/mt/freebuf"
	"github.com/duakc/mt/freeio"
)

// Each benchmark pairs a freeio implementation against its standard-library
// counterpart (sub-benchmarks "freeio" and "std") so the two can be compared
// directly, e.g. with benchstat.

type copyFn func(io.Writer, io.Reader) (int64, error)

// benchMem times an in-memory bytes.Reader -> bytes.Buffer copy. When hide is
// set, both ends are wrapped to hide WriteTo/ReadFrom, forcing the buffered path.
func benchMem(b *testing.B, cp copyFn, hide bool) {
	src := bytes.NewReader(payload)
	var buf bytes.Buffer
	buf.Grow(len(payload))
	var dst io.Writer = &buf
	var rd io.Reader = src
	if hide {
		dst, rd = onlyWriter{&buf}, onlyReader{src}
	}

	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for b.Loop() {
		src.Reset(payload)
		buf.Reset()
		if _, err := cp(dst, rd); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCopyInMem — fast path (bytes.Reader.WriteTo); also the counting cost.
func BenchmarkCopyInMem(b *testing.B) {
	b.Run("freeio", func(b *testing.B) { benchMem(b, freeio.Copy, false) })
	b.Run("std", func(b *testing.B) { benchMem(b, io.Copy, false) })
	b.Run("freeio-counted", func(b *testing.B) {
		var r, w int64
		rc, wc := []freeio.CounterFunc{adder(&r)}, []freeio.CounterFunc{adder(&w)}
		benchMem(b, func(dst io.Writer, src io.Reader) (int64, error) {
			return freeio.CopyWithCounter(dst, src, wc, rc)
		}, false)
	})
}

// BenchmarkCopyGeneric — buffered staging path (freeio pooled buffer vs io.Copy's
// per-call 32KB allocation).
func BenchmarkCopyGeneric(b *testing.B) {
	b.Run("freeio", func(b *testing.B) { benchMem(b, freeio.Copy, true) })
	b.Run("std", func(b *testing.B) { benchMem(b, io.Copy, true) })
}

// BenchmarkCopyBuffer — caller-supplied scratch buffer.
func BenchmarkCopyBuffer(b *testing.B) {
	src := bytes.NewReader(payload)
	var buf bytes.Buffer
	buf.Grow(len(payload))
	dst, rd := onlyWriter{&buf}, onlyReader{src}
	b.SetBytes(int64(len(payload)))

	b.Run("freeio", func(b *testing.B) {
		scratch := freebuf.NewSerial()
		defer scratch.FreeMe()
		b.ReportAllocs()
		for b.Loop() {
			src.Reset(payload)
			buf.Reset()
			if _, err := freeio.CopyBuffer(dst, rd, scratch); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("std", func(b *testing.B) {
		scratch := make([]byte, freebuf.PartIncSize)
		b.ReportAllocs()
		for b.Loop() {
			src.Reset(payload)
			buf.Reset()
			if _, err := io.CopyBuffer(dst, rd, scratch); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func stdCopyFile(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	return err
}

// BenchmarkCopyFile — file -> file (freeio copy_file_range vs io.Copy, which on
// Linux also reaches copy_file_range through os.File.ReadFrom).
func BenchmarkCopyFile(b *testing.B) {
	src := filepath.Join(b.TempDir(), "src")
	if err := os.WriteFile(src, payload, 0o644); err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(payload)))

	b.Run("freeio", func(b *testing.B) {
		dst := filepath.Join(b.TempDir(), "dst")
		b.ReportAllocs()
		for b.Loop() {
			if _, err := freeio.CopyFile(dst, src); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("std", func(b *testing.B) {
		dst := filepath.Join(b.TempDir(), "dst")
		b.ReportAllocs()
		for b.Loop() {
			if err := stdCopyFile(dst, src); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkReadFile — whole-file read into memory (freeio pooled Buffer vs
// os.ReadFile).
func BenchmarkReadFile(b *testing.B) {
	src := filepath.Join(b.TempDir(), "src")
	if err := os.WriteFile(src, payload, 0o644); err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(payload)))

	b.Run("freeio", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			buf, err := freeio.ReadFile(src)
			if err != nil {
				b.Fatal(err)
			}
			buf.FreeMe()
		}
	})
	b.Run("std", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := os.ReadFile(src); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkCopyFS — copy a small file tree (freeio.CopyFS vs os.CopyFS).
func BenchmarkCopyFS(b *testing.B) {
	srcDir := b.TempDir()
	for _, name := range []string{"a.bin", "b.bin", "c.bin", "d.bin"} {
		if err := os.WriteFile(filepath.Join(srcDir, name), payload[:256*1024], 0o644); err != nil {
			b.Fatal(err)
		}
	}
	fsys := os.DirFS(srcDir)
	b.SetBytes(int64(4 * 256 * 1024))

	b.Run("freeio", func(b *testing.B) {
		root := b.TempDir()
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			i++
			if _, _, err := freeio.CopyFS(filepath.Join(root, strconv.Itoa(i)), fsys); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("std", func(b *testing.B) {
		root := b.TempDir()
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			i++
			if err := os.CopyFS(filepath.Join(root, strconv.Itoa(i)), fsys); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// benchConn streams payload through a fresh client->proxy->upstream TCP pipeline
// per iteration; cp is the proxy copy. With a multi-MB payload the copy
// dominates connection setup.
func benchConn(b *testing.B, cp copyFn) {
	upLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	defer upLn.Close()
	go func() {
		for {
			c, err := upLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) { _, _ = io.Copy(io.Discard, c); c.Close() }(c)
		}
	}()

	srcLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	defer srcLn.Close()
	go func() {
		for {
			c, err := srcLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				_, _ = c.Write(payload)
				_ = c.(*net.TCPConn).CloseWrite()
			}(c)
		}
	}()

	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for b.Loop() {
		dstConn, err := net.Dial("tcp", upLn.Addr().String())
		if err != nil {
			b.Fatal(err)
		}
		srcConn, err := net.Dial("tcp", srcLn.Addr().String())
		if err != nil {
			b.Fatal(err)
		}
		if _, err := cp(dstConn, srcConn); err != nil {
			b.Fatal(err)
		}
		_ = dstConn.(*net.TCPConn).CloseWrite()
		_ = dstConn.Close()
		_ = srcConn.Close()
	}
}

// BenchmarkCopyConn — socket -> socket proxy throughput. "freeio" (no counter)
// delegates to the conn's own WriteTo, the same splice path io.Copy takes, so it
// ties "std"; "freeio-counted" drives the hand-rolled splice loop to keep
// per-syscall counting and shows its cost. Run with -count to average out TCP
// connection-churn noise (loopback only; real links are network-bound).
func BenchmarkCopyConn(b *testing.B) {
	b.Run("freeio", func(b *testing.B) { benchConn(b, freeio.Copy) })
	b.Run("freeio-counted", func(b *testing.B) {
		var r, w int64
		rc, wc := []freeio.CounterFunc{adder(&r)}, []freeio.CounterFunc{adder(&w)}
		benchConn(b, func(dst io.Writer, src io.Reader) (int64, error) {
			return freeio.CopyWithCounter(dst, src, wc, rc)
		})
	})
	b.Run("std", func(b *testing.B) { benchConn(b, io.Copy) })
}
