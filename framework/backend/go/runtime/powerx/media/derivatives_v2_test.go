package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func derivativeFixture(t *testing.T, name string) ImageDerivatives {
	t.Helper()
	raw, err := os.ReadFile("testdata/media-image-derivatives.v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatal(err)
	}
	var value ImageDerivatives
	if err := json.Unmarshal(fixtures[name], &value); err != nil {
		t.Fatal(err)
	}
	return value
}
func v2Client(t *testing.T, channel, base, prefix string) *HostClient {
	t.Helper()
	cfg := HostClientConfig{BaseURL: base, APIPrefix: prefix}
	var c *HostClient
	var err error
	if channel == "apikey" {
		c, err = NewHostClientWithAPIKey(cfg, "controlled-key", nil)
	} else {
		c, err = NewHostClientWithTokenProvider(cfg, HostTokenProviderFunc(func(context.Context) (string, error) { return "controlled-sts", nil }), nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func assertMediaCredential(t *testing.T, r *http.Request, channel string) {
	t.Helper()
	want := "Bearer controlled-sts"
	if channel == "apikey" {
		want = "ApiKey controlled-key"
	}
	if r.Header.Get("Authorization") != want || r.URL.RawQuery != "" || r.Header.Get("X-Tenant-UUID") != "" {
		t.Error("MEDIA_CREDENTIAL_OR_QUERY_BOUNDARY_INVALID")
	}
}
func bindingState(w http.ResponseWriter, state any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"protocol_used": "core_internal", "fallback_used": false, "payload": map[string]any{"result": state}}})
}

func TestDerivativeV2BindingPartialFailureAndActualVariantRead(t *testing.T) {
	for _, channel := range []string{"apikey", "sts"} {
		t.Run(channel, func(t *testing.T) {
			state := derivativeFixture(t, "partial_failed")
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				assertMediaCredential(t, r, channel)
				switch {
				case strings.HasSuffix(r.URL.Path, "/tenant/invocations"):
					if r.URL.Path != "/deployment/core/custom/tenant/invocations" {
						t.Error("API_PREFIX_DUPLICATED_OR_IGNORED")
					}
					var req struct {
						Capability string                     `json:"capability_id"`
						Protocol   string                     `json:"preferred_protocol"`
						Payload    map[string]json.RawMessage `json:"payload"`
					}
					if json.NewDecoder(r.Body).Decode(&req) != nil {
						t.Fatal("MEDIA_BINDING_DECODE_FAILED")
					}
					var body map[string]any
					_ = json.Unmarshal(req.Payload["body"], &body)
					if req.Protocol != "core_internal" || len(req.Payload) != 1 || body["asset_uuid"] != state.AssetUUID {
						t.Fatal("MEDIA_BINDING_NOT_FIXED")
					}
					if calls == 1 && (req.Capability != ImageDerivativesReadCapability || body["operation"] != "status" || len(body) != 2) {
						t.Fatal("MEDIA_READ_BINDING_INVALID")
					}
					if calls == 2 && (req.Capability != ImageDerivativesEnsureCapability || body["operation"] != "ensure" || body["retry_failed"] != true) {
						t.Fatal("MEDIA_ENSURE_BINDING_INVALID")
					}
					bindingState(w, state)
				case r.Method == "GET":
					_ = json.NewEncoder(w).Encode(map[string]any{"data": Variant{AssetUUID: state.AssetUUID, VariantUUID: state.Items[0].VariantUUID, Status: "ready", MimeType: "image/png", SizeBytes: 6993}})
				default:
					if !strings.Contains(r.URL.Path, "/"+state.AssetUUID+"/variants/"+state.Items[0].VariantUUID+"/presign-download") {
						t.Fatal("VARIANT_PARENT_NOT_PRESERVED")
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"data": TransferTicket{URL: "https://objects.invalid/opaque-ticket", Method: "GET", ExpiresAt: "2026-10-22T12:00:00Z"}})
				}
			}))
			defer server.Close()
			client := v2Client(t, channel, server.URL+"/deployment", "/core/custom")
			out, err := client.GetImageDerivatives(context.Background(), state.AssetUUID)
			if err != nil {
				t.Fatal(err)
			}
			item, err := out.Item(VariantThumbnail64)
			if err != nil || item.ErrorCode != "MEDIA_THUMBNAIL_64_BYTE_LIMIT" || item.Retryable == nil || !*item.Retryable {
				t.Fatal("ITEM_ERROR_DIAGNOSTICS_DROPPED")
			}
			if _, err = client.EnsureImageDerivatives(context.Background(), state.AssetUUID, EnsureImageDerivativesInput{RetryFailed: true}); err != nil {
				t.Fatal(err)
			}
			v, ticket, err := client.ReadImageDerivative(context.Background(), out, VariantPreview)
			if err != nil || v.MimeType != "image/png" || v.SizeBytes != 6993 || ticket.Method != "GET" {
				t.Fatal("READY_ITEM_BLOCKED_BY_AGGREGATE_FAILURE")
			}
			if _, _, err := client.ReadImageDerivative(context.Background(), out, VariantThumbnail64); err == nil || calls != 4 {
				t.Fatal("FAILED_SPEC_FELL_BACK_OR_MADE_REQUEST")
			}
		})
	}
}

