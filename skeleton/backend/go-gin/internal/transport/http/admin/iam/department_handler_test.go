package iam

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	"github.com/gin-gonic/gin"
)

func TestDepartmentHandler_DelegatedRejectsWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewDepartmentHandler(nil, iamservice.IAMAdapterModeDelegated)

	router.POST("/departments", handler.Create)

	body := map[string]any{
		"tenant_uuid": "tenant-1",
		"name":        "R&D",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/departments", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status mismatch, got=%d body=%s", resp.Code, resp.Body.String())
	}

	var envelope contracts.APIResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if envelope.Error == nil || envelope.Error.Code != "IAM_DELEGATED_READ_ONLY" {
		t.Fatalf("unexpected error envelope: %#v", envelope)
	}
}
