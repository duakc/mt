package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/duakc/mt/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLifecycle struct {
	starts   []services.Stage
	closed   bool
	failAt   services.Stage // -1 = never fail
	closeErr error
}

func (f *fakeLifecycle) Start(ctx context.Context, stage services.Stage) error {
	f.starts = append(f.starts, stage)
	if stage == f.failAt {
		return errors.New("boom")
	}
	return nil
}

func (f *fakeLifecycle) Close() error {
	f.closed = true
	return f.closeErr
}

func TestStartService(t *testing.T) {
	allStages := []services.Stage{
		services.StagePreStart,
		services.StageStart,
		services.StagePostStart,
	}

	cases := []struct {
		name       string
		failAt     services.Stage
		wantStages []services.Stage
		wantErr    string // "" means no error; otherwise the Stage field expected on LifeCycleError
	}{
		{"runs every stage in order", -1, allStages, ""},
		{"halts on PreStart failure", services.StagePreStart, allStages[:1], "PreStart"},
		{"halts on Start failure", services.StageStart, allStages[:2], "Start"},
		{"halts on PostStart failure", services.StagePostStart, allStages[:3], "PostStart"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &fakeLifecycle{failAt: tc.failAt}
			err := services.StartService(context.Background(), f)

			assert.Equal(t, tc.wantStages, f.starts)
			if tc.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			var lce *services.LifeCycleError
			require.ErrorAs(t, err, &lce)
			assert.Equal(t, tc.wantErr, lce.Stage)
		})
	}
}

func TestCloseService(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name     string
		closeErr error
		wantErr  error
	}{
		{"no error", nil, nil},
		{"wraps inner error", boom, boom},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &fakeLifecycle{failAt: -1, closeErr: tc.closeErr}
			err := services.CloseService(f)

			assert.True(t, f.closed)
			if tc.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			var lce *services.LifeCycleError
			require.ErrorAs(t, err, &lce)
			assert.Equal(t, "Close", lce.Stage)
			assert.Same(t, tc.wantErr, lce.Err)
		})
	}
}

func TestStartService_NonStarterIsNoop(t *testing.T) {
	type plain struct{}
	assert.NoError(t, services.StartService(context.Background(), plain{}))
}

func TestCloseService_NonCloserIsNoop(t *testing.T) {
	type plain struct{}
	assert.NoError(t, services.CloseService(plain{}))
}
