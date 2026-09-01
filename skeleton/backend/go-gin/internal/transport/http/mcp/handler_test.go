package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	frameworkrealtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/mcp/stream"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestServeSSEStreamsFrameworkEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	broker := stream.NewBroker()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.MCPSession{}); err != nil {
		t.Fatalf("migrate session: %v", err)
	}
	if err := db.Create(&model.MCPSession{ID: "session-1", TenantUuid: "tenant-1", RuntimeAssignmentID: "assignment-1", State: "READY"}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	handler := &Handler{broker: broker, db: db, descriptors: mcpDescriptors()}
	ctx, cancel := context.WithCancel(authx.ContextWithTenantUUID(context.Background(), "tenant-1"))
	defer cancel()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/mcp/sse?session_id=session-1", nil).WithContext(ctx)
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = request
	done := make(chan struct{})
	go func() { handler.ServeSSE(ginContext); close(done) }()
	time.Sleep(10 * time.Millisecond)
	broker.Publish(stream.Event{SessionID: "session-1", Type: "progress", Payload: map[string]int{"percent": 50}})
	time.Sleep(10 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stream did not close")
	}
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "event: progress") {
		t.Fatalf("response=%d %q", recorder.Code, recorder.Body.String())
	}
}

func TestServeSSEDeniesCrossTenantSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.MCPSession{}); err != nil {
		t.Fatalf("migrate session: %v", err)
	}
	if err := db.Create(&model.MCPSession{ID: "session-private", TenantUuid: "tenant-a", RuntimeAssignmentID: "assignment-1", State: "READY"}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/mcp/sse?session_id=session-private", nil).
		WithContext(authx.ContextWithTenantUUID(context.Background(), "tenant-b"))
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = request
	(&Handler{broker: stream.NewBroker(), db: db, descriptors: mcpDescriptors()}).ServeSSE(ginContext)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestServeSSEDeniesUndeclaredChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.MCPSession{}); err != nil {
		t.Fatalf("migrate session: %v", err)
	}
	if err := db.Create(&model.MCPSession{ID: "session-1", TenantUuid: "tenant-1", RuntimeAssignmentID: "assignment-1", State: "READY"}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/mcp/sse?session_id=session-1", nil).
		WithContext(authx.ContextWithTenantUUID(context.Background(), "tenant-1"))
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = request
	(&Handler{broker: stream.NewBroker(), db: db}).ServeSSE(ginContext)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func mcpDescriptors() []frameworkrealtime.Descriptor {
	return []frameworkrealtime.Descriptor{{
		Key:        "_channel.mcp.session",
		Protocols:  []frameworkrealtime.Protocol{frameworkrealtime.ProtocolSSE},
		Actions:    []frameworkrealtime.Action{frameworkrealtime.ActionSubscribe},
		Scope:      frameworkrealtime.ScopeTenant,
		EventTypes: []string{"session.ready"},
	}}
}
