package taskqueue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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
	workerID         string
}

func NewRedisProvider(cfg RedisProviderConfig) (*RedisProvider, error) {
	if strings.TrimSpace(cfg.RedisURL) == "" {
		return nil, errors.New("TASK_QUEUE_REDIS_URL_REQUIRED")
	}
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	prefix := strings.TrimSpace(cfg.Prefix)
	if prefix == "" || len(prefix) > 256 || strings.ContainsAny(prefix, "{}") {
		return nil, errors.New("TASK_QUEUE_NAMESPACE_REQUIRED")
	}
	ttl := cfg.ProcessingExpiry
	if ttl == 0 {
		ttl = 2 * time.Minute
	}
	if ttl < time.Second || ttl > time.Hour {
		return nil, errors.New("TASK_QUEUE_LEASE_INVALID")
	}
	wait := cfg.BlockingTimeout
	if wait == 0 {
		wait = 3 * time.Second
	}
	return &RedisProvider{client: redis.NewClient(opt), prefix: prefix, processingExpiry: ttl, blockingTimeout: wait, workerID: uuid.NewString()}, nil
}
func (p *RedisProvider) Close() error { return p.client.Close() }

// ValidateSubmission is shared by both providers and durable producers.
func ValidateSubmission(m Message) error {
	for _, part := range []struct {
		value string
		limit int
	}{{m.ID, 128}, {m.TenantKey, 128}, {m.SubscriberID, 128}, {m.Topic, 255}, {m.IdempotencyKey, 255}} {
		value, limit := part.value, part.limit
		if value == "" || value != strings.TrimSpace(value) || len(value) > limit {
			return errors.New("TASK_QUEUE_INVALID_ARGUMENT")
		}
	}
	if m.Lease != nil || m.Attempt != 0 || len(m.Payload) > 1<<20 || len(m.Headers) > 32 || len(m.Metadata) > 32 || len(m.TraceID) > 256 {
		return errors.New("TASK_QUEUE_INVALID_ARGUMENT")
	}
	for key, value := range m.Headers {
		if key == "" || len(key) > 128 || len(value) > 2048 {
			return errors.New("TASK_QUEUE_INVALID_ARGUMENT")
		}
	}
	for key, value := range m.Metadata {
		if key == "" || len(key) > 128 || len(value) > 2048 || strings.HasPrefix(key, "runtime_host_") || strings.HasPrefix(key, "taskbus_") || key == "run_id" {
			return errors.New("TASK_QUEUE_INVALID_ARGUMENT")
		}
	}
	return nil
}
func validateMessage(m Message) error { return ValidateSubmission(m) }

func digest(v string) string { h := sha256.Sum256([]byte(v)); return hex.EncodeToString(h[:]) }
func (p *RedisProvider) scope(t, s string) string {
	return p.prefix + ":v2:{" + digest(t+"\x00"+s) + "}"
}
func (p *RedisProvider) record(t, s, id string) string {
	return p.scope(t, s) + ":record:" + digest(id)
}

var localEnqueue = redis.NewScript(`
local previous=redis.call('HGET',KEYS[1],'fingerprint')
local keyed=redis.call('GET',KEYS[2])
if previous or keyed then
 if previous~=ARGV[1] or keyed~=ARGV[1] then return 0 end
 return 1
end
redis.call('HSET',KEYS[1],'fingerprint',ARGV[1],'raw',ARGV[2],'status','queued','attempt',0,'fence',0)
redis.call('SET',KEYS[2],ARGV[1])
if tonumber(ARGV[3])>tonumber(ARGV[4]) then redis.call('ZADD',KEYS[4],ARGV[3],ARGV[2]) else redis.call('XADD',KEYS[3],'*','message',ARGV[2]) end
return 1`)

func (p *RedisProvider) Enqueue(ctx context.Context, req EnqueueRequest) error {
	if err := validateMessage(req.Message); err != nil {
		return err
	}
	m := req.Message
	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}
	base := p.scope(m.TenantKey, m.SubscriberID)
	n, err := localEnqueue.Run(ctx, p.client, []string{p.record(m.TenantKey, m.SubscriberID, m.ID), base + ":key:" + digest(m.IdempotencyKey), base + ":stream", base + ":delay"}, digest(string(raw)), string(raw), m.VisibleAt.UnixMilli(), time.Now().UnixMilli()).Int()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("TASK_QUEUE_IDEMPOTENCY_CONFLICT")
	}
	return nil
}

