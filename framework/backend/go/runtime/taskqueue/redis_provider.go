package taskqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultRedisPrefix            = "event_fabric:task"
	defaultRedisBlockingTimeout   = 3 * time.Second
	defaultRedisProcessingExpires = 30 * time.Minute
)

type RedisProviderConfig struct {
	RedisURL         string
	Prefix           string
	BlockingTimeout  time.Duration
	ProcessingExpiry time.Duration
}

type RedisProvider struct {
	client           *redis.Client
	prefix           string
	blockingTimeout  time.Duration
	processingExpiry time.Duration
}

func NewRedisProvider(cfg RedisProviderConfig) (*RedisProvider, error) {
	if strings.TrimSpace(cfg.RedisURL) == "" {
		return nil, errors.New("taskqueue redis provider: redis_url is required")
	}
	opt, err := redis.ParseURL(strings.TrimSpace(cfg.RedisURL))
	if err != nil {
		return nil, fmt.Errorf("taskqueue redis provider: invalid redis_url: %w", err)
	}
	prefix := strings.TrimSpace(cfg.Prefix)
	if prefix == "" {
		prefix = defaultRedisPrefix
	}
	blocking := cfg.BlockingTimeout
	if blocking <= 0 {
		blocking = defaultRedisBlockingTimeout
	}
	expiry := cfg.ProcessingExpiry
	if expiry <= 0 {
		expiry = defaultRedisProcessingExpires
	}
	return &RedisProvider{
		client:           redis.NewClient(opt),
		prefix:           prefix,
		blockingTimeout:  blocking,
		processingExpiry: expiry,
	}, nil
}

func (p *RedisProvider) Enqueue(ctx context.Context, req EnqueueRequest) error {
	if err := p.validate(); err != nil {
		return err
	}
	msg := req.Message
	if err := validateMessage(msg); err != nil {
		return err
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if !msg.VisibleAt.IsZero() && msg.VisibleAt.After(time.Now().UTC()) {
		return p.client.ZAdd(ctx, p.delayKey(msg.TenantKey, msg.SubscriberID), redis.Z{
			Score:  float64(msg.VisibleAt.UnixMilli()),
			Member: string(raw),
		}).Err()
	}
	return p.client.LPush(ctx, p.queueKey(msg.TenantKey, msg.SubscriberID), string(raw)).Err()
}

func (p *RedisProvider) Dequeue(ctx context.Context, req DequeueRequest) ([]Message, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.TenantKey) == "" || strings.TrimSpace(req.SubscriberID) == "" {
		return nil, errors.New("tenant_key and subscriber_id are required")
	}
	maxItems := req.MaxItems
	if maxItems <= 0 {
		maxItems = 1
	}
	wait := req.WaitTimeout
	if wait <= 0 {
		wait = p.blockingTimeout
	}
	if err := p.flushDelayed(ctx, req.TenantKey, req.SubscriberID); err != nil {
		return nil, err
	}
	queueKey := p.queueKey(req.TenantKey, req.SubscriberID)
	processingKey := p.processingKey(req.TenantKey, req.SubscriberID)
	raw, err := p.client.BRPopLPush(ctx, queueKey, processingKey, wait).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	out := make([]Message, 0, maxItems)
	first, err := p.bindInflight(ctx, req.TenantKey, req.SubscriberID, raw)
	if err != nil {
		return nil, err
	}
	out = append(out, first)
	for len(out) < maxItems {
		nextRaw, popErr := p.client.RPopLPush(ctx, queueKey, processingKey).Result()
		if popErr != nil {
			if popErr == redis.Nil {
				break
			}
			return nil, popErr
		}
		msg, bindErr := p.bindInflight(ctx, req.TenantKey, req.SubscriberID, nextRaw)
		if bindErr != nil {
			return nil, bindErr
		}
		out = append(out, msg)
	}
	return out, nil
}

func (p *RedisProvider) Ack(ctx context.Context, req AckRequest) error {
	if err := p.validateAck(req.TenantKey, req.SubscriberID, req.MessageID); err != nil {
		return err
	}
	raw, err := p.client.HGet(ctx, p.inflightKey(req.TenantKey, req.SubscriberID), req.MessageID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return err
	}
	if err := p.client.LRem(ctx, p.processingKey(req.TenantKey, req.SubscriberID), 1, raw).Err(); err != nil {
		return err
	}
	return p.client.HDel(ctx, p.inflightKey(req.TenantKey, req.SubscriberID), req.MessageID).Err()
}

