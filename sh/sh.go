package sh

import (
	"errors"
	"fmt"
	"os/exec"
)

type ShellError struct {
	ShellPath string
	ShellArgs []string

	Become BecomeOption
	Err    error
}

func (e *ShellError) Error() string {
	formatString := "exec shell `%s %v` "
	formatArgs := []any{e.ShellPath, e.ShellArgs}

	if e.Become.Method != BecomeNone {
		command, args := BecomeCommand(e.Become)
		formatString = formatString + "with `%s %v` "
		formatArgs = append(formatArgs, command)
		formatArgs = append(formatArgs, stringArrayToAny(args)...)
	}
	return fmt.Sprintf(formatString, formatArgs...)
}

func (e *ShellError) Unwrap() error {
	return e.Err
}

func ExitCode(err error) int {
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return exitErr.ExitCode()
	}
	return -1
}
