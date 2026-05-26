package closeme

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/duakc/mt/services"
)

type StopCloser interface {
	Stop() error
}

type Manager interface {
	services.ContextInjector
	io.Closer

	add(fn func() error)
}

var _ Manager = (*DefaultManager)(nil)

func NewManager() Manager {
	return &DefaultManager{}
}

type DefaultManager struct {
	mu      sync.Mutex
	entries []func() error
	closed  atomic.Bool
}

func (m *DefaultManager) ContextInject(ctx context.Context) context.Context {
	return services.InjectMe[Manager](ctx, m)
}

func (m *DefaultManager) add(fn func() error) {
	if m.closed.Load() {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed.Load() {
		return
	}
	m.entries = append(m.entries, fn)
}

func (m *DefaultManager) Close() error {
	if !m.closed.CompareAndSwap(false, true) {
		return nil
	}
	m.mu.Lock()
	entries := m.entries
	m.entries = nil
	m.mu.Unlock()

	var err error
	for i := len(entries) - 1; i >= 0; i-- {
		err = errors.Join(err, entries[i]())
	}
	return err
}

func AddClose[T io.Closer](m Manager, v T) {
	m.add(v.Close)
}

func AddStop[T StopCloser](m Manager, v T) {
	m.add(v.Stop)
}

func Add(m Manager, v any) {
	switch x := v.(type) {
	case io.Closer:
		AddClose(m, x)
	case StopCloser:
		AddStop(m, x)
	}
}

var Default Manager = NewManager()
