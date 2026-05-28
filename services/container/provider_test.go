package container_test

import (
	"context"
	"testing"

	"github.com/duakc/mt/services"
	"github.com/duakc/mt/services/container"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_NewAttachesContainer(t *testing.T) {
	var p container.Provider = container.NewDefaultProvider()

	ctx, c := p.New(context.Background())
	got, ok := container.FromContext(ctx)
	assert.True(t, ok)
	assert.Same(t, c, got)
}

func TestProvider_NewIsIdempotent(t *testing.T) {
	var p container.Provider = container.NewDefaultProvider()

	ctx, c1 := p.New(context.Background())

	ctx, c2 := p.New(ctx)
	_ = ctx

	assert.Same(t, c1, c2)
}

func TestProvider_ContextInjectRegistersIntoServices(t *testing.T) {
	p := container.NewDefaultProvider()
	ctx := p.ContextInject(context.Background())

	got := services.Lookup[container.Provider](ctx)
	require.NotNil(t, got)
	assert.Same(t, p, got)
}

func TestNewProvider_UsesCustomFactory(t *testing.T) {
	calls := 0
	factory := container.FuncFactory(func() container.Container {
		calls++
		c := container.NewContainer()
		container.Store(c, "factory-seeded", true)
		return c
	})

	p := container.NewProvider(factory)
	ctx, _ := p.New(context.Background())

	v, ok := container.LoadContext[bool](ctx, "factory-seeded")
	require.True(t, ok)
	assert.True(t, v)
	assert.Equal(t, 1, calls, "factory should be invoked exactly once on first New")
}

func TestNewProvider_NilFactoryFallsBackToDefault(t *testing.T) {
	p := container.NewProvider(nil)
	ctx, _ := p.New(context.Background())

	_, ok := container.FromContext(ctx)
	assert.True(t, ok)
}

func TestProvider_ReleaseResetsAndPools(t *testing.T) {
	p := container.NewDefaultProvider()

	ctx, c := p.New(context.Background())
	container.Store(c, "k", "v")

	p.ReleaseContext(ctx)

	// The container we just released should be reset; we can probe it
	// directly because we still hold the reference.
	_, ok := container.Load[string](c, "k")
	assert.False(t, ok, "Release must clear container entries before pooling")
}

func TestProvider_ReleaseOnBareContextIsNoop(t *testing.T) {
	p := container.NewDefaultProvider()
	assert.NotPanics(t, func() {
		p.ReleaseContext(context.Background())
	})
}

func TestProvider_ReleaseContextBlockedByRef(t *testing.T) {
	p := container.NewDefaultProvider()

	ctx, c := p.New(context.Background())
	container.Store(c, "k", "v")
	c.IncRef()

	p.ReleaseContext(ctx)

	v, ok := container.Load[string](c, "k")
	require.True(t, ok, "entries must survive ReleaseContext while a ref is outstanding")
	assert.Equal(t, "v", v)

	c.DecRef()
	p.ReleaseContext(ctx)
	_, ok = container.Load[string](c, "k")
	assert.False(t, ok, "after the last DecRef, ReleaseContext must clear entries")
}

func TestProvider_ReleaseBlockedByRef(t *testing.T) {
	p := container.NewDefaultProvider()

	_, c := p.New(context.Background())
	container.Store(c, "k", "v")
	c.IncRef()

	p.Release(c)

	v, ok := container.Load[string](c, "k")
	require.True(t, ok, "entries must survive Release while a ref is outstanding")
	assert.Equal(t, "v", v)
}

func TestDefaultProvider_EndToEnd(t *testing.T) {
	p := container.NewDefaultProvider()
	ctx := p.ContextInject(context.Background())

	resolved := services.Lookup[container.Provider](ctx)
	ctx, _ = resolved.New(ctx)
	defer resolved.ReleaseContext(ctx)

	container.StoreContext(ctx, "tracer", "noop")
	v, ok := container.LoadContext[string](ctx, "tracer")
	require.True(t, ok)
	assert.Equal(t, "noop", v)
}
