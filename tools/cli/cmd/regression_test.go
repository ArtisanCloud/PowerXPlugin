package cmd

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	Regression "github.com/powerx-plugin/cli/internal/regression"
)

func TestRegressionRepeatedCasesReachAPI(t *testing.T) {
	first := "11111111-1111-4111-8111-111111111111"
	second := "22222222-2222-4222-8222-222222222222"
	received := make(chan []string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/admin/regression/runs" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body struct {
			Cases []string `json:"case_uuids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		received <- body.Cases
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"run": map[string]string{"uuid": "44444444-4444-4444-8444-444444444444"}}})
	}))
	defer server.Close()
	t.Setenv("REGRESSION_CLI_TEST_TOKEN", "fixture.credential")
	args := []string{"run", "--api-base", server.URL + "/api/v1", "--token-env", "REGRESSION_CLI_TEST_TOKEN", "--profile", "33333333-3333-4333-8333-333333333333", "--case", first, "--case", second}
	if err := runRegression(args); err != nil {
		t.Fatal(err)
	}
	if actual := <-received; !reflect.DeepEqual(actual, []string{first, second}) {
		t.Fatalf("case_uuids=%v", actual)
	}
	// A CSV value is not the repeated-flag contract and must fail before HTTP.
	args = append(args[:len(args)-4], "--case", first+","+second)
	err := runRegression(args)
	var failure *Regression.Error
	if !errors.As(err, &failure) || failure.Code != "REGRESSION_INVALID_REQUEST" {
		t.Fatalf("error=%v", err)
	}
	select {
	case actual := <-received:
		t.Fatalf("case_uuids=%v", actual)
	default:
	}
}
