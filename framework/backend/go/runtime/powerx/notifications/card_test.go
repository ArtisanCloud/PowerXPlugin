package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

type cardFixture struct {
	Create       CreateInput          `json:"create"`
	Notification Notification         `json:"notification"`
	Invalidate   InvalidateCardInput  `json:"invalidate"`
	Result       InvalidateCardResult `json:"invalidate_result"`
	Read         ReadReceiptInput     `json:"read_receipt"`
}

func loadCardFixture(t *testing.T) cardFixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/business-notification-card.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture cardFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}
func cardClient(t *testing.T, channel string, transport func(*http.Request) (*http.Response, error)) *Client {
	t.Helper()
	cfg := Config{BaseURL: "https://core.test/deployment", APIPrefix: "/core/custom", PluginID: "com.powerx.plugins.ai-craft"}
	httpClient := &http.Client{Transport: notificationRoundTrip(func(req *http.Request) (*http.Response, error) {
		want := "Bearer controlled-sts"
		if channel == "apikey" {
			want = "ApiKey controlled-api-key"
		}
		if req.Header.Get("Authorization") != want {
			t.Errorf("authorization scheme mismatch")
		}
		return transport(req)
	})}
	var client *Client
	var err error
	if channel == "apikey" {
		cfg.AuthScheme = "apikey"
		cfg.APIKey = "controlled-api-key"
		client, err = NewClient(cfg, httpClient)
	} else {
		client, err = NewClientWithTokenProvider(cfg, TokenProviderFunc(func(context.Context) (string, error) { return "controlled-sts", nil }), httpClient)
	}
	if err != nil {
		t.Fatal(err)
	}
	return client
}
func cardResponse(data any) (*http.Response, error) {
	raw, _ := json.Marshal(map[string]any{"success": true, "data": data})
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
}
func bindingResponse(result any) map[string]any {
	return map[string]any{"protocol_used": CardProtocol, "fallback_used": false, "payload": map[string]any{"result": result}}
}

func TestCardFixtureCreateInvalidateBindAndReadBothCredentialChannels(t *testing.T) {
	for _, channel := range []string{"apikey", "sts"} {
		t.Run(channel, func(t *testing.T) {
			fixture := loadCardFixture(t)
			calls := 0
			client := cardClient(t, channel, func(req *http.Request) (*http.Response, error) {
				calls++
				switch req.URL.Path {
				case "/deployment/core/custom/notifications":
					var got CreateInput
					if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(got, fixture.Create) {
						t.Fatalf("create lost metadata/navigation: %#v", got)
					}
					return cardResponse(fixture.Notification)
				case "/deployment/core/custom/tenant/invocations":
					var got struct {
						Capability string                     `json:"capability_id"`
						Protocol   string                     `json:"preferred_protocol"`
						Payload    map[string]json.RawMessage `json:"payload"`
					}
					if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
						t.Fatal(err)
					}
					if req.Method != "POST" || got.Capability != InvalidateCardCapability || got.Protocol != CardProtocol || len(got.Payload) != 1 {
						t.Fatal("invalid fixed binding")
					}
					var body struct {
						Operation string `json:"operation"`
						InvalidateCardInput
					}
					if err := json.Unmarshal(got.Payload["body"], &body); err != nil {
						t.Fatal(err)
					}
					if body.Operation != "invalidate_card" || body.NotificationUUID != fixture.Invalidate.NotificationUUID {
						t.Fatal("updated wrong notification")
					}
					if body.CardReference != nil && (body.Reason != AccessChanged || body.CardReference.SourceRevision != body.SourceRevision) {
						t.Fatal("binding/reference lost")
					}
					if body.CardReference == nil && (body.SourceRevision != fixture.Invalidate.SourceRevision || body.Reason != fixture.Invalidate.Reason) {
						t.Fatal("RETRY_CHANGED_SOURCE_VERSION_OR_REASON")
					}
					result := fixture.Result
					if calls > 2 {
						result.Changed = false
						result.SourceRevision = 3
					}
					return cardResponse(bindingResponse(result))
				case "/deployment/core/custom/notifications/read-receipts":
					var got ReadReceiptInput
					_ = json.NewDecoder(req.Body).Decode(&got)
					if !reflect.DeepEqual(got, fixture.Read) {
						t.Fatal("receipt lost revisions")
					}
					return cardResponse(ReadReceipt{Contract: ReadReceiptContract, MemberUUID: got.MemberUUID, NotificationUUIDs: got.NotificationUUIDs, Revisions: got.Revisions})
				default:
					t.Fatalf("unexpected URL: %s", req.URL.Path)
					return nil, nil
				}
			})
			fixture.Create.Metadata["caller_context"] = map[string]any{"retained": true}
			fixture.Notification.Metadata["caller_context"] = fixture.Create.Metadata["caller_context"]
			created, err := client.Create(context.Background(), fixture.Create)
			if err != nil || created.Revision != 1 || created.SourceRevision != 1 {
				t.Fatalf("create: %#v %v", created, err)
			}
			out, err := client.InvalidateCard(context.Background(), fixture.Invalidate)
			if err != nil || !out.Changed {
				t.Fatalf("invalidate: %#v %v", out, err)
			}
			for i := 0; i < 2; i++ {
				out, err = client.InvalidateCard(context.Background(), fixture.Invalidate)
				if err != nil || out.Changed || out.SourceRevision != 3 {
					t.Fatalf("stale/replay: %#v %v", out, err)
				}
			}
			ref, _ := CardReference(fixture.Create.Metadata)
			binding := fixture.Invalidate
			binding.SourceRevision = 1
			binding.Reason = AccessChanged
			binding.CardReference = ref
			if _, err := client.InvalidateCard(context.Background(), binding); err != nil {
				t.Fatal(err)
			}
			if _, err := client.Acknowledge(context.Background(), fixture.Read); err != nil {
				t.Fatal(err)
			}
			if calls != 6 {
				t.Fatalf("unexpected additional request: %d", calls)
			}
		})
	}
}