func TestDerivativeV2ItemOrderMissingAndResponseValidation(t *testing.T) {
	state := derivativeFixture(t, "ready")
	state.Items[0], state.Items[2] = state.Items[2], state.Items[0]
	if item, err := state.Item(VariantThumbnail64); err != nil || item.VariantUUID != "22222222-2222-4222-8222-000000000003" {
		t.Fatal("VARIANT_SELECTED_BY_INDEX")
	}
	state.Status = "processing"
	state.Items = state.Items[1:]
	if _, err := state.Item(VariantThumbnail64); err == nil {
		t.Fatal("MISSING_SPEC_SUBSTITUTED")
	}
	for _, mutation := range []string{"ready_without_uuid", "pending_with_uuid", "duplicate", "missing_retryable", "fallback", "missing_fallback", "protocol", "asset_mismatch"} {
		t.Run(mutation, func(t *testing.T) {
			state := derivativeFixture(t, "ready")
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var value any = state
				switch mutation {
				case "ready_without_uuid":
					state.Items[2].VariantUUID = ""
					value = state
				case "pending_with_uuid":
					state.Status = "processing"
					state.Items[2].Status = "processing"
					value = state
				case "duplicate":
					state.Items = append(state.Items, state.Items[0])
					value = state
				case "asset_mismatch":
					state.AssetUUID = state.Items[0].VariantUUID
					value = state
				case "missing_retryable":
					raw, _ := json.Marshal(state)
					var object map[string]any
					_ = json.Unmarshal(raw, &object)
					delete(object, "retryable")
					value = object
				}
				data := map[string]any{"protocol_used": "core_internal", "fallback_used": false, "payload": map[string]any{"result": value}}
				if mutation == "fallback" {
					data["fallback_used"] = true
				}
				if mutation == "missing_fallback" {
					delete(data, "fallback_used")
				}
				if mutation == "protocol" {
					data["protocol_used"] = "http"
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
			}))
			defer server.Close()
			c := v2Client(t, "apikey", server.URL, "")
			_, err := c.GetImageDerivatives(context.Background(), derivativeFixture(t, "ready").AssetUUID)
			var typed *HostHTTPError
			if !errors.As(err, &typed) || typed.StatusCode != 502 || calls != 1 {
				t.Fatalf("INVALID_RESPONSE_ACCEPTED:%v", err)
			}
		})
	}
}

