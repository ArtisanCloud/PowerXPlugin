package service

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/internal/compliance/scanner"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

var pluginIDPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$`)

// BootstrapHandler handles CLI bootstrap/validation requests.
type BootstrapHandler struct {
	logger     *slog.Logger
	templates  map[string]TemplateSpec
	scanner    *scanner.LicenseScanner
	defaultOrg string
}

// TemplateSpec captures registry metadata snippet for validation.
type TemplateSpec struct {
	ID          string `json:"id"`
	Backend     string `json:"backend"`
	Frontend    string `json:"frontend,omitempty"`
	MinGo       string `json:"minGo"`
	MinNode     string `json:"minNode,omitempty"`
	Recommended bool   `json:"recommended"`
}

// NewBootstrapHandler builds a handler with default registry entries.
func NewBootstrapHandler(logger *slog.Logger) *BootstrapHandler {
	if logger == nil {
		logger = slog.Default()
	}
	templates := map[string]TemplateSpec{
		"fullstack-go-nuxt": {
			ID:          "fullstack-go-nuxt",
			Backend:     "go-gin",
			Frontend:    "nuxt",
			MinGo:       "1.24",
			MinNode:     "18",
			Recommended: true,
		},
		"backend-go-lite": {
			ID:      "backend-go-lite",
			Backend: "go-gin",
			MinGo:   "1.24",
		},
	}
	return &BootstrapHandler{
		logger:     logger,
		templates:  templates,
		scanner:    scanner.NewLicenseScanner(logger),
		defaultOrg: "powerx-plugins",
	}
}

// Validate processes incoming CLI bootstrap validation requests.
func (h *BootstrapHandler) Validate(ctx bootstrap.Context) {
	var req BootstrapValidateRequest
	if err := ctx.BindJSON(&req); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "unable to parse bootstrap payload", nil)
		return
	}

	if err := h.validateRequest(req); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
		return
	}

	template := h.templates[req.TemplateID]
	scanResult, err := h.scanner.Scan(scanner.ScanRequest{
		PluginID:     req.PluginID,
		Version:      req.Version,
		SBOMPath:     req.SBOMPath,
		SourceType:   req.Source.Type,
		Dependencies: req.Dependencies,
	})
	if err != nil {
		router.RespondError(ctx, http.StatusInternalServerError, "SCAN_FAILED", err.Error(), nil)
		return
	}

	response := BootstrapValidateResponse{
		ValidationID:    fmt.Sprintf("val-%d", time.Now().UnixNano()),
		PluginID:        req.PluginID,
		Version:         req.Version,
		Template:        template,
		GitRepository:   h.buildGitRepository(req),
		ComplianceScan:  scanResult,
		Approved:        !scanResult.Blocked,
		RequiredActions: h.requiredActions(scanResult),
		AuditRecord: AuditRecord{
			ID:         fmt.Sprintf("audit-%d", time.Now().UnixNano()),
			WebhookURL: fmt.Sprintf("https://audit.powerx.dev/hooks/plugin-import/%s", req.PluginID),
			Status:     "queued",
		},
		Message: "bootstrap validation completed",
	}

	router.RespondSuccess(ctx, http.StatusOK, response, "bootstrap validation completed")
}

func (h *BootstrapHandler) validateRequest(req BootstrapValidateRequest) error {
	if req.PluginID == "" {
		return fmt.Errorf("pluginId is required")
	}
	if !pluginIDPattern.MatchString(req.PluginID) {
		return fmt.Errorf("pluginId must match %s", pluginIDPattern.String())
	}
	if req.TemplateID == "" {
		return fmt.Errorf("templateId is required")
	}
	if _, ok := h.templates[req.TemplateID]; !ok {
		return fmt.Errorf("templateId %q is not supported", req.TemplateID)
	}
	if req.Version == "" {
		return fmt.Errorf("version is required")
	}
	if req.Organization == "" {
		return fmt.Errorf("organization is required")
	}
	if len(req.Dependencies) == 0 {
		return fmt.Errorf("dependencies list must contain at least one entry")
	}
	return nil
}

func (h *BootstrapHandler) buildGitRepository(req BootstrapValidateRequest) GitRepository {
	repoName := req.Git.RepoName
	if repoName == "" {
		repoName = strings.ReplaceAll(req.PluginID, ".", "-")
	}
	org := req.Organization
	if org == "" {
		org = h.defaultOrg
	}
	provider := req.Git.Provider
	if provider == "" {
		provider = "gitlab"
	}
	url := fmt.Sprintf("https://git.powerx.dev/%s/%s", org, repoName)
	return GitRepository{
		Provider:   provider,
		Name:       repoName,
		Visibility: req.Git.Visibility,
		URL:        url,
	}
}

func (h *BootstrapHandler) requiredActions(scan scanner.ScanResult) []string {
	if scan.Blocked {
		return []string{
			"Resolve blocked licenses or vulnerabilities",
			"Resubmit scan via px-plugin doctor --fix",
		}
	}
	if len(scan.Issues) > 0 {
		return []string{"Review high severity dependencies", "Attach compliance waiver if required"}
	}
	return []string{"Create repository in Git", "Push initial commit", "Trigger CI bootstrap pipeline"}
}

// BootstrapValidateRequest is the expected CLI payload.
type BootstrapValidateRequest struct {
	PluginID     string               `json:"pluginId"`
	Version      string               `json:"version"`
	TemplateID   string               `json:"templateId"`
	Organization string               `json:"organization"`
	SBOMPath     string               `json:"sbomPath"`
	Git          GitOptions           `json:"git"`
	Source       SourceOptions        `json:"source"`
	Dependencies []scanner.Dependency `json:"dependencies"`
}

// GitOptions comes from CLI payload and describes Git registration preferences.
type GitOptions struct {
	Provider   string `json:"provider"`
	RepoName   string `json:"repoName"`
	Visibility string `json:"visibility"`
}

// SourceOptions describe third-party import metadata.
type SourceOptions struct {
	Type      string `json:"type"`
	Reference string `json:"reference"`
}

// BootstrapValidateResponse summarizes validation state for CLI.
type BootstrapValidateResponse struct {
	ValidationID    string             `json:"validationId"`
	PluginID        string             `json:"pluginId"`
	Version         string             `json:"version"`
	Template        TemplateSpec       `json:"template"`
	GitRepository   GitRepository      `json:"gitRepository"`
	ComplianceScan  scanner.ScanResult `json:"complianceScan"`
	Approved        bool               `json:"approved"`
	RequiredActions []string           `json:"requiredActions"`
	AuditRecord     AuditRecord        `json:"audit"`
	Message         string             `json:"message"`
}

// GitRepository describes the repository to create.
type GitRepository struct {
	Provider   string `json:"provider"`
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
	URL        string `json:"url"`
}

// AuditRecord represents `plugin-import-audit` webhook scheduling.
type AuditRecord struct {
	ID         string `json:"id"`
	WebhookURL string `json:"webhookUrl"`
	Status     string `json:"status"`
}
