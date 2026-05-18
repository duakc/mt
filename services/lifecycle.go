package services

import (
	"context"
	"fmt"
	"io"
)

type StartStage int

const (
	StagePreStart StartStage = iota
	StageStart
	StagePostStart
)

func (s StartStage) String() string {
	switch s {
	case StagePreStart:
		return "PreStart"
	case StageStart:
		return "Start"
	case StagePostStart:
		return "PostStart"
	default:
		return fmt.Sprintf("StartStage(%d)", s)
	}
}

type Starter interface {
	Start(ctx context.Context, stage StartStage) error
}

type Closer interface {
	Close() error
}

func Close(service any) error {
	if c, ok := service.(Closer); ok {
		return c.Close()
	}
	if c, ok := service.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func Start(ctx context.Context, stage StartStage, service any) error {
	if s, ok := service.(Starter); ok {
		return s.Start(ctx, stage)
	}
	return nil
}

type LifeCycle interface {
	Starter
	Closer
}

func StartService(ctx context.Context, ser any) error {
	for _, stage := range []StartStage{StagePreStart, StageStart, StagePostStart} {
		if err := Start(ctx, stage, ser); err != nil {
			return newServiceLifeCycleError(err, stage)
		}
	}
	return nil
}

func CloseService(ser any) error {
	if err := Close(ser); err != nil {
		return &LifeCycleError{Err: err, Stage: "Close"}
	}
	return nil
}

type LifeCycleError struct {
	Err   error
	Stage string
}

func newServiceLifeCycleError(err error, stage StartStage) error {
	return &LifeCycleError{
		Err:   err,
		Stage: stage.String(),
	}
}

func (e *LifeCycleError) Error() string {
	return fmt.Sprintf("service stage %s: %s", e.Stage, e.Err)
}

func (e *LifeCycleError) Unwrap() error {
	return e.Err
}
