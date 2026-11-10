//go:build darwin || linux

package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testOrigin = "http://localhost:3039"

func request(t *testing.T, base, method, route, token, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, base+route, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", testOrigin)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}
func decodeData[T any](t *testing.T, r *http.Response, want int) T {
	t.Helper()
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	if r.StatusCode != want {
		t.Fatalf("status=%d want=%d data=%s", r.StatusCode, want, data)
	}
	var envelope struct {
		Data T `json:"data"`
	}
	if err = json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}
func startTest(t *testing.T) (string, *Engine, *Server, Bootstrap) {
	t.Helper()
	root := workspace(t)
	engine := openTest(t, root)
	server, err := Start(context.Background(), engine, ServerOptions{Listen: "127.0.0.1:0", Origin: testOrigin})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Error(err)
		}
	})
	data, err := os.ReadFile(filepath.Join(root, SessionFile))
	if err != nil {
		t.Fatal(err)
	}
	var bootstrap Bootstrap
	if err = json.Unmarshal(data, &bootstrap); err != nil {
		t.Fatal(err)
	}
	return root, engine, server, bootstrap
}
func exchange(t *testing.T, server *Server, bootstrap Bootstrap) Session {
	t.Helper()
	return decodeData[Session](t, request(t, server.Info().Endpoint, http.MethodPost, "/v1/session", bootstrap.BootstrapToken, ""), http.StatusCreated)
}

