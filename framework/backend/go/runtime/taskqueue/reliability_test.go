package taskqueue

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestHostQueueStrictAcceptance(t *testing.T) {
	for _, body := range []string{"", "{", `{}`, `{"code":200,"success":false,"data":{}}`, `{"code":503,"error_code":"FIXTURE_UNAVAILABLE","request_id":"fixture_request"}`, `{"code":200,"data":{"status":"completed","protocol_used":"rest","fallback_used":false,"result":{"contract_version":"taskqueue.v2","accepted":true,"message_id":"job","idempotency_key":"key"}}}`, `{"code":200,"data":{"status":"completed","protocol_used":"core_internal","fallback_used":true,"result":{}}}`} {
		t.Run(uuid.NewString(), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
			t.Cleanup(server.Close)
			q, err := NewHostProvider(HostProviderConfig{BaseURL: server.URL, APIKey: "fixture_key"})
			require.NoError(t, err)
			require.Error(t, q.Enqueue(context.Background(), EnqueueRequest{Message: Message{ID: "job", TenantKey: "tenant", SubscriberID: "subscriber", Topic: "topic", IdempotencyKey: "key"}}))
		})
	}
}
func TestHostQueueTypedScopeAndAcceptance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/fixture/v2/tenant/invocations", r.URL.Path)
		require.Empty(t, r.Header.Get("tenant_uuid"), "tenant identity must come from the verified service credential")
		require.Equal(t, "ApiKey fixture_key", r.Header.Get("Authorization"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, enqueueCapability, body["capability_id"])
		op := body["payload"].(map[string]any)["body"].(map[string]any)
		require.Equal(t, ContractVersion, op["contract_version"])
		require.Equal(t, "tenant", op["tenant_key"])
		require.Equal(t, "enqueue", op["operation"])
		_, _ = w.Write([]byte(`{"code":200,"data":{"status":"completed","protocol_used":"core_internal","fallback_used":false,"result":{"contract_version":"taskqueue.v2","accepted":true,"message_id":"job","idempotency_key":"key"}}}`))
	}))
	t.Cleanup(server.Close)
	q, err := NewHostProvider(HostProviderConfig{BaseURL: server.URL, APIPrefix: "/fixture/v2", APIKey: "fixture_key", TenantUUID: "tenant"})
	require.NoError(t, err)
	require.NoError(t, q.Enqueue(context.Background(), EnqueueRequest{Message: Message{ID: "job", TenantKey: "tenant", SubscriberID: "subscriber", Topic: "topic", IdempotencyKey: "key"}}))
}
func TestRedisQueueReliableReceiptAndImmutableSubmission(t *testing.T) {
	url := os.Getenv("TASKQUEUE_TEST_REDIS_URL")
	if url == "" {
		t.Skip("TASK_QUEUE_ISOLATED_REDIS_REQUIRED")
	}
	ctx := context.Background()
	prefix := "r1_fixture_" + uuid.NewString()
	q, err := NewRedisProvider(RedisProviderConfig{RedisURL: url, Prefix: prefix, ProcessingExpiry: time.Second})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, q.Close()) })
	m := Message{ID: "job", TenantKey: "tenant", SubscriberID: "subscriber", Topic: "topic", Payload: []byte(`{"v":1}`), IdempotencyKey: "key"}
	require.NoError(t, q.Enqueue(ctx, EnqueueRequest{Message: m}))
	require.NoError(t, q.Enqueue(ctx, EnqueueRequest{Message: m}))
	changed := m
	changed.Payload = []byte(`{"v":2}`)
	require.Error(t, q.Enqueue(ctx, EnqueueRequest{Message: changed}))
	changed = m
	changed.ID = "other"
	require.Error(t, q.Enqueue(ctx, EnqueueRequest{Message: changed}))
	a, err := q.Dequeue(ctx, DequeueRequest{TenantKey: m.TenantKey, SubscriberID: m.SubscriberID, WaitTimeout: time.Millisecond})
	require.NoError(t, err)
	require.Len(t, a, 1)
	require.NotNil(t, a[0].Lease)
	lease, err := q.Renew(ctx, RenewRequest{TenantKey: m.TenantKey, SubscriberID: m.SubscriberID, MessageID: m.ID, ReceiptUUID: a[0].Lease.ReceiptUUID})
	require.NoError(t, err)
	require.Equal(t, a[0].Lease.Fence, lease.Fence)
	time.Sleep(1200 * time.Millisecond)
	other, err := NewRedisProvider(RedisProviderConfig{RedisURL: url, Prefix: prefix, ProcessingExpiry: time.Second})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, other.Close()) })
	b, err := other.Dequeue(ctx, DequeueRequest{TenantKey: m.TenantKey, SubscriberID: m.SubscriberID, WaitTimeout: time.Millisecond})
	require.NoError(t, err)
	require.Len(t, b, 1)
	require.NotEqual(t, a[0].Lease.ReceiptUUID, b[0].Lease.ReceiptUUID)
	require.Greater(t, b[0].Lease.Fence, a[0].Lease.Fence)
	ack := AckRequest{TenantKey: m.TenantKey, SubscriberID: m.SubscriberID, MessageID: m.ID, ReceiptUUID: a[0].Lease.ReceiptUUID, Outcome: "completed"}
	require.Error(t, q.Ack(ctx, ack))
	ack.ReceiptUUID = b[0].Lease.ReceiptUUID
	require.NoError(t, other.Ack(ctx, ack))
	require.NoError(t, other.Ack(ctx, ack))
	require.NoError(t, q.Enqueue(ctx, EnqueueRequest{Message: m}))
	empty, err := q.Dequeue(ctx, DequeueRequest{TenantKey: m.TenantKey, SubscriberID: m.SubscriberID, WaitTimeout: time.Millisecond})
	require.NoError(t, err)
	require.Empty(t, empty)
	otherTenant := m
	otherTenant.TenantKey = "other_tenant"
	require.NoError(t, q.Enqueue(ctx, EnqueueRequest{Message: otherTenant}))
	cross := ack
	cross.TenantKey = otherTenant.TenantKey
	require.Error(t, q.Ack(ctx, cross))
}
