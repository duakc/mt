package validator

import (
	"cmp"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/duakc/mt"
)

func NonEmpty[T comparable](t T, name string) error {
	if t == mt.Zero[T]() {
		return NewValidError(name, t, errors.New("empty"))
	}
	return nil
}

func NonEmptySlice[S ~[]E, E any](s S, name string) error {
	if len(s) == 0 {
		return NewValidError(name, s, errors.New("empty"))
	}
	return nil
}

func NonEmptyMap[M ~map[K]V, K comparable, V any](m M, name string) error {
	if len(m) == 0 {
		return NewValidError(name, m, errors.New("empty"))
	}
	return nil
}

func NotNil[T any](t *T, name string) error {
	if t == nil {
		return NewValidError(name, t, errors.New("nil"))
	}
	return nil
}

func GreaterThan[T cmp.Ordered](t, bound T, name string) error {
	if t <= bound {
		return NewValidError(name, t, fmt.Errorf("%v less than or equal to %v", t, bound))
	}
	return nil
}

func GreaterOrEqual[T cmp.Ordered](t, bound T, name string) error {
	if t < bound {
		return NewValidError(name, t, fmt.Errorf("%v less than %v", t, bound))
	}
	return nil
}

func LessThan[T cmp.Ordered](t, bound T, name string) error {
	if t >= bound {
		return NewValidError(name, t, fmt.Errorf("%v greater than or equal to %v", t, bound))
	}
	return nil
}

func LessOrEqual[T cmp.Ordered](t, bound T, name string) error {
	if t > bound {
		return NewValidError(name, t, fmt.Errorf("%v greater than %v", t, bound))
	}
	return nil
}

func Between[T cmp.Ordered](t, min, max T, name string) error {
	if t < min || t > max {
		return NewValidError(name, t, fmt.Errorf("%v not in range [%v, %v]", t, min, max))
	}
	return nil
}

func EqualWith[T comparable](t, want T, name string) error {
	if t != want {
		return NewValidError(name, t, fmt.Errorf("%v not equal to %v", t, want))
	}
	return nil
}

func NotEqualWith[T comparable](t, banned T, name string) error {
	if t == banned {
		return NewValidError(name, t, fmt.Errorf("%v equal to %v", t, banned))
	}
	return nil
}

func Contains[T comparable, S ~[]T](t T, s S, name string) error {
	for i := 0; i < len(s); i++ {
		if s[i] == t {
			return nil
		}
	}
	return NewValidError(name, t, fmt.Errorf("%v not in %v", t, s))
}

func NotContains[T comparable, S ~[]T](t T, s S, name string) error {
	for i := 0; i < len(s); i++ {
		if s[i] == t {
			return NewValidError(name, t, fmt.Errorf("%v in %v", t, s))
		}
	}
	return nil
}

func MinRune[T ~string](s T, n int, name string) error {
	if l := utf8.RuneCountInString(string(s)); l < n {
		return NewValidError(name, s, fmt.Errorf("rune count %d less than %d", l, n))
	}
	return nil
}

func MaxRune[T ~string](s T, n int, name string) error {
	if l := utf8.RuneCountInString(string(s)); l > n {
		return NewValidError(name, s, fmt.Errorf("rune count %d greater than %d", l, n))
	}
	return nil
}

func StringStartsWith[T ~string](s T, prefix, name string) error {
	if !strings.HasPrefix(string(s), prefix) {
		return NewValidError(name, s, fmt.Errorf("does not start with %q", prefix))
	}
	return nil
}

func StringEndsWith[T ~string](s T, suffix, name string) error {
	if !strings.HasSuffix(string(s), suffix) {
		return NewValidError(name, s, fmt.Errorf("does not end with %q", suffix))
	}
	return nil
}

func StringContains[T ~string](s T, substr, name string) error {
	if !strings.Contains(string(s), substr) {
		return NewValidError(name, s, fmt.Errorf("does not contain %q", substr))
	}
	return nil
}

// Implements checks that v can be asserted to T. T is typically an interface
// type — when T is a concrete type the check degenerates to dynamic-type equality.
func Implements[T any](v any, name string) error {
	if _, ok := v.(T); ok {
		return nil
	}
	return NewValidError(name, v, fmt.Errorf("%T does not implement %v", v, reflect.TypeFor[T]()))
}
