package validator

import "fmt"

type ValidError struct {
	Field string
	Value any
	err   error
}

func NewValidError(field string, value any, err error) *ValidError {
	return &ValidError{Field: field, Value: value, err: err}
}

func (v *ValidError) Error() string {
	if v == nil || v.err == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s: %s", v.Field, v.err.Error())
}

func (v *ValidError) Unwrap() error {
	if v == nil {
		return nil
	}
	return v.err
}
