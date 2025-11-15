package services

import "time"

// PublishPayload mirrors CLI publish metadata passed to Marketplace.
type PublishPayload struct {
	PublishID string   `json:"publishId"`
	VersionID string   `json:"versionId"`
	PluginID  string   `json:"pluginId"`
	Channel   string   `json:"channel"`
	Notes     string   `json:"notes,omitempty"`
	Artefacts []string `json:"artefacts"`
}

// ScanReport summarizes automated checks.
type ScanReport struct {
	TotalFiles int      `json:"totalFiles"`
	Warnings   []string `json:"warnings"`
	Errors     []string `json:"errors"`
}

// ReviewRecord is stored for reviewer console.
type ReviewRecord struct {
	PublishID    string     `json:"publishId"`
	VersionID    string     `json:"versionId"`
	Channel      string     `json:"channel"`
	SubmittedAt  time.Time  `json:"submittedAt"`
	Status       string     `json:"status"`
	ScanFindings ScanReport `json:"scanFindings"`
}

// Scanner interface decouples handler from implementation.
type Scanner interface {
	Scan(payload PublishPayload) ScanReport
}

// DefaultScanner provides minimal placeholder implementation.
type DefaultScanner struct{}

func (DefaultScanner) Scan(payload PublishPayload) ScanReport {
	warnings := []string{}
	if len(payload.Notes) == 0 {
		warnings = append(warnings, "missing release notes")
	}
	if len(payload.Artefacts) == 0 {
		return ScanReport{Errors: []string{"no artefacts attached"}}
	}
	return ScanReport{
		TotalFiles: len(payload.Artefacts),
		Warnings:   warnings,
	}
}

// Helper to instantiate default scanner.
func NewDefaultScanner() Scanner {
	return DefaultScanner{}
}
