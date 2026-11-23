package publisher

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/powerx-plugin/cli/internal/publisher/wire"
)

type reviewFixture struct {
	corruptArtifact                                                                        bool
	mu                                                                                     sync.Mutex
	closed, noWrite, missingArtifact, lost, processing, changedOptions, versionUnavailable bool
	fixedPublicationID                                                                     string
	versionStatus, publicationStatus                                                       string
	processingWait                                                                         int
	posts, operationQueries                                                                int
	operation, fingerprint                                                                 string
	input                                                                                  SubmissionInput
	receipt                                                                                map[string]any
}

func reviewInput() SubmissionInput {
	return SubmissionInput{VersionID: "00000000-0000-4000-8000-000000000003", Version: "1.2.0", Metadata: PublicationMetadata{Description: "  description with enough characters  ", License: "MIT", MinCore: "1.0.0", SupportURL: "https://support.example/help", Security: "security statement for fixture", Privacy: "privacy statement for fixture"}, Artifacts: []MarketArtifact{{ArtifactID: "00000000-0000-4000-8000-000000000004", OS: "linux", Arch: "amd64", SHA256: strings.Repeat("a", 64), Size: 10}, {ArtifactID: "00000000-0000-4000-8000-000000000005", OS: "darwin", Arch: "arm64", SHA256: strings.Repeat("b", 64), Size: 20}}}
}
func reviewServer(t *testing.T, f *reviewFixture) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch {
		case r.URL.Path == "/publisher/discovery":
			v := example(t, "DiscoveryResponse")
			if !f.closed {
				data(v)["features"] = []string{"package-upload", "idempotent-writes", "publication-submit"}
				data(v)["supported_scopes"] = []string{"plugins:read", "packages:write", "publications:read", "publications:submit"}
			} else {
				data(v)["features"] = []string{}
			}
			writeMarketFixture(w, 200, v)
		case r.URL.Path == "/publisher/me":
			v := example(t, "MeResponse")
			d := data(v)
			d["credential"].(map[string]any)["plugin_ids"] = []string{"com.example.publisher"}
			scopes := []string{"plugins:read", "publications:read"}
			if !f.noWrite {
				scopes = append(scopes, "publications:submit")
			}
			d["credential"].(map[string]any)["scopes"] = scopes
			d["credential"].(map[string]any)["expires_at"] = time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
			d["server_time"] = time.Now().UTC().Format(time.RFC3339)
			writeMarketFixture(w, 200, v)
		case r.URL.Path == "/publisher/plugins/com.example.publisher":
			v := example(t, "PluginResponse")
			data(v)["plugin_id"] = "com.example.publisher"
			writeMarketFixture(w, 200, v)
		case r.URL.Path == "/publisher/publication-options":
			v := example(t, "PublicationOptionsResponse")
			if f.changedOptions {
				data(v)["licenses"] = []string{"Apache-2.0"}
			}
			writeMarketFixture(w, 200, v)
		case strings.HasSuffix(r.URL.Path, "/submit"):
			f.posts++
			if f.operation != "" && f.operation != r.Header.Get("Idempotency-Key") {
				t.Error("operation_replaced")
			}
			f.operation = r.Header.Get("Idempotency-Key")
			if r.Header.Get("Content-Type") != "application/json" {
				t.Error("json_media_type")
			}
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				return
			}
			value, err := wire.Validate("SubmitRequest", raw)
			if err != nil {
				t.Error(err)
				return
			}
			v := value.(map[string]any)
			fingerprintRaw, _ := json.Marshal(map[string]any{"plugin_id": "com.example.publisher", "version_id": f.input.VersionID, "metadata": v["metadata"], "expected_artifacts": v["expected_artifacts"]})
			_, f.fingerprint, err = wire.Fingerprint("publication-submit", fingerprintRaw)
			if err != nil {
				t.Error(err)
			}
			if len(v["expected_artifacts"].([]any)) != 2 {
				t.Error("partial_artifact_set_sent")
			}
			if f.corruptArtifact {
				writeMarketFixture(w, 409, map[string]any{"code": 409, "error": "artifact_changed", "message": "fixture", "retryable": false})
				return
			}
			receipt := example(t, "SubmissionReceiptResponse")
			data(receipt)["operation_id"] = f.operation
			data(receipt)["publication_id"] = uuid.NewString()
			if f.fixedPublicationID != "" {
				data(receipt)["publication_id"] = f.fixedPublicationID
			}
			data(receipt)["market_publication_url"] = "https://market.example/developer/plugins/com.example.publisher?publication_id=" + data(receipt)["publication_id"].(string)
			f.receipt = data(receipt)
			if f.lost {
				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
				} else {
					conn.Close()
				}
				return
			}
			if f.processing {
				processing := example(t, "OperationProcessingResponse")
				d := data(processing)
				d["operation_id"] = f.operation
				d["type"] = "publication-submit"
				d["plugin_id"] = "com.example.publisher"
				d["request_sha256"] = f.fingerprint
				wait := 2
				if f.processingWait > 0 {
					wait = f.processingWait
				}
				d["retry_after_seconds"] = wait
				w.Header().Set("Retry-After", fmtInt(wait))
				writeMarketFixture(w, 202, processing)
				return
			}
			writeMarketFixture(w, 201, receipt)
		case strings.HasPrefix(r.URL.Path, "/publisher/operations/"):
			f.operationQueries++
			if f.receipt == nil {
				writeMarketFixture(w, 404, map[string]any{"code": 404, "error": "operation_not_found", "message": "fixture", "retryable": false})
				return
			}
			v := example(t, "OperationResponse")
			d := data(v)
			d["operation_id"] = f.operation
			d["type"] = "publication-submit"
			d["plugin_id"] = "com.example.publisher"
			d["request_sha256"] = f.fingerprint
			d["result"] = f.receipt
			writeMarketFixture(w, 200, v)
		case strings.HasSuffix(r.URL.Path, "/publications"):
			if f.versionUnavailable {
				writeMarketFixture(w, 503, map[string]any{"code": 503, "error": "storage_unavailable", "message": "fixture", "retryable": true})
				return
			}
			v := example(t, "PublicationsPageResponse")
			p := data(v)["items"].([]any)[0].(map[string]any)
			p["plugin_id"] = "com.example.publisher"
			if f.receipt != nil {
				p["publication_id"] = f.receipt["publication_id"]
				p["market_publication_url"] = f.receipt["market_publication_url"]
			}
			p["version_id"] = f.input.VersionID
			p["artifacts"] = f.input.Artifacts
			p["metadata"] = f.input.Metadata
			status := f.publicationStatus
			if status == "" {
				status = "submitted"
			}
			p["status"] = status
			if status == "rejected" {
				p["reviewed_at"] = time.Now().UTC().Format(time.RFC3339)
				p["review_reason"] = "fixture_reason"
			}
			writeMarketFixture(w, 200, v)
		case strings.Contains(r.URL.Path, "/versions/"):
			v := example(t, "MarketVersionResponse")
			d := data(v)
			d["plugin_id"] = "com.example.publisher"
			d["artifacts"] = f.input.Artifacts
			status := f.versionStatus
			if status == "" {
				status = "draft"
			}
			d["availability_status"] = status
			if f.missingArtifact {
				d["artifacts"] = f.input.Artifacts[:1]
			}
			writeMarketFixture(w, 200, v)
		default:
			t.Errorf("unexpected_path=%s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
}
func reviewEngine(t *testing.T, f *reviewFixture) (*Engine, SubmissionInput) {
	t.Helper()
	f.input = reviewInput()
	server := reviewServer(t, f)
	t.Cleanup(server.Close)
	e := openTest(t, workspace(t))
	e.credentials = &memoryCredentials{items: map[string]string{}}
	m, err := e.CreateMarket(MarketInput{Name: "fixture", APIRoot: server.URL, ConsoleBase: "https://market.example"})
	if err != nil {
		t.Fatal(err)
	}
	m, err = e.BindMarket(context.Background(), m.ConnectionID, "pxpub_"+strings.Repeat("A", 43))
	if err != nil {
		t.Fatal(err)
	}
	input := reviewInput()
	input.ConnectionID = m.ConnectionID
	return e, input
}
func TestSubmissionPreparationReadsReviewedVersions(t *testing.T) {
	for _, status := range []string{"draft", "submitted", "rejected", "approved", "published"} {
		t.Run(status, func(t *testing.T) {
			f := &reviewFixture{versionStatus: status}
			e, input := reviewEngine(t, f)
			prepared, err := e.PrepareSubmission(context.Background(), input.ConnectionID, input.VersionID)
			if err != nil || prepared.Version.Status != status || !artifactSetEqual(prepared.Version.Artifacts, input.Artifacts) || f.posts != 0 {
				t.Fatal("read_only_preparation", status, err, f.posts)
			}
		})
	}
}

