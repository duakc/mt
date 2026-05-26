package container

import (
	"context"
	"sync"

	"github.com/duakc/mt"
)

type Container interface {
	Load(k string) (any, bool)
	Store(k string, v any)
}

type defaultContainer struct {
	m sync.Map
}

func NewContainer() Container {
	return &defaultContainer{}
}

func (c *defaultContainer) Load(k string) (any, bool) {
	return c.m.Load(k)
}

func (c *defaultContainer) Store(k string, v any) {
	c.m.Store(k, v)
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

func WithContext(ctx context.Context) context.Context {
	if _, ok := FromContext(ctx); ok {
		return ctx
	}
	return context.WithValue(ctx, containerKey{}, NewContainer())
}

func FromContext(ctx context.Context) (Container, bool) {
	c, ok := ctx.Value(containerKey{}).(Container)
	if !ok {
		return nil, false
	}
	return c, true
}

func StoreContext[T any](ctx context.Context, k string, v T) context.Context {
	c, ok := FromContext(ctx)
	if !ok {
		ctx = WithContext(ctx)
		c, _ = FromContext(ctx)
	}
	Store(c, k, v)
	return ctx
}

func StorePtrContext[T any](ctx context.Context, k string, v *T) context.Context {
	c, ok := FromContext(ctx)
	if !ok {
		ctx = WithContext(ctx)
		c, _ = FromContext(ctx)
	}
	StorePtr(c, k, v)
	return ctx
}

func LoadContext[T any](ctx context.Context, k string) (T, bool) {
	c, ok := FromContext(ctx)
	if !ok {
		return mt.Zero[T](), false
	}
	return Load[T](c, k)
}

func LoadPtrContext[T any](ctx context.Context, k string) (*T, bool) {
	c, ok := FromContext(ctx)
	if !ok {
		return nil, false
	}
	return LoadPtr[T](c, k)
}
