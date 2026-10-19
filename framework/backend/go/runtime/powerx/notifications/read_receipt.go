package notifications

import (
	"context"
	"github.com/google/uuid"
	"net/http"
)

const ReadReceiptContract = "notification_read_receipt.v1"

type ReadReceiptInput struct {
	MemberUUID        string   `json:"member_uuid"`
	NotificationUUIDs []string `json:"notification_uuids"`
}
type ReadReceipt struct {
	Contract          string   `json:"contract"`
	MemberUUID        string   `json:"member_uuid"`
	NotificationUUIDs []string `json:"notification_uuids"`
}
type ReadAcknowledger interface {
	Acknowledge(context.Context, ReadReceiptInput) (*ReadReceipt, error)
}

func (c *Client) Acknowledge(ctx context.Context, input ReadReceiptInput) (*ReadReceipt, error) {
	if c == nil || c.http == nil || c.tokens == nil {
		return nil, &HTTPError{StatusCode: http.StatusServiceUnavailable, ReasonCode: "NOTIFICATION_CLIENT_UNAVAILABLE"}
	}
	invalid := func() error {
		return &HTTPError{StatusCode: http.StatusBadRequest, ReasonCode: "NOTIFICATION_READ_RECEIPT_INVALID"}
	}
	invalidResponse := func() error {
		return &HTTPError{StatusCode: http.StatusBadGateway, ReasonCode: "NOTIFICATION_READ_RECEIPT_RESPONSE_INVALID"}
	}
	valid := func(raw string) bool {
		id, err := uuid.Parse(raw)
		return err == nil && id != uuid.Nil && id.String() == raw
	}
	if !valid(input.MemberUUID) || len(input.NotificationUUIDs) == 0 || len(input.NotificationUUIDs) > 100 {
		return nil, invalid()
	}
	expected := map[string]bool{}
	for _, id := range input.NotificationUUIDs {
		if !valid(id) || expected[id] {
			return nil, invalid()
		}
		expected[id] = true
	}
	var out ReadReceipt
	if err := c.doJSON(ctx, http.MethodPost, "/notifications/read-receipts", input, &out); err != nil {
		return nil, err
	}
	if out.Contract != ReadReceiptContract || out.MemberUUID != input.MemberUUID || len(out.NotificationUUIDs) != len(expected) {
		return nil, invalidResponse()
	}
	for _, id := range out.NotificationUUIDs {
		if !expected[id] {
			return nil, invalidResponse()
		}
		delete(expected, id)
	}
	return &out, nil
}
