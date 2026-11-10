// Package eventfabric implements event_fabric.service.v1 over the fixed Core
// internal capability binding. Business transactions and deduplication belong
// to the consumer, not to this transport.
package eventfabric

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
)

const (
	Contract          = "event_fabric.service.v1"
	PublishCapability = "com.corex.eventfabric.service_publish"
	ConsumeCapability = "com.corex.eventfabric.service_consume"
	Protocol          = "core_internal"
)

type Config = capability.Config
type TokenProvider = capability.TokenProvider
type TokenProviderFunc = capability.TokenProviderFunc
type HTTPError = capability.HTTPError

type Event struct {
	EventID        string            `json:"event_id"`
	IdempotencyKey string            `json:"idempotency_key"`
	Topic          string            `json:"topic"`
	Version        string            `json:"version"`
	Payload        json.RawMessage   `json:"payload"`
	TraceID        string            `json:"trace_id,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
}
type PublishRequest struct {
	TenantUUID string
	PluginID   string
	Event      Event
}
type PublishReceipt struct {
	ContractVersion string `json:"contract_version"`
	Accepted        bool   `json:"accepted"`
	EventID         string `json:"event_id"`
	TraceID         string `json:"trace_id"`
}

// ResolveTopicsRequest selects a service direction without accepting a physical
// topic, free-form URL, or administrator credential.
type ResolveTopicsRequest struct {
	TenantUUID, PluginID, SubscriberID string
	CapabilityID                       string
	TopicKeys                          []string
}

type ResolvedTopics struct {
	ContractVersion string            `json:"contract_version"`
	TenantUUID      string            `json:"tenant_uuid"`
	PluginID        string            `json:"plugin_id"`
	TopicBindings   map[string]string `json:"topic_bindings"`
}
type SubscribeRequest struct {
	TenantUUID        string
	PluginID          string
	SubscriberID      string
	Topics            []string
	SupportedVersions []string
	CompatibilityMode string
	BatchSize         int
	// ExpectedPublishers is trusted startup policy, keyed by physical FullTopic.
	ExpectedPublishers map[string][]string
	ReconnectLimit     int
	ReconnectBackoff   time.Duration
}
type Delivery struct {
	ContractVersion   string            `json:"contract_version"`
	TenantUUID        string            `json:"tenant_uuid"`
	SubscriberID      string            `json:"subscriber_id"`
	DeliveryID        string            `json:"delivery_id"`
	EventID           string            `json:"event_id"`
	IdempotencyKey    string            `json:"idempotency_key"`
	Attributes        map[string]string `json:"attributes,omitempty"`
	Topic             string            `json:"topic"`
	Version           string            `json:"version"`
	Payload           json.RawMessage   `json:"payload"`
	TraceID           string            `json:"trace_id"`
	PublishedBy       string            `json:"published_by"`
	PublisherPluginID string            `json:"publisher_plugin_id"`
	Attempt           int               `json:"attempt"`
	MaxAttempts       int               `json:"max_attempts"`
	AckDeadline       time.Time         `json:"ack_deadline"`
}
type DeliveryReceipt struct {
	TenantUUID, PluginID, SubscriberID, DeliveryID string
	AckDeadline                                    time.Time
}

func (d Delivery) Receipt() DeliveryReceipt {
	return DeliveryReceipt{d.TenantUUID, d.SubscriberID, d.SubscriberID, d.DeliveryID, d.AckDeadline}
}

type NackResult struct {
	ContractVersion   string `json:"contract_version"`
	Confirmed         bool   `json:"confirmed"`
	DeliveryID        string `json:"delivery_id"`
	Outcome           string `json:"outcome"`
	RemainingAttempts int    `json:"remaining_attempts"`
	NextDelaySeconds  int    `json:"next_delay_seconds"`
}
type Subscription interface {
	Recv(context.Context) (Delivery, error)
	Close() error
}
type Service interface {
	ResolveTopics(context.Context, ResolveTopicsRequest) (*ResolvedTopics, error)
	Publish(context.Context, PublishRequest) (*PublishReceipt, error)
	Subscribe(context.Context, SubscribeRequest) (Subscription, error)
	Ack(context.Context, DeliveryReceipt) error
	Nack(context.Context, DeliveryReceipt, string) (*NackResult, error)
}

// SettlementError never implies that a committed business transaction failed.
// Uncertain means the server may already have persisted the receipt.
type SettlementError struct {
	Receipt   DeliveryReceipt
	Uncertain bool
	Err       error
}

func (e *SettlementError) Error() string { return e.Err.Error() }
func (e *SettlementError) Unwrap() error { return e.Err }
func failure(status int, code string) *HTTPError {
	return &HTTPError{StatusCode: status, ErrorCode: code, ReasonCode: code}
}
func uncertain(err error) bool {
	var upstream *HTTPError
	return !errors.As(err, &upstream) || upstream.StatusCode >= 500 || upstream.StatusCode == 0
}
