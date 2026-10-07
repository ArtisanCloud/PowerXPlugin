package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
)

const provisioningUUID = "11111111-1111-4111-8111-111111111111"
const profileJSON = `{"uuid":"11111111-1111-4111-8111-111111111111","key":"p1_general","version":2}`
const profilesJSON = `{"ingestion":` + profileJSON + `,"index":` + profileJSON + `,"rag":` + profileJSON + `}`
const quotasJSON = `{"cpu_cores":4,"storage_gb":200,"ingestion_concurrency":2}`
const catalogJSON = `{"data":{"catalog":{"version":"1","source":"powerx_core","scenes":[{"key":"support_faq","label":"fixture","default_bundle":"p1_general","allowed_bundles":["p1_general"]}],"strategy_packages":[{"key":"A_simple","label":"fixture","recommended_profile_key":"p1_general","recommended_scenes":["support_faq"],"profiles":` + profilesJSON + `,"available":true,"unavailable_reasons":[],"activation_dependencies":["index:dense","assets:embedding"],"dependencies":{"index":["dense"],"runtime":[],"assets":["embedding"]}}],"policy_templates":[{"uuid":"11111111-1111-4111-8111-111111111111","name":"fixture","version":"v1"}],"default_policy_template_uuid":"11111111-1111-4111-8111-111111111111","quota_defaults":` + quotasJSON + `,"quota_minimums":{"cpu_cores":1,"storage_gb":50,"ingestion_concurrency":1},"quota_override_allowed":true}}}`
const createdJSON = `{"data":{"item":{"space_uuid":"22222222-2222-4222-8222-222222222222","name":"fixture","status":"pending_iam","department_uuid":"11111111-1111-4111-8111-111111111111","strategy_key":"A_simple","scene_key":"support_faq","policy_template_uuid":"11111111-1111-4111-8111-111111111111","profiles":` + profilesJSON + `,"quotas":` + quotasJSON + `}}}`

func TestProvisioningBothServiceCredentialsAndProvider(t *testing.T) {
	for _, scheme := range []string{"ApiKey", "Bearer"} {
		t.Run(scheme, func(t *testing.T) {
			calls := 0
			transport := knowledgeRoundTrip(func(req *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, scheme+" service-credential", req.Header.Get("Authorization"))
				require.Empty(t, req.URL.RawQuery)
				require.Empty(t, req.Header.Get("tenant_uuid"))
				if req.Method == http.MethodGet {
					require.Equal(t, "/api/v1/tenant/knowledge/catalog", req.URL.Path)
					require.Nil(t, req.Body)
					return knowledgeResponse(200, catalogJSON), nil
				}
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, "/api/v1/tenant/knowledge/spaces", req.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				require.Equal(t, map[string]any{"name": "fixture", "department_uuid": provisioningUUID, "strategy_key": "A_simple"}, body)
				return knowledgeResponse(200, createdJSON), nil
			})
			var client *Client
			var err error
			if scheme == "ApiKey" {
				client, err = NewClientWithAPIKey(Config{BaseURL: "https://core.example"}, "service-credential", &http.Client{Transport: transport})
			} else {
				client, err = NewClientWithTokenProvider(Config{BaseURL: "https://core.example"}, TokenProviderFunc(func(context.Context) (string, error) { return "service-credential", nil }), &http.Client{Transport: transport})
			}
			require.NoError(t, err)
			provider := fw.NewDelegatedProvider(fw.DelegatedProviderConfig{Client: client})
			require.True(t, fw.Supports(provider.Capabilities(context.Background()), fw.OperationCatalog))
			require.True(t, fw.Supports(provider.Capabilities(context.Background()), fw.OperationCreate))
			catalog, err := provider.Catalog(context.Background())
			require.NoError(t, err)
			require.True(t, catalog.StrategyPackages[0].Available)
			require.Equal(t, 2, catalog.StrategyPackages[0].Profiles.Ingestion.Version)
			require.Equal(t, []string{"index:dense", "assets:embedding"}, catalog.StrategyPackages[0].ActivationDependencies)
			require.Equal(t, provisioningUUID, catalog.DefaultPolicyTemplateUUID)
			require.True(t, catalog.QuotaOverrideAllowed)
			item, err := provider.CreateSpace(context.Background(), fw.CreateSpaceInput{Name: "fixture", DepartmentUUID: provisioningUUID, StrategyKey: "A_simple"})
			require.NoError(t, err)
			require.Equal(t, "pending_iam", item.Status)
			require.Equal(t, 2, calls)
		})
	}
}

