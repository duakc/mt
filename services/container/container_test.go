package container_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/duakc/mt/debug"
	"github.com/duakc/mt/services/container"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreLoad_RoundTrip(t *testing.T) {
	c := container.NewContainer()
	container.Store(c, "answer", 42)

	v, ok := container.Load[int](c, "answer")
	require.True(t, ok)
	assert.Equal(t, 42, v)
}

func TestStorePtrLoadPtr_SharesUnderlying(t *testing.T) {
	c := container.NewContainer()
	x := 7
	container.StorePtr(c, "p", &x)

	p, ok := container.LoadPtr[int](c, "p")
	require.True(t, ok)
	*p = 99
	assert.Equal(t, 99, x)
}

func TestReset_ClearsEntries(t *testing.T) {
	c := container.NewContainer()
	container.Store(c, "a", 1)
	container.Store(c, "b", 2)

	c.Reset()

	_, okA := container.Load[int](c, "a")
	_, okB := container.Load[int](c, "b")
	assert.False(t, okA)
	assert.False(t, okB)
}

func TestLoad_Absent(t *testing.T) {
	cases := []struct {
		name string
		run  func(c container.Container) (any, bool)
	}{
		{
			name: "Load returns zero",
			run: func(c container.Container) (any, bool) {
				v, ok := container.Load[int](c, "absent")
				return v, ok
			},
		},
		{
			name: "LoadPtr returns nil",
			run: func(c container.Container) (any, bool) {
				v, ok := container.LoadPtr[int](c, "absent")
				return v, ok
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := container.NewContainer()
			v, ok := tc.run(c)
			assert.False(t, ok)
			assert.Zero(t, v)
		})
	}
}

func TestLoad_WrongTypePanics(t *testing.T) {
	cases := []struct {
		name  string
		store func(c container.Container)
		load  func(c container.Container)
	}{
		{
			name:  "Load int as string",
			store: func(c container.Container) { container.Store(c, "k", 1) },
			load:  func(c container.Container) { container.Load[string](c, "k") },
		},
		{
			name:  "LoadPtr on non-pointer value",
			store: func(c container.Container) { container.Store(c, "k", 1) },
			load:  func(c container.Container) { container.LoadPtr[int](c, "k") },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := container.NewContainer()
			tc.store(c)
			assert.Panics(t, func() { tc.load(c) })
		})
	}
}

func TestContextRoundTrip(t *testing.T) {
	// Library never auto-creates a Container — attach one explicitly via a Provider.
	p := container.NewDefaultProvider()
	ctx := p.New(context.Background())

	container.StoreContext(ctx, "db", "conn")

	v, ok := container.LoadContext[string](ctx, "db")
	require.True(t, ok)
	assert.Equal(t, "conn", v)
}

func TestContextHelpers_NoContainer(t *testing.T) {
	// In production builds (debug.Enabled == false) the helpers must silently
	// no-op when no Container is attached. The debug-panic path is tested
	// separately below, guarded by a build tag.
	if debug.Enabled {
		t.Skip("silent no-op path is meaningful only without -tags=debug")
	}

	cases := []struct {
		name string
		run  func() (any, bool)
	}{
		{
			name: "LoadContext returns zero",
			run: func() (any, bool) {
				v, ok := container.LoadContext[int](context.Background(), "x")
				return v, ok
			},
		},
		{
			name: "LoadPtrContext returns nil",
			run: func() (any, bool) {
				v, ok := container.LoadPtrContext[int](context.Background(), "x")
				return v, ok
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v, ok := tc.run()
			assert.False(t, ok)
			assert.Zero(t, v)
		})
	}

	// Stores against a bare context should also silently disappear, not panic.
	assert.NotPanics(t, func() {
		container.StoreContext(context.Background(), "k", 1)
		x := 1
		container.StorePtrContext(context.Background(), "p", &x)
	})
}

func TestLoadContext_WrongTypePanics(t *testing.T) {
	p := container.NewDefaultProvider()
	ctx := p.New(context.Background())
	container.StoreContext(ctx, "k", "string-value")

	assert.Panics(t, func() {
		container.LoadContext[int](ctx, "k")
	})
}

func TestContainer_ConcurrentReadWrite(t *testing.T) {
	c := container.NewContainer()
	const n = 64
	for i := range n {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			k := strconv.Itoa(i)
			container.Store(c, k, i)
			v, ok := container.Load[int](c, k)
			require.True(t, ok)
			assert.Equal(t, i, v)
		})
	}
}
