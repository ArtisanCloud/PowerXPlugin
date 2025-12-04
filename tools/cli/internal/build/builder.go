package build

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	case "plugin":
		err = b.buildPlugin(ctx, opts)
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
	case "plugin":
		err = b.buildPlugin(ctx, opts)
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
	goDir := detectGoModuleDir(opts.EntryPath)
	if goDir == "" {
		return fmt.Errorf("go project not found")
	}
	if err := os.MkdirAll(filepath.Join(opts.OutDir, "backend"), 0o755); err != nil {
		return err
	}
	targetPath := filepath.Join(opts.OutDir, "backend", "plugin")
	buildTarget := detectGoBuildTarget(goDir)
	args := []string{"build", "-o", targetPath}
	if buildTarget != "" {
		args = append(args, buildTarget)
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = goDir

	if opts.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// buildNode builds a Node.js project
func (b *SimpleBuilder) buildNode(ctx context.Context, opts *BuildOptions) error {
	nodeDir := detectNodeProjectDir(opts.EntryPath)
	if nodeDir == "" {
		return fmt.Errorf("node project not found")
	}
	cmd := exec.CommandContext(ctx, "npm", "run", "build")
	cmd.Dir = nodeDir

	if opts.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return err
	}
	return copyNodeArtifacts(nodeDir, filepath.Join(opts.OutDir, "web-admin"))
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

// buildPlugin builds a plugin layout (backend + web-admin subdirectories).
func (b *SimpleBuilder) buildPlugin(ctx context.Context, opts *BuildOptions) error {
	if err := b.buildGo(ctx, opts); err != nil {
		return err
	}
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
	goDir := detectGoModuleDir(entryPath)
	nodeDir := detectNodeProjectDir(entryPath)

	switch {
	case goDir != "" && nodeDir != "":
		if strings.HasPrefix(goDir, filepath.Join(entryPath, "backend")) || strings.HasPrefix(nodeDir, filepath.Join(entryPath, "web-admin")) {
			return "plugin"
		}
		return "mixed"
	case goDir != "":
		return "go"
	case nodeDir != "":
		return "node"
	default:
		return "unknown"
	}
}

func detectGoModuleDir(entryPath string) string {
	candidates := []string{
		entryPath,
		filepath.Join(entryPath, "backend"),
		filepath.Join(entryPath, "server"),
	}
	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
	}
	return ""
}

func detectNodeProjectDir(entryPath string) string {
	candidates := []string{
		entryPath,
		filepath.Join(entryPath, "web-admin"),
		filepath.Join(entryPath, "frontend"),
	}
	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
			return dir
		}
	}
	return ""
}

func detectGoBuildTarget(goDir string) string {
	cmdPlugin := filepath.Join(goDir, "cmd", "plugin")
	if info, err := os.Stat(cmdPlugin); err == nil && info.IsDir() {
		return "./cmd/plugin"
	}
	if _, err := os.Stat(filepath.Join(goDir, "main.go")); err == nil {
		return "."
	}
	return ""
}

func copyNodeArtifacts(srcDir, dstDir string) error {
	prefer := []string{".output", "dist", "build"}
	var src string
	for _, candidate := range prefer {
		path := filepath.Join(srcDir, candidate)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			src = path
			break
		}
	}
	if src == "" {
		return nil
	}
	return copyDir(src, dstDir)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		dstFile, err := os.Create(target)
		if err != nil {
			return err
		}
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			dstFile.Close()
			return err
		}
		return dstFile.Close()
	})
}
