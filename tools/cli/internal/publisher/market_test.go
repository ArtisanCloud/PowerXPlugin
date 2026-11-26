//go:build darwin || linux

package publisher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/powerx-plugin/cli/internal/publisher/wire"
)

type memoryCredentials struct {
	mu    sync.Mutex
	items map[string]string
}

func (s *memoryCredentials) Set(_ context.Context, id, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[id] = token
	return nil
}
func (s *memoryCredentials) Get(_ context.Context, id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.items[id]
	if v == "" {
		return "", failure(MarketKeyring, nil)
	}
	return v, nil
}
func (s *memoryCredentials) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, id)
	return nil
}
func example(t *testing.T, name string) map[string]any {
	t.Helper()
	b, err := wire.Bundle.ReadFile("examples.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Cases []struct {
			Name string
			Data map[string]any
		}
	}
	if json.Unmarshal(b, &v) != nil {
		t.Fatal("examples")
	}
	for _, c := range v.Cases {
		if c.Name == name {
			return c.Data
		}
	}
	t.Fatal(name)
	return nil
}
func data(v map[string]any) map[string]any { return v["data"].(map[string]any) }
func writeMarketFixture(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Request-ID", uuid.NewString())
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type marketFixture struct {
	versionUnavailable bool
	retryHeader        int
	queries            int
	mu                 sync.Mutex
	posts              int
	op                 string
	receipt            map[string]any
	fingerprint        string
	lost               bool
	processing         bool
	denyFeature        bool
	wrongIdentity      bool
	corruptReceipt     bool
	seenSHA            string
}

func marketFixtureServer(t *testing.T, f *marketFixture) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("Cookie") != "" {
			t.Error("cookie_forwarded")
		}
		if r.URL.Path != "/publisher/discovery" && r.Header.Get("Authorization") != "Bearer pxpub_"+strings.Repeat("A", 43) {
			writeMarketFixture(w, 401, map[string]any{"code": 401, "error": "invalid_token", "message": "fixture", "retryable": false})
			return
		}
		switch {
		case r.URL.Path == "/publisher/discovery":
			v := example(t, "DiscoveryResponse")
			if f.denyFeature {
				data(v)["features"] = []string{}
			}
			writeMarketFixture(w, 200, v)
		case r.URL.Path == "/publisher/me":
			v := example(t, "MeResponse")
			d := data(v)
			d["credential"].(map[string]any)["plugin_ids"] = []string{"com.example.publisher"}
			d["credential"].(map[string]any)["expires_at"] = time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
			d["server_time"] = time.Now().UTC().Format(time.RFC3339)
			if f.wrongIdentity {
				d["user"].(map[string]any)["user_id"] = uuid.NewString()
			}
			writeMarketFixture(w, 200, v)
		case r.URL.Path == "/publisher/plugins/com.example.publisher":
			v := example(t, "PluginResponse")
			data(v)["plugin_id"] = "com.example.publisher"
			writeMarketFixture(w, 200, v)
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/package"):
			f.posts++
			if f.op != "" && f.op != r.Header.Get("Idempotency-Key") {
				t.Error("operation_changed")
			}
			f.op = r.Header.Get("Idempotency-Key")
			form, err := r.MultipartReader()
			if err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			fields := map[string]string{}
			var sha string
			var size int64
			count := 0
			for {
				part, err := form.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Error(err)
					return
				}
				count++
				if part.FormName() == "file" {
					h := sha256.New()
					size, err = io.Copy(h, part)
					if err != nil {
						t.Error(err)
						return
					}
					sha = hex.EncodeToString(h.Sum(nil))
				} else {
					b, err := io.ReadAll(part)
					if err != nil {
						t.Error(err)
					}
					fields[part.FormName()] = string(b)
				}
				part.Close()
			}
			if count != 8 || len(fields) != 7 || fields["sha256"] != sha || fields["size"] != fmtInt64(size) {
				t.Error("multipart_binding")
			}
			f.seenSHA = sha
			raw, _ := json.Marshal(map[string]any{"plugin_id": "com.example.publisher", "version": fields["version"], "channel": fields["channel"], "os": fields["os"], "arch": fields["arch"], "sha256": sha, "size": size, "release_notes": fields["release_notes"]})
			_, f.fingerprint, err = wire.Fingerprint("package-upload", raw)
			if err != nil {
				t.Error(err)
			}
			v := example(t, "UploadReceiptResponse")
			d := data(v)
			d["operation_id"] = f.op
			d["plugin_id"] = "com.example.publisher"
			a := d["artifact"].(map[string]any)
			a["sha256"] = sha
			a["size"] = size
			f.receipt = d
			if f.corruptReceipt {
				a["sha256"] = strings.Repeat("b", 64)
			}
			if f.lost {
				c, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
				} else {
					c.Close()
				}
				return
			}
			if f.processing {
				v = example(t, "OperationProcessingResponse")
				data(v)["operation_id"] = f.op
				data(v)["plugin_id"] = "com.example.publisher"
				data(v)["request_sha256"] = f.fingerprint
				wait := "2"
				if f.retryHeader != 0 {
					wait = fmtInt(f.retryHeader)
				}
				w.Header().Set("Retry-After", wait)
				writeMarketFixture(w, 202, v)
				return
			}
			writeMarketFixture(w, 201, v)
		case strings.HasPrefix(r.URL.Path, "/publisher/operations/"):
			f.queries++
			if f.receipt == nil {
				writeMarketFixture(w, 404, map[string]any{"code": 404, "error": "operation_not_found", "message": "fixture", "retryable": false})
				return
			}
			v := example(t, "OperationResponse")
			d := data(v)
			d["operation_id"] = f.op
			d["plugin_id"] = "com.example.publisher"
			d["request_sha256"] = f.fingerprint
			d["result"] = f.receipt
			writeMarketFixture(w, 200, v)
		case strings.Contains(r.URL.Path, "/versions/"):
			if f.versionUnavailable {
				writeMarketFixture(w, 503, map[string]any{"code": 503, "error": "storage_unavailable", "message": "fixture", "retryable": true})
				return
			}
			v := example(t, "MarketVersionResponse")
			d := data(v)
			d["plugin_id"] = "com.example.publisher"
			d["artifacts"] = []any{f.receipt["artifact"]}
			writeMarketFixture(w, 200, v)
		default:
			t.Errorf("unexpected_route=%s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
}
func boundFixture(t *testing.T, f *marketFixture) (*Engine, Build, Market) {
	t.Helper()
	_, e, p := buildFixture(t)
	b, err := e.StartBuild(context.Background(), p.ReleasePlanID)
	if err != nil {
		t.Fatal(err)
	}
	b, err = e.WaitBuild(b.BuildID)
	if err != nil || b.Stage != "ready" {
		t.Fatal(b, err)
	}
	server := marketFixtureServer(t, f)
	t.Cleanup(server.Close)
	e.credentials = &memoryCredentials{items: map[string]string{}}
	m, err := e.CreateMarket(MarketInput{Name: "fixture", APIRoot: server.URL, ConsoleBase: "https://market.example"})
	if err != nil {
		t.Fatal(err)
	}
	m, err = e.BindMarket(context.Background(), m.ConnectionID, "pxpub_"+strings.Repeat("A", 43))
	if err != nil {
		t.Fatal(err)
	}
	return e, b, m
}
func TestMarketUploadRecoveryBindings(t *testing.T) {
	for _, scenario := range []string{"success", "lost_response", "processing", "capability_closed", "receipt_mismatch"} {
		t.Run(scenario, func(t *testing.T) {
			f := &marketFixture{lost: scenario == "lost_response", processing: scenario == "processing", denyFeature: scenario == "capability_closed", corruptReceipt: scenario == "receipt_mismatch"}
			e, b, m := boundFixture(t, f)
			u, err := e.StartUpload(context.Background(), UploadInput{ConnectionID: m.ConnectionID, BuildID: b.BuildID})
			if err != nil {
				t.Fatal(err)
			}
			u, err = e.WaitUpload(u.UploadID)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "capability_closed" {
				if f.posts != 0 || u.Stage != "pending" || u.Error.Code != MarketCapability {
					t.Fatal(u, f.posts)
				}
				return
			}
			if scenario == "receipt_mismatch" {
				if u.Stage != "pending" || u.Error.Code != MarketProtocol {
					t.Fatal(u)
				}
				return
			}
			if u.Stage != "succeeded" || u.Receipt == nil || u.CurrentVersion == nil || f.posts != 1 || u.Attempts != 1 || f.seenSHA != b.Artifact.SHA256 {
				t.Fatalf("upload=%+v posts=%d", u, f.posts)
			}
			if _, err = e.StartUpload(context.Background(), u.UploadInput); AsError(err).Code != Conflict {
				t.Fatal("duplicate_upload", err)
			}
			for _, dir := range []string{"markets", "uploads"} {
				files, err := os.ReadDir(filepath.Join(e.state.path, dir))
				if err != nil {
					t.Fatal(err)
				}
				for _, file := range files {
					if file.IsDir() {
						continue
					}
					raw, err := os.ReadFile(filepath.Join(e.state.path, dir, file.Name()))
					if err != nil || strings.Contains(string(raw), "pxpub_") {
						t.Fatal("credential_persisted", err)
					}
				}
			}
		})
	}
}
func TestMarketBindingRejectsIdentityReplacementAndUntrustedURLs(t *testing.T) {
	f := &marketFixture{}
	e, _, m := boundFixture(t, f)
	f.mu.Lock()
	f.wrongIdentity = true
	f.mu.Unlock()
	_, err := e.BindMarket(context.Background(), m.ConnectionID, "pxpub_"+strings.Repeat("A", 43))
	if AsError(err).Code != MarketIdentity {
		t.Fatal(err)
	}
	for _, raw := range []string{"http://market.example", "http://localhost:1234", "https://market.example/a/../b", "https://market.example/%2e%2e", "https://market.example#x", "https://u:p@market.example"} {
		if marketURL(raw) {
			t.Fatal(raw)
		}
	}
}
func TestMarketStrictResponsesAndRedirectCredentialIsolation(t *testing.T) {
	hits := 0
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++; w.WriteHeader(500) }))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, 302) }))
	defer source.Close()
	_, err := marketRequest(context.Background(), Market{MarketInput: MarketInput{APIRoot: source.URL}}, "pxpub_"+strings.Repeat("A", 43), "/publisher/me", "MeResponse", nil, nil)
	if err == nil || hits != 0 {
		t.Fatal(err, hits)
	}
}

