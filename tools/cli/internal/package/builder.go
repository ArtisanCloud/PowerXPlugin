package pkg

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/powerx-plugin/cli/internal/manifest"
)

// Options controls how the packaging pipeline runs.
type Options struct {
	EntryPath       string
	FrontendDir     string
	BackendDir      string
	OutputDir       string
	ManifestPath    string
	RBACPath        string
	SkipFrontend    bool
	SkipBackend     bool
	Channel         string
	VersionOverride string
	CLIVersion      string
}

// Result captures the key artefacts emitted by the builder.
type Result struct {
	BuildID           string
	BuildDir          string
	PayloadDir        string
	PackagePath       string
	MetadataPath      string
	ManifestPath      string
	RBACPath          string
	BackendBinaryPath string
	FrontendPath      string
	Artifacts         []Artifact
	DistHash          string
}

// FrontendBuildFunc executes the frontend build (npm).
type FrontendBuildFunc func(ctx context.Context, opts *Options) error

// BackendBuildFunc executes the backend build (go build), writing to outputPath.
type BackendBuildFunc func(ctx context.Context, opts *Options, outputPath string) error

// Builder orchestrates the packaging workflow.
type Builder struct {
	clock           func() time.Time
	frontendBuilder FrontendBuildFunc
	backendBuilder  BackendBuildFunc
}

// BuilderOption customises the Builder.
type BuilderOption func(*Builder)

