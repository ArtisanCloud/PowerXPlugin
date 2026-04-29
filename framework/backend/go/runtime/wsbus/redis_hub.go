package wsbus

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const defaultRedisChannel = "powerx.wsbus"

type RedisHubConfig struct {
	RedisURL string
	Channel  string
	Timeout  time.Duration
	Logger   *slog.Logger
}

type redisEnvelope struct {
	Topic      string `json:"topic"`
	Payload    any    `json:"payload"`
	TenantUUID string `json:"tenant_uuid"`
	TraceID    string `json:"trace_id"`
	Origin     string `json:"origin"`
}

type RedisHub struct {
	client     *redis.Client
	channel    string
	instanceID string
	local      *MemoryHub
	logger     *slog.Logger
	logAdapter runtimelogging.Logger

	subOnce sync.Once
	subErr  error
}

func NewRedisHub(cfg RedisHubConfig) (*RedisHub, error) {
	if strings.TrimSpace(cfg.RedisURL) == "" {
		return nil, errors.New("wsbus redis: redis url is required")
	}
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	channel := strings.TrimSpace(cfg.Channel)
	if channel == "" {
		channel = defaultRedisChannel
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &RedisHub{
		client:     client,
		channel:    channel,
		instanceID: uuid.NewString(),
		local:      NewMemoryHub(),
		logger:     logger,
		logAdapter: runtimelogging.NewSlogAdapter(logger),
	}, nil
}

func (h *RedisHub) Publish(ctx context.Context, topic string, payload any, opts PublishOptions) error {
	if h == nil || h.client == nil {
		return errors.New("wsbus redis: client not configured")
	}
	env := redisEnvelope{
		Topic:      topic,
		Payload:    payload,
		TenantUUID: opts.TenantUUID,
		TraceID:    opts.TraceID,
		Origin:     h.instanceID,
	}
	body, err := json.Marshal(&env)
	if err != nil {
		h.logPublish(topic, opts, runtimelogging.StatusFailed, "marshal_failed")
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := h.client.Publish(ctx, h.channel, body).Err(); err != nil {
		h.logPublish(topic, opts, runtimelogging.StatusFailed, "redis_publish_failed")
		return err
	}
	h.logPublish(topic, opts, runtimelogging.StatusSucceeded, "")
	_ = h.local.Publish(ctx, topic, payload, opts)
	return nil
}

func (h *RedisHub) Subscribe(topic string, handler func(Event)) func() {
	if h == nil {
		return func() {}
	}
	return h.local.Subscribe(topic, handler)
}

// Start begins consuming redis pub/sub events and re-emits to local subscribers.
func (h *RedisHub) Start(ctx context.Context) error {
	if h == nil || h.client == nil {
		return errors.New("wsbus redis: client not configured")
	}
	h.subOnce.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		pubsub := h.client.Subscribe(ctx, h.channel)
		go func() {
			defer pubsub.Close()
			ch := pubsub.Channel()
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-ch:
					if !ok {
						return
					}
					h.handleMessage(ctx, msg.Payload)
				}
			}
		}()
	})
	return h.subErr
}

func (h *RedisHub) Close() error {
	if h == nil || h.client == nil {
		return nil
	}
	return h.client.Close()
}

func (h *RedisHub) handleMessage(ctx context.Context, payload string) {
	if strings.TrimSpace(payload) == "" {
		return
	}
	var env redisEnvelope
	if err := json.Unmarshal([]byte(payload), &env); err != nil {
		fields := runtimelogging.NormalizeRuntimeFields(runtimelogging.Fields{
			runtimelogging.FieldTraceID:    "unknown",
			runtimelogging.FieldTaskID:     "unknown",
			runtimelogging.FieldTenantUUID: "unknown",
			runtimelogging.FieldTenantKey:  "unknown",
			runtimelogging.FieldSubscriber: "wsbus.redis_hub",
			runtimelogging.FieldTopic:      "unknown",
			runtimelogging.FieldStatus:     runtimelogging.StatusFailed,
			runtimelogging.FieldReason:     "invalid_payload",
		})
		runtimelogging.NewFacade(nil, h.logAdapter).Warn("wsbus redis: invalid payload", runtimelogging.Entry{
			Fields: fields,
			Context: runtimelogging.Fields{
				"module":     "wsbus.redis_hub",
				"biz_scene":  "ws_publish",
				"biz_domain": "runtime",
			},
		})
		return
	}
	if env.Origin != "" && env.Origin == h.instanceID {
		return
	}
	opts := PublishOptions{
		TenantUUID: strings.TrimSpace(env.TenantUUID),
		TraceID:    strings.TrimSpace(env.TraceID),
	}
	_ = h.local.Publish(ctx, env.Topic, env.Payload, opts)
}

func (h *RedisHub) logPublish(topic string, opts PublishOptions, status, reason string) {
	if h == nil || h.logAdapter == nil {
		return
	}
	fields := runtimelogging.NormalizeRuntimeFields(runtimelogging.Fields{
		runtimelogging.FieldTraceID:    strings.TrimSpace(opts.TraceID),
		runtimelogging.FieldTaskID:     strings.TrimSpace(opts.TraceID),
		runtimelogging.FieldTenantUUID: strings.TrimSpace(opts.TenantUUID),
		runtimelogging.FieldTenantKey:  runtimelogging.TenantKeyFromUUID(opts.TenantUUID),
		runtimelogging.FieldSubscriber: "wsbus.redis_hub",
		runtimelogging.FieldTopic:      strings.TrimSpace(topic),
		runtimelogging.FieldStatus:     status,
		runtimelogging.FieldReason:     reason,
	})
	runtimelogging.NewFacade(nil, h.logAdapter).Info("wsbus redis publish", runtimelogging.Entry{
		Fields: fields,
		Context: runtimelogging.Fields{
			"module":     "wsbus.redis_hub",
			"biz_scene":  "ws_publish",
			"biz_domain": "runtime",
		},
	})
}
