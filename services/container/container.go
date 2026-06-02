package container

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/duakc/mt"
	"github.com/duakc/mt/debug"
)

type Container interface {
	Load(k string) (any, bool)
	Store(k string, v any)
	Delete(k string)

	IncRef()
	DecRef()

	Release() bool
}

var _ Container = (*defaultContainer)(nil)

type defaultContainer struct {
	m sync.Map

	ref atomic.Int32
}

func (c *defaultContainer) Release() bool {
	if c.ref.Load() == 0 {
		// reset
		c.m.Clear()
		return true
	}
	return false
}

func NewContainer() Container {
	return &defaultContainer{}
}

func (c *defaultContainer) IncRef() {
	c.ref.Add(1)
}

func (c *defaultContainer) DecRef() {
	c.ref.Add(-1)
}

func (c *defaultContainer) Load(k string) (any, bool) {
	return c.m.Load(k)
}

func (c *defaultContainer) Store(k string, v any) {
	c.m.Store(k, v)
}

func (c *defaultContainer) Delete(k string) {
	c.m.Delete(k)
}

func Store[T any](c Container, k string, v T) {
	c.Store(k, v)
}

func StorePtr[T any](c Container, k string, v *T) {
	c.Store(k, v)
}

func Load[T any](c Container, k string) (T, bool) {
	l, ok := c.Load(k)
	if !ok {
		return mt.Zero[T](), false
	}
	return l.(T), true
}

func LoadPtr[T any](c Container, k string) (*T, bool) {
	l, ok := c.Load(k)
	if !ok {
		return nil, false
	}
	return l.(*T), true
}

type containerKey struct{}

func FromContext(ctx context.Context) (Container, bool) {
	c, ok := ctx.Value(containerKey{}).(Container)
	if !ok {
		return nil, false
	}
	return c, true
}

func mustContainer(ctx context.Context) Container {
	c, ok := FromContext(ctx)
	if ok {
		return c
	}
	if debug.Enabled {
		panic("container: no Container on context — call Provider.New(ctx) before Store/Load helpers")
	}
	return nil
}

func StoreContext[T any](ctx context.Context, k string, v T) {
	if c := mustContainer(ctx); c != nil {
		Store(c, k, v)
	}
}

func StorePtrContext[T any](ctx context.Context, k string, v *T) {
	if c := mustContainer(ctx); c != nil {
		StorePtr(c, k, v)
	}
}

func LoadContext[T any](ctx context.Context, k string) (T, bool) {
	c := mustContainer(ctx)
	if c == nil {
		return mt.Zero[T](), false
	}
	return Load[T](c, k)
}

func LoadPtrContext[T any](ctx context.Context, k string) (*T, bool) {
	c := mustContainer(ctx)
	if c == nil {
		return nil, false
	}
	return LoadPtr[T](c, k)
}
