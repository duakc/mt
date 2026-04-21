package mt

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"
)

func Zero[T any]() T {
	var v T
	return v
}

func Comparable[T comparable](v T) T {
	return v
}

func PtrValueOrDefault[T any](v *T) T {
	if v == nil {
		return Zero[T]()
	}
	return *v
}

func All[T any, S ~[]T](arr S, fn func(T) bool) bool {
	for i := 0; i < len(arr); i++ {
		if !fn(arr[i]) {
			return false
		}
	}
	return true
}

func Or[T any, S ~[]T](arr S, fn func(T) bool) bool {
	for i := 0; i < len(arr); i++ {
		if fn(arr[i]) {
			return true
		}
	}
	return false
}

func Filter[T any, S ~[]T](arr S, fn func(T) bool) S {
	return slices.DeleteFunc(slices.Clone(arr), func(t T) bool {
		return !fn(t)
	})
}

func Set[T comparable, S ~[]T](arr S) map[T]bool {
	m := make(map[T]bool)
	for i := 0; i < len(arr); i++ {
		m[arr[i]] = true
	}
	return m
}

func Reduce[T any, S ~[]T](arr S, fn func(v1, v2 T) T) T {
	if len(arr) == 0 {
		return Zero[T]()
	}
	piovt := arr[0]
	for i := 1; i < len(arr); i++ {
		piovt = fn(piovt, arr[i])
	}
	return piovt
}

func Sum[T cmp.Ordered, S ~[]T](arr S) T {
	var su T
	for i := 0; i < len(arr); i++ {
		su += arr[i]
	}
	return su
}

func UnquoteString(s string) string {
	trim := strings.Trim(s, "'\"\n\r\t")
	return strings.TrimSpace(trim)
}

func Distinct[T comparable, S ~[]T](arr S) S {
	v := make(S, 0, len(arr))
	hash := make(map[T]bool)
	for i := 0; i < len(arr); i++ {
		if !hash[arr[i]] {
			v = append(v, arr[i])
		}
	}
	return v[:]
}

func Map[S any, D any](arr []S, fn func(S) D) []D {
	retArr := make([]D, 0, len(arr))
	for i := 0; i < len(arr); i++ {
		retArr = append(retArr, fn(arr[i]))
	}
	return retArr
}

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func Must0[T1 any, T2 any](v1 T1, v2 T2, err error) (T1, T2) {
	if err != nil {
		panic(err)
	}
	return v1, v2
}

func Done(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func emptyfn() {}

func Timeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if Done(ctx) {
		return ctx, emptyfn
	}
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		now := time.Now()
		// use short timeout first
		if deadline.Before(now) || deadline.Sub(now) <= timeout {
			return ctx, emptyfn
		}
	}
	return context.WithTimeout(ctx, timeout)
}
