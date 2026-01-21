package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	models "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	integrationModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/integration"
	runtimeModel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/mcp/stream"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	integrationService "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/integration"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	runtimehttp "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/runtime_ops"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestMCPAgentModeInterop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	models.ForceSchemaForTests("")
	db := setupMCPTestDB(t)
	toolScope := "agent.template"
	insertGrantOverride(t, db, toolScope, "MCP")
	insertGrantOverride(t, db, toolScope, "REST")

	cfg := &config.Config{
		Server:      &config.ServerConfig{APIPrefix: "/api/v1"},
		Integration: &config.IntegrationConfig{},
	}
	deps := &app.Deps{DB: db, Config: cfg, Ctx: context.Background()}
	handler := runtimehttp.NewSessionsHandler(deps)

	tenantID := "00000000-0000-0000-0000-000000009999"
	runtimeAssignment := uuid.NewString()

	session := registerSession(t, handler, tenantID, runtimeAssignment)
	if session.RuntimeAssignmentID != runtimeAssignment {
		t.Fatalf("runtime assignment mismatch, got %s", session.RuntimeAssignmentID)
	}

	ackSession(t, handler, tenantID, session.ID)
	heartbeatSession(t, handler, tenantID, session.ID)

	events, cancel := stream.DefaultBroker().Subscribe(session.ID)
	defer cancel()

	invokeReply := invokeSession(t, handler, tenantID, session.ID, toolScope)
	if invokeReply.Status == "" {
		t.Fatalf("invoke response missing status")
	}
	if invokeReply.Replay {
		t.Fatalf("expected fresh invoke, got replay")
	}

	evt := waitForEvent(t, events, "invoke.completed")
	payload, ok := evt.Payload.(map[string]any)
	if !ok || payload["session_id"] != session.ID {
		t.Fatalf("invoke.completed payload missing session context: %#v", evt.Payload)
	}

	dispatcher := integrationService.BuildDispatchService(deps, nil)
	envelope := buildEnvelope(tenantID, toolScope)
	ctx := authx.ContextWithTenantUUID(context.Background(), tenantID)
	outcome, err := dispatcher.Dispatch(ctx, "REST", "/integration/dispatch", "CALL", envelope)
	if err != nil {
		t.Fatalf("REST channel dispatch failed: %v", err)
	}
	if outcome.Status == "" {
		t.Fatalf("REST dispatch missing status")
	}
}

func setupMCPTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(dbx.SQLiteDialector("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	statements := []string{
		`CREATE TABLE IF NOT EXISTS mcp_sessions (
			id TEXT PRIMARY KEY,
			runtime_assignment_id TEXT NOT NULL,
			tenant_uuid TEXT NOT NULL,
			state TEXT NOT NULL,
			jwt_id TEXT,
			capabilities_hash TEXT,
			missed_heartbeats INTEGER NOT NULL DEFAULT 0,
			last_ping_at DATETIME,
			closed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS runtime_audit_events (
			id TEXT PRIMARY KEY,
			plugin_id TEXT NOT NULL,
			tenant_uuid TEXT,
			event_type TEXT NOT NULL,
			payload TEXT,
			occurred_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS integration_grant_matrix_overrides (
			id TEXT PRIMARY KEY,
			scope TEXT NOT NULL,
			channel TEXT NOT NULL,
			resource TEXT NOT NULL,
			action TEXT NOT NULL,
			constraints TEXT NOT NULL,
			status TEXT NOT NULL,
			version INTEGER NOT NULL,
			approved_by TEXT,
			approved_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create table failed: %v", err)
		}
	}
	sqlDB, err := db.DB()
	if err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	return db
}

func insertGrantOverride(t *testing.T, db *gorm.DB, scope, channel string) {
	t.Helper()
	entry := &integrationModel.GrantMatrixOverride{
		ID:          uuid.NewString(),
		Scope:       scope,
		Channel:     channel,
		Resource:    "/integration/dispatch",
		Action:      "CALL",
		Constraints: datatypes.JSON([]byte("{}")),
		Status:      "APPROVED",
		Version:     1,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatalf("insert grant override: %v", err)
	}
}

func registerSession(t *testing.T, handler *runtimehttp.SessionsHandler, tenantID, assignment string) runtimeModel.MCPSession {
	payload := map[string]string{
		"runtime_assignment_id": assignment,
		"state":                 "registering",
		"jwt_id":                "jwt-demo",
		"capabilities_hash":     "hash-v1",
	}
	resp := performSessionRequest(t, handler.Register, tenantID, "", payload)
	if resp.Code != 200 {
		t.Fatalf("register returned %d: %s", resp.Code, resp.Body.String())
	}
	return decodeResponse[runtimeModel.MCPSession](t, resp)
}

func ackSession(t *testing.T, handler *runtimehttp.SessionsHandler, tenantID, sessionID string) {
	resp := performSessionRequest(t, handler.Ack, tenantID, sessionID, map[string]string{"state": "ready"})
	if resp.Code != 200 {
		t.Fatalf("ack returned %d: %s", resp.Code, resp.Body.String())
	}
}

func heartbeatSession(t *testing.T, handler *runtimehttp.SessionsHandler, tenantID, sessionID string) {
	resp := performSessionRequest(t, handler.Heartbeat, tenantID, sessionID, map[string]int{"missed_heartbeats": 0})
	if resp.Code != 200 {
		t.Fatalf("heartbeat returned %d: %s", resp.Code, resp.Body.String())
	}
}

func invokeSession(t *testing.T, handler *runtimehttp.SessionsHandler, tenantID, sessionID, scope string) invokeResponse {
	now := time.Now().UTC()
	payload := map[string]any{
		"message_id":     uuid.NewString(),
		"trace_id":       uuid.NewString(),
		"correlation_id": uuid.NewString(),
		"tenant_uuid":    tenantID,
		"tool_scope":     scope,
		"issued_at":      now.Format(time.RFC3339),
		"payload_ref":    `{"input":"demo"}`,
		"signature":      "sig-test",
		"metadata":       map[string]any{"session_id": sessionID},
	}
	resp := performSessionRequest(t, handler.Invoke, tenantID, sessionID, payload)
	if resp.Code != 200 {
		t.Fatalf("invoke returned %d: %s", resp.Code, resp.Body.String())
	}
	return decodeResponse[invokeResponse](t, resp)
}

func waitForEvent(t *testing.T, ch <-chan stream.Event, eventType string) stream.Event {
	t.Helper()
	select {
	case evt := <-ch:
		if evt.Type != eventType {
			t.Fatalf("expected event %s, got %s", eventType, evt.Type)
		}
		return evt
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for %s event", eventType)
	}
	return stream.Event{}
}

type invokeResponse struct {
	Status        string `json:"status"`
	TraceID       string `json:"trace_id"`
	CorrelationID string `json:"correlation_id"`
	LatencyMs     int64  `json:"latency_ms"`
	Replay        bool   `json:"replay"`
}

func performSessionRequest(t *testing.T, fn func(*gin.Context), tenantID, sessionID string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode payload: %v", err)
		}
	}
	req := httptest.NewRequest("POST", "/", &buf)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(authx.ContextWithTenantUUID(req.Context(), tenantID))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	if sessionID != "" {
		c.Params = gin.Params{{Key: "sessionID", Value: sessionID}}
	}
	fn(c)
	return w
}

func decodeResponse[T any](t *testing.T, resp *httptest.ResponseRecorder) T {
	t.Helper()
	var wrapper struct {
		Success bool `json:"success"`
		Data    T    `json:"data"`
		Error   *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !wrapper.Success {
		t.Fatalf("api error: %#v", wrapper.Error)
	}
	return wrapper.Data
}

func buildEnvelope(tenantID, scope string) *integrationModel.IntegrationEnvelope {
	return &integrationModel.IntegrationEnvelope{
		MessageID:     uuid.New(),
		TraceID:       uuid.New(),
		CorrelationID: uuid.New(),
		TenantUuid:    tenantID,
		ToolScope:     scope,
		IssuedAt:      time.Now().UTC(),
		PayloadRef:    `{"input":"rest"}`,
		Signature:     "sig-rest",
		Metadata:      map[string]any{"channel": "rest"},
	}
}
