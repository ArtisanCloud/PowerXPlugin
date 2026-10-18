package notifications

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestReadReceiptContractRejectsDifferentRecipientsOrWidenedRanges(t *testing.T) {
	member, id := uuid.NewString(), uuid.NewString()
	for _, scenario := range []string{"ready", "wrong_member", "extra_id", "duplicate_id", "wrong_contract", "denied"} {
		t.Run(scenario, func(t *testing.T) {
			client, err := NewClient(Config{BaseURL: "https://core.test", BearerToken: "fixture"}, &http.Client{Transport: notificationRoundTrip(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, "/api/v1/notifications/read-receipts", req.URL.Path)
				require.Equal(t, "POST", req.Method)
				var input map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&input))
				require.Len(t, input, 2)
				require.Equal(t, member, input["member_uuid"])
				receipt := ReadReceipt{Contract: ReadReceiptContract, MemberUUID: member, NotificationUUIDs: []string{id}}
				switch scenario {
				case "wrong_member":
					receipt.MemberUUID = uuid.NewString()
				case "extra_id":
					receipt.NotificationUUIDs = append(receipt.NotificationUUIDs, uuid.NewString())
				case "duplicate_id":
					receipt.NotificationUUIDs = append(receipt.NotificationUUIDs, id)
				case "wrong_contract":
					receipt.Contract = "wrong"
				case "denied":
					return notificationResponse(403, `{"error":{"code":"NOTIFICATION_SERVICE_FORBIDDEN"}}`), nil
				}
				raw, _ := json.Marshal(map[string]any{"data": receipt})
				return notificationResponse(200, string(raw)), nil
			})})
			require.NoError(t, err)
			result, err := client.Acknowledge(context.Background(), ReadReceiptInput{MemberUUID: member, NotificationUUIDs: []string{id}})
			if scenario == "ready" {
				require.NoError(t, err)
				require.Equal(t, []string{id}, result.NotificationUUIDs)
			} else {
				var typed *HTTPError
				require.ErrorAs(t, err, &typed)
				if scenario == "denied" {
					require.Equal(t, http.StatusForbidden, typed.StatusCode)
				} else {
					require.Equal(t, http.StatusBadGateway, typed.StatusCode)
					require.Equal(t, "NOTIFICATION_READ_RECEIPT_RESPONSE_INVALID", typed.ReasonCode)
				}
			}
		})
	}
}

func TestReadReceiptInvalidArgumentsAreNotUpstreamFailures(t *testing.T) {
	member, id := uuid.NewString(), uuid.NewString()
	client, err := NewClient(Config{BaseURL: "https://core.test", BearerToken: "fixture"}, &http.Client{Transport: notificationRoundTrip(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid arguments must not reach Core")
		return nil, nil
	})})
	require.NoError(t, err)
	for _, input := range []ReadReceiptInput{
		{}, {MemberUUID: member}, {MemberUUID: "invalid", NotificationUUIDs: []string{id}},
		{MemberUUID: member, NotificationUUIDs: []string{id, id}},
		{MemberUUID: member, NotificationUUIDs: []string{uuid.Nil.String()}},
		{MemberUUID: member, NotificationUUIDs: make([]string, 101)},
	} {
		_, err := client.Acknowledge(context.Background(), input)
		var typed *HTTPError
		require.ErrorAs(t, err, &typed)
		require.Equal(t, http.StatusBadRequest, typed.StatusCode)
		require.Equal(t, "NOTIFICATION_READ_RECEIPT_INVALID", typed.ReasonCode)
	}
}

func TestUnconfiguredReadAcknowledgerFailsWithoutPanicking(t *testing.T) {
	for _, client := range []*Client{nil, {}} {
		_, err := client.Acknowledge(context.Background(), ReadReceiptInput{})
		var typed *HTTPError
		require.ErrorAs(t, err, &typed)
		require.Equal(t, http.StatusServiceUnavailable, typed.StatusCode)
		require.Equal(t, "NOTIFICATION_CLIENT_UNAVAILABLE", typed.ReasonCode)
	}
}
