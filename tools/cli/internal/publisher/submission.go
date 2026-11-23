package publisher

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/powerx-plugin/cli/internal/publisher/wire"
)

const SubmissionSchema = "powerx.plugin-publisher-submission/v1"
const MaxSubmissionRecordBytes = 1 << 20
const SubmissionChanged = "PUBLISHER_SUBMISSION_CHANGED"
const SubmissionCapability = "PUBLISHER_SUBMISSION_CAPABILITY"
const SubmissionState = "PUBLISHER_SUBMISSION_STATE"

type PublicationMetadata struct {
	Description string `json:"description"`
	License     string `json:"license"`
	MinCore     string `json:"min_core"`
	SupportURL  string `json:"support_url"`
	IconURL     string `json:"icon_url"`
	Security    string `json:"security_statement"`
	Privacy     string `json:"privacy_statement"`
}
type PublicationOptions struct {
	Versions      []string `json:"versions"`
	Licenses      []string `json:"licenses"`
	Source        string   `json:"source"`
	VersionSource string   `json:"version_source"`
}
type SubmissionInput struct {
	ConnectionID string              `json:"connection_id"`
	VersionID    string              `json:"version_id"`
	Version      string              `json:"version"`
	Metadata     PublicationMetadata `json:"metadata"`
	Artifacts    []MarketArtifact    `json:"expected_artifacts"`
}
type SubmissionPreview struct {
	Input         SubmissionInput `json:"input"`
	RequestSHA256 string          `json:"request_sha256"`
}
type SubmissionPreparation struct {
	Version MarketVersion      `json:"version"`
	Options PublicationOptions `json:"options"`
}
type SubmissionReceipt struct {
	OperationID   string    `json:"operation_id"`
	PublicationID string    `json:"publication_id"`
	VersionID     string    `json:"version_id"`
	Status        string    `json:"status"`
	SubmittedAt   time.Time `json:"submitted_at"`
	URL           string    `json:"market_publication_url"`
}
type Publication struct {
	PublicationID string              `json:"publication_id"`
	PluginID      string              `json:"plugin_id"`
	VersionID     string              `json:"version_id"`
	Version       string              `json:"version"`
	Status        string              `json:"status"`
	Metadata      PublicationMetadata `json:"metadata"`
	Artifacts     []MarketArtifact    `json:"artifacts"`
	SubmittedAt   time.Time           `json:"submitted_at"`
	ReviewedAt    *time.Time          `json:"reviewed_at"`
	ReviewReason  string              `json:"review_reason"`
	PublishedAt   *time.Time          `json:"published_at"`
	PublishReason string              `json:"publish_reason"`
	URL           string              `json:"market_publication_url"`
}
type PublicationsPage struct {
	Items []Publication `json:"items"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Size  int           `json:"size"`
}
type Submission struct {
	Schema          string `json:"schema"`
	ContractVersion string `json:"contract_version"`
	BundleSHA256    string `json:"bundle_sha256"`
	SubmissionID    string `json:"submission_id"`
	OperationID     string `json:"operation_id"`
	PluginID        string `json:"plugin_id"`
	UserID          string `json:"user_id"`
	DeveloperID     string `json:"developer_id"`
	SubmissionInput
	RequestSHA256      string             `json:"request_sha256"`
	Stage              string             `json:"stage"`
	Attempts           int                `json:"attempts"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
	Receipt            *SubmissionReceipt `json:"receipt,omitempty"`
	CurrentPublication *Publication       `json:"current_publication,omitempty"`
	Error              *Error             `json:"error,omitempty"`
}
type submissionOperation struct {
	OperationID string             `json:"operation_id"`
	Type        string             `json:"type"`
	PluginID    string             `json:"plugin_id"`
	SHA256      string             `json:"request_sha256"`
	Status      string             `json:"status"`
	RetryAfter  int                `json:"retry_after_seconds"`
	Result      *SubmissionReceipt `json:"result"`
	Error       *marketError       `json:"error"`
}

