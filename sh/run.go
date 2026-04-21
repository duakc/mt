package sh

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/duakc/mt/gosys"
)

func Run(command string) error {
	return New().Run(command)
}
func RunContext(ctx context.Context, command string) error {
	return New().RunContext(ctx, command)
}

func New() *Cmd {
	return NewShell(ShellUseDefault)
}

func NewShell(shell Shell) *Cmd {
	return create(shell)
}

func create(shell Shell) *Cmd {
	c := &Cmd{Shell: shell, Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}

	return c
}

type Cmd struct {
	Shell Shell

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	ExtendEnv []string

	WorkDir string
	Become  *BecomeOption
}

func (c *Cmd) Run(command string) error {
	return c.RunContext(context.Background(), command)
}

func (c *Cmd) RunContext(ctx context.Context, command string) error {
	var (
		runName string
		runArgs []string
		becomed bool
	)

	shell, shellArgs := ShellCommand(c.Shell)
	runName = shell
	runArgs = shellArgs
	if suCommand, suCommandArg := BecomeCommand(*c.Become); c.Become != nil && suCommand != "" {
		runArgs = append(suCommandArg, shell)
		runArgs = append(runArgs, runArgs...)
		runName = suCommand
		// Force c.Stdin = os.Stdin when it hasn't been
		// explicitly set by the caller (i.e. Cmd.Stdin == nil).
		// This ensures the elevated child process (sudo, doas, pkexec, etc.) inherits
		// the parent's standard input stream so that
		// interactive prompts and piped data
		// continue to work after privilege escalation.
		//
		if c.Stdin == nil &&
			// We also skip when using pkexec on a desktop session
			// (XDG_SESSION_TYPE != "")
			// In that case pkexec automatically shows a graphical authentication dialog;
			// forcing os.Stdin would interfere with the GUI password prompt.
			!(gosys.IsLinux && c.Become.Method == BecomeUsePkexec && os.Getenv("XDG_SESSION_TYPE") != "") {
			c.Stdin = os.Stdin
		}
		becomed = true
	}

	cc := exec.CommandContext(ctx, runName, append(runArgs, command)...)

	cc.Env = append(os.Environ(), c.ExtendEnv...)
	cc.Dir = c.WorkDir
	cc.Stdin = c.Stdin
	cc.Stdout = c.Stdout
	cc.Stderr = c.Stderr

	err := cc.Run()
	if err != nil {
		shellErr := &ShellError{ShellPath: shell, ShellArgs: shellArgs, Err: err}
		if becomed {
			shellErr.Become = c.Become
		}
		return shellErr
	}
	return nil
}

func (c *Cmd) Env(k, v string) *Cmd {
	c.ExtendEnv = append(c.ExtendEnv, k+"="+v)
	return c
}

func (c *Cmd) Envs(vv []string) *Cmd {
	c.ExtendEnv = append(c.ExtendEnv, vv...)
	return c
}

func (c *Cmd) Deattach() *Cmd {
	c.Stdin = nil
	c.Stdout = io.Discard
	c.Stderr = io.Discard
	return c
}

func (c *Cmd) CD(path string) *Cmd {
	wd, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}
	c.WorkDir = filepath.Clean(filepath.Join(wd, path))
	return c
}

func (c *Cmd) BecomeUser(user string) *Cmd {
	c.Become = &BecomeOption{BecomeUseDefault, user, ""}
	return c
}

func (c *Cmd) BecomeFull(method BecomeMethod, user, group string) *Cmd {
	c.Become = &BecomeOption{method, user, group}
	return c
}

type ShellError struct {
	ShellPath string
	ShellArgs []string

	Become *BecomeOption
	Err    error
}

func (e *ShellError) Error() string {
	formatString := "exec shell `%s %v` "
	formatArgs := []any{e.ShellPath, e.ShellArgs}

	if e.Become != nil {
		command, args := BecomeCommand(*e.Become)
		formatString = formatString + "with `%s %v` "
		formatArgs = append(formatArgs, command)
		formatArgs = append(formatArgs, stringArrayToAny(args)...)
	}
	return fmt.Sprintf(formatString, formatArgs...)
}

func (e *ShellError) Unwrap() error {
	return e.Err
}

func hasProgramInPath(p string) bool {
	_, err := exec.LookPath(p)
	return err == nil
}

func stringArrayToAny(v []string) []any {
	vv := make([]any, len(v))
	for i := 0; i < len(v); i++ {
		vv[i] = v[i]
	}
	return vv
}