var localPromote = redis.NewScript(`local rows=redis.call('ZRANGEBYSCORE',KEYS[2],'-inf',ARGV[1],'LIMIT',0,100);for _,raw in ipairs(rows) do redis.call('XADD',KEYS[1],'*','message',raw);redis.call('ZREM',KEYS[2],raw) end;return #rows`)
var localClaim = redis.NewScript(`
if redis.call('EXISTS',KEYS[2])==1 then return {0,0} end
local status=redis.call('HGET',KEYS[1],'status')
if not status then return {-1,0} end
if status=='acked' or status=='dead' then redis.call('XACK',KEYS[3],ARGV[1],ARGV[2]);redis.call('XDEL',KEYS[3],ARGV[2]);return {0,0} end
local fence=redis.call('HINCRBY',KEYS[1],'fence',1)
local attempt=redis.call('HINCRBY',KEYS[1],'attempt',1)
if attempt>3 then
 redis.call('HSET',KEYS[1],'status','dead')
 redis.call('XACK',KEYS[3],ARGV[1],ARGV[2]);redis.call('XDEL',KEYS[3],ARGV[2]);redis.call('XADD',KEYS[4],'*','message',redis.call('HGET',KEYS[1],'raw'),'reason','TASK_QUEUE_ATTEMPTS_EXHAUSTED');return {0,0}
end
redis.call('SET',KEYS[2],ARGV[3],'PX',ARGV[4])
redis.call('HSET',KEYS[1],'receipt',ARGV[3],'stream_id',ARGV[2],'owner',ARGV[5],'status','running')
return {fence,attempt}`)

func (p *RedisProvider) Dequeue(ctx context.Context, req DequeueRequest) ([]Message, error) {
	if req.TenantKey == "" || req.SubscriberID == "" {
		return nil, errors.New("TASK_QUEUE_INVALID_ARGUMENT")
	}
	max := req.MaxItems
	if max == 0 {
		max = 1
	}
	if max < 1 || max > 100 {
		return nil, errors.New("TASK_QUEUE_INVALID_ARGUMENT")
	}
	wait := req.WaitTimeout
	if wait == 0 {
		wait = p.blockingTimeout
	}
	if wait < time.Millisecond || wait > 30*time.Second {
		return nil, errors.New("TASK_QUEUE_INVALID_ARGUMENT")
	}
	base := p.scope(req.TenantKey, req.SubscriberID)
	stream := base + ":stream"
	if err := p.client.XGroupCreateMkStream(ctx, stream, "workers", "0").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return nil, err
	}
	if err := localPromote.Run(ctx, p.client, []string{stream, base + ":delay"}, time.Now().UnixMilli()).Err(); err != nil {
		return nil, err
	}
	entries, _, err := p.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{Stream: stream, Group: "workers", Consumer: p.workerID, MinIdle: p.processingExpiry, Start: "0-0", Count: int64(max)}).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	if len(entries) == 0 {
		batches, err := p.client.XReadGroup(ctx, &redis.XReadGroupArgs{Group: "workers", Consumer: p.workerID, Streams: []string{stream, ">"}, Count: int64(max), Block: wait}).Result()
		if errors.Is(err, redis.Nil) {
			return []Message{}, nil
		}
		if err != nil {
			return nil, err
		}
		for _, v := range batches {
			entries = append(entries, v.Messages...)
		}
	}
	out := make([]Message, 0, len(entries))
	for _, entry := range entries {
		raw, ok := entry.Values["message"].(string)
		if !ok {
			return nil, errors.New("TASK_QUEUE_DELIVERY_STATE_INVALID")
		}
		var m Message
		if json.Unmarshal([]byte(raw), &m) != nil || validateMessage(m) != nil || m.TenantKey != req.TenantKey || m.SubscriberID != req.SubscriberID {
			return nil, errors.New("TASK_QUEUE_DELIVERY_STATE_INVALID")
		}
		receipt := uuid.NewString()
		key := p.record(req.TenantKey, req.SubscriberID, m.ID)
		nums, err := localClaim.Run(ctx, p.client, []string{key, key + ":lease", stream, base + ":dead"}, "workers", entry.ID, receipt, p.processingExpiry.Milliseconds(), p.workerID).Int64Slice()
		if err != nil {
			return nil, err
		}
		if nums[0] < 0 {
			return nil, errors.New("TASK_QUEUE_DELIVERY_STATE_INVALID")
		}
		if nums[0] == 0 {
			continue
		}
		m.Attempt = int(nums[1])
		m.Lease = &Lease{ReceiptUUID: receipt, Fence: uint64(nums[0]), ExpiresAt: time.Now().Add(p.processingExpiry)}
		out = append(out, m)
	}
	return out, nil
}

