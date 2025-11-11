package scanner

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Dependency represents a dependency entry from sbom/manifest lists.
type Dependency struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	License  string `json:"license"`
	Source   string `json:"source,omitempty"`
	Severity string `json:"severity,omitempty"`
}

// ScanRequest captures the payload required to run a compliance scan.
type ScanRequest struct {
	PluginID     string       `json:"pluginId"`
	Version      string       `json:"version"`
	SBOMPath     string       `json:"sbomPath,omitempty"`
	SourceType   string       `json:"sourceType,omitempty"`
	Dependencies []Dependency `json:"dependencies"`
}

// ScanIssue describes a single compliance finding.
type ScanIssue struct {
	Dependency Dependency `json:"dependency"`
	Severity   string     `json:"severity"`
	Code       string     `json:"code"`
	Message    string     `json:"message"`
}

// ScanResult summarizes the outcome of a compliance scan.
type ScanResult struct {
	Status      string      `json:"status"`
	Blocked     bool        `json:"blocked"`
	Issues      []ScanIssue `json:"issues"`
	ReportURL   string      `json:"reportUrl"`
	GeneratedAt time.Time   `json:"generatedAt"`
}

// LicenseScanner performs lightweight policy evaluation for dependencies.
type LicenseScanner struct {
	logger          *slog.Logger
	blockedLicenses map[string]struct{}
}

// NewLicenseScanner builds a new scanner with default deny-lists.
func NewLicenseScanner(logger *slog.Logger) *LicenseScanner {
	if logger == nil {
		logger = slog.Default()
	}
	blocked := map[string]struct{}{
		"gpl-3.0":  {},
		"agpl-3.0": {},
		"lgpl-3.0": {},
	}
	return &LicenseScanner{
		logger:          logger,
		blockedLicenses: blocked,
	}
}

// Scan evaluates the provided dependencies and returns a structured result.
func (s *LicenseScanner) Scan(req ScanRequest) (ScanResult, error) {
	if req.PluginID == "" {
		return ScanResult{}, errors.New("plugin id is required")
	}
	if req.Version == "" {
		return ScanResult{}, errors.New("version is required")
	}
	result := ScanResult{
		Status:      "pass",
		Blocked:     false,
		Issues:      []ScanIssue{},
		ReportURL:   fmt.Sprintf("https://compliance.powerx.dev/reports/%s/%s", req.PluginID, req.Version),
		GeneratedAt: time.Now().UTC(),
	}

	for _, dep := range req.Dependencies {
		license := strings.ToLower(dep.License)
		if _, blocked := s.blockedLicenses[license]; blocked {
			result.Blocked = true
			result.Status = "blocked"
			result.Issues = append(result.Issues, ScanIssue{
				Dependency: dep,
				Severity:   "critical",
				Code:       "LICENSE_DENYLIST",
				Message:    fmt.Sprintf("license %s is not permitted for tenant distribution", dep.License),
			})
			continue
		}
		if dep.Severity == "high" || dep.Severity == "critical" {
			result.Status = "needs_review"
			result.Issues = append(result.Issues, ScanIssue{
				Dependency: dep,
				Severity:   "high",
				Code:       "VULNERABILITY",
				Message:    fmt.Sprintf("dependency %s@%s flagged as %s severity", dep.Name, dep.Version, dep.Severity),
			})
		}
	}

	if result.Blocked {
		s.logger.Warn("compliance scan blocked publish candidate",
			slog.String("plugin", req.PluginID),
			slog.Int("issues", len(result.Issues)))
	} else if len(result.Issues) > 0 {
		s.logger.Info("compliance scan requires follow-up",
			slog.String("plugin", req.PluginID),
			slog.Int("issues", len(result.Issues)))
	} else {
		s.logger.Info("compliance scan passed", slog.String("plugin", req.PluginID))
	}

	return result, nil
}
