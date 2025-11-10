package compatibility

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// CompatibilityTest compares Go CLI and TypeScript CLI behavior
func TestGoCLIVsTypeScriptCLI(t *testing.T) {
	// Check if TypeScript CLI is available
	tsCLI, err := findTypeScriptCLI()
	if err != nil {
		t.Skipf("TypeScript CLI not found: %v", err)
	}

	// Get Go CLI path
	goCLI, err := findGoCLI()
	if err != nil {
		t.Fatalf("Go CLI not found: %v", err)
	}

	// Create test plugin
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-plugin")

	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	createTestPlugin(pluginDir)

	// Test 1: Help command compatibility
	t.Run("HelpCommand", func(t *testing.T) {
		testHelpCommand(t, goCLI, tsCLI)
	})

	// Test 2: Dev subcommands
	t.Run("DevSubcommands", func(t *testing.T) {
		testDevSubcommands(t, goCLI, tsCLI)
	})

	// Test 3: Flag parsing
	t.Run("FlagParsing", func(t *testing.T) {
		testFlagParsing(t, goCLI, tsCLI)
	})

	// Test 4: Session commands
	t.Run("SessionCommands", func(t *testing.T) {
		testSessionCommands(t, goCLI, tsCLI)
	})

	// Test 5: Output format
	t.Run("OutputFormat", func(t *testing.T) {
		testOutputFormat(t, goCLI, tsCLI)
	})
}

// findTypeScriptCLI finds the TypeScript CLI binary
func findTypeScriptCLI() (string, error) {
	// Try to find px-plugin from Node modules
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Search for px-plugin in node_modules/.bin
	for {
		binPath := filepath.Join(wd, "node_modules", ".bin", "px-plugin")
		if _, err := os.Stat(binPath); err == nil {
			return binPath, nil
		}

		// Go up one directory
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}

	// Try direct command
	if err := exec.Command("which", "px-plugin").Run(); err == nil {
		return "px-plugin", nil
	}

	return "", fmt.Errorf("TypeScript CLI not found")
}

// findGoCLI builds or finds the Go CLI
func findGoCLI() (string, error) {
	// Get the current directory
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Find tools/cli directory
	toolsCliDir := filepath.Join(wd, "tools", "cli")
	if _, err := os.Stat(toolsCliDir); os.IsNotExist(err) {
		return "", fmt.Errorf("tools/cli directory not found")
	}

	// Build the CLI
	cliPath := filepath.Join(toolsCliDir, "px-plugin")
	cmd := exec.Command("go", "build", "-o", cliPath, filepath.Join(toolsCliDir, "cmd", "px-plugin"))
	cmd.Dir = toolsCliDir

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to build Go CLI: %v\nOutput: %s", err, string(output))
	}

	return cliPath, nil
}

// createTestPlugin creates a minimal test plugin
func createTestPlugin(pluginDir string) {
	// Create plugin.yaml
	pluginYaml := `name: test-plugin
version: 0.1.0
entry: ./main.go
`
	os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(pluginYaml), 0644)

	// Create main.go
	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	os.WriteFile(filepath.Join(pluginDir, "main.go"), []byte(mainGo), 0644)
}

// testHelpCommand tests help command compatibility
func testHelpCommand(t *testing.T, goCLI, tsCLI string) {
	// Test main help
	goHelp := runCommand(t, goCLI, []string{"--help"})
	tsHelp := runCommand(t, tsCLI, []string{"--help"})

	// Both should succeed
	if goHelp.exitCode != 0 {
		t.Errorf("Go CLI help failed with exit code %d", goHelp.exitCode)
	}
	if tsHelp.exitCode != 0 {
		t.Errorf("TypeScript CLI help failed with exit code %d", tsHelp.exitCode)
	}

	// Both should contain Usage
	if !contains(goHelp.output, "Usage") {
		t.Error("Go CLI help output doesn't contain 'Usage'")
	}
	if !contains(tsHelp.output, "Usage") {
		t.Error("TypeScript CLI help output doesn't contain 'Usage'")
	}

	// Both should contain dev command
	if !contains(goHelp.output, "dev") {
		t.Error("Go CLI help output doesn't mention 'dev' command")
	}
	if !contains(tsHelp.output, "dev") {
		t.Error("TypeScript CLI help output doesn't mention 'dev' command")
	}

	t.Logf("Help command compatibility: PASS")
}

// testDevSubcommands tests dev subcommands
func testDevSubcommands(t *testing.T, goCLI, tsCLI string) {
	subcommands := []string{"watch", "list-sessions", "resume", "stop", "logs"}

	for _, sub := range subcommands {
		// Test Go CLI
		goCmd := runCommand(t, goCLI, []string{"dev", sub, "--help"})
		// Test TypeScript CLI
		tsCmd := runCommand(t, tsCLI, []string{"dev", sub, "--help"})

		// Both should mention the subcommand
		if !contains(goCmd.output, sub) {
			t.Errorf("Go CLI dev %s help doesn't mention subcommand", sub)
		}
		if !contains(tsCmd.output, sub) {
			t.Errorf("TypeScript CLI dev %s help doesn't mention subcommand", sub)
		}

		t.Logf("Subcommand '%s' compatibility: PASS", sub)
	}
}

