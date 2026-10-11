package pluginrelease

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientStartsInstallSessionWithSTS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tenant/plugin-release/install-sessions" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer service-sts" {
			t.Fatalf("Authorization = %q", got)
		}
		var in StartInstallSessionInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Fatal(err)
		}
		if in.ArtifactURI != "pxp://package-uuid" {
			t.Fatalf("input = %#v", in)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"session_uuid":"session-uuid","plugin_id":"example","service_actor":"plugin:example","status":"queued"}}`))
	}))
	defer server.Close()

	client, err := NewClientWithTokenProvider(Config{BaseURL: server.URL}, TokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	session, err := client.StartInstallSession(context.Background(), StartInstallSessionInput{ArtifactURI: "pxp://package-uuid"})
	if err != nil {
		t.Fatal(err)
	}
	if session.SessionUUID != "session-uuid" || session.ServiceActor != "plugin:example" {
		t.Fatalf("session = %#v", session)
	}
}

func TestClientPreservesCoreConflictReason(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"reason_code":"PLUGIN_RELEASE_ACTIVE_SESSION_CONFLICT"}}`))
	}))
	defer server.Close()
	client, err := NewClientWithTokenProvider(Config{BaseURL: server.URL}, TokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.StartInstallSession(context.Background(), StartInstallSessionInput{ArtifactURI: "pxp://package-uuid"})
	var hostErr *HTTPError
	if !errors.As(err, &hostErr) || hostErr.StatusCode != http.StatusConflict || hostErr.ReasonCode != "PLUGIN_RELEASE_ACTIVE_SESSION_CONFLICT" {
		t.Fatalf("error = %#v", err)
	}
}