func TestCardRejectsIncompleteBindingAndResult(t *testing.T) {
	fixture := loadCardFixture(t)
	for _, mutation := range []string{"missing_changed", "missing_uuid", "mismatched_uuid", "missing_revision", "invalid_source", "lower_source", "missing_protocol", "missing_fallback", "fallback", "wrong_protocol", "null_result"} {
		t.Run(mutation, func(t *testing.T) {
			calls := 0
			client := cardClient(t, "sts", func(*http.Request) (*http.Response, error) {
				calls++
				raw, _ := json.Marshal(fixture.Result)
				var result map[string]any
				_ = json.Unmarshal(raw, &result)
				data := bindingResponse(result)
				switch mutation {
				case "missing_changed":
					delete(result, "changed")
				case "missing_uuid":
					delete(result, "notification_uuid")
				case "mismatched_uuid":
					result["notification_uuid"] = fixture.Read.MemberUUID
				case "missing_revision":
					delete(result, "revision")
				case "invalid_source":
					result["source_revision"] = float64(MaxRevision) + 1
				case "lower_source":
					result["source_revision"] = 1
				case "missing_protocol":
					delete(data, "protocol_used")
				case "missing_fallback":
					delete(data, "fallback_used")
				case "fallback":
					data["fallback_used"] = true
				case "wrong_protocol":
					data["protocol_used"] = "http"
				case "null_result":
					data["payload"] = map[string]any{"result": nil}
				}
				return cardResponse(data)
			})
			_, err := client.InvalidateCard(context.Background(), fixture.Invalidate)
			var typed *HTTPError
			if !errors.As(err, &typed) || typed.StatusCode != 502 || calls != 1 {
				t.Fatalf("contract failure: %v calls=%d", err, calls)
			}
		})
	}
}

