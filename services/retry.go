package services

import (
	"context"
	"errors"
	"fmt"
)

type RetryStartLifeCycle struct {
	LifeCycle

	Attempt int

	attempted int
	err       []error

	nextStage Stage
}

func (rs *RetryStartLifeCycle) Reset(n int, stage Stage) {
	rs.Attempt = n
	rs.attempted = 0
	rs.err = nil
	rs.nextStage = stage
}

func (rs *RetryStartLifeCycle) StartOnce(ctx context.Context) (error, bool) {
	if rs.Attempt <= 0 {
		return errors.New("attempt must be greater than zero"), false
	}

	if rs.LifeCycle == nil {
		return errors.New("nil lifecycle"), false
	}

	if rs.nextStage >= _stageMax {
		return nil, false
	}

	if rs.attempted >= rs.Attempt {
		if len(rs.err) == 0 {
			return fmt.Errorf(
				"reached max attempt %d at stage %s",
				rs.Attempt,
				rs.nextStage,
			), false
		}
		return errors.Join(rs.err...), false
	}

	rs.attempted++

	err := rs.LifeCycle.Start(ctx, rs.nextStage)
	if err != nil {
		err = fmt.Errorf(
			"attempt %d stage %s: %w",
			rs.attempted,
			rs.nextStage,
			err,
		)
		rs.err = append(rs.err, err)

		if rs.attempted < rs.Attempt {
			return err, true
		}

		return errors.Join(rs.err...), false
	}

	rs.nextStage++
	rs.attempted = 0

	if rs.nextStage >= _stageMax {
		return nil, false
	}

	return nil, true
}

func (rs *RetryStartLifeCycle) reachMax() bool {
	return rs.attempted >= rs.Attempt
}

func StartRetry(
	ctx context.Context,
	lifecycles []LifeCycle,
	attempt int,
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
		retry.Reset(attempt, StagePreStart)

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
