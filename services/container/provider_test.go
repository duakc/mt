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
	var p container.Provider = &container.DefaultProvider{}

	ctx := p.New(context.Background())
	_, ok := container.FromContext(ctx)
	assert.True(t, ok)
}

func TestProvider_NewIsIdempotent(t *testing.T) {
	var p container.Provider = &container.DefaultProvider{}

	ctx := p.New(context.Background())
	c1, _ := container.FromContext(ctx)

	ctx = p.New(ctx)
	c2, _ := container.FromContext(ctx)

	assert.Same(t, c1, c2)
}

func TestProvider_ContextInjectRegistersIntoServices(t *testing.T) {
	p := &container.DefaultProvider{}
	ctx := p.ContextInject(context.Background())

	got := services.Lookup[container.Provider](ctx)
	require.NotNil(t, got)
	assert.Same(t, p, got)
}

func TestDefaultProvider_EndToEnd(t *testing.T) {
	p := &container.DefaultProvider{}
	ctx := p.ContextInject(context.Background())

	resolved := services.Lookup[*container.DefaultProvider](ctx)
	ctx = resolved.New(ctx)

	c, ok := container.FromContext(ctx)
	require.True(t, ok)

	container.Store(c, "tracer", "noop")
	v, ok := container.Load[string](c, "tracer")
	require.True(t, ok)
	assert.Equal(t, "noop", v)
}
