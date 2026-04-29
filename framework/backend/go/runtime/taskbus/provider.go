package taskbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/redis/go-redis/v9"
)

type HostProviderConfig struct {
	BaseURL        string
	APIPrefix      string
	AuthScheme     string
	Token          string
	APIKey         string
	TenantUUID     string
	UserAgent      string
	Timeout        time.Duration
	PayloadVersion string
	SourcePlugin   string
}

type HostProvider struct {
	cfg HostProviderConfig
}

func NewHostProvider(cfg HostProviderConfig) *HostProvider {
	return &HostProvider{cfg: cfg}
}

func NewHostProviderFromApp(app *bootstrap.App, sourcePlugin, payloadVersion string) *HostProvider {
	if app == nil || app.Config == nil {
		return NewHostProvider(HostProviderConfig{
			SourcePlugin:   strings.TrimSpace(sourcePlugin),
			PayloadVersion: strings.TrimSpace(payloadVersion),
		})
	}

	timeout := app.Config.Gateway.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return NewHostProvider(HostProviderConfig{
		BaseURL:        strings.TrimSpace(app.Config.Gateway.BaseURL),
		APIPrefix:      strings.TrimSpace(app.Config.Gateway.APIPrefix),
		AuthScheme:     strings.TrimSpace(app.Config.Gateway.AuthScheme),
		Token:          strings.TrimSpace(app.Config.Gateway.ToolToken),
		APIKey:         strings.TrimSpace(app.Config.Gateway.APIKey),
		TenantUUID:     strings.TrimSpace(app.Config.Gateway.TenantID),
		UserAgent:      strings.TrimSpace(app.Config.Gateway.UserAgent),
		Timeout:        timeout,
		SourcePlugin:   strings.TrimSpace(sourcePlugin),
		PayloadVersion: strings.TrimSpace(payloadVersion),
	})
}

func (p *HostProvider) NewEmitter() (eventbridge.Emitter, error) {
	if p == nil {
		return nil, errors.New("taskbus host provider is nil")
	}

	client, err := wsbus.NewHostClient(wsbus.HostClientConfig{
		BaseURL:    p.cfg.BaseURL,
		APIPrefix:  p.cfg.APIPrefix,
		AuthScheme: p.cfg.AuthScheme,
		Token:      p.cfg.Token,
		APIKey:     p.cfg.APIKey,
		TenantUUID: p.cfg.TenantUUID,
		UserAgent:  p.cfg.UserAgent,
		Timeout:    p.cfg.Timeout,
	})
	if err != nil {
		return nil, err
	}

	metaBuilder := event.NewMetaBuilder(p.cfg.SourcePlugin, p.cfg.PayloadVersion)
	return &hostEmitter{
		publisher:   wsbus.NewAdapter(client, p.cfg.TenantUUID, nil),
		metaBuilder: metaBuilder,
		logger:      runtimelogging.NewSlogAdapter(slog.Default()),
	}, nil
}

type hostEmitter struct {
	publisher   wsbus.Publisher
	metaBuilder event.MetaBuilder
	logger      runtimelogging.Logger
}

func (e *hostEmitter) Emit(ctx context.Context, ev event.Event) error {
	if e == nil || e.publisher == nil {
		return errors.New("taskbus host emitter is not configured")
	}

	topic := strings.TrimSpace(string(ev.Topic))
	if topic == "" {
		return errors.New("event topic is required")
	}

	meta, err := e.ensureMeta(ev.Meta)
	if err != nil {
		return err
	}

	payload, err := decodePayload(ev.Payload)
	if err != nil {
		return err
	}

	result := e.publisher.Publish(ctx, topic, payload, wsbus.PublishOptions{
		TenantUUID: meta.TenantUUID,
		TraceID:    firstNonEmpty(meta.TraceID, meta.RequestID),
	})
	if !result.OK {
		e.logEmit(topic, meta, runtimelogging.StatusFailed, result.ErrorCode)
		return fmt.Errorf("taskbus host publish failed: %s (%s)", result.ErrorCode, result.ErrorMessage)
	}
	e.logEmit(topic, meta, runtimelogging.StatusSucceeded, "")

	return nil
}

