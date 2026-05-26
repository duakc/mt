package services

import (
	"context"

	"github.com/duakc/mt"
)

type ContextInjector interface {
	ContextInject(ctx context.Context) context.Context
}

type Registry interface {
	Store(k, v any)
	Load(k any) any

	Seal()
}

type registryKey struct{}

func LookupDefault[K ContextInjector](ctx context.Context, dft K) K {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		return dft
	}
	if v := registry.Load(mt.Zero[*K]()); v != nil {
		return v.(K)
	}
	return dft
}

func Lookup[K ContextInjector](ctx context.Context) K {
	return LookupDefault(ctx, mt.Zero[K]())
}

func LookupPtrDefault[K ContextInjector](ctx context.Context, dft *K) *K {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		return dft
	}
	if v := registry.Load(mt.Zero[*K]()); v != nil {
		return v.(*K)
	}
	return dft
}

func LookupPtr[K ContextInjector](ctx context.Context) *K {
	return LookupPtrDefault[K](ctx, nil)
}

func Store[K ContextInjector](ctx context.Context, value K) context.Context {
	return value.ContextInject(ctx)
}

func StorePtr[K ContextInjector](ctx context.Context, value *K) context.Context {
	return (*value).ContextInject(ctx)
}

func NewRegistry(ctx context.Context, r Registry) context.Context {
	return context.WithValue(ctx, registryKey{}, r)
}

func RegistryFromContext(ctx context.Context) Registry {
	registry := ctx.Value(registryKey{})
	if registry == nil {
		return nil
	}
	return registry.(Registry)
}

func InjectMe[T ContextInjector](ctx context.Context, v T) context.Context {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		registry = NewDefaultRegistry()
		ctx = context.WithValue(ctx, registryKey{}, registry)
	}
	registry.Store(mt.Zero[*T](), v)
	return ctx
}