func (p *RedisProvider) Nack(ctx context.Context, req NackRequest) error {
	if err := p.validateAck(req.TenantKey, req.SubscriberID, req.MessageID); err != nil {
		return err
	}
	raw, err := p.client.HGet(ctx, p.inflightKey(req.TenantKey, req.SubscriberID), req.MessageID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return err
	}
	if err := p.client.LRem(ctx, p.processingKey(req.TenantKey, req.SubscriberID), 1, raw).Err(); err != nil {
		return err
	}
	if err := p.client.HDel(ctx, p.inflightKey(req.TenantKey, req.SubscriberID), req.MessageID).Err(); err != nil {
		return err
	}
	if !req.RetryAt.IsZero() && req.RetryAt.After(time.Now().UTC()) {
		return p.client.ZAdd(ctx, p.delayKey(req.TenantKey, req.SubscriberID), redis.Z{
			Score:  float64(req.RetryAt.UnixMilli()),
			Member: raw,
		}).Err()
	}
	return p.client.LPush(ctx, p.queueKey(req.TenantKey, req.SubscriberID), raw).Err()
}

func (p *RedisProvider) Retry(ctx context.Context, req RetryRequest) error {
	msg := req.Message
	if msg.VisibleAt.IsZero() {
		msg.VisibleAt = req.RetryAt
	}
	return p.Enqueue(ctx, EnqueueRequest{Message: msg})
}

func (p *RedisProvider) validate() error {
	if p == nil || p.client == nil {
		return errors.New("taskqueue redis provider is not configured")
	}
	return nil
}

func (p *RedisProvider) validateAck(tenantKey, subscriberID, messageID string) error {
	if err := p.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(tenantKey) == "" || strings.TrimSpace(subscriberID) == "" || strings.TrimSpace(messageID) == "" {
		return errors.New("tenant_key, subscriber_id and message_id are required")
	}
	return nil
}

func validateMessage(msg Message) error {
	if strings.TrimSpace(msg.ID) == "" || strings.TrimSpace(msg.TenantKey) == "" || strings.TrimSpace(msg.SubscriberID) == "" {
		return errors.New("message id, tenant_key and subscriber_id are required")
	}
	return nil
}

func (p *RedisProvider) bindInflight(ctx context.Context, tenantKey, subscriberID, raw string) (Message, error) {
	var msg Message
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return Message{}, err
	}
	if strings.TrimSpace(msg.ID) == "" {
		return Message{}, errors.New("message id is required")
	}
	if err := p.client.HSet(ctx, p.inflightKey(tenantKey, subscriberID), msg.ID, raw).Err(); err != nil {
		return Message{}, err
	}
	if err := p.client.Expire(ctx, p.inflightKey(tenantKey, subscriberID), p.processingExpiry).Err(); err != nil {
		return Message{}, err
	}
	return msg, nil
}

func (p *RedisProvider) flushDelayed(ctx context.Context, tenantKey, subscriberID string) error {
	key := p.delayKey(tenantKey, subscriberID)
	now := fmt.Sprintf("%d", time.Now().UTC().UnixMilli())
	items, err := p.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{Min: "-inf", Max: now, Offset: 0, Count: 100}).Result()
	if err != nil {
		return err
	}
	for _, raw := range items {
		if err := p.client.LPush(ctx, p.queueKey(tenantKey, subscriberID), raw).Err(); err != nil {
			return err
		}
		if err := p.client.ZRem(ctx, key, raw).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (p *RedisProvider) queueKey(tenantKey, subscriberID string) string {
	return p.prefix + ":pending:" + strings.TrimSpace(tenantKey) + ":" + strings.TrimSpace(subscriberID)
}

func (p *RedisProvider) processingKey(tenantKey, subscriberID string) string {
	return p.prefix + ":processing:" + strings.TrimSpace(tenantKey) + ":" + strings.TrimSpace(subscriberID)
}

func (p *RedisProvider) inflightKey(tenantKey, subscriberID string) string {
	return p.prefix + ":inflight:" + strings.TrimSpace(tenantKey) + ":" + strings.TrimSpace(subscriberID)
}

func (p *RedisProvider) delayKey(tenantKey, subscriberID string) string {
	return p.prefix + ":deferred:" + strings.TrimSpace(tenantKey) + ":" + strings.TrimSpace(subscriberID)
}
