package host_contract

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ai"
	fwcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/capability"
	fwintegration "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/integration"
	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	fwmedia "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/media"
	fwnotifications "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/notifications"
	fwpluginruntime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/pluginruntime"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	powerxcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	hostcontract "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
	powerxintegration "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/integration"
	powerxmedia "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/media"
	powerxnotifications "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/notifications"
	powerxruntime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/pluginruntime"
	powerxskills "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/skills"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	frameworkrealtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
	fwskills "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
	authmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

const probeTenantUUID = "11111111-1111-1111-1111-111111111111"
const probeMemberUUID = "22222222-2222-2222-2222-222222222222"

type directoryStub struct {
	tenant       string
	member       string
	batchMembers []string
	displayNames []string
}

type mediaStub struct{ input powerxmedia.ListAssetsInput }

type knowledgeStub struct {
	jobQuery fwknowledge.IndexJobQuery
	reindex  fwknowledge.ReindexInput
}

func (*knowledgeStub) DelegatedCapabilities(context.Context) fwknowledge.ProviderCapabilities {
	return fwknowledge.BasicCapabilities("test", "delegated", fwknowledge.OperationRetrieve, fwknowledge.OperationReindex)
}

func (s *knowledgeStub) ListKnowledgeSpaces(context.Context, fwknowledge.ListSpacesInput) ([]fwknowledge.KnowledgeSpace, error) {
	return []fwknowledge.KnowledgeSpace{}, nil
}
func (s *knowledgeStub) SearchKnowledge(context.Context, fwknowledge.KnowledgeQuery) (*fwknowledge.KnowledgeSearchResult, error) {
	return &fwknowledge.KnowledgeSearchResult{}, nil
}
func (s *knowledgeStub) UpsertKnowledgeDocument(context.Context, fwknowledge.KnowledgeDocument) (*fwknowledge.KnowledgeIndexJob, error) {
	return &fwknowledge.KnowledgeIndexJob{JobID: "33333333-3333-3333-3333-333333333333", Status: fwknowledge.IndexStatusQueued}, nil
}
func (s *knowledgeStub) DeleteKnowledgeDocument(context.Context, fwknowledge.DeleteDocumentInput) (*fwknowledge.KnowledgeIndexJob, error) {
	return &fwknowledge.KnowledgeIndexJob{JobID: "33333333-3333-3333-3333-333333333333", Status: fwknowledge.IndexStatusQueued}, nil
}
func (s *knowledgeStub) ReindexKnowledgeDocument(_ context.Context, input fwknowledge.ReindexInput) (*fwknowledge.KnowledgeIndexJob, error) {
	s.reindex = input
	return &fwknowledge.KnowledgeIndexJob{JobID: "33333333-3333-3333-3333-333333333333", SpaceID: input.SpaceID, Operation: fwknowledge.IndexOperationReindex, Status: fwknowledge.IndexStatusQueued}, nil
}
func (s *knowledgeStub) GetKnowledgeIndexJob(_ context.Context, input fwknowledge.IndexJobQuery) (*fwknowledge.KnowledgeIndexJob, error) {
	s.jobQuery = input
	return &fwknowledge.KnowledgeIndexJob{JobID: input.JobID, Status: fwknowledge.IndexStatusSucceeded}, nil
}