func normalizeSubmission(plugin string, input SubmissionInput) (SubmissionPreview, error) {
	if !canonicalUUID(input.ConnectionID) || !canonicalUUID(input.VersionID) || len(input.Version) > 50 || !stableVersion.MatchString(input.Version) {
		return SubmissionPreview{}, invalid("submission")
	}
	raw, err := json.Marshal(map[string]any{"plugin_id": plugin, "version_id": input.VersionID, "metadata": input.Metadata, "expected_artifacts": input.Artifacts})
	if err != nil {
		return SubmissionPreview{}, invalid("submission")
	}
	canonical, digest, err := wire.Fingerprint("publication-submit", raw)
	if err != nil {
		return SubmissionPreview{}, invalid("submission")
	}
	var normalized struct {
		Metadata  PublicationMetadata `json:"metadata"`
		Artifacts []MarketArtifact    `json:"expected_artifacts"`
	}
	if json.Unmarshal([]byte(canonical), &normalized) != nil {
		return SubmissionPreview{}, failure(MarketProtocol, nil)
	}
	input.Metadata = normalized.Metadata
	input.Artifacts = normalized.Artifacts
	return SubmissionPreview{Input: input, RequestSHA256: digest}, nil
}
func (e *Engine) PreviewSubmission(input SubmissionInput) (SubmissionPreview, error) {
	return normalizeSubmission(e.plugin.ID, input)
}
func submissionAllowed(m Market) bool {
	return slices.Contains(m.Features, "publication-submit") && slices.Contains(m.Features, "idempotent-writes")
}
func submissionScope(m Market, write bool) bool {
	return m.Identity != nil && slices.Contains(m.Identity.Credential.Scopes, "publications:read") && (!write || slices.Contains(m.Identity.Credential.Scopes, "publications:submit"))
}
func readMarketVersion(ctx context.Context, m Market, token, plugin, id string) (MarketVersion, error) {
	var v MarketVersion
	_, err := marketRequest(ctx, m, token, "/publisher/plugins/"+plugin+"/versions/"+id, "MarketVersionResponse", &v, nil)
	if err != nil {
		return v, err
	}
	if v.VersionID != id || v.PluginID != plugin || !wire.TrustedMarketLink(m.ConsoleBase, v.URL) {
		return v, failure(MarketProtocol, nil)
	}
	return v, nil
}
func readPublicationOptions(ctx context.Context, m Market, token string) (PublicationOptions, error) {
	var o PublicationOptions
	_, err := marketRequest(ctx, m, token, "/publisher/publication-options", "PublicationOptionsResponse", &o, nil)
	return o, err
}
func (e *Engine) PrepareSubmission(ctx context.Context, connectionID, versionID string) (SubmissionPreparation, error) {
	// Reading the complete collection remains valid after submission. Eligibility
	// for a new write is checked separately immediately before sending it.
	if !canonicalUUID(versionID) {
		return SubmissionPreparation{}, invalid("version_id")
	}
	m, err := e.VerifyMarket(ctx, connectionID)
	if err != nil {
		return SubmissionPreparation{}, err
	}
	if !submissionAllowed(m) {
		return SubmissionPreparation{}, failure(SubmissionCapability, nil)
	}
	if !submissionScope(m, true) {
		return SubmissionPreparation{}, failure(MarketCredential, nil)
	}
	token, err := e.credentials.Get(ctx, connectionID)
	if err != nil {
		return SubmissionPreparation{}, failure(MarketKeyring, err)
	}
	v, err := readMarketVersion(ctx, m, token, e.plugin.ID, versionID)
	if err != nil {
		return SubmissionPreparation{}, err
	}
	o, err := readPublicationOptions(ctx, m, token)
	return SubmissionPreparation{Version: v, Options: o}, err
}
func publicationPage(ctx context.Context, m Market, token, plugin string, page, size int) (PublicationsPage, error) {
	var out PublicationsPage
	_, err := marketRequest(ctx, m, token, "/publisher/plugins/"+plugin+"/publications?"+url.Values{"page": {fmtInt(page)}, "size": {fmtInt(size)}}.Encode(), "PublicationsPageResponse", &out, nil)
	if err != nil {
		return out, err
	}
	for _, p := range out.Items {
		if p.PluginID != plugin || !wire.TrustedMarketLink(m.ConsoleBase, p.URL) {
			return out, failure(MarketProtocol, nil)
		}
	}
	return out, nil
}
func (e *Engine) MarketPublications(ctx context.Context, id string, page, size int) (PublicationsPage, error) {
	if page < 1 || page > 1000000 || size < 1 || size > 100 {
		return PublicationsPage{}, invalid("pagination")
	}
	m, err := e.VerifyMarket(ctx, id)
	if err != nil {
		return PublicationsPage{}, err
	}
	if !submissionScope(m, false) {
		return PublicationsPage{}, failure(MarketCredential, nil)
	}
	token, err := e.credentials.Get(ctx, id)
	if err != nil {
		return PublicationsPage{}, failure(MarketKeyring, err)
	}
	return publicationPage(ctx, m, token, e.plugin.ID, page, size)
}
func artifactSetEqual(a, b []MarketArtifact) bool {
	aa := slices.Clone(a)
	bb := slices.Clone(b)
	less := func(x, y MarketArtifact) int { return strings.Compare(x.OS+"\x00"+x.Arch, y.OS+"\x00"+y.Arch) }
	slices.SortFunc(aa, less)
	slices.SortFunc(bb, less)
	return slices.Equal(aa, bb)
}
func submissionActive(stage string) bool {
	return uploadActive(stage) || stage == "sending" || stage == "submitted"
}
func (e *Engine) submissionDir() (*directory, error) { return e.state.child("submissions", true) }
func (e *Engine) saveSubmission(s *Submission, replace bool) error {
	s.UpdatedAt = time.Now().UTC()
	raw, err := json.Marshal(s)
	if err != nil {
		return failure(StorageFailure, err)
	}
	if len(raw) > MaxSubmissionRecordBytes {
		return failure(TooLarge, nil)
	}
	d, err := e.submissionDir()
	if err != nil {
		return err
	}
	defer d.close()
	return d.write(s.SubmissionID+".json", raw, replace)
}
func (e *Engine) GetSubmission(id string) (Submission, error) {
	if !canonicalUUID(id) {
		return Submission{}, invalid("submission_id")
	}
	d, err := e.submissionDir()
	if err != nil {
		return Submission{}, err
	}
	defer d.close()
	raw, err := d.read(id+".json", MaxSubmissionRecordBytes)
	var s Submission
	if err != nil {
		return s, err
	}
	if decodeStrict(raw, &s) != nil || s.Schema != SubmissionSchema || s.ContractVersion != MarketContractVersion || s.BundleSHA256 != MarketBundleSHA256 || s.SubmissionID != id || s.PluginID != e.plugin.ID || !canonicalUUID(s.OperationID) || !canonicalUUID(s.UserID) || !canonicalUUID(s.DeveloperID) {
		return Submission{}, failure(InvalidRecord, nil)
	}
	normalized, err := normalizeSubmission(s.PluginID, s.SubmissionInput)
	if err != nil || normalized.RequestSHA256 != s.RequestSHA256 {
		return Submission{}, failure(InvalidRecord, nil)
	}
	e.mu.Lock()
	_, active := e.submissionJobs[id]
	e.mu.Unlock()
	if submissionActive(s.Stage) && !active {
		s.Stage = "pending"
		s.Error = failure(MarketPending, nil)
	}
	return s, nil
}
func (e *Engine) ListSubmissions() ([]Submission, error) {
	d, err := e.submissionDir()
	if err != nil {
		return nil, err
	}
	defer d.close()
	names, err := d.names()
	if err != nil {
		return nil, err
	}
	out := []Submission{}
	for _, name := range names {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		s, err := e.GetSubmission(strings.TrimSuffix(name, ".json"))
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
func (e *Engine) StartSubmission(ctx context.Context, input SubmissionInput) (Submission, error) {
	preview, err := e.PreviewSubmission(input)
	if err != nil {
		return Submission{}, err
	}
	m, err := e.GetMarket(input.ConnectionID)
	if err != nil {
		return Submission{}, err
	}
	if m.Identity == nil || m.VerifiedAt == nil {
		return Submission{}, failure(MarketCredential, nil)
	}
	d, err := e.submissionDir()
	if err != nil {
		return Submission{}, err
	}
	defer d.close()
	release, err := d.lock()
	if err != nil {
		return Submission{}, failure(Conflict, err)
	}
	defer release()
	old, err := e.ListSubmissions()
	if err != nil {
		return Submission{}, err
	}
	for _, s := range old {
		if s.ConnectionID == input.ConnectionID && s.VersionID == input.VersionID && s.Stage != "failed" {
			if s.Stage != "succeeded" || s.CurrentPublication == nil || s.CurrentPublication.Status != "rejected" {
				return Submission{}, failure(Conflict, nil)
			}
		}
	}
	now := time.Now().UTC()
	s := Submission{Schema: SubmissionSchema, ContractVersion: MarketContractVersion, BundleSHA256: MarketBundleSHA256, SubmissionID: uuid.NewString(), OperationID: uuid.NewString(), PluginID: e.plugin.ID, UserID: m.Identity.User.ID, DeveloperID: m.Identity.Developer.DeveloperID, SubmissionInput: preview.Input, RequestSHA256: preview.RequestSHA256, Stage: "queued", CreatedAt: now, UpdatedAt: now}
	if err = e.saveSubmission(&s, false); err != nil {
		return Submission{}, err
	}
	return e.launchSubmission(ctx, s, false)
}
func (e *Engine) ResumeSubmission(ctx context.Context, id string) (Submission, error) {
	s, err := e.GetSubmission(id)
	if err != nil {
		return s, err
	}
	return e.launchSubmission(ctx, s, true)
}
func (e *Engine) launchSubmission(ctx context.Context, s Submission, reconcile bool) (Submission, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closing {
		return s, failure(Conflict, nil)
	}
	if _, active := e.submissionJobs[s.SubmissionID]; active {
		return s, failure(Conflict, nil)
	}
	d, err := e.submissionDir()
	if err != nil {
		return s, err
	}
	owner, err := d.child(s.SubmissionID, true)
	d.close()
	if err != nil {
		return s, err
	}
	release, err := owner.lock()
	if err != nil {
		owner.close()
		return s, failure(Conflict, err)
	}
	run, cancel := context.WithCancel(ctx)
	e.submissionJobs[s.SubmissionID] = cancel
	s.Stage = "reconciling"
	s.Error = nil
	if err = e.saveSubmission(&s, true); err != nil {
		delete(e.submissionJobs, s.SubmissionID)
		cancel()
		release()
		owner.close()
		return s, err
	}
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer release()
		defer owner.close()
		defer cancel()
		defer func() { e.mu.Lock(); delete(e.submissionJobs, s.SubmissionID); e.mu.Unlock() }()
		e.runSubmission(run, s, reconcile)
	}()
	return s, nil
}
func (e *Engine) CancelSubmission(id string) (Submission, error) {
	e.mu.Lock()
	cancel := e.submissionJobs[id]
	e.mu.Unlock()
	if cancel == nil {
		s, err := e.GetSubmission(id)
		if err == nil && s.Stage == "pending" {
			err = failure(Conflict, nil)
		}
		return s, err
	}
	cancel()
	return e.GetSubmission(id)
}
func (e *Engine) WaitSubmission(id string) (Submission, error) {
	for {
		e.mu.Lock()
		_, active := e.submissionJobs[id]
		e.mu.Unlock()
		if !active {
			return e.GetSubmission(id)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
func (e *Engine) updatePublication(ctx context.Context, s *Submission, m Market, token string) error {
	s.CurrentPublication = nil
	if err := e.saveSubmission(s, true); err != nil {
		return err
	}
	for page := 1; page <= 1000000; page++ {
		list, err := publicationPage(ctx, m, token, s.PluginID, page, 100)
		if err != nil {
			return err
		}
		for _, p := range list.Items {
			if p.PublicationID != s.Receipt.PublicationID {
				continue
			}
			if p.PluginID != s.PluginID || p.VersionID != s.VersionID || p.Version != s.Version {
				return failure(MarketProtocol, nil)
			}
			preview, err := normalizeSubmission(s.PluginID, SubmissionInput{ConnectionID: s.ConnectionID, VersionID: p.VersionID, Version: p.Version, Metadata: p.Metadata, Artifacts: p.Artifacts})
			if err != nil || preview.RequestSHA256 != s.RequestSHA256 {
				return failure(MarketProtocol, err)
			}
			s.CurrentPublication = &p
			return nil
		}
		if page*100 >= list.Total {
			return failure(MarketProtocol, nil)
		}
	}
	return failure(MarketPending, nil)
}
func (e *Engine) acceptSubmission(ctx context.Context, s *Submission, m Market, token string, r *SubmissionReceipt) error {
	if r == nil || r.OperationID != s.OperationID || r.VersionID != s.VersionID || r.Status != "submitted" || !canonicalUUID(r.PublicationID) || !wire.TrustedMarketLink(m.ConsoleBase, r.URL) {
		return failure(MarketProtocol, nil)
	}

	previous, err := e.ListSubmissions()
	if err != nil {
		return err
	}
	for _, old := range previous {
		if old.SubmissionID != s.SubmissionID && old.ConnectionID == s.ConnectionID && old.VersionID == s.VersionID && old.Receipt != nil && old.Receipt.PublicationID == r.PublicationID {
			return failure(MarketProtocol, nil)
		}
	}
	s.Receipt = r
	s.CurrentPublication = nil
	s.Stage = "submitted"
	if err := e.saveSubmission(s, true); err != nil {
		return err
	}
	return e.updatePublication(ctx, s, m, token)
}
func (e *Engine) runSubmission(ctx context.Context, s Submission, reconcile bool) {
	ctx = e.traceRequests(ctx, "submissions", s.SubmissionID, s.OperationID)
	finish := func(stage string, err error) {
		s.Stage = stage
		s.Error = nil
		if err != nil {
			if ctx.Err() != nil && stage == "pending" {
				err = failure(MarketPending, nil)
			}
			s.Error = AsError(err)
		}
		_ = e.saveSubmission(&s, true)
	}
	defer func() {
		if ctx.Err() == nil {
			return
		}
		e.mu.Lock()
		closing := e.closing
		e.mu.Unlock()
		if closing {
			return
		}
		read, cancel := context.WithTimeout(context.WithoutCancel(ctx), 8*time.Second)
		defer cancel()
		m, err := e.VerifyMarket(read, s.ConnectionID)
		if err != nil || m.Identity == nil || m.Identity.User.ID != s.UserID || m.Identity.Developer.DeveloperID != s.DeveloperID || !submissionScope(m, false) {
			return
		}
		token, err := e.credentials.Get(read, s.ConnectionID)
		if err != nil {
			return
		}
		if s.Receipt != nil {
			if e.updatePublication(read, &s, m, token) == nil {
				finish("succeeded", nil)
			}
			return
		}
		var op submissionOperation
		_, err = marketRequest(read, m, token, "/publisher/operations/"+s.OperationID, "OperationResponse", &op, nil)
		if err != nil || op.OperationID != s.OperationID || op.Type != "publication-submit" || op.PluginID != s.PluginID || op.SHA256 != s.RequestSHA256 {
			return
		}
		if op.Status == "succeeded" && e.acceptSubmission(read, &s, m, token, op.Result) == nil {
			finish("succeeded", nil)
		}
	}()
	s.CurrentPublication = nil
	if e.saveSubmission(&s, true) != nil {
		return
	}
	m, err := e.VerifyMarket(ctx, s.ConnectionID)
	if err != nil {
		finish("pending", err)
		return
	}
	if m.Identity.User.ID != s.UserID || m.Identity.Developer.DeveloperID != s.DeveloperID {
		finish("pending", failure(MarketIdentity, nil))
		return
	}
	if !submissionScope(m, false) {
		finish("pending", failure(MarketCredential, nil))
		return
	}
	token, err := e.credentials.Get(ctx, s.ConnectionID)
	if err != nil {
		finish("pending", failure(MarketKeyring, err))
		return
	}
	if s.Receipt != nil {
		read, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err = e.updatePublication(read, &s, m, token); err != nil {
			finish("pending", err)
		} else {
			finish("succeeded", nil)
		}
		return
	}
	pollStart := time.Now()
	status := "not_found"
	retry := 0
	if reconcile || s.Attempts > 0 {
		status = "unknown"
	}
	for {
		if ctx.Err() != nil {
			finish("pending", failure(MarketPending, nil))
			return
		}
		if status == "unknown" || status == "processing" {
			var op submissionOperation
			httpStatus, readErr := marketRequest(ctx, m, token, "/publisher/operations/"+s.OperationID, "OperationResponse", &op, nil)
			if readErr != nil {
				if httpStatus == 404 {
					status = "not_found"
					retry = 0
				} else {
					finish("pending", readErr)
					return
				}
			} else {
				if op.OperationID != s.OperationID || op.PluginID != s.PluginID || op.Type != "publication-submit" || op.SHA256 != s.RequestSHA256 {
					finish("pending", failure(MarketProtocol, nil))
					return
				}
				switch op.Status {
				case "succeeded":
					read, cancel := context.WithTimeout(ctx, 30*time.Second)
					err = e.acceptSubmission(read, &s, m, token, op.Result)
					cancel()
					if err != nil {
						finish("pending", err)
					} else {
						finish("succeeded", nil)
					}
					return
				case "processing":
					status = "processing"
					retry = op.RetryAfter
				case "failed":
					if op.Error == nil {
						finish("pending", failure(MarketProtocol, nil))
						return
					}
					if !op.Error.Retryable {
						finish("failed", op.Error)
						return
					}
					status = "failed_retryable"
				default:
					finish("pending", failure(MarketProtocol, nil))
					return
				}
			}
		}
		elapsed := int(time.Since(s.CreatedAt).Seconds())
		if elapsed < 0 {
			finish("pending", failure(MarketProtocol, nil))
			return
		}
		raw, _ := json.Marshal(map[string]any{"credential_status": "active", "identity_matches": true, "plugin_allowed": true, "scope_allowed": status == "processing" || submissionScope(m, true), "request_matches": true, "operation_status": status, "retention_expired": time.Since(s.CreatedAt) > 7*24*time.Hour, "feature_available": submissionAllowed(m), "attempts": s.Attempts, "elapsed_seconds": elapsed, "poll_elapsed_seconds": int(time.Since(pollStart).Seconds()), "retry_after_seconds": retry})
		action, policyErr := wire.RecoveryAction(raw)
		if policyErr != nil {
			finish("pending", failure(MarketProtocol, policyErr))
			return
		}
		switch action {
		case "poll_operation":
			s.Stage = "processing"
			if e.saveSubmission(&s, true) != nil {
				return
			}
			if !waitUpload(ctx, time.Duration(max(2, retry))*time.Second) {
				finish("pending", failure(MarketPending, nil))
				return
			}
			continue
		case "retry_same_operation":
			if s.Attempts > 0 {
				n, err := rand.Int(rand.Reader, big.NewInt(int64(1<<min(s.Attempts-1, 4))*1000+1))
				if err != nil {
					finish("pending", failure(StorageFailure, err))
					return
				}
				delay := max(time.Duration(n.Int64())*time.Millisecond, time.Duration(retry)*time.Second)
				if delay >= time.Until(s.CreatedAt.Add(7200*time.Second)) || !waitUpload(ctx, delay) {
					finish("pending", failure(MarketPending, nil))
					return
				}
			}
		case "wait_capability":
			finish("pending", failure(SubmissionCapability, nil))
			return
		case "deny", "reauthorize":
			finish("pending", failure(MarketCredential, nil))
			return
		default:
			finish("pending", failure(MarketPending, nil))
			return
		}
		// Fresh scopes, capability, version state, options and the entire immutable
		// artifact set are checked before every JSON write, including same-key retries.
		fresh, err := e.PrepareSubmission(ctx, s.ConnectionID, s.VersionID)
		if err != nil {
			finish("pending", err)
			return
		}
		if fresh.Version.Channel != "stable" || (fresh.Version.Status != "draft" && fresh.Version.Status != "rejected") {
			finish("pending", failure(SubmissionState, nil))
			return
		}
		if fresh.Version.Version != s.Version || !artifactSetEqual(fresh.Version.Artifacts, s.Artifacts) {
			finish("failed", failure(SubmissionChanged, nil))
			return
		}
		if !slices.Contains(fresh.Options.Versions, s.Metadata.MinCore) || !slices.Contains(fresh.Options.Licenses, s.Metadata.License) {
			finish("failed", invalid("publication_options"))
			return
		}
		m, err = e.GetMarket(s.ConnectionID)
		if err != nil {
			finish("pending", err)
			return
		}
		token, err = e.credentials.Get(ctx, s.ConnectionID)
		if err != nil {
			finish("pending", failure(MarketKeyring, err))
			return
		}
		if m.Identity == nil || m.Identity.User.ID != s.UserID || m.Identity.Developer.DeveloperID != s.DeveloperID {
			finish("pending", failure(MarketIdentity, nil))
			return
		}
		body, _ := json.Marshal(map[string]any{"metadata": s.Metadata, "expected_artifacts": s.Artifacts})
		if _, err = wire.Validate("SubmitRequest", body); err != nil {
			finish("failed", invalid("submission"))
			return
		}
		s.Attempts++
		s.Stage = "sending"
		if e.saveSubmission(&s, true) != nil {
			return
		}
		write, writeCancel := context.WithDeadline(ctx, s.CreatedAt.Add(7200*time.Second))
		request, err := http.NewRequestWithContext(write, http.MethodPost, m.APIRoot+"/publisher/plugins/"+s.PluginID+"/versions/"+s.VersionID+"/submit", bytes.NewReader(body))
		if err != nil {
			writeCancel()
			finish("failed", failure(MarketProtocol, err))
			return
		}
		request.GetBody = nil
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", s.OperationID)
		var receipt SubmissionReceipt
		httpStatus, writeErr := marketRequest(write, m, token, "", "SubmissionReceiptResponse", &receipt, request)
		writeCancel()
		if writeErr == nil && (httpStatus == 200 || httpStatus == 201) {
			read, cancel := context.WithTimeout(ctx, 30*time.Second)
			err = e.acceptSubmission(read, &s, m, token, &receipt)
			cancel()
			if err != nil {
				finish("pending", err)
			} else {
				finish("succeeded", nil)
			}
			return
		}
		wait := 0
		if writeErr != nil {
			if AsError(writeErr).Code == MarketPending {
				finish("pending", writeErr)
				return
			}
			var processing *marketProcessing
			if errors.As(writeErr, &processing) {
				op := processing.Operation
				if op.OperationID != s.OperationID || op.Type != "publication-submit" || op.PluginID != s.PluginID || op.SHA256 != s.RequestSHA256 {
					finish("pending", failure(MarketProtocol, nil))
					return
				}
				wait = max(2, processing.Wait)
			}
			var remote *marketError
			if errors.As(writeErr, &remote) {
				wait = remote.RetryAfter
				if !remote.Retryable {
					if remote.Status == 401 || remote.Status == 403 {
						finish("pending", writeErr)
					} else {
						finish("failed", writeErr)
					}
					return
				}
			}
		}
		pollStart = time.Now()
		if wait > 0 {
			s.Stage = "reconciling"
			var processing *marketProcessing
			if errors.As(writeErr, &processing) {
				s.Stage = "processing"
			}
			if e.saveSubmission(&s, true) != nil {
				return
			}
			if !responseWaitAllowed(s.CreatedAt, pollStart, time.Now(), wait, errors.As(writeErr, &processing)) || !waitUpload(ctx, time.Duration(wait)*time.Second) {
				finish("pending", failure(MarketPending, nil))
				return
			}
		}
		status = "unknown"
		retry = 0
		s.Stage = "reconciling"
		if e.saveSubmission(&s, true) != nil {
			return
		}
	}
}
