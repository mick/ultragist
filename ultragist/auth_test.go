package ultragist

import (
	"testing"
)

type testallowedCommand struct {
	allowed bool
	command string
	args    string
}

var testallowedCommands = map[string]testallowedCommand{
	"git-receive-pack": {
		allowed: true,
		command: "git-receive-pack",
		args:    "",
	},
	"git-receive-pack someone/something.git": {
		allowed: true,
		command: "git-receive-pack",
		args:    "someone/something.git",
	},
	"git-upload-pack": {
		allowed: true,
		command: "git-upload-pack",
		args:    "",
	},
	"git-upload-archive": {
		allowed: true,
		command: "git-upload-archive",
		args:    "",
	},
	"git receive-pack": {
		allowed: true,
		command: "git-receive-pack",
		args:    "",
	},
	"git upload-pack": {
		allowed: true,
		command: "git-upload-pack",
		args:    "",
	},
	"git upload-archive": {
		allowed: true,
		command: "git-upload-archive",
		args:    "",
	},
	"git something": {
		allowed: false,
		command: "",
		args:    "",
	},
	"something": {
		allowed: false,
		command: "",
		args:    "",
	},
	"git-receive-pack 'someone/something.git'": {
		allowed: true,
		command: "git-receive-pack",
		args:    "someone/something.git",
	},
	"git-receive-pack 'someone/something.git' something else": {
		allowed: false,
		command: "",
		args:    "",
	},
	"git-receive-pack '/someone/something.git'": {
		allowed: false,
		command: "",
		args:    "",
	},
	"git-receive-pack '../someone.git'": {
		allowed: false,
		command: "",
		args:    "",
	},
}

func TestGistShellParse(t *testing.T) {

	for command, allowedExpected := range testallowedCommands {
		allowed, parsedCommand, args := GistShellParse(command)
		if allowed != allowedExpected.allowed {
			t.Errorf("GistShellParse(%s) allowed = %t, want %t", command, allowed, allowedExpected.allowed)
		}
		if parsedCommand != allowedExpected.command {
			t.Errorf("GistShellParse(%s) command = %s, want %s", command, parsedCommand, allowedExpected.command)
		}
		if args != allowedExpected.args {
			t.Errorf("GistShellParse(%s) args = %s, want %s", command, args, allowedExpected.args)
		}
	}
}