func (s *mediaStub) ListAssets(_ context.Context, input powerxmedia.ListAssetsInput) (*powerxmedia.ListAssetsOutput, error) {
	s.input = input
	return &powerxmedia.ListAssetsOutput{Items: []powerxmedia.Asset{}, Page: input.Page, PageSize: input.PageSize, TraceID: "media-trace"}, nil
}
func (*mediaStub) GetAsset(context.Context, string) (*powerxmedia.HostAsset, error) {
	return &powerxmedia.HostAsset{}, nil
}
func (*mediaStub) CreateAsset(context.Context, powerxmedia.CreateAssetInput) (*powerxmedia.HostAsset, error) {
	return &powerxmedia.HostAsset{}, nil
}
func (*mediaStub) UpdateAsset(context.Context, string, powerxmedia.UpdateAssetInput) (*powerxmedia.HostAsset, error) {
	return &powerxmedia.HostAsset{}, nil
}
func (*mediaStub) DeleteAsset(context.Context, string) error { return nil }
func (*mediaStub) PresignUpload(context.Context, string) (*powerxmedia.TransferTicket, error) {
	return &powerxmedia.TransferTicket{}, nil
}
func (*mediaStub) CompleteUpload(context.Context, string, powerxmedia.CompleteUploadInput) (*powerxmedia.HostAsset, error) {
	return &powerxmedia.HostAsset{}, nil
}
func (*mediaStub) PresignDownload(context.Context, string) (*powerxmedia.TransferTicket, error) {
	return &powerxmedia.TransferTicket{}, nil
}
func (*mediaStub) CreateVariant(context.Context, string, powerxmedia.CreateVariantInput) (*powerxmedia.Variant, error) {
	return &powerxmedia.Variant{}, nil
}
func (*mediaStub) PresignVariantUpload(context.Context, string, string, powerxmedia.VariantTicketInput) (*powerxmedia.TransferTicket, error) {
	return &powerxmedia.TransferTicket{}, nil
}
func (*mediaStub) PresignVariantDownload(context.Context, string, string, powerxmedia.VariantTicketInput) (*powerxmedia.TransferTicket, error) {
	return &powerxmedia.TransferTicket{}, nil
}
func (*mediaStub) CompleteVariantUpload(context.Context, string, string, powerxmedia.CompleteUploadInput) (*powerxmedia.Variant, error) {
	return &powerxmedia.Variant{}, nil
}
func (*mediaStub) GetVariant(context.Context, string) (*powerxmedia.Variant, error) {
	return &powerxmedia.Variant{}, nil
}

func (s *directoryStub) GetTenant(_ context.Context, tenant string) (*fwiamcontracts.Tenant, error) {
	s.tenant = tenant
	return &fwiamcontracts.Tenant{TenantUUID: tenant, Name: "tenant"}, nil
}
func (s *directoryStub) ListDepartments(context.Context, string) ([]fwiamcontracts.Department, error) {
	return nil, nil
}
func (s *directoryStub) ListMembers(context.Context, string) ([]fwiamcontracts.Member, error) {
	return nil, nil
}
func (s *directoryStub) ListMembersPage(context.Context, string, fwiamcontracts.MemberPageRequest) (*fwiamcontracts.MemberPage, error) {
	return &fwiamcontracts.MemberPage{}, nil
}
func (s *directoryStub) GetMember(_ context.Context, tenant, member string) (*fwiamcontracts.Member, error) {
	s.tenant, s.member = tenant, member
	return &fwiamcontracts.Member{TenantUUID: tenant, MemberUUID: member, DisplayName: "member"}, nil
}
func (s *directoryStub) BatchGetMembers(_ context.Context, tenant string, memberUUIDs []string) ([]fwiamcontracts.Member, error) {
	s.tenant = tenant
	s.batchMembers = append([]string(nil), memberUUIDs...)
	return []fwiamcontracts.Member{{TenantUUID: tenant, MemberUUID: memberUUIDs[0], DisplayName: "member"}}, nil
}
func (s *directoryStub) BatchResolveMembers(context.Context, string, []string) (*fwiamcontracts.MemberResolution, error) {
	return &fwiamcontracts.MemberResolution{}, nil
}
func (s *directoryStub) BatchResolveMembersByDisplayNames(_ context.Context, tenant string, displayNames []string) (*fwiamcontracts.MemberDisplayNameResolution, error) {
	s.tenant = tenant
	s.displayNames = append([]string(nil), displayNames...)
	return &fwiamcontracts.MemberDisplayNameResolution{Items: []fwiamcontracts.MemberDisplayNameResolutionItem{
		{DisplayName: displayNames[0], Status: fwiamcontracts.MemberDisplayNameResolutionFound, Member: &fwiamcontracts.Member{TenantUUID: tenant, MemberUUID: probeMemberUUID, DisplayName: displayNames[0]}},
		{DisplayName: displayNames[1], Status: fwiamcontracts.MemberDisplayNameResolutionAmbiguous},
	}}, nil
}
func (s *directoryStub) ListRoles(context.Context, string) ([]fwiamcontracts.Role, error) {
	return nil, nil
}
func (s *directoryStub) ListPermissions(context.Context, string) ([]fwiamcontracts.Permission, error) {
	return nil, nil
}

