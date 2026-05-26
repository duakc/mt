package services_test

import (
	"context"
	"testing"

	"github.com/duakc/mt/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeServiceKey struct{}

type fakeService struct {
	name string
}

func (s fakeService) ContextInject(ctx context.Context) context.Context {
	return context.WithValue(ctx, fakeServiceKey{}, s)
}

func TestInjectMe_StoresAndLookup(t *testing.T) {
	ctx := services.InjectMe(context.Background(), fakeService{name: "hello"})

	v := services.Lookup[fakeService](ctx)
	assert.Equal(t, "hello", v.name)
}

func TestInjectMe_LastWriteWins(t *testing.T) {
	ctx := context.Background()
	ctx = services.InjectMe(ctx, fakeService{name: "A"})
	ctx = services.InjectMe(ctx, fakeService{name: "B"})

	v := services.Lookup[fakeService](ctx)
	assert.Equal(t, "B", v.name)
}

func TestLookupDefault(t *testing.T) {
	cases := []struct {
		name     string
		setup    func() context.Context
		wantName string
	}{
		{
			name:     "no registry on context",
			setup:    func() context.Context { return context.Background() },
			wantName: "dft",
		},
		{
			name: "registry present but entry missing",
			setup: func() context.Context {
				return services.NewRegistry(context.Background(), services.NewDefaultRegistry())
			},
			wantName: "dft",
		},
		{
			name: "entry present overrides default",
			setup: func() context.Context {
				return services.InjectMe(context.Background(), fakeService{name: "real"})
			},
			wantName: "real",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := services.LookupDefault(tc.setup(), fakeService{name: "dft"})
			assert.Equal(t, tc.wantName, v.name)
		})
	}
}

func TestStore_DispatchesContextInject(t *testing.T) {
	ctx := services.Store(context.Background(), fakeService{name: "via-store"})

	v, ok := ctx.Value(fakeServiceKey{}).(fakeService)
	require.True(t, ok)
	assert.Equal(t, "via-store", v.name)
}