var localSettle = redis.NewScript(`
if ARGV[1]=='ack' and redis.call('HGET',KEYS[1],'status')=='acked' and redis.call('HGET',KEYS[1],'receipt')==ARGV[2] then return 1 end
if redis.call('GET',KEYS[2])~=ARGV[2] then return 0 end
local id=redis.call('HGET',KEYS[1],'stream_id')
if redis.call('XACK',KEYS[3],'workers',id)~=1 then return 0 end
redis.call('XDEL',KEYS[3],id);redis.call('DEL',KEYS[2])
if ARGV[1]=='ack' then redis.call('HSET',KEYS[1],'status','acked');return 1 end
redis.call('HSET',KEYS[1],'status','queued','reason',ARGV[5])
local raw=redis.call('HGET',KEYS[1],'raw')
if tonumber(ARGV[3])>tonumber(ARGV[4]) then redis.call('ZADD',KEYS[4],ARGV[3],raw) else redis.call('XADD',KEYS[3],'*','message',raw) end
return 1`)

func (p *RedisProvider) settle(ctx context.Context, op, t, s, id, receipt, reason string, retry time.Time) error {
	if t == "" || s == "" || id == "" || receipt == "" {
		return errors.New("TASK_QUEUE_RECEIPT_REQUIRED")
	}
	key := p.record(t, s, id)
	base := p.scope(t, s)
	n, err := localSettle.Run(ctx, p.client, []string{key, key + ":lease", base + ":stream", base + ":delay"}, op, receipt, retry.UnixMilli(), time.Now().UnixMilli(), reason).Int()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("TASK_QUEUE_RECEIPT_STALE")
	}
	return nil
}
func (p *RedisProvider) Ack(ctx context.Context, req AckRequest) error {
	if req.Outcome != "completed" && req.Outcome != "failed" {
		return errors.New("TASK_QUEUE_RESULT_REQUIRED")
	}
	return p.settle(ctx, "ack", req.TenantKey, req.SubscriberID, req.MessageID, req.ReceiptUUID, "", time.Time{})
}
func (p *RedisProvider) Nack(ctx context.Context, req NackRequest) error {
	if req.Reason == "" || len(req.Reason) > 128 || req.RetryAt.After(time.Now().Add(24*time.Hour)) {
		return errors.New("TASK_QUEUE_INVALID_ARGUMENT")
	}
	return p.settle(ctx, "nack", req.TenantKey, req.SubscriberID, req.MessageID, req.ReceiptUUID, req.Reason, req.RetryAt)
}

var localRenew = redis.NewScript(`
if redis.call('GET',KEYS[2])~=ARGV[1] then return 0 end
local id=redis.call('HGET',KEYS[1],'stream_id');local owner=redis.call('HGET',KEYS[1],'owner')
local ids=redis.call('XCLAIM',KEYS[3],'workers',owner,0,id,'JUSTID');if #ids~=1 then return 0 end
redis.call('PEXPIRE',KEYS[2],ARGV[2]);return tonumber(redis.call('HGET',KEYS[1],'fence'))`)

func (p *RedisProvider) Renew(ctx context.Context, req RenewRequest) (Lease, error) {
	if req.TenantKey == "" || req.SubscriberID == "" || req.MessageID == "" || req.ReceiptUUID == "" {
		return Lease{}, errors.New("TASK_QUEUE_RECEIPT_REQUIRED")
	}
	key := p.record(req.TenantKey, req.SubscriberID, req.MessageID)
	n, err := localRenew.Run(ctx, p.client, []string{key, key + ":lease", p.scope(req.TenantKey, req.SubscriberID) + ":stream"}, req.ReceiptUUID, p.processingExpiry.Milliseconds()).Uint64()
	if err != nil {
		return Lease{}, err
	}
	if n == 0 {
		return Lease{}, errors.New("TASK_QUEUE_RECEIPT_STALE")
	}
	return Lease{ReceiptUUID: req.ReceiptUUID, Fence: n, ExpiresAt: time.Now().Add(p.processingExpiry)}, nil
}
func (p *RedisProvider) Retry(ctx context.Context, req RetryRequest) error {
	if req.Message.Lease == nil {
		return errors.New("TASK_QUEUE_RECEIPT_REQUIRED")
	}
	return p.Nack(ctx, NackRequest{TenantKey: req.Message.TenantKey, SubscriberID: req.Message.SubscriberID, MessageID: req.Message.ID, ReceiptUUID: req.Message.Lease.ReceiptUUID, RetryAt: req.RetryAt, Reason: req.Reason})
}
