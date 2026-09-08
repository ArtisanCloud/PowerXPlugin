package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestServiceSessionNoRedirectAndIdempotencyIsCallerOwned(t *testing.T) {
	ctx := context.Background()
	t.Run("redirect", func(t *testing.T) {
		calls := 0
		c := sessionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Location", sessionRoot+"/"+testSessionUUID)
			w.WriteHeader(307)
		})
		_, e := c.CreateSession(ctx, CreateSessionInput{AgentUUID: testAgentUUID})
		if e == nil || calls != 1 {
			t.Fatalf("%v %d", e, calls)
		}
	})
	t.Run("idempotency", func(t *testing.T) {
		calls := 0
		var previous string
		c := sessionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			raw, _ := io.ReadAll(r.Body)
			if r.Header.Get("Idempotency-Key") != "caller-key" {
				t.Error("key changed")
			}
			if calls == 1 {
				previous = string(raw)
				w.WriteHeader(202)
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 202, "data": invocationFixture()})
				return
			}
			if previous != string(raw) {
				t.Error("body changed")
			}
			w.WriteHeader(409)
			fmt.Fprint(w, `{"reason_code":"AGENT_SESSION_IDEMPOTENCY_EXPIRED"}`)
		})
		if _, e := c.InvokeSession(ctx, testSessionUUID, testMessageUUID, "caller-key"); e != nil {
			t.Fatal(e)
		}
		_, e := c.InvokeSession(ctx, testSessionUUID, testMessageUUID, "caller-key")
		var target *Error
		if !errors.As(e, &target) || target.ReasonCode != "AGENT_SESSION_IDEMPOTENCY_EXPIRED" || calls != 2 {
			t.Fatal(e, calls)
		}
	})
	t.Run("response_scope", func(t *testing.T) {
		c := sessionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			s := sessionFixture()
			s.SessionUUID = testAgentUUID
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": s})
		})
		if _, e := c.GetSession(ctx, testSessionUUID); e == nil {
			t.Fatal("unexpected resource accepted")
		}
	})
}

func TestServiceSessionTokenFailures(t *testing.T) {
	for _, kind := range []string{"empty", "failure", "cancelled", "typed_nil"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c := sessionTestClient(t, func(http.ResponseWriter, *http.Request) { t.Error("unexpected request") })
			c.tokens = TokenProviderFunc(func(context.Context) (string, error) {
				switch kind {
				case "failure":
					return "", errors.New("test.secret")
				case "cancelled":
					cancel()
					return "", ctx.Err()
				}
				return "", nil
			})
			if kind == "typed_nil" {
				var provider TokenProviderFunc
				c.tokens = provider
			}
			_, e := c.GetSession(ctx, testSessionUUID)
			if e == nil {
				t.Fatal("missing error")
			}
			if kind == "cancelled" && !errors.Is(e, context.Canceled) {
				t.Fatal(e)
			}
			if strings.Contains(e.Error(), "test.secret") {
				t.Fatal("secret leak")
			}
		})
	}
}

const testSessionUUID = "12345678-1234-4234-8234-123456789abc"
const testAgentUUID = "12345678-1234-4234-8234-123456789abd"
const testMessageUUID = "12345678-1234-4234-8234-123456789abe"
const testInvocationUUID = "12345678-1234-4234-8234-123456789abf"

type sessionRoundTrip func(*http.Request) (*http.Response, error)

func (f sessionRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type sessionBrokenBody struct {
	closed bool
	cancel context.CancelFunc
}

func (b *sessionBrokenBody) Read([]byte) (int, error) {
	if b.cancel != nil {
		b.cancel()
	}
	return 0, io.ErrUnexpectedEOF
}
func (b *sessionBrokenBody) Close() error { b.closed = true; return nil }

func TestServiceSessionTransportFailureClosesBody(t *testing.T) {
	for _, kind := range []string{"read", "read_cancel", "network"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			body := &sessionBrokenBody{}
			if kind == "read_cancel" {
				body.cancel = cancel
			}
			c := sessionTestClient(t, func(http.ResponseWriter, *http.Request) { t.Error("unexpected network") })
			calls := 0
			c.http = &http.Client{Transport: sessionRoundTrip(func(*http.Request) (*http.Response, error) {
				calls++
				if kind == "network" {
					return nil, errors.New("test.secret")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: body}, nil
			})}
			_, e := c.GetSession(ctx, testSessionUUID)
			if e == nil || calls != 1 {
				t.Fatalf("%v %d", e, calls)
			}
			if kind != "network" && !body.closed {
				t.Fatal("body not closed")
			}
			if kind == "read_cancel" && !errors.Is(e, context.Canceled) {
				t.Fatal(e)
			}
			if strings.Contains(e.Error(), "test.secret") {
				t.Fatal("secret leak")
			}
		})
	}
}

