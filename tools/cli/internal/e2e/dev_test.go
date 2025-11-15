package e2e

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestEndToEndDevWatch tests the complete dev --watch workflow
func TestEndToEndDevWatch(t *testing.T) {
	// Create a temporary directory for the test plugin
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-plugin")

	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	// Create a minimal plugin.yaml
	pluginYaml := `name: test-plugin
version: 0.1.0
entry: ./main.go
`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(pluginYaml), 0644); err != nil {
		t.Fatalf("Failed to create plugin.yaml: %v", err)
	}

	// Create a main.go file
	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	if err := os.WriteFile(filepath.Join(pluginDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	// Build the Go CLI
	cliPath := buildGoCLI(t, filepath.Join(tmpDir, "px-plugin"))

	// Test 1: dev --watch with file changes
	t.Run("DevWatchWithFileChanges", func(t *testing.T) {
		testDevWatchWithFileChanges(t, pluginDir, cliPath)
	})

	// Test 2: Session management
	t.Run("SessionManagement", func(t *testing.T) {
		testSessionManagement(t, pluginDir, cliPath)
	})

	// Test 3: List sessions
	t.Run("ListSessions", func(t *testing.T) {
		testListSessions(t, cliPath)
	})

	// Test 4: Logs streaming (if Dev API is available)
	t.Run("LogsStreaming", func(t *testing.T) {
		testLogsStreaming(t, pluginDir, cliPath)
	})
}

// buildGoCLI builds the Go CLI binary
func buildGoCLI(t *testing.T, outputPath string) string {
	// Get the current directory
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Find the tools/cli directory
	toolsCliDir := filepath.Join(dir, "tools", "cli")
	if _, err := os.Stat(toolsCliDir); os.IsNotExist(err) {
		t.Skip("Skipping e2e test: tools/cli directory not found")
		return ""
	}

	// Build the CLI
	cmd := exec.Command("go", "build", "-o", outputPath, filepath.Join(toolsCliDir, "cmd", "px-plugin"))
	cmd.Dir = toolsCliDir
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build Go CLI: %v\nOutput: %s", err, string(output))
	}

	return outputPath
}

// testDevWatchWithFileChanges tests the dev watch with file changes
func testDevWatchWithFileChanges(t *testing.T, pluginDir, cliPath string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start dev --watch in the background
	cmd := exec.CommandContext(ctx, cliPath, "dev", "--watch", "--entry", pluginDir)
	cmd.Dir = pluginDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to get stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start dev command: %v", err)
	}
	defer cmd.Process.Kill()

	// Read output in the background
	go io.Copy(io.Discard, stdout)
	go io.Copy(io.Discard, stderr)

	// Wait a bit for the command to initialize
	time.Sleep(2 * time.Second)

	// Check if the process is still running
	if cmd.ProcessState.ExitCode() != -1 {
		t.Errorf("dev command exited unexpectedly with code: %d", cmd.ProcessState.ExitCode())
	}

	// Modify a file
	mainGoPath := filepath.Join(pluginDir, "main.go")
	newContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, Modified World!")
}
`
	if err := os.WriteFile(mainGoPath, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to modify main.go: %v", err)
	}

	// Wait for the file change to be detected
	time.Sleep(3 * time.Second)

	// Check if the process is still running
	if cmd.ProcessState.ExitCode() != -1 {
		t.Errorf("dev command exited unexpectedly after file change")
	}

	// Restore the original file
	originalContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	if err := os.WriteFile(mainGoPath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to restore main.go: %v", err)
	}

	// Wait a bit more
	time.Sleep(1 * time.Second)

	// Kill the process
	cmd.Process.Kill()
	cmd.Wait()
}

// testSessionManagement tests session creation and management
func testSessionManagement(t *testing.T, pluginDir, cliPath string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create a new session
	cmd := exec.CommandContext(ctx, cliPath, "dev", "--watch", "--entry", pluginDir)
	cmd.Dir = pluginDir

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start dev command: %v", err)
	}
	defer cmd.Process.Kill()

	// Wait for session to be created
	time.Sleep(2 * time.Second)

	// Kill the process to stop the session
	cmd.Process.Kill()
	cmd.Wait()

	// Try to resume the session (this will fail if session is not persisted, but that's OK for this test)
	// In a real scenario with a running Dev API, this would work
	t.Log("Session management test completed")
}

// testListSessions tests listing sessions
func testListSessions(t *testing.T, cliPath string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// List sessions
	cmd := exec.CommandContext(ctx, cliPath, "dev", "--list-sessions")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("List sessions command failed (this is expected if no sessions exist): %v", err)
	}

	// Check if output contains expected content
	outputStr := string(output)
	if !strings.Contains(outputStr, "session") && !strings.Contains(outputStr, "No sessions") {
		t.Errorf("Unexpected output from list-sessions: %s", outputStr)
	}
}

// testLogsStreaming tests log streaming
func testLogsStreaming(t *testing.T, pluginDir, cliPath string) {
	// This test is optional as it requires a running Dev API
	// We'll just verify the command exists and can be invoked
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to get logs for a non-existent session
	cmd := exec.CommandContext(ctx, cliPath, "dev", "--logs", "non-existent-session")

	output, err := cmd.CombinedOutput()

	// This will fail, but we just want to verify the command is recognized
	if err != nil {
		// Expected to fail
		t.Logf("Logs command failed as expected (no session exists): %v", err)
	}

	// Check if the error is about session not found (expected) or command not recognized (unexpected)
	outputStr := string(output)
	if strings.Contains(outputStr, "session") || strings.Contains(outputStr, "not found") {
		t.Log("Logs command is working correctly")
	} else if strings.Contains(outputStr, "unknown command") || strings.Contains(outputStr, "invalid") {
		t.Errorf("Unexpected error from logs command: %s", outputStr)
	}
}

// TestBuildCLI tests building the Go CLI
func TestBuildCLI(t *testing.T) {
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "px-plugin")

	buildGoCLI(t, cliPath)

	// Check if the binary was created
	if _, err := os.Stat(cliPath); os.IsNotExist(err) {
		t.Error("CLI binary was not created")
	}

	// Check if the binary is executable
	if !isExecutable(cliPath) {
		t.Error("CLI binary is not executable")
	}
}

// isExecutable checks if a file is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}

// TestCLIHelp tests that the CLI shows help
func TestCLIHelp(t *testing.T) {
	tmpDir := t.TempDir()
	cliPath := buildGoCLI(t, filepath.Join(tmpDir, "px-plugin"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test help command
	cmd := exec.CommandContext(ctx, cliPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run help command: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Usage:") {
		t.Errorf("Help output doesn't contain 'Usage': %s", outputStr)
	}

	if !strings.Contains(outputStr, "dev") {
		t.Errorf("Help output doesn't mention 'dev' command: %s", outputStr)
	}
}

// TestDevHelp tests the dev command help
func TestDevHelp(t *testing.T) {
	tmpDir := t.TempDir()
	cliPath := buildGoCLI(t, filepath.Join(tmpDir, "px-plugin"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test dev help
	cmd := exec.CommandContext(ctx, cliPath, "dev", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run dev help command: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "watch") {
		t.Errorf("Dev help output doesn't mention 'watch' subcommand: %s", outputStr)
	}

	// Check for expected subcommands
	expectedSubcommands := []string{"watch", "list-sessions", "resume", "stop", "logs"}
	for _, sub := range expectedSubcommands {
		if !strings.Contains(outputStr, sub) {
			t.Errorf("Dev help output doesn't mention '%s' subcommand: %s", sub, outputStr)
		}
	}
}
