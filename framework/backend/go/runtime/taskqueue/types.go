package taskqueue

import (
	"context"
	"time"
)

const ContractVersion = "taskqueue.v2"

type Lease struct {
	ReceiptUUID string    `json:"receipt_uuid"`
	Fence       uint64    `json:"fence"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type Message struct {
	Lease          *Lease            `json:"lease,omitempty"`
	ID             string            `json:"id"`
	TenantKey      string            `json:"tenant_key"`
	SubscriberID   string            `json:"subscriber_id"`
	Topic          string            `json:"topic"`
	Payload        []byte            `json:"payload"`
	Headers        map[string]string `json:"headers,omitempty"`
	Attempt        int               `json:"attempt,omitempty"`
	TraceID        string            `json:"trace_id,omitempty"`
	VisibleAt      time.Time         `json:"visible_at,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

type EnqueueRequest struct {
	Message Message `json:"message"`
}

type DequeueRequest struct {
	TenantKey    string        `json:"tenant_key"`
	SubscriberID string        `json:"subscriber_id"`
	MaxItems     int           `json:"max_items,omitempty"`
	WaitTimeout  time.Duration `json:"wait_timeout,omitempty"`
}

type AckRequest struct {
	Outcome      string            `json:"outcome"`
	ReceiptUUID  string            `json:"receipt_uuid"`
	TenantKey    string            `json:"tenant_key"`
	SubscriberID string            `json:"subscriber_id"`
	MessageID    string            `json:"message_id"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type NackRequest struct {
	ReceiptUUID  string            `json:"receipt_uuid"`
	TenantKey    string            `json:"tenant_key"`
	SubscriberID string            `json:"subscriber_id"`
	MessageID    string            `json:"message_id"`
	Reason       string            `json:"reason,omitempty"`
	RetryAt      time.Time         `json:"retry_at,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type RetryRequest struct {
	Message Message   `json:"message"`
	RetryAt time.Time `json:"retry_at,omitempty"`
	Reason  string    `json:"reason,omitempty"`
}

type RenewRequest struct {
	TenantKey    string `json:"tenant_key"`
	SubscriberID string `json:"subscriber_id"`
	MessageID    string `json:"message_id"`
	ReceiptUUID  string `json:"receipt_uuid"`
}

type Queue interface {
	Renew(context.Context, RenewRequest) (Lease, error)
	Enqueue(ctx context.Context, req EnqueueRequest) error
	Dequeue(ctx context.Context, req DequeueRequest) ([]Message, error)
	Ack(ctx context.Context, req AckRequest) error
	Nack(ctx context.Context, req NackRequest) error
	Retry(ctx context.Context, req RetryRequest) error
}