func sessionFixture() ServiceSession {
	return ServiceSession{SessionUUID: testSessionUUID, AgentUUID: testAgentUUID, Status: "active", Revision: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
}
func invocationFixture() ServiceInvocation {
	return ServiceInvocation{InvocationUUID: testInvocationUUID, SessionUUID: testSessionUUID, MessageUUID: testMessageUUID, TraceUUID: testAgentUUID, Status: "running", CreatedAt: time.Now().UTC(), DeadlineAt: time.Now().UTC().Add(time.Minute)}
}
func sessionTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	c, e := NewClientWithTokenProvider(PowerXAgentClientConfig{BaseURL: s.URL, Mode: ModeDelegated}, TokenProviderFunc(func(context.Context) (string, error) { return "test-sts", nil }))
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func TestServiceSessionOperations(t *testing.T) {
	ctx := context.Background()
	s := sessionFixture()
	m := ServiceMessage{SessionUUID: testSessionUUID, MessageUUID: testMessageUUID, Role: "user", Content: "test.input", Sequence: 1, CreatedAt: time.Now().UTC()}
	i := invocationFixture()
	cases := []struct {
		name, method, path, key string
		status                  int
		output                  any
		call                    func(*Client) error
	}{
		{"create", "POST", sessionRoot, "", 201, s, func(c *Client) error {
			_, e := c.CreateSession(ctx, CreateSessionInput{AgentUUID: testAgentUUID})
			return e
		}},
		{"list", "GET", sessionRoot + "?page=1&page_size=20", "", 200, ServiceSessionList{Items: []ServiceSession{s}, Total: 1, Page: 1, PageSize: 20}, func(c *Client) error { _, e := c.ListSessions(ctx, SessionPage{}); return e }},
		{"get", "GET", sessionRoot + "/" + testSessionUUID, "", 200, s, func(c *Client) error { _, e := c.GetSession(ctx, testSessionUUID); return e }},
		{"rename", "PATCH", sessionRoot + "/" + testSessionUUID, "", 200, s, func(c *Client) error { _, e := c.RenameSession(ctx, testSessionUUID, "test.title"); return e }},
		{"archive", "POST", sessionRoot + "/" + testSessionUUID + "/archive", "", 200, s, func(c *Client) error { _, e := c.ArchiveSession(ctx, testSessionUUID); return e }},
		{"delete", "DELETE", sessionRoot + "/" + testSessionUUID, "", 200, s, func(c *Client) error { _, e := c.DeleteSession(ctx, testSessionUUID); return e }},
		{"append", "POST", sessionRoot + "/" + testSessionUUID + "/messages", "append-key", 201, m, func(c *Client) error {
			_, e := c.AppendSessionMessage(ctx, testSessionUUID, "append-key", AppendSessionMessageInput{Role: "user", Content: "test.input"})
			return e
		}},
		{"messages", "GET", sessionRoot + "/" + testSessionUUID + "/messages?page=1&page_size=20", "", 200, ServiceMessageList{Items: []ServiceMessage{m}, Total: 1, Page: 1, PageSize: 20}, func(c *Client) error { _, e := c.ListSessionMessages(ctx, testSessionUUID, SessionPage{}); return e }},
		{"invoke", "POST", sessionRoot + "/" + testSessionUUID + "/invocations", "invoke-key", 202, i, func(c *Client) error {
			_, e := c.InvokeSession(ctx, testSessionUUID, testMessageUUID, "invoke-key")
			return e
		}},
		{"invocation", "GET", sessionRoot + "/" + testSessionUUID + "/invocations/" + testInvocationUUID, "", 200, i, func(c *Client) error {
			_, e := c.GetSessionInvocation(ctx, testSessionUUID, testInvocationUUID)
			return e
		}},
		{"cancel", "POST", sessionRoot + "/" + testSessionUUID + "/invocations/" + testInvocationUUID + "/cancel", "", 202, i, func(c *Client) error {
			_, e := c.CancelSessionInvocation(ctx, testSessionUUID, testInvocationUUID)
			return e
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := sessionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != tc.method || r.URL.RequestURI() != tc.path || r.Header.Get("Idempotency-Key") != tc.key || r.Header.Get("Authorization") != "Bearer test-sts" {
					t.Errorf("request mismatch: %s %s", r.Method, r.URL.RequestURI())
				}
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				for _, key := range []string{"tenant_uuid", "plugin_id", "service_actor", "user_id"} {
					if _, ok := body[key]; ok {
						t.Errorf("identity override: %s", key)
					}
				}
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(map[string]any{"code": tc.status, "data": tc.output})
			})
			if e := tc.call(c); e != nil {
				t.Fatal(e)
			}
			if calls != 1 {
				t.Fatal(calls)
			}
		})
	}
}
func TestServiceSessionErrorAndRevocation(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 409, 429, 502, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			c := sessionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(status)
				fmt.Fprint(w, `{"error_code":"AGENT_SESSION_FORBIDDEN","reason_code":"AGENT_SESSION_FORBIDDEN"}`)
			})
			_, e := c.GetSession(context.Background(), testSessionUUID)
			var target *Error
			if !errors.As(e, &target) || target.StatusCode != status || target.ReasonCode != "AGENT_SESSION_FORBIDDEN" || calls != 1 {
				t.Fatalf("%v calls=%d", e, calls)
			}
		})
	}
	calls := 0
	c := sessionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": sessionFixture()})
			return
		}
		w.WriteHeader(403)
		fmt.Fprint(w, `{"reason_code":"AGENT_SESSION_FORBIDDEN"}`)
	})
	if _, e := c.GetSession(context.Background(), testSessionUUID); e != nil {
		t.Fatal(e)
	}
	if _, e := c.GetSession(context.Background(), testSessionUUID); e == nil {
		t.Fatal("cached authorization")
	}
}
func TestServiceSessionRejectsInvalidInputsAndOldDelegatedEntrypoints(t *testing.T) {
	c := sessionTestClient(t, func(http.ResponseWriter, *http.Request) { t.Error("unexpected request") })
	ctx := context.Background()
	checks := []func() error{
		func() error { _, e := c.GetSession(ctx, "123"); return e },
		func() error { _, e := c.CreateSession(ctx, CreateSessionInput{AgentUUID: "1"}); return e },
		func() error {
			_, e := c.AppendSessionMessage(ctx, testSessionUUID, "key", AppendSessionMessageInput{Role: "assistant", Content: "test.input"})
			return e
		},
		func() error { _, e := c.InvokeSession(ctx, testSessionUUID, testMessageUUID, ""); return e },
		func() error { _, e := c.ListSessions(ctx, SessionPage{Page: -1}); return e },
		func() error { _, e := c.Invoke(ctx, AgentInvokeRequest{}); return e },
		func() error { return c.StreamSSE(ctx, nil, func(AgentStreamEvent) error { return nil }) },
	}
	for _, check := range checks {
		if check() == nil {
			t.Fatal("expected rejection")
		}
	}
	c.cfg.Mode = ModeStandalone
	if _, e := c.GetSession(ctx, testSessionUUID); e == nil {
		t.Fatal("standalone credential accepted")
	}
}
func TestServiceSessionMalformedSuccess(t *testing.T) {
	for _, body := range []string{`{}`, `{"code":200,"data":null}`, `{"code":200,"data":{}}`, `{"code":0,"data":{}}`, `not-json`} {
		c := sessionTestClient(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) })
		_, e := c.GetSession(context.Background(), testSessionUUID)
		var target *Error
		if !errors.As(e, &target) || target.StatusCode != 502 {
			t.Fatalf("%s: %v", body, e)
		}
	}
}
func TestServiceSessionEvents(t *testing.T) {
	run := invocationFixture()
	run.Status = "succeeded"
	raw, _ := json.Marshal(run)
	good := "event: state\ndata: " + string(raw) + "\n\nevent: final\ndata: " + string(raw) + "\n\nevent: end\ndata: {\"status\":\"succeeded\"}\n\n"
	for _, tc := range []struct {
		name, stream string
		wantErr      bool
	}{
		{"success", good, false},
		{"missing_end", strings.Split(good, "event: end")[0], true},
		{"missing_final", "event: end\ndata: {\"status\":\"succeeded\"}\n\n", true},
		{"failure", "event: error\ndata: {\"reason_code\":\"AGENT_SESSION_UPSTREAM_DEPENDENCY\"}\n\nevent: end\ndata: {\"status\":\"failed\"}\n\n", true},
		{"wrong_subject", strings.ReplaceAll(good, testSessionUUID, testAgentUUID), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := sessionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || !strings.HasSuffix(r.URL.Path, "/events") {
					t.Error(r.URL.Path)
				}
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, tc.stream)
			})
			e := c.StreamSessionEvents(context.Background(), testSessionUUID, testInvocationUUID, func(SessionEvent) error { return nil })
			if (e != nil) != tc.wantErr {
				t.Fatal(e)
			}
		})
	}
	sentinel := errors.New("test.callback")
	if e := consumeSessionEvents(strings.NewReader(good), testSessionUUID, testInvocationUUID, func(SessionEvent) error { return sentinel }); e != sentinel {
		t.Fatal(e)
	}
}
func TestServiceSessionSubscriptionCancellationDoesNotInvokeCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	c := sessionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if !strings.HasSuffix(r.URL.Path, "/events") {
			t.Error(r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		cancel()
		<-r.Context().Done()
	})
	e := c.StreamSessionEvents(ctx, testSessionUUID, testInvocationUUID, func(SessionEvent) error { return nil })
	if !errors.Is(e, context.Canceled) || calls != 1 {
		t.Fatalf("%v calls=%d", e, calls)
	}
}
