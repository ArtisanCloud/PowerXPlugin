package notifications

import (
	"context"
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
		require.Equal(t, "/api/v1/tenant/notifications", req.URL.Path)
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

type notificationRoundTrip func(*http.Request) (*http.Response, error)

func (f notificationRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func notificationResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}
