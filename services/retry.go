package services

import (
	"context"
	"errors"
	"fmt"
)

type RetryStartLifeCycle struct {
	LifeCycle

	attempt int

	attempted int

	err   []error
	stage Stage
}

func (rs *RetryStartLifeCycle) Reset(n int, stage Stage) {
	rs.attempt = n
	rs.attempted = 0
	for i := 0; i < len(rs.err); i++ {
		// avoid memory leak
		rs.err[i] = nil
	}

	rs.err = nil
	rs.stage = stage
}

func (rs *RetryStartLifeCycle) StartOnce(ctx context.Context) (error, bool) {
	if rs.attempt <= 0 {
		return errors.New("attempt must be greater than zero"), false
	}

	if rs.LifeCycle == nil {
		return errors.New("nil lifecycle"), false
	}

	if rs.stage >= _stageMax {
		return nil, false
	}

	if rs.attempted >= rs.attempt {
		if len(rs.err) == 0 {
			return fmt.Errorf(
				"reached max attempt %d at stage %s",
				rs.attempt,
				rs.stage.String(),
			), false
		}
		return errors.Join(rs.err...), false
	}

	rs.attempted++

	err := rs.LifeCycle.Start(ctx, rs.stage)
	if err != nil {
		err = fmt.Errorf(
			"attempt %d stage %s: %w",
			rs.attempted,
			rs.stage.String(),
			err,
		)
		rs.err = append(rs.err, err)

		if rs.attempted < rs.attempt {
			return err, true
		}

		return errors.Join(rs.err...), false
	}

	return nil, true
}

func StartRetry[T LifeCycle](
	ctx context.Context,
	stage Stage,
	attempt int,
	lifecycles ...T,
) error {
	if attempt <= 0 {
		return errors.New("attempt must be greater than zero")
	}

	type indexedLifecycle struct {
		index int
		value *RetryStartLifeCycle
	}

	queue := make([]indexedLifecycle, 0, len(lifecycles))

	for index, lifecycle := range lifecycles {
		retry := &RetryStartLifeCycle{
			LifeCycle: lifecycle,
		}
		retry.Reset(attempt, stage)

		queue = append(queue, indexedLifecycle{
			index: index,
			value: retry,
		})
	}

	var result error

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		err, retry := current.value.StartOnce(ctx)

		if retry {
			queue = append(queue, current)
			continue
		}

		if err == nil {
			continue
		}

		result = errors.Join(
			result,
			fmt.Errorf(
				"lifecycles[%d]: reached max attempt %d: %w",
				current.index,
				attempt,
				err,
			),
		)
	}

	return result
}
