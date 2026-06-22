package filehelper

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/duakc/mt/services"
)

type Helper interface {
	io.Closer
	services.ContextInjector

	Root() *os.Root
	Create(name string) (*os.File, error)
	Open(name string) (*os.File, error)
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)

	MkdirAll(path string, perm os.FileMode) error
	Path(name string) string
	Stat(name string) (os.FileInfo, error)

	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	MustReadFile(name string) []byte
	MustWriteFile(name string, data []byte, perm os.FileMode)
}

var _ Helper = (*DefaultFileHelper)(nil)

type DefaultFileHelper struct {
	dir string

	root *os.Root

	closeOnce sync.Once
	closeErr  error
}

func (h *DefaultFileHelper) ReadFile(name string) ([]byte, error) {
	return h.root.ReadFile(name)
}

func (h *DefaultFileHelper) WriteFile(name string, data []byte, perm os.FileMode) error {
	return h.root.WriteFile(name, data, perm)
}

func (h *DefaultFileHelper) MustReadFile(name string) []byte {
	data, err := h.root.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return data
}

func (h *DefaultFileHelper) MustWriteFile(name string, data []byte, perm os.FileMode) {
	err := h.root.WriteFile(name, data, perm)
	if err != nil {
		panic(err)
	}
}

func (h *DefaultFileHelper) ContextInject(ctx context.Context) context.Context {
	return services.InjectMe[Helper](ctx, h)
}

func (h *DefaultFileHelper) Stat(name string) (os.FileInfo, error) {
	return h.root.Stat(name)
}

func New(dir string) (*DefaultFileHelper, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(absDir)
	if err != nil {
		return nil, err
	}
	return &DefaultFileHelper{
		dir:  absDir,
		root: root,
	}, nil
}

func (h *DefaultFileHelper) Close() error {
	h.closeOnce.Do(func() {
		h.closeErr = h.root.Close()
	})
	return h.closeErr
}

func (h *DefaultFileHelper) Root() *os.Root {
	return h.root
}

func (h *DefaultFileHelper) Create(name string) (*os.File, error) {
	if err := h.mkdir(name); err != nil {
		return nil, err
	}
	return h.root.Create(name)
}

func (h *DefaultFileHelper) Open(name string) (*os.File, error) {
	return h.root.Open(name)
}

func (h *DefaultFileHelper) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	if flag&os.O_CREATE > 0 {
		err := h.mkdir(name)
		if err != nil {
			return nil, err
		}
	}
	return h.root.OpenFile(name, flag, perm)
}

func (h *DefaultFileHelper) MkdirAll(path string, perm os.FileMode) error {
	return h.root.MkdirAll(path, perm)
}

func (h *DefaultFileHelper) Path(name string) string {
	return filepath.Join(h.dir, name)
}

func (h *DefaultFileHelper) mkdir(name string) error {
	dir := filepath.Dir(name)
	return h.root.MkdirAll(dir, 0o777)
}
