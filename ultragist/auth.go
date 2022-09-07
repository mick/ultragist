package ultragist

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
)

var globalOptions = []string{
	"restrict",
	// "pty",
}

// MarshalAuthorizedKey serializes key for inclusion in an OpenSSH
// authorized_keys file. The return value ends with newline.
func MarshalAuthorizedKey(key ssh.PublicKey, options []string) []byte {
	b := &bytes.Buffer{}
	b.WriteString(strings.Join(options, ","))
	b.WriteByte(' ')
	b.WriteString(key.Type())
	b.WriteByte(' ')
	e := base64.NewEncoder(base64.StdEncoding, b)
	e.Write(key.Marshal())
	e.Close()
	b.WriteByte('\n')
	return b.Bytes()
}

func AuthorizedKeys(fingerprint string) error {
	key, err := GetKeyByFingerprint(fingerprint)
	if err != nil {
		return err
	}
	var options []string
	copy(globalOptions, options)

	options = append(options, fmt.Sprintf("environment=\"UGUSER=%s\"", key.UserId))

	fmt.Print(string(MarshalAuthorizedKey(key.pk, options)))
	return nil
}

var allowedCommanded = []string{
	"git-receive-pack",
	"git-upload-pack",
	"git-upload-archive",
}
var ExpectedRepoPath = regexp.MustCompile(`^[a-zA-Z0-9\-\_]+\/[a-zA-Z0-9\-\_]+\.git$`).MatchString

func GistShellParse(command string) (bool, string, string) {
	if command[3] == ' ' {
		command = command[:3] + "-" + command[4:]
	}

	allowed := false
	var args string
	var parsedCommand string
	for _, allowedCommand := range allowedCommanded {
		if strings.HasPrefix(command, allowedCommand) {
			allowed = true
			if (len(command) - len(allowedCommand)) > 2 {
				args = command[len(allowedCommand)+1:]
			}

			parsedCommand = allowedCommand
			break
		}
	}

	args = strings.Trim(args, "'")

	if args != "" && !ExpectedRepoPath(args) {
		return false, "", ""
	}

	return allowed, parsedCommand, args
}

func parseArgs(args string) (string, string) {
	parts := strings.Split(args, "/")
	repoParts := strings.Split(parts[1], ".")
	return parts[0], repoParts[0]
}

func GistShellAuthorization(userId string, args string) (bool, error) {
	if userId == "" {
		return false, fmt.Errorf("no user id")
	}
	username, gistId := parseArgs(args)

	//TODO load gist metadata and check permissions

	if username != userId {
		return false, fmt.Errorf("user id mismatch")
	}

	if gistId == "unauthorized" {
		return false, fmt.Errorf("not allowed to write to gist")
	}

	return true, nil
}

func GistShell(command string) error {
	allowed, parsedCommand, args := GistShellParse(command)
	if !allowed {
		return fmt.Errorf("command not allowed: %s", command)
	}

	userId, ok := os.LookupEnv("UGUSER")
	if !ok {
		return fmt.Errorf("no user id")
	}

	ok, err := GistShellAuthorization(userId, args)
	if !ok {
		return err
	}

	cmd := exec.Command(parsedCommand, args)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
