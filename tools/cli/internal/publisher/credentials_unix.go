//go:build darwin || linux

package publisher

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// No file/environment fallback. The protected local binding API accepts a
// transient token and returns only verified identity metadata.
// Implementations pass secrets over stdin, never process arguments.
type credentialStore interface {
	Set(context.Context, string, string) error
	Get(context.Context, string) (string, error)
	Delete(context.Context, string) error
}
type systemCredentials struct{ keychain string }

const credentialService = "com.powerx.plugin.publisher.v1"

func (s systemCredentials) command(ctx context.Context, action, id, token string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if !canonicalUUID(id) {
		return "", errors.New(MarketKeyring)
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		// security's interactive parser reads the command from the pipe. UUIDs and
		// validated base64url tokens contain no parser metacharacters. A named
		// keychain is solely an internal test injection, never a product setting.
		if strings.ContainsAny(s.keychain, "\"\n\r\\") {
			return "", errors.New(MarketKeyring)
		}
		suffix := ""
		if s.keychain != "" {
			suffix = " \"" + s.keychain + "\""
		}
		args := []string{action + "-generic-password", "-s", credentialService, "-a", id}
		if action == "add" {
			cmd = exec.CommandContext(ctx, "/usr/bin/security", "-i")
			cmd.Stdin = strings.NewReader("add-generic-password -U -s " + credentialService + " -a " + id + " -w " + token + suffix + "\n")
		} else {
			if action == "find" {
				args = append(args, "-w")
			}
			if s.keychain != "" {
				args = append(args, s.keychain)
			}
			cmd = exec.CommandContext(ctx, "/usr/bin/security", args...)
		}
	} else if runtime.GOOS == "linux" {
		binary, err := exec.LookPath("secret-tool")
		if err != nil {
			return "", err
		}
		actionName := map[string]string{"add": "store", "find": "lookup", "delete": "clear"}[action]
		args := []string{actionName}
		if action == "add" {
			args = append(args, "--label=com.powerx.plugin.publisher.v1")
		}
		args = append(args, "service", credentialService, "account", id)
		cmd = exec.CommandContext(ctx, binary, args...)
		if action == "add" {
			cmd.Stdin = strings.NewReader(token)
		}
	} else {
		return "", errors.New(MarketKeyring)
	}
	var out limitedSecretBuffer
	cmd.Stdout = &out
	cmd.Stderr = &limitedSecretBuffer{}
	if err := cmd.Run(); err != nil {
		return "", errors.New(MarketKeyring)
	}
	return strings.TrimSpace(out.String()), nil
}

type limitedSecretBuffer struct{ bytes.Buffer }

func (b *limitedSecretBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if b.Len()+n > 4096 {
		return 0, errors.New(MarketKeyring)
	}
	_, err := b.Buffer.Write(p)
	return n, err
}
func (s systemCredentials) Set(ctx context.Context, id, token string) error {
	if _, err := s.command(ctx, "add", id, token); err != nil {
		return err
	}
	// security -i can exit zero after a failed command. Verify the saved value.
	value, err := s.Get(ctx, id)
	if err != nil || value != token {
		return errors.New(MarketKeyring)
	}
	return nil
}
func (s systemCredentials) Get(ctx context.Context, id string) (string, error) {
	value, err := s.command(ctx, "find", id, "")
	if err != nil || !publishTokenPattern.MatchString(value) {
		return "", errors.New(MarketKeyring)
	}
	return value, nil
}
func (s systemCredentials) Delete(ctx context.Context, id string) error {
	_, err := s.command(ctx, "delete", id, "")
	return err
}
