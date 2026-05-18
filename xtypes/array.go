package xtypes

import "strings"

type Joined[T any] struct {
	Array []T
	Join  T
}

func NewJoinedString(array []string, sep string) Joined[string] {
	return Joined[string]{
		Array: array,
		Join:  strings.Join(array, sep),
	}
}

func NewJoined[T any](array []T, joinFunc JoinFunc[T]) *Joined[T] {
	return &Joined[T]{Array: array, Join: joinFunc(array)}
}
