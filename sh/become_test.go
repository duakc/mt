package sh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBecomeCommand_None(t *testing.T) {
	cmd, args := BecomeCommand(BecomeOption{Method: BecomeNone})
	assert.Empty(t, cmd)
	assert.Nil(t, args)
}

// --sudo

func TestBecomeCommand_Sudo_NoUserNoGroup(t *testing.T) {
	cmd, args := BecomeCommand(BecomeOption{Method: BecomeUseSudo})
	assert.Equal(t, "sudo", cmd)
	assert.Equal(t, []string{"-E"}, args)
}

func TestBecomeCommand_Sudo_UserOnly(t *testing.T) {
	cmd, args := BecomeCommand(BecomeOption{Method: BecomeUseSudo, User: "alice"})
	assert.Equal(t, "sudo", cmd)
	assert.Equal(t, []string{"-E", "-u", "alice"}, args)
}

func TestBecomeCommand_Sudo_GroupOnly(t *testing.T) {
	cmd, args := BecomeCommand(BecomeOption{Method: BecomeUseSudo, Group: "wheel"})
	assert.Equal(t, "sudo", cmd)
	assert.Equal(t, []string{"-E", "-g", "wheel"}, args)
}

func TestBecomeCommand_Sudo_UserAndGroup(t *testing.T) {
	cmd, args := BecomeCommand(BecomeOption{
		Method: BecomeUseSudo,
		User:   "alice",
		Group:  "admins",
	})
	assert.Equal(t, "sudo", cmd)
	assert.Equal(t, []string{"-E", "-g", "admins", "-u", "alice"}, args)
}

// --su

func TestBecomeCommand_Su_DefaultsToRoot(t *testing.T) {
	cmd, args := BecomeCommand(BecomeOption{Method: BecomeUseSu})
	assert.Equal(t, "su", cmd)
	assert.Contains(t, args, "root")
	assert.Contains(t, args, "-c")
}

func TestBecomeCommand_Su_WithUser(t *testing.T) {
	cmd, args := BecomeCommand(BecomeOption{Method: BecomeUseSu, User: "bob"})
	assert.Equal(t, "su", cmd)
	assert.Contains(t, args, "bob")
	assert.Contains(t, args, "-c")
	assert.NotContains(t, args, "root")
}

func TestBecomeCommand_Su_WithGroup(t *testing.T) {
	cmd, args := BecomeCommand(BecomeOption{
		Method: BecomeUseSu,
		User:   "alice",
		Group:  "wheel",
	})
	assert.Equal(t, "su", cmd)
	assert.Contains(t, args, "--group")
	assert.Contains(t, args, "wheel")
	assert.Contains(t, args, "alice")
}

func TestBecomeCommand_Su_PreservesEnvironmentFlag(t *testing.T) {
	_, args := BecomeCommand(BecomeOption{Method: BecomeUseSu, User: "alice"})
	assert.Contains(t, args, "--preserve-environment",
		"su should always pass --preserve-environment")
}

// --doas

func TestBecomeCommand_Doas_NoUser(t *testing.T) {
	cmd, args := BecomeCommand(BecomeOption{Method: BecomeUseDoas})
	assert.Equal(t, "doas", cmd)
	assert.Empty(t, args, "doas can not be pass args without user")
}

func TestBecomeCommand_Doas_WithUser(t *testing.T) {
	cmd, args := BecomeCommand(BecomeOption{Method: BecomeUseDoas, User: "carol"})
	assert.Equal(t, "doas", cmd)
	assert.Equal(t, []string{"-u", "carol"}, args)
}

func TestBecomeCommand_Doas_GroupIsIgnored(t *testing.T) {
	_, args := BecomeCommand(BecomeOption{
		Method: BecomeUseDoas,
		User:   "carol",
		Group:  "wheel",
	})
	assert.NotContains(t, args, "wheel",
		"doas doesn't support set group")
}

// --pkexec

func TestBecomeCommand_Pkexec_NoUser(t *testing.T) {
	cmd, args := BecomeCommand(BecomeOption{Method: BecomeUsePkexec})
	assert.Equal(t, "pkexec", cmd)
	assert.Equal(t, []string{"--keep-cwd"}, args)
}

func TestBecomeCommand_Pkexec_WithUser(t *testing.T) {
	cmd, args := BecomeCommand(BecomeOption{Method: BecomeUsePkexec, User: "dave"})
	assert.Equal(t, "pkexec", cmd)
	assert.Equal(t, []string{"--keep-cwd", "--user", "dave"}, args)
}

func TestBecomeCommand_Pkexec_AlwaysKeepsCwd(t *testing.T) {
	_, args := BecomeCommand(BecomeOption{Method: BecomeUsePkexec})
	assert.Equal(t, "--keep-cwd", args[0],
		"pkexec should start with --keep-cwd")
}

func TestDefaultBecomeMethod_IsValid(t *testing.T) {
	m := DefaultBecomeMethod()
	valid := []BecomeMethod{
		BecomeNone,
		BecomeUseSudo,
		BecomeUseSu,
		BecomeUseDoas,
		BecomeUsePkexec,
	}
	assert.Contains(t, valid, m,
		"DefaultBecomeMethod should return known BecomeMethod")
}

func TestDefaultBecomeMethod_IsIdempotent(t *testing.T) {
	assert.Equal(t, DefaultBecomeMethod(), DefaultBecomeMethod())
}

func TestBecomeCommand_DefaultMethodResolvesToConcreteMethod(t *testing.T) {
	// BecomeUseDefault will call DefaultBecomeMethod() and resolve a actually method.
	// so, the DefaultBecomeMethod can not return BecomeUseDefault.

	cmd, _ := BecomeCommand(BecomeOption{Method: BecomeUseDefault, User: "root"})
	if DefaultBecomeMethod() != BecomeNone {
		// if DefaultBecomeMethod returned a Method that not is BecomeNone
		// a non-empty cmd should returned by BecomeCommand().
		assert.NotEmpty(t, cmd)
	}
}