func TestMarketRecoveryRetainsOperationAcrossRestartAndBudget(t *testing.T) {
	f := &marketFixture{denyFeature: true}
	e, b, m := boundFixture(t, f)
	u, err := e.StartUpload(context.Background(), UploadInput{ConnectionID: m.ConnectionID, BuildID: b.BuildID})
	if err != nil {
		t.Fatal(err)
	}
	u, err = e.WaitUpload(u.UploadID)
	if err != nil {
		t.Fatal(err)
	}
	original := u.OperationID
	// A reopened engine has no active transport; it must reconcile the saved UUID.
	other, err := Open(e.root)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	other.credentials = e.credentials
	f.mu.Lock()
	f.denyFeature = false
	f.mu.Unlock()
	if _, err = other.ResumeUpload(context.Background(), u.UploadID); err != nil {
		t.Fatal(err)
	}
	u, err = other.WaitUpload(u.UploadID)
	if err != nil || u.Stage != "succeeded" || u.OperationID != original || u.Attempts != 1 {
		t.Fatal(u.Stage, err, u.Error)
	}
	// A pending task with an expired total budget may query, but cannot POST even
	// when the operation query is not found and capabilities are now available.
	u.UploadID = uuid.NewString()
	u.OperationID = uuid.NewString()
	u.CreatedAt = time.Now().UTC().Add(-7201 * time.Second)
	u.Stage = "pending"
	u.Attempts = 0
	u.Receipt = nil
	u.CurrentVersion = nil
	if err = other.saveUpload(&u, false); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.receipt = nil
	before := f.posts
	f.mu.Unlock()
	if _, err = other.ResumeUpload(context.Background(), u.UploadID); err != nil {
		t.Fatal(err)
	}
	u, err = other.WaitUpload(u.UploadID)
	if err != nil || u.Stage != "pending" {
		t.Fatal(err, u.Stage)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts != before {
		t.Fatal("budget_bypassed")
	}
}

func TestMarketExcessRetryAfterNeverTriggersAutomaticQuery(t *testing.T) {
	f := &marketFixture{processing: true, retryHeader: 301}
	e, b, m := boundFixture(t, f)
	u, err := e.StartUpload(context.Background(), UploadInput{ConnectionID: m.ConnectionID, BuildID: b.BuildID})
	if err != nil {
		t.Fatal(err)
	}
	u, err = e.WaitUpload(u.UploadID)
	if err != nil || u.Stage != "pending" || u.Error.Code != MarketPending {
		t.Fatal(err, u.Stage, u.Error)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts != 1 || f.queries != 0 {
		t.Fatal("retry_after_not_honored", f.posts, f.queries)
	}
}

func TestLocalMarketBindingUsesProtectedRouteAndNeverReturnsToken(t *testing.T) {
	e, _, bound := boundFixture(t, &marketFixture{})
	server, err := Start(context.Background(), e, ServerOptions{Listen: "127.0.0.1:0", Origin: testOrigin})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	raw, err := e.state.read("tool-session.json", MaxRecordBytes)
	if err != nil {
		t.Fatal(err)
	}
	var bootstrap Bootstrap
	if json.Unmarshal(raw, &bootstrap) != nil {
		t.Fatal("bootstrap")
	}
	session := exchange(t, server, bootstrap)
	input, _ := json.Marshal(MarketInput{Name: "fixture", APIRoot: bound.APIRoot, ConsoleBase: bound.ConsoleBase})
	create := string(input)
	m := decodeData[Market](t, request(t, server.Info().Endpoint, http.MethodPost, "/v1/markets", session.Token, create), http.StatusCreated)
	if m.ContractVersion != MarketContractVersion || m.BundleSHA256 != MarketBundleSHA256 {
		t.Fatal("contract_provenance")
	}
	read := decodeData[[]Market](t, request(t, server.Info().Endpoint, http.MethodGet, "/v1/markets", session.Token, ""), http.StatusOK)
	if len(read) != 2 {
		t.Fatal("connection_read")
	}
	response := request(t, server.Info().Endpoint, http.MethodPost, "/v1/markets/"+m.ConnectionID+"/bind", session.Token, `{"token":"pxpub_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}`)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatal("protected_token_binding", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	if strings.Contains(string(body), "pxpub_") {
		t.Fatal("binding_response_leaks_token")
	}
	if _, err = e.credentials.Get(context.Background(), m.ConnectionID); err != nil {
		t.Fatal("binding_not_persisted", err)
	}
	response = request(t, server.Info().Endpoint, http.MethodPost, "/v1/markets", session.Token, `{"name":"fixture","api_root":"https://api.example/v1","console_base":"https://market.example","token":"pxpub_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}`)
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatal("token_in_connection", response.StatusCode)
	}
	uploads := decodeData[[]Upload](t, request(t, server.Info().Endpoint, http.MethodGet, "/v1/uploads", session.Token, ""), http.StatusOK)
	if uploads == nil || len(uploads) != 0 {
		t.Fatal("upload_list")
	}
}

func TestMarketSuccessfulRecoveryDoesNotRequireLostLocalArtifact(t *testing.T) {
	f := &marketFixture{}
	e, b, m := boundFixture(t, f)
	u, err := e.StartUpload(context.Background(), UploadInput{ConnectionID: m.ConnectionID, BuildID: b.BuildID})
	if err != nil {
		t.Fatal(err)
	}
	u, err = e.WaitUpload(u.UploadID)
	if err != nil || u.Stage != "succeeded" {
		t.Fatal(err, u.Error)
	}
	if err = os.Remove(filepath.Join(e.work.path, b.BuildID, "artifact.zip")); err != nil {
		t.Fatal(err)
	}
	if _, err = e.ResumeUpload(context.Background(), u.UploadID); err != nil {
		t.Fatal(err)
	}
	restored, err := e.WaitUpload(u.UploadID)
	if err != nil || restored.Stage != "succeeded" || restored.OperationID != u.OperationID || restored.Receipt.VersionID != u.Receipt.VersionID {
		t.Fatal(err, restored.Stage, restored.Error)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts != 1 {
		t.Fatal("recovery_reuploaded")
	}
}

func TestMarketPaginationBounds(t *testing.T) {
	for _, raw := range []string{"page=0", "page=1000001", "size=101", "page=1&page=2", "size=20&unknown=1", "page=01"} {
		if _, _, err := marketPage(raw); err == nil {
			t.Fatal("pagination_accepted", raw)
		}
	}
	page, size, err := marketPage("page=1000000&size=100")
	if err != nil || page != 1000000 || size != 100 {
		t.Fatal("pagination_boundary", err, page, size)
	}
}

func TestMarketRecoveryNeverPresentsAnOldVersionSnapshotAsCurrent(t *testing.T) {
	f := &marketFixture{}
	e, b, m := boundFixture(t, f)
	u, err := e.StartUpload(context.Background(), UploadInput{ConnectionID: m.ConnectionID, BuildID: b.BuildID})
	if err != nil {
		t.Fatal(err)
	}
	u, err = e.WaitUpload(u.UploadID)
	if err != nil || u.Stage != "succeeded" {
		t.Fatal(err, u.Error)
	}
	f.mu.Lock()
	f.versionUnavailable = true
	f.mu.Unlock()
	if _, err = e.ResumeUpload(context.Background(), u.UploadID); err != nil {
		t.Fatal(err)
	}
	u, err = e.WaitUpload(u.UploadID)
	if err != nil || u.Stage != "pending" || u.Receipt == nil || u.CurrentVersion != nil {
		t.Fatal("stale_current_version", err, u.Stage)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts != 1 {
		t.Fatal("unexpected_reupload")
	}
}