func TestProbeIAMMemberGetUsesCredentialTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	directory := &directoryStub{}
	handler := NewHandler(&app.Deps{IAMDirectoryService: directory, ProviderMode: fwprovider.ModeDelegated})
	response := executeProbe(t, handler, map[string]any{"module": "iam", "operation": "member.get", "input": map[string]any{"member_uuid": probeMemberUUID}})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if directory.tenant != probeTenantUUID || directory.member != probeMemberUUID {
		t.Fatalf("directory call tenant=%q member=%q", directory.tenant, directory.member)
	}
	var envelope struct {
		Data hostcontract.ProbeResult `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.CapabilityID != "com.corex.iam.members.read" || envelope.Data.ProviderMode != "delegated" {
		t.Fatalf("probe=%#v", envelope.Data)
	}
}

func TestProbeRejectsTenantOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(&app.Deps{IAMDirectoryService: &directoryStub{}})
	response := executeProbe(t, handler, map[string]any{"module": "iam", "operation": "member.get", "input": map[string]any{"member_uuid": probeMemberUUID, "tenant_uuid": "33333333-3333-3333-3333-333333333333"}})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	assertReason(t, response, string(hostcontract.ReasonTenantOverrideForbidden))
}

func TestProbeRejectsMissingCredentialTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(&app.Deps{IAMDirectoryService: &directoryStub{}})
	response := executeProbeWithoutTenant(t, handler, map[string]any{"module": "iam", "operation": "tenant.get"})
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	assertReason(t, response, string(hostcontract.ReasonUnauthorized))
}

func TestProbeReportsUnavailableDirectoryWithoutLocalFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(&app.Deps{ProviderMode: fwprovider.ModeDelegated})
	response := executeProbe(t, handler, map[string]any{"module": "iam", "operation": "tenant.get"})
	if response.Code != http.StatusFailedDependency {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	assertReason(t, response, string(hostcontract.ReasonUpstreamDependency))
}

func TestProbeKnowledgeWriteRequiresConfirmation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := executeProbe(t, NewHandler(&app.Deps{KnowledgeProvider: fwknowledge.NewDelegatedProvider(fwknowledge.DelegatedProviderConfig{Client: &knowledgeStub{}})}), map[string]any{
		"module": "knowledge", "operation": "index.rebuild", "input": map[string]any{"space_uuid": probeTenantUUID},
	})
	if response.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	assertReason(t, response, string(hostcontract.ReasonConfirmationRequired))
}

func TestProbeKnowledgeIndexJobUsesTenantScopedTypedClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	knowledge := &knowledgeStub{}
	jobUUID := "33333333-3333-3333-3333-333333333333"
	response := executeProbe(t, NewHandler(&app.Deps{KnowledgeProvider: fwknowledge.NewDelegatedProvider(fwknowledge.DelegatedProviderConfig{Client: knowledge})}), map[string]any{
		"module": "knowledge", "operation": "index_job.get", "input": map[string]any{"job_uuid": jobUUID},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if knowledge.jobQuery.JobID != jobUUID || knowledge.jobQuery.TenantUUID != probeTenantUUID {
		t.Fatalf("query=%#v", knowledge.jobQuery)
	}
	if !strings.Contains(response.Body.String(), `"status":"succeeded"`) {
		t.Fatalf("body=%s", response.Body.String())
	}
}

func TestProbeIAMDisplayNameResolutionPreservesPerItemResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	directory := &directoryStub{}
	response := executeProbe(t, NewHandler(&app.Deps{IAMDirectoryService: directory}), map[string]any{
		"module": "iam", "operation": "members.resolve_display_names", "input": map[string]any{"display_names": []any{"Ada", "Alex"}},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if directory.tenant != probeTenantUUID || strings.Join(directory.displayNames, ",") != "Ada,Alex" {
		t.Fatalf("tenant=%q displayNames=%#v", directory.tenant, directory.displayNames)
	}
	if !strings.Contains(response.Body.String(), `"status":"ambiguous"`) {
		t.Fatalf("body=%s", response.Body.String())
	}
}

func TestProbeIAMBatchGetUsesCredentialTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	directory := &directoryStub{}
	response := executeProbe(t, NewHandler(&app.Deps{IAMDirectoryService: directory}), map[string]any{
		"module": "iam", "operation": "members.batch_get", "input": map[string]any{"member_uuids": []any{probeMemberUUID}},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if directory.tenant != probeTenantUUID || strings.Join(directory.batchMembers, ",") != probeMemberUUID {
		t.Fatalf("tenant=%q memberUUIDs=%#v", directory.tenant, directory.batchMembers)
	}
}

func TestProbeAIModelsUsesTypedClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	aiClient, err := powerxai.NewClient(powerxai.Config{BaseURL: "http://core.test", BearerToken: "test-token"}, &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/v1/ai/llm/models" {
			t.Fatalf("request=%s %s", request.Method, request.URL.String())
		}
		if request.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("authorization=%q", request.Header.Get("Authorization"))
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"data":{"env":"dev","items":[]}}`))}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	aiRuntime, err := fwai.NewRuntime(fwprovider.ModeDelegated, nil, aiClient)
	if err != nil {
		t.Fatal(err)
	}
	response := executeProbe(t, NewHandler(&app.Deps{AIInvocation: aiRuntime}), map[string]any{"module": "ai", "operation": "models.list"})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestProbeAIModelsPreservesForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	aiClient, err := powerxai.NewClient(powerxai.Config{BaseURL: "http://core.test", BearerToken: "test-token"}, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusForbidden, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":{"reason_code":"AI_FORBIDDEN"}}`))}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	aiRuntime, err := fwai.NewRuntime(fwprovider.ModeDelegated, nil, aiClient)
	if err != nil {
		t.Fatal(err)
	}
	response := executeProbe(t, NewHandler(&app.Deps{AIInvocation: aiRuntime}), map[string]any{"module": "ai", "operation": "models.list"})
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	assertReason(t, response, string(hostcontract.ReasonForbidden))
}