func TestCardConflictsPreserveDiagnosticsWithoutRetry(t *testing.T) {
	fixture := loadCardFixture(t)
	for _, channel := range []string{"apikey", "sts"} {
		for _, code := range []string{"NOTIFICATION_CARD_BINDING_CONFLICT", "NOTIFICATION_REVISION_CONFLICT"} {
			t.Run(channel+"/"+code, func(t *testing.T) {
				calls := 0
				client := cardClient(t, channel, func(*http.Request) (*http.Response, error) {
					calls++
					raw, _ := json.Marshal(map[string]any{"error": map[string]any{"error_code": code, "reason_code": code, "trace_id": "core-trace", "request_id": "core-request", "details": map[string]any{"revision": 3}}})
					return &http.Response{StatusCode: 409, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
				})
				var err error
				if code == "NOTIFICATION_REVISION_CONFLICT" {
					_, err = client.Acknowledge(context.Background(), fixture.Read)
				} else {
					ref, _ := CardReference(fixture.Create.Metadata)
					in := fixture.Invalidate
					in.SourceRevision = ref.SourceRevision
					in.CardReference = ref
					_, err = client.InvalidateCard(context.Background(), in)
				}
				var typed *HTTPError
				if !errors.As(err, &typed) || typed.StatusCode != 409 || typed.ErrorCode != code || typed.ReasonCode != code || typed.TraceID != "core-trace" || typed.RequestID != "core-request" || typed.Details["revision"] != float64(3) || typed.Body == "" || calls != 1 {
					t.Fatalf("lost conflict: %#v requests=%d", err, calls)
				}
			})
		}
	}
}

func TestCardValidationAndMetadataCopy(t *testing.T) {
	fixture := loadCardFixture(t)
	ref, _ := CardReference(fixture.Create.Metadata)
	original := CreateInput{Metadata: map[string]any{"caller": true}}
	with, err := original.WithBusinessCard(*ref)
	if err != nil || with.Metadata["caller"] != true || len(original.Metadata) != 1 {
		t.Fatal("metadata overwritten")
	}
	if _, err := with.WithBusinessCard(*ref); err == nil {
		t.Fatal("existing reference replaced")
	}
	client := cardClient(t, "sts", func(*http.Request) (*http.Response, error) { t.Fatal("invalid input made request"); return nil, nil })
	for _, change := range []func(*InvalidateCardInput){
		func(in *InvalidateCardInput) { in.SourceRevision = 0 }, func(in *InvalidateCardInput) { in.SourceRevision = MaxRevision + 1 }, func(in *InvalidateCardInput) { in.MemberUUID = "invalid" }, func(in *InvalidateCardInput) { in.Reason = "unknown" }, func(in *InvalidateCardInput) { in.CardReference = ref },
	} {
		in := fixture.Invalidate
		change(&in)
		if _, err := client.InvalidateCard(context.Background(), in); err == nil {
			t.Fatal("invalid command accepted")
		}
	}
	for _, delivery := range []string{"../escape", "encoded%2Fpath", "key?query", "", strings.Repeat("x", 257)} {
		bad := *ref
		bad.DeliveryKey = delivery
		if bad.Validate() == nil {
			t.Fatal("invalid delivery key")
		}
	}
	for _, version := range []int64{0, -1, MaxRevision + 1} {
		in := fixture.Read
		in.Revisions = map[string]int64{in.NotificationUUIDs[0]: version}
		if _, err := client.Acknowledge(context.Background(), in); err == nil {
			t.Fatal("invalid receipt version")
		}
	}
	fixture.Create.MemberUUID = ""
	if _, err := client.Create(context.Background(), fixture.Create); err == nil {
		t.Fatal("card without member")
	}
}

func TestReceiptResponseMustEchoExactRevisionMap(t *testing.T) {
	fixture := loadCardFixture(t)
	for _, mapping := range []map[string]int64{nil, {fixture.Read.NotificationUUIDs[0]: 1}, {fixture.Read.NotificationUUIDs[0]: 3}, {fixture.Read.MemberUUID: 2}} {
		client := cardClient(t, "sts", func(*http.Request) (*http.Response, error) {
			return cardResponse(ReadReceipt{Contract: ReadReceiptContract, MemberUUID: fixture.Read.MemberUUID, NotificationUUIDs: fixture.Read.NotificationUUIDs, Revisions: mapping})
		})
		if _, err := client.Acknowledge(context.Background(), fixture.Read); err == nil {
			t.Fatal("accepted changed receipt revision map")
		}
	}
}

func TestCreateCardRejectsIncompleteOrMismatchedResponse(t *testing.T) {
	for _, mutation := range []string{"uuid", "member", "revision", "source_revision", "reference", "metadata", "navigation"} {
		t.Run(mutation, func(t *testing.T) {
			fixture := loadCardFixture(t)
			fixture.Create.Metadata["caller"] = true
			fixture.Notification.Metadata["caller"] = true
			switch mutation {
			case "uuid":
				fixture.Notification.UUID = ""
			case "member":
				fixture.Notification.MemberUUID = ""
			case "revision":
				fixture.Notification.Revision = 0
			case "source_revision":
				fixture.Notification.SourceRevision = 0
			case "reference":
				delete(fixture.Notification.Metadata, "business_card")
			case "metadata":
				delete(fixture.Notification.Metadata, "caller")
			case "navigation":
				fixture.Notification.Actions = nil
			}
			calls := 0
			client := cardClient(t, "apikey", func(*http.Request) (*http.Response, error) { calls++; return cardResponse(fixture.Notification) })
			if _, err := client.Create(context.Background(), fixture.Create); err == nil || calls != 1 {
				t.Fatal("INCOMPLETE_CREATE_RESPONSE_ACCEPTED")
			}
		})
	}
}

func TestNotificationClientRequiresActualSTSProvider(t *testing.T) {
	if _, err := NewClient(Config{BaseURL: "https://core.test", STSClientID: "controlled", STSClientSecret: "controlled", STSTokenURL: "https://core.test/sts"}, nil); err == nil {
		t.Fatal("PLACEHOLDER_STS_CREDENTIAL_ACCEPTED")
	}
}

func TestCreateCardRequiresPublishedResponseFields(t *testing.T) {
	for _, field := range []string{"isRead", "isImportant", "createdAt", "updatedAt", "title", "content", "category", "type"} {
		fixture := loadCardFixture(t)
		raw, _ := json.Marshal(fixture.Notification)
		var out map[string]any
		_ = json.Unmarshal(raw, &out)
		delete(out, field)
		client := cardClient(t, "apikey", func(*http.Request) (*http.Response, error) { return cardResponse(out) })
		if _, err := client.Create(context.Background(), fixture.Create); err == nil {
			t.Fatalf("REQUIRED_RESPONSE_FIELD_MISSING:%s", field)
		}
	}
}