func TestSubmissionWholeCollectionAndOriginalOperation(t *testing.T) {
	for _, scenario := range []string{"success", "lost", "processing", "processing_over_budget", "closed", "readonly", "changed_set", "changed_options", "submitted_version", "approved_version", "published_version", "corrupt_second"} {
		t.Run(scenario, func(t *testing.T) {
			f := &reviewFixture{corruptArtifact: scenario == "corrupt_second", lost: scenario == "lost", processing: scenario == "processing", closed: scenario == "closed", noWrite: scenario == "readonly", missingArtifact: scenario == "changed_set", changedOptions: scenario == "changed_options"}
			if scenario == "processing_over_budget" {
				f.processing = true
				f.processingWait = 120
			}
			if strings.HasSuffix(scenario, "_version") {
				f.versionStatus = strings.TrimSuffix(scenario, "_version")
			}
			e, input := reviewEngine(t, f)
			s, err := e.StartSubmission(context.Background(), input)
			if err != nil {
				t.Fatal(err)
			}
			s, err = e.WaitSubmission(s.SubmissionID)
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "closed", "readonly", "submitted_version", "approved_version", "published_version":
				if s.Stage != "pending" || f.posts != 0 {
					t.Fatal(s.Stage, s.Error, f.posts)
				}
				if strings.HasSuffix(scenario, "_version") && (s.Error == nil || s.Error.Code != SubmissionState) {
					t.Fatal("write_state_gate", s.Error)
				}
			case "processing_over_budget":
				if s.Stage != "pending" || f.posts != 1 || f.operationQueries != 0 || s.Receipt != nil {
					t.Fatal("processing_wait_exceeded_poll_budget", s.Stage, f.posts, f.operationQueries)
				}
			case "corrupt_second":
				if s.Stage != "failed" || f.posts != 1 || s.Receipt != nil || s.Error.Params["reason"] != "artifact_changed" {
					t.Fatal(s.Stage, s.Error, f.posts)
				}
			case "changed_set", "changed_options":
				if s.Stage != "failed" || f.posts != 0 {
					t.Fatal(s.Stage, s.Error, f.posts)
				}
			default:
				if s.Stage != "succeeded" || s.Receipt == nil || s.CurrentPublication == nil || f.posts != 1 || s.Attempts != 1 {
					t.Fatal(s.Stage, s.Error, f.posts)
				}
				if strings.HasPrefix(s.Metadata.Description, " ") {
					t.Fatal("metadata_not_normalized")
				}
				if _, err = e.StartSubmission(context.Background(), input); AsError(err).Code != Conflict {
					t.Fatal("duplicate", err)
				}
				operation := s.OperationID
				f.mu.Lock()
				f.closed = true
				f.noWrite = true
				f.mu.Unlock()
				if _, err = e.ResumeSubmission(context.Background(), s.SubmissionID); err != nil {
					t.Fatal(err)
				}
				s, err = e.WaitSubmission(s.SubmissionID)
				if err != nil || s.Stage != "succeeded" || s.OperationID != operation || f.posts != 1 {
					t.Fatal(s.Stage, s.Error, err)
				}
			}
		})
	}
}
func TestRejectedSubmissionCreatesNewOperationAndPreservesSnapshot(t *testing.T) {
	f := &reviewFixture{}
	e, input := reviewEngine(t, f)
	s, err := e.StartSubmission(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	s, _ = e.WaitSubmission(s.SubmissionID)
	f.mu.Lock()
	f.publicationStatus = "rejected"
	f.versionStatus = "rejected"
	f.mu.Unlock()
	if _, err = e.ResumeSubmission(context.Background(), s.SubmissionID); err != nil {
		t.Fatal(err)
	}
	s, _ = e.WaitSubmission(s.SubmissionID)
	if s.CurrentPublication == nil || s.CurrentPublication.Status != "rejected" {
		t.Fatal(s.Error)
	}
	oldOperation := s.OperationID
	input.Metadata.Description = "revised description for resubmission"
	f.mu.Lock()
	f.operation = ""
	f.input.Metadata = input.Metadata
	f.mu.Unlock()
	next, err := e.StartSubmission(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	next, _ = e.WaitSubmission(next.SubmissionID)
	if next.Stage != "succeeded" || next.OperationID == oldOperation || next.Receipt == nil || next.Receipt.PublicationID == s.Receipt.PublicationID {
		t.Fatal(next.Stage, next.Error)
	}
	old, _ := e.GetSubmission(s.SubmissionID)
	if old.Metadata.Description == next.Metadata.Description || old.OperationID != oldOperation {
		t.Fatal("old_snapshot_mutated")
	}
}
func TestSubmissionRequestValidationAndSetOrdering(t *testing.T) {
	input := reviewInput()
	input.ConnectionID = uuid.NewString()
	a, err := normalizeSubmission("com.example.publisher", input)
	if err != nil {
		t.Fatal(err)
	}
	input.Artifacts[0], input.Artifacts[1] = input.Artifacts[1], input.Artifacts[0]
	b, err := normalizeSubmission("com.example.publisher", input)
	if err != nil || a.RequestSHA256 != b.RequestSHA256 || !artifactSetEqual(a.Input.Artifacts, input.Artifacts) {
		t.Fatal("set_order", err)
	}
	input.Artifacts = append(input.Artifacts, input.Artifacts[0])
	if _, err = normalizeSubmission("com.example.publisher", input); err == nil {
		t.Fatal("duplicate_artifact_accepted")
	}
}

func TestSubmissionRawJSONBoundary(t *testing.T) {
	for _, raw := range [][]byte{[]byte("\xef\xbb\xbf{}"), []byte(`{"metadata":{"description":"\ud800"}}`), []byte(`{"connection_id":"x","connection_id":"y"}`), []byte(strings.Repeat(" ", MaxRequestBytes+1))} {
		if _, err := DecodeSubmission(raw); err == nil {
			t.Fatal("invalid_raw_json_accepted")
		}
	}
}

func TestOperationResponseWaitBudget(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name                string
		total, poll, wait   int
		processing, allowed bool
	}{
		{"processing_first_120", 0, 0, 120, true, false},
		{"processing_remaining_one", 0, 119, 2, true, false},
		{"processing_total_one", 7199, 0, 2, true, false},
		{"processing_retry_after_301", 0, 0, 301, true, false},
		{"processing_allowed", 0, 0, 2, true, true},
		{"retryable_300", 0, 0, 300, false, true},
		{"retryable_total_one", 7199, 0, 2, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := responseWaitAllowed(now.Add(-time.Duration(tc.total)*time.Second), now.Add(-time.Duration(tc.poll)*time.Second), now, tc.wait, tc.processing); got != tc.allowed {
				t.Fatal("response_wait_budget", got)
			}
		})
	}
}

func TestRejectedResubmissionRejectsReusedPublicationUUID(t *testing.T) {
	f := &reviewFixture{}
	e, input := reviewEngine(t, f)
	old, err := e.StartSubmission(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	old, _ = e.WaitSubmission(old.SubmissionID)
	f.mu.Lock()
	f.publicationStatus = "rejected"
	f.versionStatus = "rejected"
	f.fixedPublicationID = old.Receipt.PublicationID
	f.mu.Unlock()
	if _, err = e.ResumeSubmission(context.Background(), old.SubmissionID); err != nil {
		t.Fatal(err)
	}
	old, _ = e.WaitSubmission(old.SubmissionID)
	f.mu.Lock()
	f.operation = ""
	f.mu.Unlock()
	next, err := e.StartSubmission(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	next, _ = e.WaitSubmission(next.SubmissionID)
	if next.Stage != "pending" || next.Receipt != nil || next.Error == nil || next.Error.Code != MarketProtocol {
		t.Fatal("reused_publication_accepted", next.Stage, next.Error)
	}
	original, _ := e.GetSubmission(old.SubmissionID)
	if original.Receipt.PublicationID != old.Receipt.PublicationID || original.OperationID != old.OperationID {
		t.Fatal("old_receipt_overwritten")
	}
}
