package auth

import (
	"sync"
	"time"
)

// FederatedAuditEvent 定义联邦登录审计事件。
type FederatedAuditEvent struct {
	PluginID         string            `json:"plugin_id"`
	Provider         string            `json:"provider"`
	TenantUUID       string            `json:"tenant_uuid"`
	ExternalIdentity string            `json:"external_identity"`
	BindingOutcome   string            `json:"binding_outcome"`
	RiskDecision     string            `json:"risk_decision"`
	ReasonCode       string            `json:"reason_code"`
	TraceID          string            `json:"trace_id"`
	OccurredAt       time.Time         `json:"occurred_at"`
	Evidence         map[string]string `json:"evidence,omitempty"`
}

// FederatedAuditSink 是事件落地接口。
type FederatedAuditSink interface {
	Emit(event FederatedAuditEvent)
}

type federatedNoopSink struct{}

func (f federatedNoopSink) Emit(FederatedAuditEvent) {}

var (
	federatedAuditSinkMu sync.RWMutex
	federatedAuditSink   FederatedAuditSink = federatedNoopSink{}
)

// SetFederatedAuditSinkForTests 仅用于测试覆盖审计 sink。
func SetFederatedAuditSinkForTests(sink FederatedAuditSink) {
	federatedAuditSinkMu.Lock()
	defer federatedAuditSinkMu.Unlock()
	if sink == nil {
		federatedAuditSink = federatedNoopSink{}
		return
	}
	federatedAuditSink = sink
}

// FederatedAuditService 负责上报联邦登录审计事件。
type FederatedAuditService struct {
	pluginID string
}

func NewFederatedAuditService(pluginID string) *FederatedAuditService {
	return &FederatedAuditService{pluginID: normalizedPluginID(pluginID)}
}

func (s *FederatedAuditService) Record(event FederatedAuditEvent) {
	event.PluginID = normalizedPluginID(firstNonEmpty(event.PluginID, s.pluginID))
	event.Provider = normalizedValue(event.Provider)
	event.TenantUUID = normalizedTenantUUID(event.TenantUUID)
	event.RiskDecision = normalizedValue(event.RiskDecision)
	event.BindingOutcome = normalizedValue(event.BindingOutcome)
	event.ReasonCode = normalizedValue(event.ReasonCode)
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now()
	}

	federatedAuditSinkMu.RLock()
	sink := federatedAuditSink
	federatedAuditSinkMu.RUnlock()
	sink.Emit(event)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
