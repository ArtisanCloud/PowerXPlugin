//go:build darwin || linux

package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

func buildEnv() []string {
	allowed := map[string]bool{"PATH": true, "HOME": true, "TMPDIR": true, "LANG": true, "LC_ALL": true, "GOCACHE": true, "GOMODCACHE": true, "GOPATH": true}
	out := []string{}
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if allowed[key] {
			out = append(out, item)
		}
	}
	return append(out, "GOWORK=off", "GOENV=off", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local", "CGO_ENABLED=0", "NPM_CONFIG_UPDATE_NOTIFIER=false", "NPM_CONFIG_AUDIT=false", "NUXT_TELEMETRY_DISABLED=1", "CI=true")
}
func toolOutput(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = buildEnv()
	data, err := cmd.Output()
	if err != nil {
		return nil, failure(BuildFailed, err)
	}
	return data, nil
}
func probeTools() (BuildTools, error) {
	goVersion, err := toolOutput("go", "env", "GOVERSION")
	if err != nil {
		return BuildTools{}, err
	}
	data, err := toolOutput("node", "-p", `JSON.stringify({version:process.version,os:process.platform,arch:process.arch})`)
	if err != nil {
		return BuildTools{}, err
	}
	var node struct {
		Version string `json:"version"`
		OS      string `json:"os"`
		Arch    string `json:"arch"`
	}
	if json.Unmarshal(data, &node) != nil {
		return BuildTools{}, failure(BuildFailed, nil)
	}
	var major, minor, nodeMajor int
	if _, err = fmt.Sscanf(strings.TrimPrefix(strings.TrimSpace(string(goVersion)), "go"), "%d.%d", &major, &minor); err != nil || major != 1 || minor < 24 {
		return BuildTools{}, failure(BuildFailed, nil)
	}
	if _, err = fmt.Sscanf(strings.TrimPrefix(node.Version, "v"), "%d", &nodeMajor); err != nil || nodeMajor < 20 {
		return BuildTools{}, failure(BuildFailed, nil)
	}
	return BuildTools{Go: strings.TrimSpace(string(goVersion)), Node: node.Version, NodeOS: node.OS, NodeArch: node.Arch}, nil
}
func runBuildCommand(ctx context.Context, dir string, args []string, out io.Writer, started func(int) error) error {
	cmd := exec.CommandContext(ctx, "make", args...)
	cmd.Dir = dir
	cmd.Env = buildEnv()
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 2 * time.Second
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
	if err := cmd.Start(); err != nil {
		return failure(BuildFailed, err)
	}
	pid := cmd.Process.Pid
	if err := started(pid); err != nil {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		_ = cmd.Wait()
		return err
	}
	err := cmd.Wait()
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	if err != nil {
		return failure(BuildFailed, err)
	}
	return nil
}

type buildLog struct {
	mu         sync.Mutex
	file       *os.File
	pending    []byte
	written    int
	roots      []string
	privateKey bool
	truncated  bool
}

const MaxBuildLog = 4 << 20

func newBuildLog(file string, roots ...string) (*buildLog, error) {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, failure(StorageFailure, err)
	}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, home)
	}
	return &buildLog{file: f, roots: roots}, nil
}
func (l *buildLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(p)
	l.pending = append(l.pending, p...)
	for {
		i := strings.IndexByte(string(l.pending), '\n')
		if i < 0 {
			if len(l.pending) > 16<<10 {
				l.pending = []byte("[PUBLISHER_LOG_LINE_TRUNCATED]")
			}
			break
		}
		line := string(l.pending[:i])
		l.pending = l.pending[i+1:]
		if err := l.line(line); err != nil {
			return 0, err
		}
	}
	return n, nil
}
func (l *buildLog) line(s string) error {
	upper := strings.ToUpper(s)
	if strings.Contains(upper, "PRIVATE KEY") {
		l.privateKey = !strings.Contains(upper, "END ")
		s = "[PUBLISHER_LOG_REDACTED]"
	} else if l.privateKey {
		s = "[PUBLISHER_LOG_REDACTED]"
	}
	for _, marker := range []string{"TOKEN", "SECRET", "PASSWORD", "AUTHORIZATION", "COOKIE", "API_KEY", "APIKEY", "://"} {
		if strings.Contains(upper, marker) {
			s = "[PUBLISHER_LOG_REDACTED]"
			break
		}
	}
	for _, root := range l.roots {
		s = strings.ReplaceAll(s, root, "[workspace]")
	}
	if l.truncated {
		return nil
	}
	if l.written+len(s)+1 > MaxBuildLog-128 {
		l.truncated = true
		_, err := io.WriteString(l.file, "[PUBLISHER_LOG_BUDGET_REACHED]\n")
		return err
	}
	n, err := io.WriteString(l.file, s+"\n")
	l.written += n
	return err
}
func (l *buildLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.pending) > 0 {
		if err := l.line(string(l.pending)); err != nil {
			l.file.Close()
			return err
		}
	}
	err := l.file.Sync()
	closeErr := l.file.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func (e *Engine) BuildLog(id string) (string, error) {
	if _, err := e.GetBuild(id); err != nil {
		return "", err
	}
	d, err := e.work.existing(id, true)
	if err != nil {
		return "", err
	}
	defer d.close()
	data, err := d.read("build.log", MaxBuildLog)
	return string(data), err
}
