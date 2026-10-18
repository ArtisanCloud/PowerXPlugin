package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientCreateUsesTenantNotificationsContract(t *testing.T) {
	transport := notificationRoundTrip(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "/api/v1/notifications", req.URL.Path)
		require.Equal(t, "Bearer service-token", req.Header.Get("Authorization"))
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.JSONEq(t, `{"title":"任务完成","content":"导出已完成","type":"success","member_uuid":"7c50b003-da2f-4514-aa4a-6f095fc4dc6b","metadata":{"job_uuid":"4ef5fb5e-97e4-4db6-a683-50e590589716"}}`, string(body))
		return notificationResponse(http.StatusOK, `{"data":{"id":"1a2370d5-0f7c-4693-a6a8-f46b2799fc9d","title":"任务完成","content":"导出已完成","type":"success","category":"job","isRead":false,"isImportant":false,"createdAt":"2026-08-31T10:00:00Z","updatedAt":"2026-08-31T10:00:00Z","userId":"7c50b003-da2f-4514-aa4a-6f095fc4dc6b"}}`), nil
	})
	client, err := NewClientWithTokenProvider(Config{BaseURL: "https://core.example"}, TokenProviderFunc(func(context.Context) (string, error) {
		return "service-token", nil
	}), &http.Client{Transport: transport})
	require.NoError(t, err)

	notification, err := client.Create(context.Background(), CreateInput{Title: "任务完成", Content: "导出已完成", Type: "success", MemberUUID: "7c50b003-da2f-4514-aa4a-6f095fc4dc6b", Metadata: map[string]any{"job_uuid": "4ef5fb5e-97e4-4db6-a683-50e590589716"}})
	require.NoError(t, err)
	require.Equal(t, "1a2370d5-0f7c-4693-a6a8-f46b2799fc9d", notification.UUID)
	require.False(t, notification.IsRead)
}

func TestClientCreateReturnsTypedHTTPError(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "https://core.example", BearerToken: "service-token"}, &http.Client{Transport: notificationRoundTrip(func(*http.Request) (*http.Response, error) {
		return notificationResponse(http.StatusForbidden, `{"reason_code":"NOTIFICATION_FORBIDDEN"}`), nil
	})})
	require.NoError(t, err)

	_, err = client.Create(context.Background(), CreateInput{Title: "任务完成", Content: "导出已完成"})
	var httpErr *HTTPError
	require.True(t, errors.As(err, &httpErr))
	require.Equal(t, http.StatusForbidden, httpErr.StatusCode)
	require.Contains(t, httpErr.Body, "NOTIFICATION_FORBIDDEN")
}

func TestNotificationCommandsShareConfiguredCoreAPIPrefix(t *testing.T) {
	member, id := "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"
	for _, tc := range []struct{ prefix, expected string }{
		{"", "/api/v1"}, {"/api/v1", "/api/v1"}, {"/api/v2", "/api/v2"},
		{"api/v2/", "/api/v2"}, {"/core/custom", "/core/custom"}, {"/", ""},
	} {
		t.Run(tc.prefix, func(t *testing.T) {
			var paths []string
			client, err := NewClientWithTokenProvider(Config{BaseURL: "https://core.test/deployment/", APIPrefix: tc.prefix}, TokenProviderFunc(func(context.Context) (string, error) { return "fixture", nil }), &http.Client{Transport: notificationRoundTrip(func(req *http.Request) (*http.Response, error) {
				paths = append(paths, req.URL.Path)
				require.Equal(t, "Bearer fixture", req.Header.Get("Authorization"))
				if strings.HasSuffix(req.URL.Path, "/read-receipts") {
					raw, _ := json.Marshal(map[string]any{"data": ReadReceipt{Contract: ReadReceiptContract, MemberUUID: member, NotificationUUIDs: []string{id}}})
					return notificationResponse(200, string(raw)), nil
				}
				return notificationResponse(200, `{"data":{"id":"`+id+`"}}`), nil
			})})
			require.NoError(t, err)
			_, err = client.Create(context.Background(), CreateInput{Title: "fixture_title", Content: "fixture_content"})
			require.NoError(t, err)
			_, err = client.Acknowledge(context.Background(), ReadReceiptInput{MemberUUID: member, NotificationUUIDs: []string{id}})
			require.NoError(t, err)
			require.Equal(t, []string{"/deployment" + tc.expected + "/notifications", "/deployment" + tc.expected + "/notifications/read-receipts"}, paths)
		})
	}
}

func TestNotificationClientRejectsInvalidGatewayAddressesBeforeUsingCredentials(t *testing.T) {
	for _, tc := range []Config{
		{BaseURL: "https://core.test", APIPrefix: "/api/../v1"},
		{BaseURL: "https://core.test", APIPrefix: "/api//v1"},
		{BaseURL: "https://core.test", APIPrefix: "/api/v1?token=invalid"},
		{BaseURL: "https://core.test", APIPrefix: "/api/%2fv1"},
		{BaseURL: "https://core.test?"}, {BaseURL: "https://core.test#"},
		{BaseURL: "https://user@core.test"}, {BaseURL: "file:///tmp/core"},
	} {
		_, err := NewClientWithTokenProvider(tc, TokenProviderFunc(func(context.Context) (string, error) {
			t.Fatal("unexpected credential access")
			return "", nil
		}), nil)
		require.Error(t, err)
	}
}

type notificationRoundTrip func(*http.Request) (*http.Response, error)

func (f notificationRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func notificationResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestNavigationPassesThroughPublishedCreateAndActionsResponse(t *testing.T) {
	path := "/_p/plugin.test/admin/order?target=opaque"
	client, err := NewClient(Config{BaseURL: "https://core.test", BearerToken: "fixture"}, &http.Client{Transport: notificationRoundTrip(func(req *http.Request) (*http.Response, error) {
		var input CreateInput
		require.NoError(t, json.NewDecoder(req.Body).Decode(&input))
		require.Equal(t, path, input.Navigation.Path)
		require.Equal(t, "fixture_action", input.Navigation.Label)
		return notificationResponse(200, `{"data":{"id":"fixture","actions":[{"id":"view_business","label":"fixture_action","type":"primary","action":"navigate","params":{"path":"/_p/plugin.test/admin/order?target=opaque"}}]}}`), nil
	})})
	require.NoError(t, err)
	result, err := client.Create(context.Background(), CreateInput{Title: "fixture_title", Content: "fixture_content", Navigation: &Navigation{Label: "fixture_action", Path: path}})
	require.NoError(t, err)
	require.Len(t, result.Actions, 1)
	require.Equal(t, path, result.Actions[0].Params["path"])
}

func TestNavigationCannotBeSilentlyDroppedByHost(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "https://core.test", BearerToken: "fixture"}, &http.Client{Transport: notificationRoundTrip(func(*http.Request) (*http.Response, error) {
		return notificationResponse(200, `{"data":{"id":"fixture"}}`), nil
	})})
	require.NoError(t, err)
	_, err = client.Create(context.Background(), CreateInput{Title: "fixture_title", Content: "fixture_content", Navigation: &Navigation{Label: "fixture_action", Path: "/orders"}})
	var contract *HTTPError
	require.ErrorAs(t, err, &contract)
	require.Equal(t, "NOTIFICATION_NAVIGATION_RESPONSE_INVALID", contract.ReasonCode)
}
