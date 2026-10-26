package host_contract

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	runtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/notifications"
	core "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/notifications"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

type cardPublisher struct {
	create     core.CreateInput
	invalidate core.InvalidateCardInput
	read       core.ReadReceiptInput
	err        error
}

func (p *cardPublisher) Create(_ context.Context, in core.CreateInput) (*core.Notification, error) {
	p.create = in
	return &core.Notification{Revision: 1, SourceRevision: 1, Metadata: in.Metadata}, p.err
}
func (p *cardPublisher) InvalidateCard(_ context.Context, in core.InvalidateCardInput) (*core.InvalidateCardResult, error) {
	p.invalidate = in
	return &core.InvalidateCardResult{NotificationUUID: in.NotificationUUID, Revision: 2, SourceRevision: in.SourceRevision, Changed: true}, p.err
}
func (p *cardPublisher) Acknowledge(_ context.Context, in core.ReadReceiptInput) (*core.ReadReceipt, error) {
	p.read = in
	return &core.ReadReceipt{Contract: core.ReadReceiptContract, MemberUUID: in.MemberUUID, NotificationUUIDs: in.NotificationUUIDs, Revisions: in.Revisions}, p.err
}

func TestHostNotificationCardOperationsAndErrorDetails(t *testing.T) {
	publisher := &cardPublisher{}
	rt, err := runtime.NewRuntime(provider.ModeDelegated, nil, publisher)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(&app.Deps{NotificationDelivery: rt})
	member := "22222222-2222-4222-8222-222222222222"
	id := "33333333-3333-4333-8333-333333333333"
	for operation, input := range map[string]map[string]any{
		"notification.create":           {"title": "fixture_title", "content": "fixture_content", "member_uuid": member, "navigation": map[string]any{"label": "fixture_navigation", "path": "/fixture"}, "metadata": map[string]any{"caller": true}},
		"notification.invalidate_card":  {"member_uuid": member, "notification_uuid": id, "source_revision": 2, "reason": "access_changed", "card_reference": map[string]any{"contract": core.BusinessCardContract, "source_plugin": "com.powerx.plugins.ai-craft", "business_type": "sample_track", "delivery_key": "opaque-member-delivery", "source_revision": 2}},
		"notification.acknowledge_read": {"member_uuid": member, "notification_uuids": []string{id}, "revisions": map[string]int64{id: 2}},
	} {
		response := executeProbe(t, handler, map[string]any{"module": "notifications", "operation": operation, "input": input, "confirm": true})
		if response.Code != 200 {
			t.Fatalf("HOST_NOTIFICATION_DISPATCH_FAILED:%s %s", operation, response.Body.String())
		}
	}
	if publisher.create.Navigation == nil || publisher.create.Navigation.Path != "/fixture" || publisher.create.Metadata["caller"] != true {
		t.Fatal("HOST_CREATE_FIELDS_LOST")
	}
	if publisher.invalidate.Reason != core.AccessChanged || publisher.invalidate.CardReference == nil {
		t.Fatal("HOST_BINDING_FIELDS_LOST")
	}
	if publisher.read.Revisions[id] != 2 {
		t.Fatal("HOST_RECEIPT_REVISIONS_LOST")
	}
	publisher.err = &core.HTTPError{StatusCode: 409, ErrorCode: "NOTIFICATION_CARD_BINDING_CONFLICT", ReasonCode: "NOTIFICATION_CARD_BINDING_CONFLICT", TraceID: "core-trace", RequestID: "core-request", Details: map[string]any{"revision": 3}}
	response := executeProbe(t, handler, map[string]any{"module": "notifications", "operation": "notification.invalidate_card", "input": map[string]any{"member_uuid": member, "notification_uuid": id, "source_revision": 2}, "confirm": true})
	var envelope struct {
		Error struct {
			Status  int            `json:"http_status"`
			Code    string         `json:"error_code"`
			Trace   string         `json:"trace_id"`
			Request string         `json:"request_id"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if response.Code != 409 || envelope.Error.Status != 409 || envelope.Error.Code != "NOTIFICATION_CARD_BINDING_CONFLICT" || envelope.Error.Trace != "core-trace" || envelope.Error.Request != "core-request" || envelope.Error.Details["revision"] != float64(3) {
		t.Fatal("HOST_UPSTREAM_DIAGNOSTICS_LOST")
	}
	if rt.Mode() != provider.ModeDelegated {
		t.Fatal("CHANNEL_CHANGED_AFTER_FAILURE")
	}
}

func TestHostLocalCardUnsupportedAndUnknownFields(t *testing.T) {
	rt, _ := runtime.NewRuntime(provider.ModeLocal, notificationPublisherStub{}, nil)
	handler := NewHandler(&app.Deps{NotificationDelivery: rt})
	for _, operation := range []string{"notification.invalidate_card", "notification.acknowledge_read"} {
		response := executeProbe(t, handler, map[string]any{"module": "notifications", "operation": operation, "confirm": true, "input": map[string]any{}})
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("LOCAL_CARD_FABRICATED:%d", response.Code)
		}
		response = executeProbe(t, handler, map[string]any{"module": "notifications", "operation": operation, "confirm": true, "input": map[string]any{"endpoint": "core://elsewhere"}})
		if response.Code != 400 {
			t.Fatal("UNKNOWN_HOST_FIELD_ACCEPTED")
		}
	}
}
