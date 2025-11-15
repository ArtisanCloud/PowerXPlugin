package build

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// SimpleBuilder is a simple build implementation
type SimpleBuilder struct {
	name string
}

// NewSimpleBuilder creates a new simple builder
func NewSimpleBuilder() *SimpleBuilder {
	return &SimpleBuilder{
		name: "simple",
	}
}

// Name returns the builder name
func (b *SimpleBuilder) Name() string {
	return b.name
}

// Description returns the builder description
func (b *SimpleBuilder) Description() string {
	return "Simple builder for Go/Node projects"
}

// Build performs a build
func (b *SimpleBuilder) Build(ctx context.Context, opts *BuildOptions) (*BuildResult, error) {
	if opts == nil {
		return nil, fmt.Errorf("build options are required")
	}

	if opts.EntryPath == "" {
		return nil, fmt.Errorf("entry path is required")
	}

	startTime := time.Now()
	result := NewBuildResult(startTime)

	// Check if entry path exists
	if _, err := os.Stat(opts.EntryPath); err != nil {
		result.Complete(time.Now(), false, err)
		return result, err
	}

	// Determine build strategy
	var err error
	switch opts.Strategy {
	case StrategyFull:
		err = b.buildFull(ctx, opts, result)
	case StrategyIncremental:
		err = b.buildIncremental(ctx, opts, result)
	case StrategyDiff:
		err = b.buildDiff(ctx, opts, result)
	default:
		err = b.buildIncremental(ctx, opts, result)
	}

	if err != nil {
		result.Complete(time.Now(), false, err)
		return result, err
	}

	result.Complete(time.Now(), true, nil)
	return result, nil
}

// buildFull performs a full build
func (b *SimpleBuilder) buildFull(ctx context.Context, opts *BuildOptions, result *BuildResult) error {
	fmt.Println("Performing full build...")

	// Detect project type and build accordingly
	projectType := detectProjectType(opts.EntryPath)

	var err error
	switch projectType {
	case "go":
		err = b.buildGo(ctx, opts)
	case "node":
		err = b.buildNode(ctx, opts)
	case "mixed":
		err = b.buildMixed(ctx, opts)
	default:
		err = fmt.Errorf("unknown project type")
	}

	if err != nil {
		return err
	}

	// Calculate bundle hash and size
	return b.calculateBundleInfo(opts, result)
}

// buildIncremental performs an incremental build
func (b *SimpleBuilder) buildIncremental(ctx context.Context, opts *BuildOptions, result *BuildResult) error {
	fmt.Println("Performing incremental build...")

	// For incremental build, we still run the full build
	// but mark it as incremental in the result
	result.Artifacts = append(result.Artifacts, "incremental:true")

	// Detect project type
	projectType := detectProjectType(opts.EntryPath)

	var err error
	switch projectType {
	case "go":
		err = b.buildGo(ctx, opts)
	case "node":
		err = b.buildNode(ctx, opts)
	case "mixed":
		err = b.buildMixed(ctx, opts)
	default:
		err = fmt.Errorf("unknown project type")
	}

	if err != nil {
		return err
	}

	// Calculate bundle hash and size
	return b.calculateBundleInfo(opts, result)
}

// buildDiff performs a diff build
func (b *SimpleBuilder) buildDiff(ctx context.Context, opts *BuildOptions, result *BuildResult) error {
	fmt.Println("Performing diff build...")

	if len(opts.ChangedFiles) == 0 {
		// No changes, skip build
		fmt.Println("No changes detected, skipping build")
		result.Complete(time.Now(), true, nil)
		return nil
	}

	// For diff build, we only rebuild affected files
	result.Artifacts = append(result.Artifacts, fmt.Sprintf("changed_files:%d", len(opts.ChangedFiles)))

	// Detect project type
	projectType := detectProjectType(opts.EntryPath)

	var err error
	switch projectType {
	case "go":
		err = b.buildGo(ctx, opts)
	case "node":
		err = b.buildNode(ctx, opts)
	case "mixed":
		err = b.buildMixed(ctx, opts)
	default:
		err = fmt.Errorf("unknown project type")
	}

	if err != nil {
		return err
	}

	// Calculate bundle hash and size
	return b.calculateBundleInfo(opts, result)
}

// buildGo builds a Go project
func (b *SimpleBuilder) buildGo(ctx context.Context, opts *BuildOptions) error {
	cmd := exec.CommandContext(ctx, "go", "build", "-o", filepath.Join(opts.OutDir, "plugin"), opts.EntryPath)
	cmd.Dir = opts.EntryPath

	if opts.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// buildNode builds a Node.js project
func (b *SimpleBuilder) buildNode(ctx context.Context, opts *BuildOptions) error {
	// For Node.js, we run npm run build if available
	cmd := exec.CommandContext(ctx, "npm", "run", "build")
	cmd.Dir = opts.EntryPath

	if opts.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// buildMixed builds a mixed Go/Node project
func (b *SimpleBuilder) buildMixed(ctx context.Context, opts *BuildOptions) error {
	// Build backend (Go)
	if err := b.buildGo(ctx, opts); err != nil {
		return err
	}

	// Build frontend (Node)
	return b.buildNode(ctx, opts)
}

// calculateBundleInfo calculates bundle hash and size
func (b *SimpleBuilder) calculateBundleInfo(opts *BuildOptions, result *BuildResult) error {
	var totalSize int64
	var allHashes []byte

	// Walk output directory and calculate hash and size
	err := filepath.Walk(opts.OutDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Add file size
		totalSize += info.Size()

		// Read file and add to hash
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		allHashes = append(allHashes, data...)
		result.Artifacts = append(result.Artifacts, path)

		return nil
	})

	if err != nil {
		return err
	}

	// Calculate bundle hash
	hash := sha256.Sum256(allHashes)
	result.BundleHash = fmt.Sprintf("%x", hash)
	result.BundleSize = totalSize

	return nil
}

// detectProjectType detects the project type
func detectProjectType(entryPath string) string {
	goModPath := filepath.Join(entryPath, "go.mod")
	packageJSONPath := filepath.Join(entryPath, "package.json")

	hasGoMod := false
	hasPackageJSON := false

	if _, err := os.Stat(goModPath); err == nil {
		hasGoMod = true
	}

	if _, err := os.Stat(packageJSONPath); err == nil {
		hasPackageJSON = true
	}

	if hasGoMod && hasPackageJSON {
		return "mixed"
	} else if hasGoMod {
		return "go"
	} else if hasPackageJSON {
		return "node"
	}

	return "unknown"
}