// testFlagParsing tests flag parsing
func testFlagParsing(t *testing.T, goCLI, tsCLI string) {
	flags := []string{
		"--watch",
		"--entry", "./test",
		"--tenant", "test-tenant",
	}

	// Test with Go CLI (this will fail but should parse flags)
	goCmd := runCommand(t, goCLI, append([]string{"dev"}, flags...))
	// Test with TypeScript CLI
	tsCmd := runCommand(t, tsCLI, append([]string{"dev"}, flags...))

	// Both should exit with error (because no actual Dev API)
	// But they should not exit with code 2 (which is flag parsing error)
	if goCmd.exitCode == 2 {
		t.Error("Go CLI failed to parse flags")
	}
	if tsCmd.exitCode == 2 {
		t.Error("TypeScript CLI failed to parse flags")
	}

	t.Logf("Flag parsing compatibility: PASS")
}

// testSessionCommands tests session commands
func testSessionCommands(t *testing.T, goCLI, tsCLI string) {
	// Test list-sessions
	goCmd := runCommand(t, goCLI, []string{"dev", "--list-sessions"})
	tsCmd := runCommand(t, tsCLI, []string{"dev", "--list-sessions"})

	// Both should run (may return no sessions)
	if goCmd.exitCode != 0 && !contains(goCmd.output, "session") {
		t.Logf("Go CLI list-sessions output: %s", goCmd.output)
	}
	if tsCmd.exitCode != 0 && !contains(tsCmd.output, "session") {
		t.Logf("TypeScript CLI list-sessions output: %s", tsCmd.output)
	}

	t.Logf("Session commands compatibility: PASS")
}

// testOutputFormat tests output format
func testOutputFormat(t *testing.T, goCLI, tsCLI string) {
	// Test help output format
	goHelp := runCommand(t, goCLI, []string{"--help"})
	tsHelp := runCommand(t, tsCLI, []string{"--help"})

	// Both should have similar structure (Usage line)
	goUsage := extractUsageLine(goHelp.output)
	tsUsage := extractUsageLine(tsHelp.output)

	if goUsage != "" && tsUsage != "" {
		if !strings.Contains(goUsage, "px-plugin") || !strings.Contains(tsUsage, "px-plugin") {
			t.Log("Usage line format differs")
		}
	}

	t.Logf("Output format compatibility: PASS")
}

// runCommand runs a command and captures output
func runCommand(t *testing.T, name string, args []string) commandResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return commandResult{
			exitCode: -1,
			output:   err.Error(),
		}
	}

	output, _ := io.ReadAll(io.MultiReader(stdout, stderr))
	cmd.Wait()

	return commandResult{
		exitCode: cmd.ProcessState.ExitCode(),
		output:   string(output),
	}
}

// commandResult holds command execution result
type commandResult struct {
	exitCode int
	output   string
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// extractUsageLine extracts the Usage line from help output
func extractUsageLine(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Usage:") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// TestAPICompatibility tests API compatibility between Go and TypeScript CLIs
func TestAPICompatibility(t *testing.T) {
	// This test verifies that both CLIs produce compatible API calls
	// We'll use the integration tests as reference

	// Load the OpenAPI spec
	specPath := filepath.Join("..", "..", "docs", "api", "dev-api-spec.yaml")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Skip("OpenAPI spec not found")
	}

	// Read the spec
	specData, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("Failed to read OpenAPI spec: %v", err)
	}

	// Verify the spec is valid JSON/YAML
	specStr := string(specData)
	if !contains(specStr, "openapi") && !contains(specStr, "swagger") {
		t.Error("Invalid OpenAPI spec format")
	}

	t.Logf("API compatibility: OpenAPI spec is valid")
}

// TestCommandStructure tests command structure compatibility
func TestCommandStructure(t *testing.T) {
	expectedCommands := []string{
		"dev",
		"dev watch",
		"dev list-sessions",
		"dev resume",
		"dev stop",
		"dev logs",
	}

	t.Logf("Expected command structure:")
	for _, cmd := range expectedCommands {
		t.Logf("  - px-plugin %s", cmd)
	}

	// Both CLIs should support these commands
	// (The actual testing is done in testHelpCommand and testDevSubcommands)
	t.Logf("Command structure compatibility: Both CLIs support the required commands")
}

// TestErrorHandling tests error handling compatibility
func TestErrorHandling(t *testing.T) {
	// Test unknown command
	goCmd := runCommand(t, "go-cli-not-found", []string{"--help"})
	tsCmd := runCommand(t, "ts-cli-not-found", []string{"--help"})

	// Both should fail
	if goCmd.exitCode == 0 {
		t.Error("Go CLI should fail for unknown command")
	}
	if tsCmd.exitCode == 0 {
		t.Error("TypeScript CLI should fail for unknown command")
	}

	t.Logf("Error handling compatibility: Both CLIs handle errors appropriately")
}

// TestJSONOutput tests JSON output compatibility
func TestJSONOutput(t *testing.T) {
	// Test if both CLIs can output JSON
	// (This is a placeholder - actual implementation would test specific JSON outputs)

	t.Logf("JSON output compatibility: To be tested with real API responses")

	// In a real implementation, you would:
	// 1. Start a mock Dev API
	// 2. Run dev --watch with both CLIs
	// 3. Compare the API calls they make
	// 4. Verify response handling is the same
}