// NewBuilder constructs a Builder with sensible defaults.
func NewBuilder(opts ...BuilderOption) *Builder {
	b := &Builder{
		clock:           time.Now,
		frontendBuilder: defaultFrontendBuild,
		backendBuilder:  defaultBackendBuild,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// WithClock overrides the clock (primarily for tests).
func WithClock(clock func() time.Time) BuilderOption {
	return func(b *Builder) {
		if clock != nil {
			b.clock = clock
		}
	}
}

// WithFrontendBuild allows overriding how the frontend build runs.
func WithFrontendBuild(fn FrontendBuildFunc) BuilderOption {
	return func(b *Builder) {
		if fn != nil {
			b.frontendBuilder = fn
		}
	}
}

// WithBackendBuild allows overriding how the backend build runs.
func WithBackendBuild(fn BackendBuildFunc) BuilderOption {
	return func(b *Builder) {
		if fn != nil {
			b.backendBuilder = fn
		}
	}
}

// Build executes the package workflow and returns a Result describing produced artefacts.
func (b *Builder) Build(ctx context.Context, userOpts *Options) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if userOpts == nil {
		return nil, fmt.Errorf("package options are required")
	}
	if userOpts.EntryPath == "" {
		return nil, fmt.Errorf("entry path is required")
	}

	optsCopy := *userOpts
	opts := &optsCopy

	var err error
	opts.EntryPath, err = filepath.Abs(opts.EntryPath)
	if err != nil {
		return nil, fmt.Errorf("resolve entry path: %w", err)
	}

	if opts.OutputDir == "" {
		opts.OutputDir = filepath.Join(opts.EntryPath, ".px-plugin", "build")
	}
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create build root: %w", err)
	}

	if opts.FrontendDir == "" {
		opts.FrontendDir = filepath.Join(opts.EntryPath, "web-admin")
	}
	if opts.BackendDir == "" {
		opts.BackendDir = filepath.Join(opts.EntryPath, "backend")
	}
	if opts.ManifestPath == "" {
		opts.ManifestPath = filepath.Join(opts.EntryPath, "docs", "contracts", "manifest.json")
	}
	if opts.RBACPath == "" {
		opts.RBACPath = filepath.Join(opts.EntryPath, "docs", "contracts", "rbac.json")
	}
	if opts.Channel == "" {
		opts.Channel = "dev"
	}
	if opts.CLIVersion == "" {
		opts.CLIVersion = "dev"
	}

	manifestFile, err := manifest.Load(opts.EntryPath)
	if err != nil {
		return nil, fmt.Errorf("load plugin manifest: %w", err)
	}
	version := manifestFile.Version
	if opts.VersionOverride != "" {
		version = opts.VersionOverride
	}

	if err := ensureFileExists(opts.ManifestPath, "manifest.json", "run 'npm run sync:manifest' first"); err != nil {
		return nil, err
	}
	if err := ensureFileExists(opts.RBACPath, "rbac.json", "run 'npm run sync:manifest' first"); err != nil {
		return nil, err
	}

	now := b.clock().UTC()
	buildID := now.Format("20060102T150405Z")
	buildDir := filepath.Join(opts.OutputDir, buildID)
	payloadDir := filepath.Join(buildDir, "payload")
	if err := os.MkdirAll(payloadDir, 0o755); err != nil {
		return nil, fmt.Errorf("create payload dir: %w", err)
	}

	result := &Result{
		BuildID:      buildID,
		BuildDir:     buildDir,
		PayloadDir:   payloadDir,
		ManifestPath: filepath.Join(payloadDir, "manifest.json"),
		RBACPath:     filepath.Join(payloadDir, "rbac.json"),
		MetadataPath: filepath.Join(buildDir, "metadata.json"),
		PackagePath:  filepath.Join(buildDir, "package.tar.gz"),
	}

	var artifacts []Artifact
	var distHash string

	if !opts.SkipFrontend {
		if err := ensureFileExists(filepath.Join(opts.FrontendDir, "package.json"), "frontend package.json", "run 'npm install' in web-admin"); err != nil {
			return nil, err
		}
		if err := b.frontendBuilder(ctx, opts); err != nil {
			return nil, fmt.Errorf("frontend build failed: %w", err)
		}

		distPath, err := detectFrontendDist(opts.FrontendDir)
		if err != nil {
			return nil, err
		}
		result.FrontendPath = filepath.Join(payloadDir, "frontend")
		size, err := copyDir(distPath, result.FrontendPath)
		if err != nil {
			return nil, fmt.Errorf("copy frontend dist: %w", err)
		}
		distHash, err = hashDirectory(result.FrontendPath)
		if err != nil {
			return nil, fmt.Errorf("hash frontend dist: %w", err)
		}
		artifacts = append(artifacts, Artifact{
			Name: "frontend",
			Path: rel(result.FrontendPath, buildDir),
			Size: size,
			Hash: distHash,
		})
	}

	if !opts.SkipBackend {
		if err := ensureFileExists(filepath.Join(opts.BackendDir, "go.mod"), "backend go.mod", "run 'go mod tidy' inside backend"); err != nil {
			return nil, err
		}
		binaryName := "plugin"
		if runtime.GOOS == "windows" {
			binaryName += ".exe"
		}
		buildBinary := filepath.Join(buildDir, "backend", "bin", binaryName)
		if err := os.MkdirAll(filepath.Dir(buildBinary), 0o755); err != nil {
			return nil, fmt.Errorf("create backend build dir: %w", err)
		}
		if err := b.backendBuilder(ctx, opts, buildBinary); err != nil {
			return nil, fmt.Errorf("backend build failed: %w", err)
		}

		result.BackendBinaryPath = filepath.Join(payloadDir, "backend", "bin", binaryName)
		if err := os.MkdirAll(filepath.Dir(result.BackendBinaryPath), 0o755); err != nil {
			return nil, fmt.Errorf("create backend payload dir: %w", err)
		}
		if _, err := copyFile(buildBinary, result.BackendBinaryPath); err != nil {
			return nil, fmt.Errorf("copy backend binary: %w", err)
		}
		hash, size, err := hashFile(result.BackendBinaryPath)
		if err != nil {
			return nil, fmt.Errorf("hash backend binary: %w", err)
		}
		artifacts = append(artifacts, Artifact{
			Name: "backend/bin/" + filepath.Base(result.BackendBinaryPath),
			Path: rel(result.BackendBinaryPath, buildDir),
			Size: size,
			Hash: hash,
		})
	}

	if _, err := copyFile(opts.ManifestPath, result.ManifestPath); err != nil {
		return nil, fmt.Errorf("copy manifest.json: %w", err)
	}
	if _, err := copyFile(opts.RBACPath, result.RBACPath); err != nil {
		return nil, fmt.Errorf("copy rbac.json: %w", err)
	}

	if hash, size, err := hashFile(result.ManifestPath); err == nil {
		artifacts = append(artifacts, Artifact{
			Name: "manifest.json",
			Path: rel(result.ManifestPath, buildDir),
			Size: size,
			Hash: hash,
		})
	} else {
		return nil, fmt.Errorf("hash manifest.json: %w", err)
	}

	if hash, size, err := hashFile(result.RBACPath); err == nil {
		artifacts = append(artifacts, Artifact{
			Name: "rbac.json",
			Path: rel(result.RBACPath, buildDir),
			Size: size,
			Hash: hash,
		})
	} else {
		return nil, fmt.Errorf("hash rbac.json: %w", err)
	}

	metaDir := filepath.Join(payloadDir, "meta")
	copyIfExists(filepath.Join(opts.FrontendDir, "package.json"), filepath.Join(metaDir, "frontend.package.json"))
	copyIfExists(filepath.Join(opts.FrontendDir, "package-lock.json"), filepath.Join(metaDir, "frontend.package-lock.json"))

	if err := tarDirectory(payloadDir, result.PackagePath); err != nil {
		return nil, fmt.Errorf("create package: %w", err)
	}
	if hash, size, err := hashFile(result.PackagePath); err == nil {
		artifacts = append(artifacts, Artifact{
			Name: "package.tar.gz",
			Path: rel(result.PackagePath, buildDir),
			Size: size,
			Hash: hash,
		})
	} else {
		return nil, fmt.Errorf("hash package.tar.gz: %w", err)
	}

	if err := WriteMetadata(result.MetadataPath, MetadataOptions{
		Version:    version,
		Channel:    opts.Channel,
		BuildTime:  now,
		CLIVersion: opts.CLIVersion,
		GitCommit:  detectGitCommit(opts.EntryPath),
		DistHash:   distHash,
		Artifacts:  artifacts,
	}); err != nil {
		return nil, fmt.Errorf("write metadata: %w", err)
	}

	result.Artifacts = artifacts
	result.DistHash = distHash
	return result, nil
}