func TestDerivativeV2StreamRenewalRevocationAndNoIdleQueries(t *testing.T) {
	for _, channel := range []string{"apikey", "sts"} {
		t.Run(channel, func(t *testing.T) {
			state := derivativeFixture(t, "partial_processing")
			calls := 0
			callbacks := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				assertMediaCredential(t, r, channel)
				if r.Method != "GET" || !strings.HasSuffix(r.URL.Path, "/derivatives/events") || r.Header.Get("Last-Event-ID") != "" {
					t.Fatal("SSE_CURSOR_OR_POLLING_USED")
				}
				w.Header().Set("Content-Type", "text/event-stream")
				raw, _ := json.Marshal(state)
				fmt.Fprintf(w, ": keepalive\n\nevent: status\ndata: %s\n\n", raw)
				if calls == 1 {
					fmt.Fprint(w, "event: reconnect\ndata: {\"reason\":\"reauthorize\"}\n\n")
				} else {
					fmt.Fprintf(w, "event: invalidate\ndata: {\"reason\":\"access_or_state_changed\",\"asset_uuid\":%q}\n\n", state.AssetUUID)
				}
			}))
			defer server.Close()
			client := v2Client(t, channel, server.URL, "")
			err := client.SubscribeImageDerivatives(context.Background(), state.AssetUUID, ImageDerivativeSubscriptionOptions{ReconnectDelay: time.Millisecond}, func(event ImageDerivativeEvent) error {
				if event.Type == "status" {
					callbacks++
				}
				return nil
			})
			var typed *HostHTTPError
			if !errors.As(err, &typed) || typed.ReasonCode != "MEDIA_DERIVATIVES_INVALIDATED" || calls != 2 || callbacks != 2 {
				t.Fatalf("INVALIDATION_DID_NOT_STOP:%v", err)
			}
		})
	}
}

func TestDerivativeStreamCancellationAndNoLateCallback(t *testing.T) {
	state := derivativeFixture(t, "ready")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	callbacks := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		raw, _ := json.Marshal(state)
		fmt.Fprintf(w, "event: status\ndata: %s\n\nevent: status\ndata: %s\n\n", raw, raw)
	}))
	defer server.Close()
	err := v2Client(t, "sts", server.URL, "").SubscribeImageDerivatives(ctx, state.AssetUUID, ImageDerivativeSubscriptionOptions{}, func(ImageDerivativeEvent) error { callbacks++; cancel(); return nil })
	if !errors.Is(err, context.Canceled) || callbacks != 1 {
		t.Fatal("LATE_SNAPSHOT_AFTER_CANCEL")
	}
}

func TestThumbnail64UploadValidationAndCoreGeneratedReuse(t *testing.T) {
	state := derivativeFixture(t, "ready")
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{"data": Variant{AssetUUID: state.AssetUUID, VariantUUID: state.Items[2].VariantUUID, Status: "ready", MimeType: "image/png", SizeBytes: 208}})
	}))
	defer server.Close()
	c := v2Client(t, "apikey", server.URL, "")
	input := CreateVariantInput{VariantType: VariantThumbnail64, Checksum: strings.Repeat("a", 64), MimeType: "image/jpeg", SizeBytes: 1024}
	v, err := c.CreateVariant(context.Background(), state.AssetUUID, input)
	if err != nil || v.MimeType != "image/png" || v.SizeBytes != 208 || v.Status != "ready" {
		t.Fatal("CORE_GENERATED_VARIANT_METADATA_OVERRIDDEN")
	}
	input.SizeBytes = 32769
	if _, err := c.CreateVariant(context.Background(), state.AssetUUID, input); err == nil || calls != 1 {
		t.Fatal("THUMBNAIL_BYTE_LIMIT_NOT_ENFORCED")
	}
}

func TestDerivativeErrorsPreserveStatusCodeTraceAndDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		fmt.Fprint(w, `{"error":{"error_code":"MEDIA_VARIANT_CONFLICT","reason_code":"MEDIA_VARIANT_CONFLICT","trace_id":"core-trace","request_id":"core-request","details":{"retryable":false}}}`)
	}))
	defer server.Close()
	_, err := v2Client(t, "apikey", server.URL, "").GetImageDerivatives(context.Background(), derivativeFixture(t, "ready").AssetUUID)
	var typed *HostHTTPError
	if !errors.As(err, &typed) || typed.StatusCode != 409 || typed.ErrorCode != "MEDIA_VARIANT_CONFLICT" || typed.TraceID != "core-trace" || typed.RequestID != "core-request" || typed.Details["retryable"] != false {
		t.Fatal("MEDIA_DIAGNOSTICS_LOST")
	}
}
