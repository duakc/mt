package container

import (
	"context"

	"github.com/duakc/mt/common/generic"
	"github.com/duakc/mt/services"
)

type Provider interface {
	services.ContextInjector

	New(ctx context.Context) context.Context
	Release(ctx context.Context)
}

type Factory interface {
	NewContainer() Container
}

var _ Factory = (*FuncFactory)(nil)

type FuncFactory func() Container

func (f FuncFactory) NewContainer() Container {
	return f()
}

var DefaultFactory = FuncFactory(NewContainer)

var _ Provider = (*DefaultProvider)(nil)

type DefaultProvider struct {
	factory Factory
	pool    *generic.Pool[Container]
}

func NewDefaultProvider() *DefaultProvider {
	return NewProvider(DefaultFactory)
}

func NewProvider(factory Factory) *DefaultProvider {
	if factory == nil {
		factory = DefaultFactory
	}

	p := &DefaultProvider{factory: factory}
	p.pool = generic.NewPool(factory.NewContainer)
	return p
}

func (p *DefaultProvider) New(ctx context.Context) context.Context {
	if _, ok := FromContext(ctx); ok {
		return ctx
	}
	return context.WithValue(ctx, containerKey{}, p.pool.Get())
}

func (p *DefaultProvider) Release(ctx context.Context) {
	c, ok := FromContext(ctx)
	if !ok && !c.Release() {
		return
	}

	p.pool.Put(c)
}

func (p *DefaultProvider) ContextInject(ctx context.Context) context.Context {
	return services.InjectMe[Provider](ctx, p)
}
