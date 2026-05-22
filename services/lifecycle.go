package services

import (
	"context"
	"fmt"
	"io"
)

type Stage int

const (
	StagePreStart Stage = iota
	StageStart
	StagePostStart

	StageClose
)

type LifeCycle interface {
	Starter
	io.Closer
}

func (s Stage) String() string {
	switch s {
	case StagePreStart:
		return "PreStart"
	case StageStart:
		return "Start"
	case StagePostStart:
		return "PostStart"
	case StageClose:
		return "Close"
	default:
		return fmt.Sprintf("StartStage(%d)", s)
	}
}

type Starter interface {
	Start(ctx context.Context, stage Stage) error
}

func Close(service any) error {
	if c, ok := service.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func Start(ctx context.Context, stage Stage, service any) error {
	if s, ok := service.(Starter); ok {
		return s.Start(ctx, stage)
	}
	return nil
}

func StartService(ctx context.Context, ser any) error {
	for _, stage := range []Stage{StagePreStart, StageStart, StagePostStart} {
		if err := Start(ctx, stage, ser); err != nil {
			return newServiceLifeCycleError(err, stage)
		}
	}
	return nil
}

func CloseService(ser any) error {
	if err := Close(ser); err != nil {
		return &LifeCycleError{Err: err, Stage: StageClose}
	}
	return nil
}

type LifeCycleError struct {
	Err   error
	Stage Stage
}

func newServiceLifeCycleError(err error, stage Stage) error {
	return &LifeCycleError{
		Err:   err,
		Stage: stage,
	}
}

func (e *LifeCycleError) Error() string {
	return fmt.Sprintf("service stage %s: %s", e.Stage.String(), e.Err)
}

func (e *LifeCycleError) Unwrap() error {
	return e.Err
}
