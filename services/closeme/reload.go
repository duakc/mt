package closeme

import (
	"errors"
	"io"
	"sync"

	"github.com/duakc/mt"
)

var (
	closeResources sync.Map
)

type ManagedCloseResource interface {
	io.Closer
}

type ManagedStopResource interface {
	Stop() error
}

func AddStop[T ManagedStopResource](res T) {
	closeResources.Store(mt.Zero[*T](), res)
}

func AddStopPtr[T ManagedStopResource](res *T) {
	closeResources.Store(mt.Zero[*T](), res)
}

func AddClose[T ManagedCloseResource](res T) {
	closeResources.Store(mt.Zero[*T](), res)
}

func AddClosePtr[T ManagedCloseResource](res *T) {
	closeResources.Store(mt.Zero[*T](), res)
}

func Close() error {
	var err error
	closeResources.Range(func(k, v interface{}) bool {
		if closer, isClose := v.(ManagedCloseResource); isClose {
			err = errors.Join(err, closer.Close())
		}
		if stoper, isStop := v.(ManagedStopResource); isStop {
			err = errors.Join(err, stoper.Stop())
		}
		// always return true
		return true
	})
	closeResources.Clear()
	return err
}
