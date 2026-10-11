package local_knowledge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIngestionProgressUpdatesOnlySelectedJobAndPersistsFailure(t *testing.T) {
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Open("file:ingestion_jobs_"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.LocalKnowledgeDocument{}, &models.LocalKnowledgeIngestionJob{}, &models.LocalKnowledgeJobChunk{}); err != nil {
		t.Fatal(err)
	}
	tenantUUID, spaceUUID := uuid.NewString(), uuid.NewString()
	document := models.LocalKnowledgeDocument{TenantUUID: tenantUUID, SpaceUUID: spaceUUID, Title: "source", URI: "local://source", Content: "content", Status: "queued"}
	if err := db.Create(&document).Error; err != nil {
		t.Fatal(err)
	}
	first := models.LocalKnowledgeIngestionJob{TenantUUID: tenantUUID, SpaceUUID: spaceUUID, SourceID: document.UUID, SourceType: "manual", Status: "completed"}
	second := models.LocalKnowledgeIngestionJob{TenantUUID: tenantUUID, SpaceUUID: spaceUUID, SourceID: document.UUID, SourceType: "manual", Status: "queued"}
	for _, job := range []*models.LocalKnowledgeIngestionJob{&first, &second} {
		if err := db.Create(job).Error; err != nil {
			t.Fatal(err)
		}
	}
	h := &handler{db: db}
	h.reportIngestionProgress(context.Background(), tenantUUID, second.UUID, document.UUID, spaceUUID, "failed", "failed", 25, 2, "INDEX_WRITE_FAILED")
	var gotFirst, gotSecond models.LocalKnowledgeIngestionJob
	if err := db.Where("uuid=?", first.UUID).First(&gotFirst).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("uuid=?", second.UUID).First(&gotSecond).Error; err != nil {
		t.Fatal(err)
	}
	if gotFirst.Status != "completed" || gotSecond.Status != "failed" || gotSecond.ErrorCode != "INDEX_WRITE_FAILED" {
		t.Fatalf("job states: first=%s second=%s error=%s", gotFirst.Status, gotSecond.Status, gotSecond.ErrorCode)
	}
	if gotSecond.ProgressPercent != 25 || gotSecond.ChunkCoveredPct != 0 || gotSecond.EmbeddingSuccessPct != 0 {
		t.Fatalf("job metrics: progress=%d chunk_coverage=%v embedding_success=%v", gotSecond.ProgressPercent, gotSecond.ChunkCoveredPct, gotSecond.EmbeddingSuccessPct)
	}
	if err := db.Where("uuid=?", document.UUID).First(&document).Error; err != nil {
		t.Fatal(err)
	}
	if document.Status != "failed" {
		t.Fatalf("document status=%s", document.Status)
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/spaces/"+spaceUUID+"/ingestion-jobs?page=1&page_size=1", nil).WithContext(authx.ContextWithTenantUUID(context.Background(), tenantUUID))
	ctx.Params = gin.Params{{Key: "uuid", Value: spaceUUID}}
	h.listIngestionJobs(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			Total int `json:"total"`
			Items []struct {
				UUID            string `json:"uuid"`
				CreatedAt       string `json:"created_at"`
				ProgressPercent int    `json:"progress_percent"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Total != 2 || len(response.Data.Items) != 1 || response.Data.Items[0].UUID != second.UUID || response.Data.Items[0].CreatedAt == "" || response.Data.Items[0].ProgressPercent != 25 {
		t.Fatalf("page total=%d items=%v", response.Data.Total, response.Data.Items)
	}
	chunk := models.LocalKnowledgeChunk{TenantUUID: tenantUUID, SpaceUUID: spaceUUID, DocumentUUID: document.UUID, Ordinal: 0, Kind: "chunk", Content: "first attempt content"}
	if err := persistLocalJobChunk(db, first.UUID, chunk); err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		jobUUID string
		total   int
	}{{first.UUID, 1}, {second.UUID, 0}} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/spaces/"+spaceUUID+"/ingestion-jobs/"+testCase.jobUUID+"/chunks", nil).WithContext(authx.ContextWithTenantUUID(context.Background(), tenantUUID))
		ctx.Params = gin.Params{{Key: "uuid", Value: spaceUUID}, {Key: "jobUUID", Value: testCase.jobUUID}}
		h.listIngestionJobChunks(ctx)
		var page struct {
			Data struct {
				Total int `json:"total"`
			} `json:"data"`
		}
		if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &page) != nil || page.Data.Total != testCase.total {
			t.Fatalf("job=%s status=%d body=%s", testCase.jobUUID, recorder.Code, recorder.Body.String())
		}
	}
}

func TestListIngestionJobsFiltersByDocumentBeforePagination(t *testing.T) {
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Open("file:ingestion_filter_"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.LocalKnowledgeIngestionJob{}); err != nil {
		t.Fatal(err)
	}
	tenantUUID, spaceUUID, sourceUUID, otherSourceUUID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, sourceID := range []string{sourceUUID, otherSourceUUID, sourceUUID} {
		job := models.LocalKnowledgeIngestionJob{TenantUUID: tenantUUID, SpaceUUID: spaceUUID, SourceID: sourceID, SourceType: "manual", Status: "completed"}
		if err := db.Create(&job).Error; err != nil {
			t.Fatal(err)
		}
	}
	h := &handler{db: db}
	for page, expectedCount := range map[int]int{1: 1, 2: 1, 3: 0} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/spaces/"+spaceUUID+"/ingestion-jobs?source_id="+sourceUUID+"&page="+strconv.Itoa(page)+"&page_size=1", nil).WithContext(authx.ContextWithTenantUUID(context.Background(), tenantUUID))
		ctx.Params = gin.Params{{Key: "uuid", Value: spaceUUID}}
		h.listIngestionJobs(ctx)
		var response struct {
			Data struct {
				Total int `json:"total"`
				Items []struct {
					SourceID string `json:"source_id"`
				} `json:"items"`
			} `json:"data"`
		}
		if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &response) != nil || response.Data.Total != 2 || len(response.Data.Items) != expectedCount {
			t.Fatalf("page=%d status=%d body=%s", page, recorder.Code, recorder.Body.String())
		}
		for _, item := range response.Data.Items {
			if item.SourceID != sourceUUID {
				t.Fatalf("page=%d source=%s", page, item.SourceID)
			}
		}
	}
}

func TestRepeatedIndexAttemptsCreateDistinctIngestionJobs(t *testing.T) {
	models.ForceSchemaForTests("")
	t.Cleanup(func() { models.ForceSchemaForTests("public") })
	db, err := gorm.Open(sqlite.Open("file:ingestion_attempts_"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.LocalKnowledgeSpace{}, &models.LocalKnowledgeDocument{}, &models.LocalKnowledgeIngestionJob{}); err != nil {
		t.Fatal(err)
	}
	tenantUUID := uuid.NewString()
	space := models.LocalKnowledgeSpace{TenantUUID: tenantUUID, SpaceName: "space", DepartmentCode: "test", IngestionProfileKey: "profile", IndexProfileKey: "profile", RAGProfileKey: "profile", Status: "active"}
	if err := db.Create(&space).Error; err != nil {
		t.Fatal(err)
	}
	document := models.LocalKnowledgeDocument{TenantUUID: tenantUUID, SpaceUUID: space.UUID, Title: "source", URI: "local://source", Content: "content", Status: "queued"}
	if err := db.Create(&document).Error; err != nil {
		t.Fatal(err)
	}
	queued := models.LocalKnowledgeIngestionJob{TenantUUID: tenantUUID, SpaceUUID: space.UUID, SourceID: document.UUID, SourceType: "manual", Status: "queued"}
	if err := db.Create(&queued).Error; err != nil {
		t.Fatal(err)
	}
	h := &handler{db: db}
	for attempt := 0; attempt < 2; attempt++ {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/documents/"+document.UUID+"/index", nil).WithContext(authx.ContextWithTenantUUID(context.Background(), tenantUUID))
		ctx.Params = gin.Params{{Key: "uuid", Value: document.UUID}}
		h.indexDocument(ctx)
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("attempt=%d status=%d body=%s", attempt, recorder.Code, recorder.Body.String())
		}
	}
	var jobs []models.LocalKnowledgeIngestionJob
	if err := db.Where("tenant_uuid=? AND source_id=?", tenantUUID, document.UUID).Order("created_at asc, uuid asc").Find(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 || jobs[0].UUID == jobs[1].UUID || jobs[0].Status != "failed" || jobs[1].Status != "failed" {
		t.Fatalf("jobs=%+v", jobs)
	}
}
