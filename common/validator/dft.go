package validator

type GenericValidator[T any] interface {
	Validf(t T, format string, args ...any) bool
	Valid(T, string) bool

	Err() error
}