func (e *hostEmitter) logEmit(topic string, meta event.Meta, status, reason string) {
	if e == nil || e.logger == nil {
		return
	}
	fields := runtimelogging.NormalizeRuntimeFields(runtimelogging.Fields{
		runtimelogging.FieldTraceID:    firstNonEmpty(meta.TraceID, meta.RequestID),
		runtimelogging.FieldTaskID:     firstNonEmpty(meta.RequestID, meta.TraceID),
		runtimelogging.FieldTenantUUID: strings.TrimSpace(meta.TenantUUID),
		runtimelogging.FieldTenantKey:  runtimelogging.TenantKeyFromUUID(meta.TenantUUID),
		runtimelogging.FieldSubscriber: "taskbus.host_emitter",
		runtimelogging.FieldTopic:      strings.TrimSpace(topic),
		runtimelogging.FieldStatus:     status,
		runtimelogging.FieldReason:     reason,
	})
	facade := runtimelogging.NewFacade(nil, e.logger)
	if status == runtimelogging.StatusFailed {
		facade.Error("taskbus host emit failed", runtimelogging.Entry{
			Fields: fields,
			Context: runtimelogging.Fields{
				"module":     "taskbus.host_emitter",
				"plugin_id":  strings.TrimSpace(meta.SourcePlugin),
				"biz_scene":  "event_emit",
				"biz_domain": "runtime",
			},
		})
		return
	}
	facade.Info("taskbus host emit completed", runtimelogging.Entry{
		Fields: fields,
		Context: runtimelogging.Fields{
			"module":     "taskbus.host_emitter",
			"plugin_id":  strings.TrimSpace(meta.SourcePlugin),
			"biz_scene":  "event_emit",
			"biz_domain": "runtime",
		},
	})
}

func (e *hostEmitter) ensureMeta(meta event.Meta) (event.Meta, error) {
	tenantUUID := strings.TrimSpace(meta.TenantUUID)
	requestID := strings.TrimSpace(meta.RequestID)
	traceID := strings.TrimSpace(meta.TraceID)

	if tenantUUID == "" {
		return event.Meta{}, errors.New("event tenant_uuid is required")
	}

	if requestID != "" && traceID != "" {
		if strings.TrimSpace(meta.SourcePlugin) == "" {
			meta.SourcePlugin = e.metaBuilder.SourcePlugin
		}
		if strings.TrimSpace(meta.PayloadVersion) == "" {
			meta.PayloadVersion = e.metaBuilder.PayloadVersion
		}
		if meta.OccurredAt.IsZero() {
			meta.OccurredAt = e.metaBuilder.Now()
		}
		return meta, nil
	}

	built, err := e.metaBuilder.Build(tenantUUID, requestID, traceID)
	if err != nil {
		return event.Meta{}, err
	}
	return built, nil
}

func decodePayload(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}

	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, errors.New("event payload must be valid JSON")
	}
	if payload == nil {
		return map[string]any{}, nil
	}
	return payload, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type RedisProviderConfig struct {
	RedisURL       string
	StreamKey      string
	SourcePlugin   string
	PayloadVersion string
	MaxLenApprox   int64
}

type RedisProvider struct {
	cfg RedisProviderConfig
}

func NewRedisProvider(cfg RedisProviderConfig) *RedisProvider {
	return &RedisProvider{cfg: cfg}
}

func (p *RedisProvider) NewEmitter() (eventbridge.Emitter, error) {
	if p == nil {
		return nil, errors.New("taskbus redis provider is nil")
	}
	redisURL := strings.TrimSpace(p.cfg.RedisURL)
	if redisURL == "" {
		return nil, errors.New("taskbus redis provider: redis_url is required")
	}
	streamKey := strings.TrimSpace(p.cfg.StreamKey)
	if streamKey == "" {
		streamKey = "powerx.taskbus.events"
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("taskbus redis provider: invalid redis_url: %w", err)
	}
	client := redis.NewClient(opts)
	metaBuilder := event.NewMetaBuilder(p.cfg.SourcePlugin, p.cfg.PayloadVersion)
	return &redisEmitter{
		client:       client,
		streamKey:    streamKey,
		metaBuilder:  metaBuilder,
		maxLenApprox: p.cfg.MaxLenApprox,
		logger:       runtimelogging.NewSlogAdapter(slog.Default()),
	}, nil
}

type redisEmitter struct {
	client       *redis.Client
	streamKey    string
	metaBuilder  event.MetaBuilder
	maxLenApprox int64
	logger       runtimelogging.Logger
}

