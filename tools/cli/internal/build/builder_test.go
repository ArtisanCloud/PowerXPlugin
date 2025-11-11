package build

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewSimpleBuilder tests creating a new simple builder
func TestNewSimpleBuilder(t *testing.T) {
	builder := NewSimpleBuilder()

	if builder == nil {
		t.Error("NewSimpleBuilder returned nil")
	}

	if builder.Name() != "simple" {
		t.Errorf("Expected name 'simple', got %q", builder.Name())
	}

	expectedDesc := "Simple builder for Go/Node projects"
	if builder.Description() != expectedDesc {
		t.Errorf("Expected description %q, got %q", expectedDesc, builder.Description())
	}
}

// TestNewBuildResult tests creating a new build result
func TestNewBuildResult(t *testing.T) {
	startTime := time.Now()
	result := NewBuildResult(startTime)

	if result == nil {
		t.Error("NewBuildResult returned nil")
	}

	if !result.StartTime.Equal(startTime) {
		t.Errorf("Expected StartTime %v, got %v", startTime, result.StartTime)
	}

	if result.Success != true {
		t.Errorf("Expected Success true, got %v", result.Success)
	}

	if len(result.Artifacts) != 0 {
		t.Errorf("Expected empty Artifacts, got %v", result.Artifacts)
	}
}

// TestBuildResult_Complete tests marking a build result as complete
func TestBuildResult_Complete(t *testing.T) {
	startTime := time.Now()
	result := NewBuildResult(startTime)

	// Test successful completion
	endTime := startTime.Add(5 * time.Second)
	var err error
	result.Complete(endTime, true, err)

	if !result.EndTime.Equal(endTime) {
		t.Errorf("Expected EndTime %v, got %v", endTime, result.EndTime)
	}

	if result.BuildDuration != 5*time.Second {
		t.Errorf("Expected BuildDuration 5s, got %v", result.BuildDuration)
	}

	if result.Success != true {
		t.Errorf("Expected Success true, got %v", result.Success)
	}

	if result.Error != "" {
		t.Errorf("Expected empty Error, got %q", result.Error)
	}

	// Test failed completion
	result2 := NewBuildResult(startTime)
	endTime2 := startTime.Add(3 * time.Second)
	testErr := os.ErrNotExist
	result2.Complete(endTime2, false, testErr)

	if !result2.EndTime.Equal(endTime2) {
		t.Errorf("Expected EndTime %v, got %v", endTime2, result2.EndTime)
	}

	if result2.BuildDuration != 3*time.Second {
		t.Errorf("Expected BuildDuration 3s, got %v", result2.BuildDuration)
	}

	if result2.Success != false {
		t.Errorf("Expected Success false, got %v", result2.Success)
	}

	expectedError := testErr.Error()
	if result2.Error != expectedError {
		t.Errorf("Expected Error %q, got %q", expectedError, result2.Error)
	}
}

// TestBuild_Validation tests build parameter validation
func TestBuild_Validation(t *testing.T) {
	builder := NewSimpleBuilder()
	ctx := context.Background()

	// Test nil options
	_, err := builder.Build(ctx, nil)
	if err == nil {
		t.Error("Build should fail with nil options")
	}

	// Test empty entry path
	opts := &BuildOptions{
		Strategy:  StrategyFull,
		OutDir:    "/tmp/out",
		EntryPath: "",
	}
	_, err = builder.Build(ctx, opts)
	if err == nil {
		t.Error("Build should fail with empty entry path")
	}
}

// TestDetectProjectType tests project type detection
func TestDetectProjectType(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "project-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test Go project
	goProject := filepath.Join(tmpDir, "go-project")
	os.MkdirAll(goProject, 0755)
	goModFile := filepath.Join(goProject, "go.mod")
	os.WriteFile(goModFile, []byte("module test\n"), 0644)

	projectType := detectProjectType(goProject)
	if projectType != "go" {
		t.Errorf("Expected project type 'go', got %q", projectType)
	}

	// Test Node project
	nodeProject := filepath.Join(tmpDir, "node-project")
	os.MkdirAll(nodeProject, 0755)
	packageJSONFile := filepath.Join(nodeProject, "package.json")
	os.WriteFile(packageJSONFile, []byte("{}"), 0644)

	projectType = detectProjectType(nodeProject)
	if projectType != "node" {
		t.Errorf("Expected project type 'node', got %q", projectType)
	}

	// Test mixed project
	mixedProject := filepath.Join(tmpDir, "mixed-project")
	os.MkdirAll(mixedProject, 0755)
	os.WriteFile(filepath.Join(mixedProject, "go.mod"), []byte("module mixed\n"), 0o644)
	os.WriteFile(filepath.Join(mixedProject, "package.json"), []byte("{}"), 0o644)

	projectType = detectProjectType(mixedProject)
	if projectType != "mixed" {
		t.Errorf("Expected project type 'mixed', got %q", projectType)
	}

	// Test unknown project
	emptyProject := filepath.Join(tmpDir, "empty-project")
	os.MkdirAll(emptyProject, 0755)

	projectType = detectProjectType(emptyProject)
	if projectType != "unknown" {
		t.Errorf("Expected project type 'unknown', got %q", projectType)
	}
}

// TestBuildOptions tests BuildOptions structure
func TestBuildOptions(t *testing.T) {
	opts := &BuildOptions{
		EntryPath: "/path/to/entry",
		Strategy:  StrategyIncremental,
		OutDir:    "/path/to/output",
		Verbose:   true,
		ChangedFiles: []FileEvent{
			{Path: "file1.go", Type: "modify"},
			{Path: "file2.go", Type: "create"},
		},
	}

	if opts.EntryPath != "/path/to/entry" {
		t.Errorf("Expected EntryPath '/path/to/entry', got %q", opts.EntryPath)
	}

	if opts.Strategy != StrategyIncremental {
		t.Errorf("Expected Strategy StrategyIncremental, got %v", opts.Strategy)
	}

	if opts.OutDir != "/path/to/output" {
		t.Errorf("Expected OutDir '/path/to/output', got %q", opts.OutDir)
	}

	if opts.Verbose != true {
		t.Errorf("Expected Verbose true, got %v", opts.Verbose)
	}

	if len(opts.ChangedFiles) != 2 {
		t.Errorf("Expected 2 ChangedFiles, got %d", len(opts.ChangedFiles))
	}
}

// TestBuildStrategy tests BuildStrategy constants
func TestBuildStrategy(t *testing.T) {
	if StrategyFull != 0 {
		t.Errorf("Expected StrategyFull = 0, got %d", StrategyFull)
	}

	if StrategyIncremental != 1 {
		t.Errorf("Expected StrategyIncremental = 1, got %d", StrategyIncremental)
	}

	if StrategyDiff != 2 {
		t.Errorf("Expected StrategyDiff = 2, got %d", StrategyDiff)
	}
}

// TestFileEvent tests FileEvent structure
func TestFileEvent(t *testing.T) {
	now := time.Now()
	event := FileEvent{
		Type:      "modify",
		Path:      "/path/to/file.go",
		Hash:      "abc123",
		Timestamp: now,
	}

	if event.Type != "modify" {
		t.Errorf("Expected Type 'modify', got %q", event.Type)
	}

	if event.Path != "/path/to/file.go" {
		t.Errorf("Expected Path '/path/to/file.go', got %q", event.Path)
	}

	if event.Hash != "abc123" {
		t.Errorf("Expected Hash 'abc123', got %q", event.Hash)
	}

	if !event.Timestamp.Equal(now) {
		t.Errorf("Expected Timestamp %v, got %v", now, event.Timestamp)
	}
}
