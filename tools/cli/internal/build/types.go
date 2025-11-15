package build

import (
	"context"
	"time"
)

// BuildStrategy represents the build strategy
type BuildStrategy int

const (
	StrategyFull       BuildStrategy = iota // 完整构建
	StrategyIncremental                     // 增量构建
	StrategyDiff                            // 差异构建（仅 changed files）
)

func (s BuildStrategy) String() string {
	switch s {
	case StrategyFull:
		return "full"
	case StrategyIncremental:
		return "incremental"
	case StrategyDiff:
		return "diff"
	default:
		return "incremental"
	}
}

// FileEvent represents a file change (reused from watch package)
type FileEvent struct {
	Type      string    `json:"type"`
	Path      string    `json:"path"`
	Hash      string    `json:"hash,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// BuildResult represents the result of a build
type BuildResult struct {
	BundleHash   string    `json:"bundleHash"`
	BundleSize   int64     `json:"bundleSize"`
	BuildDuration time.Duration `json:"buildDuration"`
	Artifacts    []string  `json:"artifacts"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
}

// BuildOptions holds build options
type BuildOptions struct {
	EntryPath   string
	Strategy    BuildStrategy
	ChangedFiles []FileEvent
	OutDir      string
	Verbose     bool
}

// Builder interface for building plugins
type Builder interface {
	Build(ctx context.Context, opts *BuildOptions) (*BuildResult, error)
	Name() string
	Description() string
}

// NewBuildResult creates a new build result
func NewBuildResult(startTime time.Time) *BuildResult {
	return &BuildResult{
		StartTime: startTime,
		Success:   true,
		Artifacts: []string{},
	}
}

// Complete marks the build as complete
func (r *BuildResult) Complete(endTime time.Time, success bool, err error) {
	r.EndTime = endTime
	r.BuildDuration = endTime.Sub(r.StartTime)
	r.Success = success
	if !success && err != nil {
		r.Error = err.Error()
	}
}