func (e *redisEmitter) Emit(ctx context.Context, ev event.Event) error {
	if e == nil || e.client == nil {
		return errors.New("taskbus redis emitter is not configured")
	}
	topic := strings.TrimSpace(string(ev.Topic))
	if topic == "" {
		return errors.New("event topic is required")
	}
	meta, err := ensureMeta(e.metaBuilder, ev.Meta)
	if err != nil {
		return err
	}
	payload, err := decodePayload(ev.Payload)
	if err != nil {
		return err
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return errors.New("event payload must be JSON serializable")
	}
	args := &redis.XAddArgs{
		Stream: e.streamKey,
		Values: map[string]any{
			"topic":           topic,
			"tenant_uuid":     meta.TenantUUID,
			"request_id":      meta.RequestID,
			"trace_id":        meta.TraceID,
			"source_plugin":   meta.SourcePlugin,
			"payload_version": meta.PayloadVersion,
			"occurred_at":     meta.OccurredAt.UTC().Format(time.RFC3339Nano),
			"payload":         string(payloadBytes),
		},
	}
	if e.maxLenApprox > 0 {
		args.MaxLen = e.maxLenApprox
		args.Approx = true
	}
	if err := e.client.XAdd(ctx, args).Err(); err != nil {
		e.logEmit(topic, meta, runtimelogging.StatusFailed, "redis_enqueue_failed")
		return fmt.Errorf("taskbus redis enqueue failed: %w", err)
	}
	e.logEmit(topic, meta, runtimelogging.StatusQueued, "")
	return nil
}

func (e *redisEmitter) logEmit(topic string, meta event.Meta, status, reason string) {
	if e == nil || e.logger == nil {
		return
	}
	fields := runtimelogging.NormalizeRuntimeFields(runtimelogging.Fields{
		runtimelogging.FieldTraceID:    firstNonEmpty(meta.TraceID, meta.RequestID),
		runtimelogging.FieldTaskID:     firstNonEmpty(meta.RequestID, meta.TraceID),
		runtimelogging.FieldTenantUUID: strings.TrimSpace(meta.TenantUUID),
		runtimelogging.FieldTenantKey:  runtimelogging.TenantKeyFromUUID(meta.TenantUUID),
		runtimelogging.FieldSubscriber: "taskbus.redis_emitter",
		runtimelogging.FieldTopic:      strings.TrimSpace(topic),
		runtimelogging.FieldStatus:     status,
		runtimelogging.FieldReason:     reason,
	})
	facade := runtimelogging.NewFacade(nil, e.logger)
	if status == runtimelogging.StatusFailed {
		facade.Error("taskbus redis emit failed", runtimelogging.Entry{
			Fields: fields,
			Context: runtimelogging.Fields{
				"module":     "taskbus.redis_emitter",
				"biz_scene":  "event_emit",
				"biz_domain": "runtime",
			},
		})
		return
	}
	facade.Info("taskbus redis emit queued", runtimelogging.Entry{
		Fields: fields,
		Context: runtimelogging.Fields{
			"module":     "taskbus.redis_emitter",
			"biz_scene":  "event_emit",
			"biz_domain": "runtime",
		},
	})
}

type RedisConsumerConfig struct {
	RedisURL    string
	StreamKey   string
	Group       string
	Consumer    string
	Block       time.Duration
	BatchSize   int64
	StartID     string
	DeleteOnAck bool
}

type RedisConsumer struct {
	client      *redis.Client
	streamKey   string
	group       string
	consumer    string
	block       time.Duration
	batchSize   int64
	startID     string
	deleteOnAck bool
}

func NewRedisConsumer(cfg RedisConsumerConfig) (*RedisConsumer, error) {
	redisURL := strings.TrimSpace(cfg.RedisURL)
	if redisURL == "" {
		return nil, errors.New("taskbus redis consumer: redis_url is required")
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("taskbus redis consumer: invalid redis_url: %w", err)
	}
	streamKey := strings.TrimSpace(cfg.StreamKey)
	if streamKey == "" {
		streamKey = "powerx.taskbus.events"
	}
	group := strings.TrimSpace(cfg.Group)
	if group == "" {
		group = "powerx.plugin"
	}
	consumer := strings.TrimSpace(cfg.Consumer)
	if consumer == "" {
		consumer = defaultRedisConsumerID()
	}
	block := cfg.Block
	if block <= 0 {
		block = 2 * time.Second
	}
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}
	startID := strings.TrimSpace(cfg.StartID)
	if startID == "" {
		startID = "$"
	}
	client := redis.NewClient(opts)
	return &RedisConsumer{
		client:      client,
		streamKey:   streamKey,
		group:       group,
		consumer:    consumer,
		block:       block,
		batchSize:   batchSize,
		startID:     startID,
		deleteOnAck: cfg.DeleteOnAck,
	}, nil
}

