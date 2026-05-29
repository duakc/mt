package validator

import (
	"cmp"
	"errors"
	"fmt"

	"github.com/duakc/mt"
)

type GenericValidFunc[T any] func(T) error

var _ GenericValidator[int] = (*DefaultGenericValidator[int])(nil)

type DefaultGenericValidator[T any] struct {
	fn  GenericValidFunc[T]
	err error
}

func (v *DefaultGenericValidator[T]) Err() error {
	return v.err
}

func (v *DefaultGenericValidator[T]) Validf(t T, format string, args ...any) bool {
	if err := v.fn(t); err != nil {
		v.err = errors.Join(v.err,
			fmt.Errorf("%s: %s", fmt.Sprintf(format, args), err.Error()))
		return false
	}

	return true
}

func (v *DefaultGenericValidator[T]) Valid(t T, msg string) bool {
	if err := v.fn(t); err != nil {
		v.err = errors.Join(v.err,
			fmt.Errorf("%s: %s", msg, err.Error()))
	}

	return v.err == nil
}

func NewGenericValidator[T any](fn GenericValidFunc[T]) *DefaultGenericValidator[T] {
	return &DefaultGenericValidator[T]{fn: fn}
}

func DisallowEmpty[T comparable]() GenericValidFunc[T] {
	return func(t T) error {
		if t == mt.Zero[T]() {
			return errors.New("empty")
		}
		return nil
	}
}

func GreaterThan[T cmp.Ordered](v T) GenericValidFunc[T] {
	return func(t T) error {
		if t <= v {
			return fmt.Errorf("%d less than: %d", t, v)
		}
		return nil
	}
}

func EqualWith[T comparable](v T) GenericValidFunc[T] {
	return func(t T) error {
		if t != v {
			return fmt.Errorf("%s not equal: %q", t, v)
		}
		return nil
	}
}

func Contains[T comparable, S ~[]T](s S) GenericValidFunc[T] {
	return func(t T) error {
		for i := 0; i < len(s); i++ {
			if s[i] == t {
				return nil
			}
		}
		return fmt.Errorf("%q not contains in: %+q", t, s)
	}
}

func LessThan[T cmp.Ordered](v T) GenericValidFunc[T] {
	return func(t T) error {
		if t >= v {
			return fmt.Errorf("%d greater than: %d", t, v)
		}
		return nil
	}
}
