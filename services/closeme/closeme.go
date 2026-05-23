package closeme

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

// Manager collects resources and closes them all at once.
// Use [AddClose], [AddClosePtr], [AddStop], or [AddStopPtr]
// to register resources. The pointer itself is used as the
// key, so every registered value is unique and GC-safe.
//
// Each resource must be registered only once — registering
// the same pointer via different methods will overwrite the
// previous entry.
type Manager interface {
	store(key any, closer io.Closer)
	storeStop(key any, stopper StopCloser)

	Close() error
}

// StopCloser is a resource that can be stopped.
// A value implementing both [StopCloser] and [io.Closer]
// will only have Stop called during shutdown.
type StopCloser interface {
	Stop() error
}

// NewManager creates a new Manager.
func NewManager() Manager {
	return &defaultManager{}
}

type defaultManager struct {
	m      sync.Map
	closed atomic.Bool
}

func (m *defaultManager) store(key any, closer io.Closer) {
	if m.closed.Load() {
		return
	}
	m.m.Store(key, closer)
}

func (m *defaultManager) storeStop(key any, stopper StopCloser) {
	if m.closed.Load() {
		return
	}
	m.m.Store(key, stopper)
}

func (m *defaultManager) Close() error {
	m.closed.Store(true)

	var err error
	m.m.Range(func(_, v any) bool {
		if stopper, ok := v.(StopCloser); ok {
			err = errors.Join(err, stopper.Stop())
		} else if closer, ok := v.(io.Closer); ok {
			err = errors.Join(err, closer.Close())
		}
		return true
	})
	m.m.Clear()
	return err
}

// AddClose registers v with m. The pointer v itself is used as
// the key — every instance is unique and safe from GC reclamation.
// Use this for types that directly implement [io.Closer]
// (e.g. *os.File or any interface embedding io.Closer).
func AddClose[T io.Closer](m Manager, v T) {
	m.store(v, v)
}

// AddClosePtr registers v with m. The pointer v itself is used as
// the key. Use this for concrete types whose pointer receiver
// implements [io.Closer].
//
//	type myService struct{ ... }
//	func (*myService) Close() error { return nil }
//
//	svc := &myService{}
//	closeme.AddClosePtr(mgr, svc)
func AddClosePtr[T any, PT interface {
	*T
	io.Closer
}](m Manager, v PT) {
	m.store(v, v)
}

// AddStop registers v with m. The pointer v itself is used as
// the key. Use this for types that directly implement [StopCloser].
func AddStop[T StopCloser](m Manager, v T) {
	m.storeStop(v, v)
}

// AddStopPtr registers v with m. The pointer v itself is used as
// the key. Use this for concrete types whose pointer receiver
// implements [StopCloser].
func AddStopPtr[T any, PT interface {
	*T
	StopCloser
}](m Manager, v PT) {
	m.storeStop(v, v)
}