func (c *RedisConsumer) Consume(ctx context.Context, handler func(context.Context, event.Event) error) error {
	if c == nil || c.client == nil {
		return errors.New("taskbus redis consumer is not configured")
	}
	if handler == nil {
		return errors.New("taskbus redis consumer handler is required")
	}
	if err := c.ensureGroup(ctx); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.group,
			Consumer: c.consumer,
			Streams:  []string{c.streamKey, ">"},
			Count:    c.batchSize,
			Block:    c.block,
			NoAck:    false,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				ev, parseErr := parseRedisEvent(message.Values)
				if parseErr != nil {
					_ = c.client.XAck(ctx, c.streamKey, c.group, message.ID).Err()
					if c.deleteOnAck {
						_ = c.client.XDel(ctx, c.streamKey, message.ID).Err()
					}
					continue
				}
				if err := handler(ctx, ev); err != nil {
					continue
				}
				if err := c.client.XAck(ctx, c.streamKey, c.group, message.ID).Err(); err != nil {
					continue
				}
				if c.deleteOnAck {
					_ = c.client.XDel(ctx, c.streamKey, message.ID).Err()
				}
			}
		}
	}
}

func (c *RedisConsumer) ensureGroup(ctx context.Context) error {
	if c == nil || c.client == nil {
		return errors.New("taskbus redis consumer is not configured")
	}
	err := c.client.XGroupCreateMkStream(ctx, c.streamKey, c.group, c.startID).Err()
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToUpper(err.Error()), "BUSYGROUP") {
		return nil
	}
	return err
}

func parseRedisEvent(values map[string]any) (event.Event, error) {
	topic := strings.TrimSpace(redisValueString(values["topic"]))
	if topic == "" {
		return event.Event{}, errors.New("taskbus redis message missing topic")
	}
	tenantUUID := strings.TrimSpace(redisValueString(values["tenant_uuid"]))
	if tenantUUID == "" {
		return event.Event{}, errors.New("taskbus redis message missing tenant_uuid")
	}
	requestID := strings.TrimSpace(redisValueString(values["request_id"]))
	traceID := strings.TrimSpace(redisValueString(values["trace_id"]))
	sourcePlugin := strings.TrimSpace(redisValueString(values["source_plugin"]))
	payloadVersion := strings.TrimSpace(redisValueString(values["payload_version"]))
	occurredRaw := strings.TrimSpace(redisValueString(values["occurred_at"]))
	occurredAt, err := time.Parse(time.RFC3339Nano, occurredRaw)
	if err != nil {
		occurredAt = time.Now().UTC()
	}
	payloadRaw := redisValueString(values["payload"])
	if strings.TrimSpace(payloadRaw) == "" {
		payloadRaw = "{}"
	}
	payload := json.RawMessage(payloadRaw)
	if !json.Valid(payload) {
		return event.Event{}, errors.New("taskbus redis message payload is not valid json")
	}
	return event.Event{
		Topic: event.Topic(topic),
		Meta: event.Meta{
			TenantUUID:     tenantUUID,
			RequestID:      requestID,
			SourcePlugin:   sourcePlugin,
			TraceID:        traceID,
			OccurredAt:     occurredAt,
			PayloadVersion: payloadVersion,
		},
		Payload: payload,
	}, nil
}

func redisValueString(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return fmt.Sprint(value)
	}
}

func ensureMeta(metaBuilder event.MetaBuilder, meta event.Meta) (event.Meta, error) {
	tenantUUID := strings.TrimSpace(meta.TenantUUID)
	requestID := strings.TrimSpace(meta.RequestID)
	traceID := strings.TrimSpace(meta.TraceID)

	if tenantUUID == "" {
		return event.Meta{}, errors.New("event tenant_uuid is required")
	}
	if requestID != "" && traceID != "" {
		if strings.TrimSpace(meta.SourcePlugin) == "" {
			meta.SourcePlugin = metaBuilder.SourcePlugin
		}
		if strings.TrimSpace(meta.PayloadVersion) == "" {
			meta.PayloadVersion = metaBuilder.PayloadVersion
		}
		if meta.OccurredAt.IsZero() {
			meta.OccurredAt = metaBuilder.Now()
		}
		return meta, nil
	}
	built, err := metaBuilder.Build(tenantUUID, requestID, traceID)
	if err != nil {
		return event.Meta{}, err
	}
	return built, nil
}

func defaultRedisConsumerID() string {
	host, _ := os.Hostname()
	host = strings.TrimSpace(host)
	if host == "" {
		host = "localhost"
	}
	return host + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}