func TestProbeAIModelsMapsUpstreamServerFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	aiClient, err := powerxai.NewClient(powerxai.Config{BaseURL: "http://core.test", BearerToken: "test-token"}, &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":{"reason_code":"AI_UPSTREAM_DEPENDENCY"}}`))}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	aiRuntime, err := fwai.NewRuntime(fwprovider.ModeDelegated, nil, aiClient)
	if err != nil {
		t.Fatal(err)
	}
	response := executeProbe(t, NewHandler(&app.Deps{AIInvocation: aiRuntime}), map[string]any{"module": "ai", "operation": "models.list"})
	if response.Code != http.StatusFailedDependency {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	assertReason(t, response, string(hostcontract.ReasonUpstreamDependency))
}

func TestProbeMediaAssetsUsesTenantFreeTypedClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	media := &mediaStub{}
	mediaRuntime, err := fwmedia.NewRuntime(fwprovider.ModeLocal, media, nil)
	if err != nil {
		t.Fatal(err)
	}
	response := executeProbe(t, NewHandler(&app.Deps{MediaCatalog: mediaRuntime}), map[string]any{"module": "media", "operation": "assets.list", "input": map[string]any{"page": 2, "page_size": 10, "keyword": "image"}})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if media.input.Page != 2 || media.input.PageSize != 10 || media.input.Keyword != "image" {
		t.Fatalf("input=%#v", media.input)
	}
	var envelope struct {
		Data hostcontract.ProbeResult `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.TraceID != "media-trace" {
		t.Fatalf("trace=%q", envelope.Data.TraceID)
	}
}

func TestProbePluginRuntimeReportsDescriptors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := fwpluginruntime.NewRuntime(fwprovider.ModeLocal, pluginRuntimeStub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(&app.Deps{PluginRuntime: runtime, RealtimeDescriptors: []frameworkrealtime.Descriptor{{Key: "_topic.test", Scope: frameworkrealtime.ScopeTenant}}})
	response := executeProbe(t, handler, map[string]any{"module": "plugin_runtime", "operation": "status"})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestProbePluginRuntimeListsThroughSelectedRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &recordingPluginRuntime{}
	runtime, err := fwpluginruntime.NewRuntime(fwprovider.ModeLocal, service, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(&app.Deps{PluginRuntime: runtime})
	response := executeProbe(t, handler, map[string]any{"module": "plugin_runtime", "operation": "knowledge_spaces.list", "input": map[string]any{"page": 2, "page_size": 10, "keyword": "sales"}})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if service.spaces.Page != 2 || service.spaces.PageSize != 10 || service.spaces.Keyword != "sales" {
		t.Fatalf("spaces=%#v", service.spaces)
	}
	response = executeProbe(t, handler, map[string]any{"module": "plugin_runtime", "operation": "agents.list", "input": map[string]any{"env": "production", "status": "active"}})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if service.env != "production" || service.status != "active" {
		t.Fatalf("agents=%q/%q", service.env, service.status)
	}
}

func TestProbePluginRuntimeInstantiateRequiresConfirmationAndUUIDRelations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &recordingPluginRuntime{}
	runtime, err := fwpluginruntime.NewRuntime(fwprovider.ModeLocal, service, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(&app.Deps{PluginRuntime: runtime})
	request := map[string]any{"module": "plugin_runtime", "operation": "agent.instantiate", "input": map[string]any{"name": "Agent", "skill_uuids": []any{"11111111-1111-1111-1111-111111111111"}}}
	response := executeProbe(t, handler, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("unconfirmed status=%d body=%s", response.Code, response.Body.String())
	}
	request["confirm"] = true
	response = executeProbe(t, handler, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if service.instantiate.Name != "Agent" || len(service.instantiate.SkillIDs) != 1 {
		t.Fatalf("agent=%#v", service.instantiate)
	}
}

func TestProbeIntegrationUsesSelectedRuntimeAdapter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := fwintegration.NewRuntime(fwprovider.ModeLocal, integrationGatewayStub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	response := executeProbe(t, NewHandler(&app.Deps{IntegrationGateway: runtime}), map[string]any{"module": "integration_gateway", "operation": "status"})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestProbeIntegrationInvokeRequiresConfirmationAndUsesRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &recordingIntegrationGateway{}
	runtime, err := fwintegration.NewRuntime(fwprovider.ModeLocal, gateway, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(&app.Deps{IntegrationGateway: runtime})
	request := map[string]any{"module": "integration_gateway", "operation": "route.invoke", "input": map[string]any{"route_slug": "orders.sync", "payload": map[string]any{"dry_run": true}}}
	response := executeProbe(t, handler, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("unconfirmed status=%d body=%s", response.Code, response.Body.String())
	}
	request["confirm"] = true
	response = executeProbe(t, handler, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if gateway.slug != "orders.sync" || gateway.input.Payload["dry_run"] != true {
		t.Fatalf("input=%q %#v", gateway.slug, gateway.input)
	}
}

func TestProbeCapabilityListUsesSelectedRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	registry := &recordingCapabilityRegistry{}
	runtime, err := fwcapability.NewRuntime(fwprovider.ModeLocal, registry, nil)
	if err != nil {
		t.Fatal(err)
	}
	response := executeProbe(t, NewHandler(&app.Deps{CapabilityAccess: runtime}), map[string]any{"module": "capability_registry", "operation": "capabilities.list", "input": map[string]any{"page": 2, "page_size": 10, "plugin_id": "plugin.test"}})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if registry.input.Page != 2 || registry.input.PageSize != 10 || registry.input.PluginID != "plugin.test" {
		t.Fatalf("input=%#v", registry.input)
	}
}

func TestProbeCommandModulesUseSelectedRuntimeAdapters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	skillsRuntime, err := fwskills.NewRuntime(fwprovider.ModeLocal, skillInvokerStub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	notificationsRuntime, err := fwnotifications.NewRuntime(fwprovider.ModeLocal, notificationPublisherStub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(&app.Deps{SkillInvocation: skillsRuntime, NotificationDelivery: notificationsRuntime})
	for _, module := range []string{"skills", "notifications"} {
		response := executeProbe(t, handler, map[string]any{"module": module, "operation": "status"})
		if response.Code != http.StatusOK {
			t.Fatalf("module=%s status=%d body=%s", module, response.Code, response.Body.String())
		}
	}
}

func TestProbeSkillInvokeRequiresConfirmationAndUsesRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	invoker := &recordingSkillInvoker{}
	runtime, err := fwskills.NewRuntime(fwprovider.ModeLocal, invoker, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(&app.Deps{SkillInvocation: runtime})
	request := map[string]any{"module": "skills", "operation": "skill.invoke", "input": map[string]any{"skill_id": "skill-1", "payload": map[string]any{"value": "x"}}}
	response := executeProbe(t, handler, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("unconfirmed status=%d body=%s", response.Code, response.Body.String())
	}
	request["confirm"] = true
	response = executeProbe(t, handler, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if invoker.input.SkillID != "skill-1" || invoker.input.Payload["value"] != "x" {
		t.Fatalf("input=%#v", invoker.input)
	}
}

func TestProbeNotificationCreateRequiresConfirmationAndUsesRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	publisher := &recordingNotificationPublisher{}
	runtime, err := fwnotifications.NewRuntime(fwprovider.ModeLocal, publisher, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(&app.Deps{NotificationDelivery: runtime})
	request := map[string]any{"module": "notifications", "operation": "notification.create", "input": map[string]any{"title": "title", "content": "content", "is_important": true}}
	response := executeProbe(t, handler, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("unconfirmed status=%d body=%s", response.Code, response.Body.String())
	}
	request["confirm"] = true
	response = executeProbe(t, handler, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if publisher.input.Title != "title" || publisher.input.Content != "content" || !publisher.input.IsImportant {
		t.Fatalf("input=%#v", publisher.input)
	}
}

type skillInvokerStub struct{}

func (skillInvokerStub) Invoke(context.Context, powerxskills.InvokeInput) (*powerxskills.InvokeOutput, error) {
	return nil, nil
}

type recordingSkillInvoker struct{ input powerxskills.InvokeInput }

func (s *recordingSkillInvoker) Invoke(_ context.Context, input powerxskills.InvokeInput) (*powerxskills.InvokeOutput, error) {
	s.input = input
	return &powerxskills.InvokeOutput{TraceID: "skill-trace", Status: "succeeded"}, nil
}

type notificationPublisherStub struct{}

func (notificationPublisherStub) Create(context.Context, powerxnotifications.CreateInput) (*powerxnotifications.Notification, error) {
	return nil, nil
}

type recordingNotificationPublisher struct {
	input powerxnotifications.CreateInput
}

func (p *recordingNotificationPublisher) Create(_ context.Context, input powerxnotifications.CreateInput) (*powerxnotifications.Notification, error) {
	p.input = input
	return &powerxnotifications.Notification{UUID: "notification-uuid"}, nil
}

type integrationGatewayStub struct{}

func (integrationGatewayStub) ListRoutes(context.Context, powerxintegration.ListRoutesInput) ([]powerxintegration.RouteSummary, error) {
	return nil, nil
}
func (integrationGatewayStub) GetRoute(context.Context, string) (*powerxintegration.RouteDetail, error) {
	return nil, nil
}
func (integrationGatewayStub) InvokeRoute(context.Context, string, powerxintegration.InvokeRouteInput) (*powerxintegration.InvokeRouteOutput, error) {
	return nil, nil
}

type recordingIntegrationGateway struct {
	slug  string
	input powerxintegration.InvokeRouteInput
}

type recordingCapabilityRegistry struct{ input powerxcapability.ListInput }

func (r *recordingCapabilityRegistry) List(_ context.Context, input powerxcapability.ListInput) ([]powerxcapability.Capability, error) {
	r.input = input
	return nil, nil
}
func (*recordingCapabilityRegistry) GrantStatus(context.Context, powerxcapability.GrantStatusInput) ([]powerxcapability.GrantStatusItem, error) {
	return nil, nil
}
func (*recordingCapabilityRegistry) Resolve(context.Context, powerxcapability.ResolveInput) (*powerxcapability.ResolveResult, error) {
	return nil, nil
}
func (*recordingCapabilityRegistry) Invoke(context.Context, powerxcapability.InvokeInput) (*powerxcapability.InvokeResult, error) {
	return nil, nil
}
func (*recordingCapabilityRegistry) GetInvocation(context.Context, string) (*powerxcapability.Invocation, error) {
	return nil, nil
}

func (recordingIntegrationGateway) ListRoutes(context.Context, powerxintegration.ListRoutesInput) ([]powerxintegration.RouteSummary, error) {
	return nil, nil
}
func (recordingIntegrationGateway) GetRoute(context.Context, string) (*powerxintegration.RouteDetail, error) {
	return nil, nil
}
func (g *recordingIntegrationGateway) InvokeRoute(_ context.Context, slug string, input powerxintegration.InvokeRouteInput) (*powerxintegration.InvokeRouteOutput, error) {
	g.slug, g.input = slug, input
	return &powerxintegration.InvokeRouteOutput{TraceID: "integration-trace"}, nil
}

type pluginRuntimeStub struct{}

func (pluginRuntimeStub) ListKnowledgeSpaces(context.Context, powerxruntime.ListKnowledgeSpacesInput) (*powerxruntime.ListKnowledgeSpacesOutput, error) {
	return nil, nil
}
func (pluginRuntimeStub) InstantiateAgent(context.Context, powerxruntime.InstantiateAgentInput) (*powerxruntime.Agent, error) {
	return nil, nil
}
func (pluginRuntimeStub) ListAgents(context.Context, string, string) ([]powerxruntime.Agent, error) {
	return nil, nil
}

type recordingPluginRuntime struct {
	spaces      powerxruntime.ListKnowledgeSpacesInput
	env, status string
	instantiate powerxruntime.InstantiateAgentInput
}

func (s *recordingPluginRuntime) ListKnowledgeSpaces(_ context.Context, input powerxruntime.ListKnowledgeSpacesInput) (*powerxruntime.ListKnowledgeSpacesOutput, error) {
	s.spaces = input
	return &powerxruntime.ListKnowledgeSpacesOutput{}, nil
}
func (s *recordingPluginRuntime) InstantiateAgent(_ context.Context, input powerxruntime.InstantiateAgentInput) (*powerxruntime.Agent, error) {
	s.instantiate = input
	return &powerxruntime.Agent{UUID: "agent-uuid"}, nil
}
func (s *recordingPluginRuntime) ListAgents(_ context.Context, env, status string) ([]powerxruntime.Agent, error) {
	s.env, s.status = env, status
	return nil, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func executeProbe(t *testing.T, handler *Handler, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/admin/host-contract/probe", bytes.NewReader(raw))
	request = request.WithContext(authmw.ContextWithTenantUUID(request.Context(), probeTenantUUID))
	return executeProbeRequest(t, handler, request)
}

func executeProbeWithoutTenant(t *testing.T, handler *Handler, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return executeProbeRequest(t, handler, httptest.NewRequest(http.MethodPost, "/admin/host-contract/probe", bytes.NewReader(raw)))
}

func executeProbeRequest(t *testing.T, handler *Handler, request *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = request
	handler.Probe(context)
	return response
}

func assertReason(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	var envelope struct {
		Error struct {
			ReasonCode string `json:"reason_code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.ReasonCode != want {
		t.Fatalf("reason=%q want=%q", envelope.Error.ReasonCode, want)
	}
}
