package freeio

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/duakc/mt/freebuf"
)

// CopyFile copies the regular file at src to dst and returns the bytes copied.
// dst is created (or truncated) with src's permission bits. On Linux/FreeBSD a
// same-filesystem copy uses copy_file_range(2); elsewhere it falls back to a
// buffered copy. Copying a file onto itself is refused.
func CopyFile(dst, src string) (int64, error) {
	return CopyFileWithCounter(dst, src, nil, nil)
}

// CopyFS copies the file tree rooted at fsys into the directory dst (created if
// needed), returning the total bytes copied, the number of regular files
// copied, and the first error. Regular files go through Copy, so an os.DirFS
// source still reaches copy_file_range/sendfile; directories are recreated.
// Non-regular entries (symlinks, devices) are rejected. Paths from fsys are
// validated by fs.WalkDir, so none can escape dst.
func CopyFS(dst string, fsys fs.FS) (written int64, files int, err error) {
	err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		target := filepath.Join(dst, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		if !d.Type().IsRegular() {
			return fmt.Errorf("freeio: cannot copy %q: unsupported file mode %s", path, d.Type())
		}

		info, e := d.Info()
		if e != nil {
			return e
		}

		in, e := fsys.Open(path)
		if e != nil {
			return e
		}

		defer in.Close()
		out, e := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
		if e != nil {
			return e
		}

		n, e := Copy(out, in)
		written += n
		if cerr := out.Close(); e == nil {
			e = cerr
		}
		if e != nil {
			return e
		}
		files++
		return nil
	})
	return written, files, err
}

// CopyFileWithCounter is CopyFile that also reports progress to the counters
// (see CounterFunc), for observing copy throughput, and returns the bytes copied.
func CopyFileWithCounter(dst, src string, writeCounters, readCounters []CounterFunc) (written int64, err error) {
	srcStat, srcStatErr := os.Stat(src)
	if srcStatErr != nil {
		return 0, err
	}

	if srcStat.IsDir() {
		return 0, fmt.Errorf("freeio: cannot copy directory %q", src)
	}

	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}

	defer in.Close()

	// Refuse to copy a file onto itself: O_TRUNC would wipe the source before a
	// single byte is read.
	if dstStat, statErr := os.Stat(dst); statErr == nil && os.SameFile(srcStat, dstStat) {
		return 0, fmt.Errorf("freeio: src and dst are the same file: %q", src)
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcStat.Mode().Perm())
	if err != nil {
		return 0, err
	}

	defer out.Close()
	written, err = CopyWithCounter(out, in, writeCounters, readCounters)
	return written, err
}

// ReadFile reads the named file in full and returns it in a fresh Buffer. The
// returned Buffer is the caller's to release with FreeMe. Unlike os.ReadFile it
// draws its backing storage from freebuf's pool.
func ReadFile(path string) (freebuf.Buffer, error) {
	return ReadFileWithCounter(path, nil)
}

// ReadFileWithCounter is ReadFile that also reports bytes read to readCounter
// (see CounterFunc) as the file is consumed, for observability.
func ReadFileWithCounter(path string, readCounters []CounterFunc) (freebuf.Buffer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Only a regular file reports a meaningful size; pipes/devices report 0 and
	// must grow as they read.
	var size int64
	if info, statErr := f.Stat(); statErr == nil && info.Mode().IsRegular() {
		size = info.Size()
	}

	var buf freebuf.Buffer
	if size > 0 {
		// pre-reserve; New picks Serial (<=64KB) or MultiPart
		buf = freebuf.New(int(size))
	} else {
		// unknown size: grow in pooled chunks
		buf = freebuf.NewMultiPart()
	}

	var r io.Reader = f
	if len(readCounters) > 0 {
		r = NewCounterReader(f, readCounters...)
	}

	if _, err := buf.ReadFrom(r); err != nil {
		buf.FreeMe()
		return nil, err
	}
	return buf, nil
}

// ReadFileBuffer reads the named file in full into the caller-supplied buf,
// appending to whatever buf already holds, and returns the bytes read. Unlike
// ReadFile it allocates no Buffer of its own; buf's lifecycle stays the
// caller's. A regular file's size is reserved on buf up front so the read needs
// no incremental growth.
func ReadFileBuffer(path string, buf freebuf.Buffer) (int64, error) {
	return ReadFileBufferWithCounter(path, buf, nil)
}

// ReadFileBufferWithCounter is ReadFileBuffer that also reports bytes read to
// readCounters (see CounterFunc) as the file is consumed, for observability.
func ReadFileBufferWithCounter(path string, buf freebuf.Buffer, readCounters []CounterFunc) (int64, error) {
	// Stat before Open: it is usually much lighter, and lets us learn the size
	// without an extra fstat. Only a regular file reports a meaningful size;
	// pipes/devices report 0 and grow as they read.
	var size int64
	if info, statErr := os.Stat(path); statErr == nil && info.Mode().IsRegular() {
		size = info.Size()
	}

	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// reserve only once the open has succeeded — Stat alone does not mean the
	// file is readable.
	if size > 0 {
		buf.Grow(int(size))
	}

	var r io.Reader = f
	if len(readCounters) > 0 {
		r = NewCounterReader(f, readCounters...)
	}
	return buf.ReadFrom(r)
}

// WriteFile writes the unread contents of buf to the named file, creating it
// (or truncating an existing file) with mode 0644. The buffer is drained
// (WriteTo advances its read cursor); pass buf.Copy() first if you need to keep
// the contents.
func WriteFile(path string, buf freebuf.Buffer, perm os.FileMode) error {
	return WriteFileWithCounter(path, buf, perm, nil)
}

// WriteFileWithCounter is WriteFile that also reports bytes written to
// writeCounter (see CounterFunc) as the buffer is flushed, for observability.
func WriteFileWithCounter(path string, buf freebuf.Buffer, perm os.FileMode, writeCounters []CounterFunc) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer f.Close()

	var w io.Writer = f
	if len(writeCounters) > 0 {
		w = NewCounterWriter(f, writeCounters...)
	}

	// Buffer.WriteTo drains buf into f, returning io.EOF when buf is already
	// empty; treat that as a successful no-op.
	if _, werr := buf.WriteTo(w); werr != nil && werr != io.EOF {
		return werr
	}
	return nil
}