func TestLocalServiceHandshakePlansAndExplicitCapabilities(t *testing.T) {
	root, engine, server, bootstrap := startTest(t)
	info := server.Info()
	encoded, _ := json.Marshal(info)
	if bytes.Contains(encoded, []byte(bootstrap.BootstrapToken)) || bytes.Contains(encoded, []byte(root)) {
		t.Fatal("startup_secret_or_path_leaked")
	}
	mode, err := os.Stat(filepath.Join(root, SessionFile))
	if err != nil || mode.Mode().Perm() != 0600 {
		t.Fatalf("bootstrap_mode=%v %v", mode, err)
	}
	session := exchange(t, server, bootstrap)
	if session.Token == "" || !session.ExpiresAt.After(time.Now()) {
		t.Fatal("invalid_session")
	}
	r := request(t, info.Endpoint, http.MethodPost, "/v1/session", bootstrap.BootstrapToken, "")
	if r.StatusCode != http.StatusUnauthorized {
		t.Fatal("bootstrap_replayed")
	}
	capabilities := decodeData[LocalInfo](t, request(t, info.Endpoint, http.MethodGet, "/v1/info", session.Token, ""), http.StatusOK)
	if capabilities.Protocol != Protocol || len(capabilities.Features) != 2 || capabilities.Features[0] != "release-plans" || capabilities.BuildTargets == nil || len(capabilities.BuildTargets) != 0 {
		t.Fatalf("capabilities=%+v", capabilities)
	}
	data, _ := json.Marshal(input())
	p := decodeData[Plan](t, request(t, info.Endpoint, http.MethodPost, "/v1/plans", session.Token, string(data)), http.StatusCreated)
	stored, err := engine.Get(p.ReleasePlanID)
	if err != nil || stored != p {
		t.Fatalf("shared_engine=%+v %v", stored, err)
	}
	got := decodeData[Plan](t, request(t, info.Endpoint, http.MethodGet, "/v1/plans/"+p.ReleasePlanID, session.Token, ""), http.StatusOK)
	if got != p {
		t.Fatal("get_mismatch")
	}
	page := decodeData[PlanPage](t, request(t, info.Endpoint, http.MethodGet, "/v1/plans?page=1&size=1", session.Token, ""), http.StatusOK)
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("page=%+v", page)
	}
	for _, route := range []string{"/v1/build", "/v1/upload", "/v1/submit", "/internal/plugins/releases"} {
		r := request(t, info.Endpoint, http.MethodPost, route, session.Token, "{}")
		if r.StatusCode != http.StatusNotFound {
			t.Fatalf("unimplemented_route=%s status=%d", route, r.StatusCode)
		}
	}
}
func TestLocalServiceRejectsForeignHostsOriginsAndCredentials(t *testing.T) {
	_, _, server, bootstrap := startTest(t)
	session := exchange(t, server, bootstrap)
	for name, mutate := range map[string]func(*http.Request){
		"host":             func(r *http.Request) { r.Host = "attacker.example" },
		"origin":           func(r *http.Request) { r.Header.Set("Origin", "http://attacker.example") },
		"missing_origin":   func(r *http.Request) { r.Header.Del("Origin") },
		"null_origin":      func(r *http.Request) { r.Header.Set("Origin", "null") },
		"duplicate_origin": func(r *http.Request) { r.Header.Add("Origin", testOrigin) },
		"missing_token":    func(r *http.Request) { r.Header.Del("Authorization") },
		"duplicate_token":  func(r *http.Request) { r.Header.Add("Authorization", "Bearer "+session.Token) },
		"cookie": func(r *http.Request) {
			r.Header.Del("Authorization")
			r.AddCookie(&http.Cookie{Name: "session", Value: session.Token})
		},
		"bootstrap_as_session": func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+bootstrap.BootstrapToken) },
	} {
		t.Run(name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, server.Info().Endpoint+"/v1/info", nil)
			req.Header.Set("Origin", testOrigin)
			req.Header.Set("Authorization", "Bearer "+session.Token)
			mutate(req)
			resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("guard=%s status=%d", name, resp.StatusCode)
			}
			var envelope struct {
				Error Error `json:"error"`
			}
			if err = json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Error.MessageKey != envelope.Error.Code {
				t.Fatal("unstructured_error")
			}
		})
	}
}
func TestLocalServiceRejectsAmbiguousRequests(t *testing.T) {
	_, _, server, bootstrap := startTest(t)
	session := exchange(t, server, bootstrap)
	data, _ := json.Marshal(input())
	for name, body := range map[string]string{
		"unknown":   strings.Replace(string(data), "{", "{\"command\":\"id\",", 1),
		"duplicate": strings.Replace(string(data), "{", "{\"version\":\"2.0.0\",", 1),
		"trailing":  string(data) + "{}",
	} {
		t.Run(name, func(t *testing.T) {
			r := request(t, server.Info().Endpoint, http.MethodPost, "/v1/plans", session.Token, body)
			if r.StatusCode != http.StatusBadRequest {
				t.Fatalf("accepted=%s status=%d", name, r.StatusCode)
			}
		})
	}
	for _, query := range []string{"?size=101", "?size=0", "?page=0", "?page=1&page=2", "?command=id", "?size=no"} {
		r := request(t, server.Info().Endpoint, http.MethodGet, "/v1/plans"+query, session.Token, "")
		if r.StatusCode != http.StatusBadRequest {
			t.Fatalf("accepted_query=%s", query)
		}
	}
	r := request(t, server.Info().Endpoint, http.MethodPost, "/v1/plans", session.Token, strings.Repeat("x", MaxRequestBytes+1))
	if r.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize=%d", r.StatusCode)
	}
	r = request(t, server.Info().Endpoint, http.MethodPost, "/v1/plans", session.Token, "")
	if r.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("media_type=%d", r.StatusCode)
	}
}
func TestSessionAndBootstrapExpiryUseIndependentCredentials(t *testing.T) {
	e := openTest(t, workspace(t))
	now := time.Now()
	boot, _ := secret()
	session, _ := secret()
	h := &localHandler{engine: e, host: "127.0.0.1:3000", origin: testOrigin, bootstrapExpires: now.Add(BootstrapTTL), now: func() time.Time { return now }}
	h.bootstrapHash = sha256ForTest(boot)
	h.sessionHash = sha256ForTest(session)
	h.sessionExpires = now.Add(SessionTTL)
	call := func(method, route, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:3000"+route, nil)
		r.Header.Set("Origin", testOrigin)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := call(http.MethodGet, "/v1/info", session); w.Code != 200 {
		t.Fatal(w.Code)
	}
	now = now.Add(SessionTTL)
	if w := call(http.MethodGet, "/v1/info", session); w.Code != 401 {
		t.Fatal("expired_session_accepted")
	}
	if w := call(http.MethodPost, "/v1/session", boot); w.Code != 401 {
		t.Fatal("expired_bootstrap_accepted")
	}
}
func sha256ForTest(token string) [32]byte {
	var hash [32]byte
	copy(hash[:], hashToken(token))
	return hash
}
func TestServiceShutdownReleasesPortSessionAndWorkspaceLease(t *testing.T) {
	root := workspace(t)
	engine := openTest(t, root)
	ctx, cancel := context.WithCancel(context.Background())
	server, err := Start(ctx, engine, ServerOptions{Listen: "127.0.0.1:0", Origin: testOrigin})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	other := openTest(t, root)
	_, err = Start(context.Background(), other, ServerOptions{Listen: "127.0.0.1:0", Origin: testOrigin})
	assertCode(t, err, Conflict)
	endpoint, _ := url.Parse(server.Info().Endpoint)
	cancel()
	if err = server.Wait(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, SessionFile)); !os.IsNotExist(err) {
		t.Fatalf("bootstrap_not_removed=%v", err)
	}
	listener, err := net.Listen("tcp", endpoint.Host)
	if err != nil {
		t.Fatalf("port_not_released=%v", err)
	}
	_ = listener.Close()
	restarted, err := Start(context.Background(), other, ServerOptions{Listen: endpoint.Host, Origin: testOrigin})
	if err != nil {
		t.Fatal(err)
	}
	if err = restarted.Close(); err != nil {
		t.Fatal(err)
	}
}
func TestEndpointConfigurationDoesNotAllowPublicBindings(t *testing.T) {
	for _, listen := range []string{"0.0.0.0:3000", ":3000", "localhost:3000", "192.0.2.1:3000", "127.0.0.1:-1", "127.0.0.1:65536"} {
		assertCode(t, validateEndpoint(listen, testOrigin), InvalidInput)
	}
	for _, origin := range []string{"http://attacker.example", "http://localhost:3039/", "http://localhost:3039?", "http://localhost:", "http://user@localhost:3039", "null", "*", "http://localhost:3039?token=x"} {
		assertCode(t, validateEndpoint("127.0.0.1:0", origin), InvalidInput)
	}
	for _, origin := range []string{testOrigin, "https://127.0.0.1", "http://[::1]:3039"} {
		if err := validateEndpoint("127.0.0.1:0", origin); err != nil {
			t.Fatal(err)
		}
	}
}