func TestProvisioningPreservesCoreErrorsThroughProvider(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
	}{{401, "KNOWLEDGE_UNAUTHORIZED"}, {403, "KNOWLEDGE_FORBIDDEN"}, {400, "KNOWLEDGE_INVALID_ARGUMENT"}, {409, "KNOWLEDGE_SPACE_CONFLICT"}, {412, "KNOWLEDGE_STRATEGY_UNAVAILABLE"}, {503, "KNOWLEDGE_MIGRATION_REQUIRED"}, {500, "CORE_CUSTOM_FAILURE"}} {
		t.Run(tc.code, func(t *testing.T) {
			count := 0
			client, err := NewClientWithAPIKey(Config{BaseURL: "https://core.example"}, "service-credential", &http.Client{Transport: knowledgeRoundTrip(func(*http.Request) (*http.Response, error) {
				count++
				response := knowledgeResponse(tc.status, `{"error_code":"`+tc.code+`","request_id":"body-request"}`)
				response.Header.Set("X-Trace-Id", "core-trace")
				return response, nil
			})})
			require.NoError(t, err)
			provider := fw.NewDelegatedProvider(fw.DelegatedProviderConfig{Client: client})
			_, err = provider.CreateSpace(context.Background(), fw.CreateSpaceInput{Name: "fixture", DepartmentUUID: provisioningUUID, StrategyKey: "A_simple"})
			require.Equal(t, fw.ErrorCode(tc.code), fw.CodeOf(err))
			require.Equal(t, tc.status, fw.HTTPStatus(err))
			var typed *fw.Error
			require.True(t, errors.As(err, &typed))
			require.Equal(t, "core-trace", typed.TraceID)
			require.Equal(t, fw.OperationCreate, typed.Operation)
			require.Equal(t, 1, count)
		})
	}
}
func TestProvisioningRejectsInvalidInputsBeforeHTTP(t *testing.T) {
	client, err := NewClientWithAPIKey(Config{BaseURL: "https://core.example"}, "service-credential", &http.Client{Transport: knowledgeRoundTrip(func(*http.Request) (*http.Response, error) { t.Fatal("invalid input reached Core"); return nil, nil })})
	require.NoError(t, err)
	valid := fw.CreateSpaceInput{Name: "fixture", DepartmentUUID: provisioningUUID, StrategyKey: "A_simple"}
	for _, mutate := range []func(*fw.CreateSpaceInput){func(in *fw.CreateSpaceInput) { in.Name = " " }, func(in *fw.CreateSpaceInput) { in.Name = strings.Repeat("界", 43) }, func(in *fw.CreateSpaceInput) { in.DepartmentUUID = "1" }, func(in *fw.CreateSpaceInput) { in.PolicyTemplateUUID = "default/v1" }, func(in *fw.CreateSpaceInput) {
		in.Quotas = &fw.SpaceQuotas{CPUCores: 1, StorageGB: 49, IngestionConcurrency: 1}
	}} {
		in := valid
		mutate(&in)
		_, err := client.CreateKnowledgeSpace(context.Background(), in)
		require.Equal(t, fw.CodeInvalidArgument, fw.CodeOf(err))
	}
}
func TestProvisioningMissingEnvelopeAndUUIDFailClosed(t *testing.T) {
	for _, raw := range []string{`{"data":{}}`, strings.Replace(createdJSON, `"space_uuid":"22222222-2222-4222-8222-222222222222"`, `"space_uuid":"2"`, 1)} {
		client, _ := NewClientWithAPIKey(Config{BaseURL: "https://core.example"}, "credential", &http.Client{Transport: knowledgeRoundTrip(func(*http.Request) (*http.Response, error) { return knowledgeResponse(200, raw), nil })})
		_, err := client.CreateKnowledgeSpace(context.Background(), fw.CreateSpaceInput{Name: "fixture", DepartmentUUID: provisioningUUID, StrategyKey: "A_simple"})
		require.Equal(t, fw.CodeInvalidResponse, fw.CodeOf(err))
	}
	client, _ := NewClientWithAPIKey(Config{BaseURL: "https://core.example"}, "credential", &http.Client{Transport: knowledgeRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 403, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"reason_code":"DENIED","request_id":"core-request"}`))}, nil
	})})
	_, err := client.GetKnowledgeCatalog(context.Background())
	var typed *fw.Error
	require.True(t, errors.As(err, &typed))
	require.Equal(t, "core-request", typed.TraceID)
	require.Equal(t, 403, fw.HTTPStatus(err))
}

func TestProvisioningOptionalUUIDsAndQuotasRemainTyped(t *testing.T) {
	input := fw.CreateSpaceInput{Name: "fixture", DepartmentUUID: provisioningUUID, StrategyKey: "A_simple", SceneKey: "support_faq", PolicyTemplateUUID: provisioningUUID, IngestionProfileUUID: provisioningUUID, IndexProfileUUID: provisioningUUID, RAGProfileUUID: provisioningUUID, Quotas: &fw.SpaceQuotas{CPUCores: 4, StorageGB: 200, IngestionConcurrency: 2}}
	client, _ := NewClientWithAPIKey(Config{BaseURL: "https://core.example"}, "credential", &http.Client{Transport: knowledgeRoundTrip(func(req *http.Request) (*http.Response, error) {
		raw, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(raw, &body))
		require.Len(t, body, 9)
		for _, key := range []string{"department_uuid", "policy_template_uuid", "ingestion_profile_uuid", "index_profile_uuid", "rag_profile_uuid"} {
			require.Equal(t, provisioningUUID, body[key])
		}
		require.Equal(t, map[string]any{"cpu_cores": float64(4), "storage_gb": float64(200), "ingestion_concurrency": float64(2)}, body["quotas"])
		return knowledgeResponse(200, createdJSON), nil
	})})
	_, err := client.CreateKnowledgeSpace(context.Background(), input)
	require.NoError(t, err)
}
func TestUnavailableStrategyPreservesMissingProfilesAndReasons(t *testing.T) {
	raw := strings.Replace(catalogJSON, profilesJSON, `{}`, 1)
	raw = strings.Replace(raw, `"available":true`, `"available":false`, 1)
	raw = strings.Replace(raw, `"unavailable_reasons":[]`, `"unavailable_reasons":["rag_profile_not_published"]`, 1)
	client, _ := NewClientWithAPIKey(Config{BaseURL: "https://core.example"}, "credential", &http.Client{Transport: knowledgeRoundTrip(func(*http.Request) (*http.Response, error) { return knowledgeResponse(200, raw), nil })})
	catalog, err := client.GetKnowledgeCatalog(context.Background())
	require.NoError(t, err)
	require.False(t, catalog.StrategyPackages[0].Available)
	require.Nil(t, catalog.StrategyPackages[0].Profiles.RAG)
	require.Equal(t, []string{"rag_profile_not_published"}, catalog.StrategyPackages[0].UnavailableReasons)
}
