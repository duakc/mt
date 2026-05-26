package closeme_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/duakc/mt/services"
	"github.com/duakc/mt/services/closeme"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recorder struct {
	name      string
	calledAs  string // set to "Close" or "Stop" by the cleanup call
	returnErr error
	log       *[]string
	mu        *sync.Mutex
}

func (r *recorder) record(method string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calledAs = method
	*r.log = append(*r.log, r.name+":"+method)
	return r.returnErr
}

func (r *recorder) Close() error { return r.record("Close") }
func (r *recorder) Stop() error  { return r.record("Stop") }

type closerOnly struct{ rec *recorder }

func (c *closerOnly) Close() error { return c.rec.record("Close") }

type stopperOnly struct{ rec *recorder }

func (s *stopperOnly) Stop() error { return s.rec.record("Stop") }

func newLog() (*[]string, *sync.Mutex) {
	log := []string{}
	return &log, &sync.Mutex{}
}

func TestManager_ClosesInLIFOOrder(t *testing.T) {
	log, mu := newLog()
	m := closeme.NewManager()

	closeme.AddClose(m, &closerOnly{rec: &recorder{name: "a", log: log, mu: mu}})
	closeme.AddClose(m, &closerOnly{rec: &recorder{name: "b", log: log, mu: mu}})
	closeme.AddClose(m, &closerOnly{rec: &recorder{name: "c", log: log, mu: mu}})

	require.NoError(t, m.Close())
	assert.Equal(t, []string{"c:Close", "b:Close", "a:Close"}, *log)
}

func TestManager_CallsStopForStopCloser(t *testing.T) {
	log, mu := newLog()
	m := closeme.NewManager()

	closeme.AddStop(m, &stopperOnly{rec: &recorder{name: "s", log: log, mu: mu}})

	require.NoError(t, m.Close())
	assert.Equal(t, []string{"s:Stop"}, *log)
}

func TestManager_AddDispatchesByInterface(t *testing.T) {
	cases := []struct {
		name   string
		make   func(*recorder) any
		wantAs string
	}{
		{"closer only -> Close", func(r *recorder) any { return &closerOnly{rec: r} }, "Close"},
		{"stopper only -> Stop", func(r *recorder) any { return &stopperOnly{rec: r} }, "Stop"},
		{"both -> Close wins", func(r *recorder) any { return r }, "Close"},
		{"neither -> dropped", func(r *recorder) any { return struct{}{} }, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			log, mu := newLog()
			rec := &recorder{name: "r", log: log, mu: mu}
			m := closeme.NewManager()
			closeme.Add(m, tc.make(rec))
			require.NoError(t, m.Close())
			if tc.wantAs == "" {
				assert.Empty(t, *log)
			} else {
				assert.Equal(t, []string{"r:" + tc.wantAs}, *log)
			}
		})
	}
}

func TestManager_JoinsErrors(t *testing.T) {
	log, mu := newLog()
	m := closeme.NewManager()
	errA := errors.New("err-a")
	errB := errors.New("err-b")

	closeme.AddClose(m, &closerOnly{rec: &recorder{name: "a", returnErr: errA, log: log, mu: mu}})
	closeme.AddClose(m, &closerOnly{rec: &recorder{name: "b", returnErr: errB, log: log, mu: mu}})

	err := m.Close()
	require.Error(t, err)
	assert.True(t, errors.Is(err, errA))
	assert.True(t, errors.Is(err, errB))
}

func TestManager_CloseIsIdempotent(t *testing.T) {
	log, mu := newLog()
	m := closeme.NewManager()
	closeme.AddClose(m, &closerOnly{rec: &recorder{name: "a", log: log, mu: mu}})

	require.NoError(t, m.Close())
	require.NoError(t, m.Close())
	assert.Equal(t, []string{"a:Close"}, *log, "Close should only fire registered cleanups once")
}

func TestManager_AddAfterCloseIsDropped(t *testing.T) {
	log, mu := newLog()
	m := closeme.NewManager()
	require.NoError(t, m.Close())

	closeme.AddClose(m, &closerOnly{rec: &recorder{name: "late", log: log, mu: mu}})
	assert.Empty(t, *log, "registrations after Close must be ignored, not buffered")
}

func TestManager_ConcurrentAdd(t *testing.T) {
	log, mu := newLog()
	m := closeme.NewManager()

	const n = 64
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func() {
			defer wg.Done()
			closeme.AddClose(m, &closerOnly{rec: &recorder{name: "x", log: log, mu: mu}})
			_ = i
		}()
	}
	wg.Wait()

	require.NoError(t, m.Close())
	assert.Len(t, *log, n)
}

func TestManager_RegisteredAsServiceViaContextInject(t *testing.T) {
	m := closeme.NewManager()
	ctx := m.ContextInject(context.Background())

	got := services.Lookup[closeme.Manager](ctx)
	require.NotNil(t, got)
	assert.Same(t, m, got)
}

func TestDefault_IsUsableAsGlobalManager(t *testing.T) {
	// Default is mutable so tests can isolate themselves; always restore.
	prev := closeme.Default
	closeme.Default = closeme.NewManager()
	t.Cleanup(func() { closeme.Default = prev })

	log, mu := newLog()
	closeme.AddClose(closeme.Default, &closerOnly{rec: &recorder{name: "c", log: log, mu: mu}})
	closeme.AddStop(closeme.Default, &stopperOnly{rec: &recorder{name: "s", log: log, mu: mu}})

	require.NoError(t, closeme.Default.Close())
	// LIFO: stopper registered last → runs first.
	assert.Equal(t, []string{"s:Stop", "c:Close"}, *log)
}

// Compile-time assertion: Manager satisfies io.Closer (so things like
// services.CloseService can be passed a Manager and just work).
var _ io.Closer = closeme.NewManager()
