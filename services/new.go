package services

import (
	"context"

	"github.com/duakc/mt"
)

type Registry interface {
	Store(k, v any)
	Load(k any) any
}

type registryKey struct{}

func LookupDefault[K comparable](ctx context.Context, dft K) K {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		return dft
	}
	if v := registry.Load(mt.Zero[*K]()); v != nil {
		return v.(K)
	}
	return dft
}

func Lookup[K comparable](ctx context.Context) K {
	return LookupDefault(ctx, mt.Zero[K]())
}

func LookupPtrDefault[K comparable](ctx context.Context, dft *K) *K {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		return dft
	}
	if v := registry.Load(mt.Zero[*K]()); v != nil {
		return v.(*K)
	}
	return dft
}

func LookupPtr[K comparable](ctx context.Context) *K {
	return LookupPtrDefault[K](ctx, nil)
}

func Store[K comparable](ctx context.Context, value K) {
	registry := RegistryFromContext(ctx)
	registry.Store(mt.Zero[*K](), value)
}

func StorePtr[K comparable](ctx context.Context, value *K) {
	registry := RegistryFromContext(ctx)
	registry.Store(mt.Zero[*K](), value)
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
