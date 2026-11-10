package ai_settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	fwai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ai"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMediaModelLimitsForwardThroughBoundRuntime(t *testing.T) {
	var fixture struct {
		Catalog json.RawMessage `json:"catalog_result"`
		Cases   []struct {
			Name   string           `json:"name"`
			Limits map[string]int64 `json:"limits"`
		} `json:"cases"`
	}
	raw, err := os.ReadFile("../../../../../../../../docs/contracts/fixtures/ai-media-model-limits.v1.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &fixture))
	for _, auth := range []string{"apikey", "sts"} {
		for _, sample := range fixture.Cases {
			t.Run(auth+"/"+sample.Name, func(t *testing.T) {
				var catalog map[string]any
				require.NoError(t, json.Unmarshal(fixture.Catalog, &catalog))
				operation := catalog["items"].([]any)[0].(map[string]any)["operations"].(map[string]any)[powerxai.ImageGeneration].(map[string]any)
				for _, key := range []string{"max_image_bytes", "max_total_image_bytes", "max_image_pixels"} {
					delete(operation, key)
				}
				for key, value := range sample.Limits {
					operation[key] = value
				}
				body, err := json.Marshal(catalog)
				require.NoError(t, err)
				calls := 0
				core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					want := "ApiKey fixture-key"
					if auth == "sts" {
						want = "Bearer fixture-sts"
					}
					require.Equal(t, want, r.Header.Get("Authorization"))
					require.Equal(t, "/api/v1/tenant/invocations", r.URL.Path)
					_, _ = fmt.Fprintf(w, `{"data":{"protocol_used":"core_internal","fallback_used":false,"payload":{"result":%s}}}`, body)
				}))
				defer core.Close()
				cfg := powerxai.Config{BaseURL: core.URL, AuthScheme: "apikey", APIKey: "fixture-key"}
				var client *powerxai.Client
				if auth == "apikey" {
					client, err = powerxai.NewClient(cfg, core.Client())
				} else {
					cfg.AuthScheme = "bearer"
					cfg.APIKey = ""
					client, err = powerxai.NewClientWithTokenProvider(cfg, powerxai.TokenProviderFunc(func(context.Context) (string, error) { return "fixture-sts", nil }), core.Client())
				}
				require.NoError(t, err)
				runtime, err := fwai.NewRuntime(provider.ModeDelegated, nil, client)
				require.NoError(t, err)
				router := gin.New()
				RegisterRoutes(router.Group("/admin"), &app.Deps{AIInvocation: runtime, ProviderMode: provider.ModeDelegated})
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest("GET", "/admin/ai-settings/media/models?capability=image_generation", nil))
				require.Equal(t, 200, response.Code)
				require.Equal(t, 1, calls)
				var result struct {
					Data struct {
						Items []struct {
							Operations map[string]map[string]json.RawMessage `json:"operations"`
						} `json:"items"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
				require.Len(t, result.Data.Items, 1)
				for _, key := range []string{"max_image_bytes", "max_total_image_bytes", "max_image_pixels"} {
					value, present := result.Data.Items[0].Operations[powerxai.ImageGeneration][key]
					expected, want := sample.Limits[key]
					require.Equal(t, want, present, key)
					if want {
						var number int64
						require.NoError(t, json.Unmarshal(value, &number))
						require.Equal(t, expected, number, key)
					}
				}
			})
		}
	}
}
