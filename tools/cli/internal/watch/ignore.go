package watch

import (
	"os"
	"path/filepath"
	"strings"
)

// Matcher checks if a path should be ignored
type Matcher struct {
	patterns []string
}

// NewMatcher creates a new ignore matcher
func NewMatcher(patterns []string) *Matcher {
	// Normalize patterns
	normalized := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		// Convert glob patterns to match filepath.Match
		normalized = append(normalized, normalizePattern(pattern))
	}

	return &Matcher{
		patterns: normalized,
	}
}

// ShouldIgnore checks if a path should be ignored
func (m *Matcher) ShouldIgnore(path string) bool {
	// Get relative path
	rel, err := filepath.Rel(".", path)
	if err != nil {
		rel = path
	}

	// Check each pattern
	for _, pattern := range m.patterns {
		if match, err := filepath.Match(pattern, rel); err == nil && match {
			return true
		}
		// Also check with filepath.Separator replacement for cross-platform
		altPattern := strings.ReplaceAll(pattern, "/", string(os.PathSeparator))
		altRel := strings.ReplaceAll(rel, "/", string(os.PathSeparator))
		if match, err := filepath.Match(altPattern, altRel); err == nil && match {
			return true
		}
	}

	return false
}

// normalizePattern normalizes a pattern for matching
func normalizePattern(pattern string) string {
	// Remove leading ./
	if strings.HasPrefix(pattern, "./") {
		pattern = pattern[2:]
	}

	// Convert to forward slashes for consistency
	pattern = strings.ReplaceAll(pattern, string(os.PathSeparator), "/")

	return pattern
}
