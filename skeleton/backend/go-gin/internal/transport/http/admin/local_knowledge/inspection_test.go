package local_knowledge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInspectDocumentReturnsPersistedContentAndExplicitUnavailableVectorState(t *testing.T) {
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Open("file:inspection_"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&models.LocalKnowledgeDocument{}, &models.LocalKnowledgeChunk{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	tenantUUID, spaceUUID := uuid.NewString(), uuid.NewString()
	document := models.LocalKnowledgeDocument{TenantUUID: tenantUUID, SpaceUUID: spaceUUID, Title: "退款政策", URI: "local://refund", Content: "七天内可退款", Status: "indexed", ChunkCount: 1}
	if err := db.Create(&document).Error; err != nil {
		t.Fatalf("create document: %v", err)
	}
	if err := db.Create(&models.LocalKnowledgeChunk{TenantUUID: tenantUUID, SpaceUUID: spaceUUID, DocumentUUID: document.UUID, Ordinal: 0, Kind: "chunk", Content: document.Content}).Error; err != nil {
		t.Fatalf("create chunk: %v", err)
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/documents/"+document.UUID+"/inspection", nil)
	ctx.Request = req.WithContext(authx.ContextWithTenantUUID(context.Background(), tenantUUID))
	ctx.Params = gin.Params{{Key: "uuid", Value: document.UUID}}
	(&handler{db: db}).inspectDocument(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	for _, expected := range []string{"退款政策", "七天内可退款", "persisted_chunk_count", "VECTOR_STORE_POSTGRES_REQUIRED", `"status":"indexed"`, `"updated_at":`} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Fatalf("response missing %q: %s", expected, recorder.Body.String())
		}
	}
}
