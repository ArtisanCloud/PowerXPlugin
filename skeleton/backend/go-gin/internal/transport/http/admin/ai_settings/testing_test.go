package ai_settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	fwai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ai"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type settingsTestAdapter struct {
	fwai.GenerativeService
	calls  []string
	inputs []powerxai.MediaInvokeInput
	err    error
	empty  bool
}

func (a *settingsTestAdapter) media(modality string, input powerxai.MediaInvokeInput) (*powerxai.MediaTask, error) {
	a.calls = append(a.calls, modality)
	a.inputs = append(a.inputs, input)
	if a.err != nil {
		return nil, a.err
	}
	if a.empty {
		return nil, nil
	}
	capability, _ := mediaCapability(modality)
	return &powerxai.MediaTask{ContractVersion: 2, TaskUUID: "33333333-3333-4333-8333-333333333333", ModelKey: input.ModelKey, Capability: capability, Revision: 1, TraceID: "fixture.trace", Status: "queued", Phase: "queued", CancelStatus: "none", Assets: []powerxai.MediaAssetResult{}, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}
func (a *settingsTestAdapter) ImageInvoke(_ context.Context, in powerxai.MediaInvokeInput) (*powerxai.MediaTask, error) {
	return a.media("image", in)
}
func (a *settingsTestAdapter) VideoInvoke(_ context.Context, in powerxai.MediaInvokeInput) (*powerxai.MediaTask, error) {
	return a.media("video", in)
}
func (a *settingsTestAdapter) TTSInvoke(_ context.Context, in powerxai.MediaInvokeInput) (*powerxai.MediaTask, error) {
	return a.media("audio_tts", in)
}
func (a *settingsTestAdapter) VLMInvoke(_ context.Context, in powerxai.MediaInvokeInput) (*powerxai.AnalysisResult, error) {
	a.calls = append(a.calls, "vlm")
	a.inputs = append(a.inputs, in)
	return &powerxai.AnalysisResult{ContractVersion: 2, Type: "analysis", Text: "fixture.analysis", Provider: "fixture.provider", Model: "fixture.model", TraceID: "fixture.trace"}, a.err
}
func (a *settingsTestAdapter) LLMInvoke(_ context.Context, in powerxai.LLMInvokeInput) (*powerxai.LLMInvokeOutput, error) {
	a.calls = append(a.calls, "llm")
	return &powerxai.LLMInvokeOutput{Type: "text", Text: "fixture.text"}, a.err
}

func TestSettingsTestsDispatchByCapabilityAndKeepOriginalAttempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, modality := range []string{"llm", "vlm", "image", "video", "audio_tts"} {
		t.Run(modality, func(t *testing.T) {
			adapter := &settingsTestAdapter{}
			runtime, err := fwai.NewRuntime(provider.ModeDelegated, nil, adapter)
			require.NoError(t, err)
			h := &Handler{mode: provider.ModeDelegated, deps: &app.Deps{AIInvocation: runtime}}
			for _, action := range []gin.HandlerFunc{h.TestConnection, h.QuickCall} {
				body := `{"source":"local","environment":"development","modality":"` + modality + `","provider":"fixture.provider","model_key":"fixture.model","endpoint":"https://must-not-select-route.invalid","inputs":[{"type":"text","content":"fixture.prompt"},{"type":"image_url","url":"https://fixture.invalid/image"}],"parameters":{"seed":0},"idempotency_key":"fixture.attempt"}`
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				c.Request = httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
				c.Request.Header.Set("Content-Type", "application/json")
				action(c)
				if modality == "llm" || modality == "vlm" {
					require.Equal(t, 200, rec.Code, rec.Body.String())
				} else {
					require.Equal(t, 202, rec.Code, rec.Body.String())
				}
				var result struct{ Data map[string]any }
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
				if modality == "vlm" {
					require.Equal(t, "analysis", result.Data["type"])
				}
				if rec.Code == 202 {
					require.Equal(t, "33333333-3333-4333-8333-333333333333", result.Data["task_uuid"])
					require.NotContains(t, result.Data, "text")
				}
			}
			require.Equal(t, []string{modality, modality}, adapter.calls)
			if modality != "llm" {
				require.Equal(t, adapter.inputs[0], adapter.inputs[1])
				require.Equal(t, "fixture.attempt", adapter.inputs[0].IdempotencyKey)
			}
		})
	}
}

func TestSettingsTestFailureNeverSelectsAnotherAdapter(t *testing.T) {
	for _, status := range []int{403, 409, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			local := &settingsTestAdapter{}
			delegated := &settingsTestAdapter{err: &powerxai.HTTPError{StatusCode: status, ErrorCode: "fixture.code", ReasonCode: "fixture.reason", TraceID: "fixture.trace", RequestID: "fixture.request", Details: map[string]any{"fixture": true}}}
			runtime, err := fwai.NewRuntime(provider.ModeDelegated, local, delegated)
			require.NoError(t, err)
			h := &Handler{mode: provider.ModeDelegated, deps: &app.Deps{AIInvocation: runtime}}
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"source":"local","environment":"development","modality":"image","provider":"fixture","model_key":"fixture","inputs":[{"type":"text","content":"fixture"}],"idempotency_key":"original"}`))
			c.Request.Header.Set("Content-Type", "application/json")
			h.QuickCall(c)
			require.Equal(t, status, rec.Code)
			require.Empty(t, local.calls)
			require.Len(t, delegated.calls, 1)
			require.Equal(t, "fixture.trace", rec.Header().Get("X-Trace-ID"))
			require.Equal(t, "fixture.request", rec.Header().Get("X-Request-ID"))
			require.Contains(t, rec.Body.String(), `"error_code":"fixture.code"`)
			require.Contains(t, rec.Body.String(), `"reason_code":"fixture.reason"`)
		})
	}
}

func TestSettingsTestsRejectUnsupportedAndIncompleteResults(t *testing.T) {
	for _, modality := range []string{"embedding", "audio_asr", "model3d", "rerank", "image"} {
		t.Run(modality, func(t *testing.T) {
			adapter := &settingsTestAdapter{empty: true}
			runtime, err := fwai.NewRuntime(provider.ModeDelegated, nil, adapter)
			require.NoError(t, err)
			h := &Handler{mode: provider.ModeDelegated, deps: &app.Deps{AIInvocation: runtime}}
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"source":"powerx","environment":"development","modality":"`+modality+`","provider":"fixture","model_key":"fixture","inputs":[{"type":"text","content":"fixture"}]}`))
			c.Request.Header.Set("Content-Type", "application/json")
			h.TestConnection(c)
			if modality == "image" {
				require.Equal(t, 502, rec.Code)
			} else {
				require.Equal(t, 422, rec.Code)
				require.Empty(t, adapter.calls)
			}
		})
	}
}
