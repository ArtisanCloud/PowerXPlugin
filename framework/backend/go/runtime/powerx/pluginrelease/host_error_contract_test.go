package pluginrelease

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostErrorContractMatrix(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 409, 422, 429, 502, 503} {
		for _, shape := range []string{"reason_code", "error_code"} {
			t.Run(fmt.Sprintf("%d/%s", status, shape), func(t *testing.T) {
				reason := "PLUGIN_RELEASE_CONTRACT_TEST"
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "Bearer sts" {
						t.Error("authorization mismatch")
					}
					w.WriteHeader(status)
					fmt.Fprintf(w, "{\"%s\":\"%s\"}", shape, reason)
				}))
				defer server.Close()
				client, err := NewClientWithTokenProvider(Config{BaseURL: server.URL}, TokenProviderFunc(func(context.Context) (string, error) { return "sts", nil }), server.Client())
				if err != nil {
					t.Fatal(err)
				}
				_, err = client.GetInstallSession(context.Background(), "00000000-0000-4000-8000-000000000001")
				var e *HTTPError
				if !errors.As(err, &e) || e.StatusCode != status || e.ReasonCode != reason {
					t.Fatalf("error=%#v", err)
				}
			})
		}
	}
}
func TestHostRejectsInvalidSuccessAndTransportFailure(t *testing.T) {
	for _, body := range []string{"", "{}", "{\"data\":null}", "{\"data\":{}}", "{\"data\":[]}", "not-json"} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, body) }))
			defer server.Close()
			client, err := NewClientWithTokenProvider(Config{BaseURL: server.URL}, TokenProviderFunc(func(context.Context) (string, error) { return "sts", nil }), server.Client())
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.GetInstallSession(context.Background(), "00000000-0000-4000-8000-000000000001")
			var e *HTTPError
			if !errors.As(err, &e) || e.StatusCode != 502 {
				t.Fatalf("error=%#v", err)
			}
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client, err := NewClientWithTokenProvider(Config{BaseURL: server.URL}, TokenProviderFunc(func(context.Context) (string, error) { return "sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	server.Close()
	_, err = client.GetInstallSession(context.Background(), "00000000-0000-4000-8000-000000000001")
	var e *HTTPError
	if !errors.As(err, &e) || e.StatusCode != 503 {
		t.Fatalf("error=%#v", err)
	}
}
