package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestCardV2CreateUpgradePresentationAndReceiptBothChannels(t *testing.T) {
	raw, err := os.ReadFile("testdata/business-notification-card.v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		cardFixture
		Presentation BusinessCardPresentationV2 `json:"presentation"`
		Pending      BusinessCardPresentationV2 `json:"presentation_pending"`
		Upgrade      InvalidateCardInput        `json:"upgrade_v1_reference"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if err := fixture.Presentation.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := fixture.Pending.Validate(); err != nil {
		t.Fatal(err)
	}
	if fixture.Pending.Image.Thumbnail.ImageKey != nil || fixture.Pending.Image.Preview.ImageKey == nil {
		t.Fatal("RENDITION_STATES_NOT_INDEPENDENT")
	}
	for _, channel := range []string{"apikey", "sts"} {
		t.Run(channel, func(t *testing.T) {
			calls := 0
			creates := 0
			client := cardClient(t, channel, func(req *http.Request) (*http.Response, error) {
				calls++
				if strings.HasSuffix(req.URL.Path, "/notifications") {
					creates++
					return cardResponse(fixture.Notification)
				}
				if strings.HasSuffix(req.URL.Path, "/read-receipts") {
					return cardResponse(ReadReceipt{Contract: ReadReceiptContract, MemberUUID: fixture.Read.MemberUUID, NotificationUUIDs: fixture.Read.NotificationUUIDs, Revisions: fixture.Read.Revisions})
				}
				var payload struct {
					Payload struct {
						Body struct {
							Operation string `json:"operation"`
							InvalidateCardInput
						} `json:"body"`
					} `json:"payload"`
				}
				if json.NewDecoder(req.Body).Decode(&payload) != nil {
					t.Fatal("UPGRADE_DECODE_FAILED")
				}
				body := payload.Payload.Body
				if body.Operation != "invalidate_card" || body.NotificationUUID != fixture.Notification.UUID || body.Reason != PresentationChanged || body.CardReference == nil || body.CardReference.Contract != BusinessCardContractV2 || body.SourceRevision != body.CardReference.SourceRevision {
					t.Fatal("UPGRADE_BINDING_OR_REASON_LOST")
				}
				result := fixture.Result
				if calls > 2 {
					result.Changed = false
				}
				return cardResponse(bindingResponse(result))
			})
			created, err := client.Create(context.Background(), fixture.Create)
			if err != nil || created.UUID != fixture.Notification.UUID {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				if _, err := client.InvalidateCard(context.Background(), fixture.Upgrade); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := client.Acknowledge(context.Background(), fixture.Read); err != nil {
				t.Fatal(err)
			}
			if creates != 1 || calls != 4 {
				t.Fatal("UPGRADE_RECREATED_OR_RETRIED_NOTIFICATION")
			}
		})
	}
}

func TestCardV2RenditionContractRejectsWrongKeysAndVariants(t *testing.T) {
	raw, _ := os.ReadFile("testdata/business-notification-card.v2.json")
	var fixture struct {
		Presentation BusinessCardPresentationV2 `json:"presentation"`
	}
	_ = json.Unmarshal(raw, &fixture)
	for _, mutation := range []string{"missing_key", "pending_key", "wrong_thumbnail", "wrong_preview", "invalid_key", "unknown_status"} {
		copyJSON, _ := json.Marshal(fixture.Presentation)
		var value BusinessCardPresentationV2
		_ = json.Unmarshal(copyJSON, &value)
		switch mutation {
		case "missing_key":
			value.Image.Thumbnail.ImageKey = nil
		case "pending_key":
			value.Image.Thumbnail.Status = "queued"
		case "wrong_thumbnail":
			value.Image.Thumbnail.Variant = "thumbnail"
		case "wrong_preview":
			value.Image.Preview.Variant = "origin"
		case "invalid_key":
			key := "https://external.invalid/image"
			value.Image.Thumbnail.ImageKey = &key
		case "unknown_status":
			value.Image.Preview.Status = "success"
		}
		if value.Validate() == nil {
			t.Fatalf("INVALID_RENDITION_ACCEPTED:%s", mutation)
		}
	}
}

func TestCardV2UpgradeConflictDoesNotRecreateOrChangeChannel(t *testing.T) {
	fixture := loadCardFixture(t)
	ref, _ := CardReference(fixture.Create.Metadata)
	ref.Contract = BusinessCardContractV2
	ref.SourceRevision = 2
	in := fixture.Invalidate
	in.Reason = PresentationChanged
	in.CardReference = ref
	calls := 0
	client := cardClient(t, "apikey", func(req *http.Request) (*http.Response, error) {
		calls++
		if !strings.HasSuffix(req.URL.Path, "/tenant/invocations") {
			t.Fatal("UPGRADE_RECREATED_NOTIFICATION")
		}
		return &http.Response{StatusCode: 409, Body: io.NopCloser(strings.NewReader(`{"error_code":"NOTIFICATION_CARD_BINDING_CONFLICT","trace_id":"upgrade-trace"}`))}, nil
	})
	_, err := client.InvalidateCard(context.Background(), in)
	var typed *HTTPError
	if !errors.As(err, &typed) || typed.StatusCode != 409 || typed.ErrorCode != "NOTIFICATION_CARD_BINDING_CONFLICT" || typed.TraceID != "upgrade-trace" || calls != 1 {
		t.Fatal("UPGRADE_CONFLICT_SWALLOWED")
	}
}
