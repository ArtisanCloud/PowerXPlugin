package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMediaModelLimitsBothCredentialsDecodeAndReserialize(t *testing.T) {
	var fixture struct {
		Catalog json.RawMessage `json:"catalog_result"`
		Cases   []struct {
			Name   string           `json:"name"`
			Limits map[string]int64 `json:"limits"`
		} `json:"cases"`
	}
	raw, err := os.ReadFile("testdata/ai-media-model-limits.v1.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &fixture))
	for _, auth := range []string{"apikey", "sts"} {
		for _, test := range fixture.Cases {
			t.Run(auth+"/"+test.Name, func(t *testing.T) {
				var catalog map[string]any
				require.NoError(t, json.Unmarshal(fixture.Catalog, &catalog))
				op := catalog["items"].([]any)[0].(map[string]any)["operations"].(map[string]any)[ImageGeneration].(map[string]any)
				for _, key := range []string{"max_image_bytes", "max_total_image_bytes", "max_image_pixels"} {
					delete(op, key)
				}
				for key, value := range test.Limits {
					op[key] = value
				}
				result, err := json.Marshal(catalog)
				require.NoError(t, err)
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					require.Equal(t, "/api/v1/tenant/invocations", r.URL.Path)
					want := "ApiKey fixture-key"
					if auth == "sts" {
						want = "Bearer fixture-sts"
					}
					require.Equal(t, want, r.Header.Get("Authorization"))
					var request struct {
						Capability string `json:"capability_id"`
						Protocol   string `json:"preferred_protocol"`
						Payload    struct {
							Body struct {
								Operation string `json:"operation"`
							} `json:"body"`
						} `json:"payload"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
					require.Equal(t, CapabilityMediaModels, request.Capability)
					require.Equal(t, "core_internal", request.Protocol)
					require.Equal(t, "list_models", request.Payload.Body.Operation)
					_, _ = fmt.Fprintf(w, `{"data":{"protocol_used":"core_internal","fallback_used":false,"payload":{"result":%s}}}`, result)
				}))
				defer server.Close()
				cfg := Config{BaseURL: server.URL, AuthScheme: "apikey", APIKey: "fixture-key"}
				var client *Client
				if auth == "apikey" {
					client, err = NewClient(cfg, server.Client())
				} else {
					cfg.AuthScheme = "bearer"
					client, err = NewClientWithTokenProvider(cfg, TokenProviderFunc(func(context.Context) (string, error) { return "fixture-sts", nil }), server.Client())
				}
				require.NoError(t, err)
				out, err := client.ListMediaModels(context.Background(), MediaCatalogFilter{Capability: ImageGeneration})
				require.NoError(t, err)
				require.Equal(t, 1, calls)
				operation := out.Items[0].Operations[ImageGeneration]
				for key, value := range map[string]*int64{"max_image_bytes": operation.MaxImageBytes, "max_total_image_bytes": operation.MaxTotalImageBytes, "max_image_pixels": operation.MaxImagePixels} {
					expected, present := test.Limits[key]
					if present {
						require.NotNil(t, value, key)
						require.Equal(t, expected, *value, key)
					} else {
						require.Nil(t, value, key)
					}
				}
				forwarded, err := json.Marshal(out)
				require.NoError(t, err)
				var roundTrip struct {
					Items []struct {
						Operations map[string]map[string]json.RawMessage `json:"operations"`
					} `json:"items"`
				}
				require.NoError(t, json.Unmarshal(forwarded, &roundTrip))
				for _, key := range []string{"max_image_bytes", "max_total_image_bytes", "max_image_pixels"} {
					value, present := roundTrip.Items[0].Operations[ImageGeneration][key]
					expected, want := test.Limits[key]
					require.Equal(t, want, present, key)
					if want {
						var decoded int64
						require.NoError(t, json.Unmarshal(value, &decoded))
						require.Equal(t, expected, decoded, key)
					}
				}
			})
		}
	}
}
