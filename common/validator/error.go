package validator

import "fmt"

type ValidError struct {
	err error

	msg  string
	args []any
}

func NewValidError(err error, msg string, args ...interface{}) *ValidError {
	return &ValidError{
		err:  err,
		msg:  msg,
		args: args,
	}
}

func (v *ValidError) Error() string {
	if v == nil || v.err == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s: %s", fmt.Sprintf(v.msg, v.args),
		v.err.Error())
}

func (v *ValidError) Unwrap() error {
	if v == nil {
		return nil
	}
	return v.err
}
