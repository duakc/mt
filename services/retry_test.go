package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryStartLifeCycleSuccess(t *testing.T) {
	lifecycle := &retryTestLifeCycle{}
	retry := &RetryStartLifeCycle{LifeCycle: lifecycle}
	retry.Reset(2, StageStart)

	err, shouldRetry := retry.StartOnce(context.Background())

	require.NoError(t, err)
	assert.False(t, shouldRetry)
	assert.Equal(t, []Stage{StageStart}, lifecycle.stages)
}

func TestRetryStartLifeCycleRetryThenSuccess(t *testing.T) {
	firstErr := errors.New("first failure")

	lifecycle := &retryTestLifeCycle{
		errs: []error{firstErr},
	}
	retry := &RetryStartLifeCycle{LifeCycle: lifecycle}
	retry.Reset(2, StageStart)

	err, shouldRetry := retry.StartOnce(context.Background())

	require.ErrorIs(t, err, firstErr)
	assert.True(t, shouldRetry)

	err, shouldRetry = retry.StartOnce(context.Background())

	require.NoError(t, err)
	assert.False(t, shouldRetry)
	assert.Equal(t, []Stage{StageStart, StageStart}, lifecycle.stages)
}

func TestRetryStartLifeCycleReturnsAllErrors(t *testing.T) {
	firstErr := errors.New("first failure")
	secondErr := errors.New("second failure")

	lifecycle := &retryTestLifeCycle{
		errs: []error{firstErr, secondErr},
	}
	retry := &RetryStartLifeCycle{LifeCycle: lifecycle}
	retry.Reset(2, StageStart)

	_, shouldRetry := retry.StartOnce(context.Background())
	assert.True(t, shouldRetry)

	err, shouldRetry := retry.StartOnce(context.Background())

	require.Error(t, err)
	assert.False(t, shouldRetry)
	assert.ErrorIs(t, err, firstErr)
	assert.ErrorIs(t, err, secondErr)
	assert.Len(t, lifecycle.stages, 2)
}

func TestStartRetryDoesNotReturnErrorForSuccessfulServices(t *testing.T) {
	lifecycle := &retryTestLifeCycle{}

	err := StartRetry(
		context.Background(),
		StagePreStart,
		2,
		lifecycle,
	)

	require.NoError(t, err)
	assert.Equal(t, []Stage{StagePreStart}, lifecycle.stages)
}

func TestStartRetryRetriesOnlyFailedServices(t *testing.T) {
	recoverableErr := errors.New("recoverable failure")
	failedFirstErr := errors.New("failed first")
	failedSecondErr := errors.New("failed second")

	success := &retryTestLifeCycle{}

	recoverable := &retryTestLifeCycle{
		errs: []error{recoverableErr},
	}

	failed := &retryTestLifeCycle{
		errs: []error{
			failedFirstErr,
			failedSecondErr,
		},
	}

	err := StartRetry(
		context.Background(),
		StageStart,
		2,
		success,
		recoverable,
		failed,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, failedFirstErr)
	assert.ErrorIs(t, err, failedSecondErr)

	assert.Equal(t, []Stage{StageStart}, success.stages)
	assert.Equal(t, []Stage{StageStart, StageStart}, recoverable.stages)
	assert.Equal(t, []Stage{StageStart, StageStart}, failed.stages)
}

type retryTestLifeCycle struct {
	stages []Stage
	errs   []error
}

func (f *retryTestLifeCycle) Start(
	_ context.Context,
	stage Stage,
) error {
	f.stages = append(f.stages, stage)

	if len(f.errs) == 0 {
		return nil
	}

	err := f.errs[0]
	f.errs = f.errs[1:]
	return err
}

func (f *retryTestLifeCycle) Close() error {
	return nil
}