func ensureFileExists(path, label, remediation string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found at %s (%s)", label, path, remediation)
		}
		return fmt.Errorf("stat %s: %w", path, err)
	}
	return nil
}

func detectFrontendDist(frontendDir string) (string, error) {
	candidates := []string{
		filepath.Join(frontendDir, "dist"),
		filepath.Join(frontendDir, ".output", "public"),
		filepath.Join(frontendDir, ".output"),
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			empty, err := dirEmpty(candidate)
			if err != nil {
				return "", err
			}
			if !empty {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("frontend build output not found (expected dist/ or .output/ under %s)", frontendDir)
}

func dirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func copyDir(src, dst string) (int64, error) {
	var total int64
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, relPath)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if _, err := copyFile(path, target); err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func copyFile(src, dst string) (int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return 0, err
	}

	out, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return 0, err
	}

	if stat, err := out.Stat(); err == nil {
		return stat.Size(), nil
	}
	return 0, nil
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

func hashDirectory(path string) (string, error) {
	h := sha256.New()
	var files []string
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, p)
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no files under %s to hash", path)
	}
	sort.Strings(files)

	for _, p := range files {
		relPath, err := filepath.Rel(path, p)
		if err != nil {
			return "", err
		}
		if _, err := h.Write([]byte(relPath)); err != nil {
			return "", err
		}
		file, err := os.Open(p)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(h, file); err != nil {
			file.Close()
			return "", err
		}
		file.Close()
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func tarDirectory(srcDir, destFile string) error {
	f, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	var entries []string
	err = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		entries = append(entries, path)
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(entries)

	for _, path := range entries {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			continue
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, file); err != nil {
			file.Close()
			return err
		}
		file.Close()
	}

	return nil
}

func rel(path, base string) string {
	r, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(r)
}

func copyIfExists(src, dst string) {
	if _, err := os.Stat(src); err == nil {
		_, _ = copyFile(src, dst)
	}
}

func detectGitCommit(entry string) string {
	cmd := exec.Command("git", "-C", entry, "rev-parse", "--short", "12", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func defaultFrontendBuild(ctx context.Context, opts *Options) error {
	cmd := exec.CommandContext(ctx, "npm", "--prefix", opts.FrontendDir, "run", "build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func defaultBackendBuild(ctx context.Context, opts *Options, outputPath string) error {
	args := []string{"build", "-o", outputPath, "./backend/cmd/plugin"}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = opts.EntryPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
