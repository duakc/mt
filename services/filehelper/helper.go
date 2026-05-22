package filehelper

import (
	"io"
	"os"
	"path/filepath"
	"sync"
)

type Helper interface {
	io.Closer

	Root() *os.Root
	Create(name string) (*os.File, error)
	Open(name string) (*os.File, error)
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)

	MkdirAll(path string, perm os.FileMode) error
}

var _ Helper = (*DefaultFileHelper)(nil)

type DefaultFileHelper struct {
	dir string

	root *os.Root

	closeOnce sync.Once
	closeErr  error
}

func New(dir string) (*DefaultFileHelper, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	return &DefaultFileHelper{
		dir:  dir,
		root: root,
	}, nil
}

func NewMkdir(dir string) (*DefaultFileHelper, error) {
	err := os.MkdirAll(filepath.Dir(dir), 0777)
	if err != nil {
		return nil, err
	}
	return New(dir)
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
	return os.OpenFile(name, flag, perm)
}

func (h *DefaultFileHelper) MkdirAll(path string, perm os.FileMode) error {
	return h.root.MkdirAll(path, perm)
}

func (h *DefaultFileHelper) mkdir(name string) error {
	dir := filepath.Dir(name)
	return h.root.MkdirAll(dir, 0777)
}
