package skills

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestSkillsErrorEnvelopeMatrix(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 409, 429, 502, 503} {
		for _, body := range []string{`{"reason_code":"TEST_REASON"}`, `{"error":{"reason_code":"TEST_REASON"}}`, `{"error_code":"TEST_REASON"}`} {
			t.Run(fmt.Sprintf("%d/%s", status, body), func(t *testing.T) {
				client, err := NewClient(Config{BaseURL: "http://test.invalid", BearerToken: "test-sts"}, &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) { return response(status, body), nil })})
				if err != nil {
					t.Fatal(err)
				}
				out, err := client.Invoke(t.Context(), InvokeInput{SkillID: "test-skill"})
				var typed *HTTPError
				if out != nil || !errors.As(err, &typed) || typed.StatusCode != status || typed.ReasonCode != "TEST_REASON" {
					t.Fatalf("out=%v err=%v", out, err)
				}
			})
		}
	}
}

func TestSkillsRejectsInvalidSuccessEnvelope(t *testing.T) {
	for _, body := range []string{"invalid", "{}", `{"data":null}`, `{"success":false,"data":{}}`, `{"trace_id":"trace"}`} {
		client, err := NewClient(Config{BaseURL: "http://test.invalid", BearerToken: "test-sts"}, &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) { return response(200, body), nil })})
		if err != nil {
			t.Fatal(err)
		}
		out, err := client.Invoke(t.Context(), InvokeInput{SkillID: "test-skill"})
		var typed *HTTPError
		if out != nil || !errors.As(err, &typed) || typed.StatusCode != 502 || typed.ReasonCode != "SKILL_UPSTREAM_DEPENDENCY" {
			t.Fatalf("body=%s out=%v err=%v", body, out, err)
		}
	}
}
