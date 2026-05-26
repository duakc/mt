package container

import (
	"context"

	"github.com/duakc/mt/services"
)

type Provider interface {
	services.ContextInjector

	New(ctx context.Context) context.Context
}

var _ Provider = (*DefaultProvider)(nil)

type DefaultProvider struct{}

func (p *DefaultProvider) New(ctx context.Context) context.Context {
	return WithContext(ctx)
}

func (p *DefaultProvider) ContextInject(ctx context.Context) context.Context {
	return services.InjectMe[Provider](ctx, p)
}
