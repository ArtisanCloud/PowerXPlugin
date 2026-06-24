package capability_test

import (
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	srvcap "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/capability"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeManager struct {
	entries []capabilities.CatalogEntry
	err     error
}

func (f *fakeManager) ListCapabilities(ctx context.Context) ([]capabilities.CatalogEntry, error) {
	return f.entries, f.err
}

func (f *fakeManager) ExportProtocols(ctx context.Context) ([]capabilities.ProtocolAsset, error) {
	return nil, nil
}

func (f *fakeManager) ExportCatalog(ctx context.Context) (*capabilities.CatalogSnapshot, error) {
	return &capabilities.CatalogSnapshot{Entries: append([]capabilities.CatalogEntry(nil), f.entries...)}, f.err
}

func (f *fakeManager) RegisterWithHost(ctx context.Context, client capabilities.HostSyncClient) error {
	return nil
}

func TestRegisterServiceValidatesDuplicates(t *testing.T) {
	svc := srvcap.NewRegisterService(&app.Deps{
		CapabilitiesManager: &fakeManager{
			entries: []capabilities.CatalogEntry{
				{ID: "com.powerx.plugins.base.template.create"},
			},
		},
	})

	input := sampleInput(false)
	input.Namespace = "com.powerx.plugins.base"
	input.Resource = "template"
	input.Action = "create"

	result, err := svc.Validate(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "capability_id", result.Errors[0].Field)
}

func TestRegisterServiceSubmitDraftAndFinal(t *testing.T) {
	svc := srvcap.NewRegisterService(&app.Deps{
		CapabilitiesManager: &fakeManager{},
	})

	draftInput := sampleInput(true)
	record, validation, err := svc.Submit(context.Background(), draftInput)
	require.NoError(t, err)
	require.Nil(t, validation)
	assert.True(t, record.Draft)
	assert.Equal(t, "com.powerx.plugins.base.template.create", record.ID)
	assert.Equal(t, "draft", record.Status)

	finalInput := sampleInput(false)
	record, validation, err = svc.Submit(context.Background(), finalInput)
	require.NoError(t, err)
	require.Nil(t, validation)
	assert.False(t, record.Draft)
	assert.Equal(t, "under_review", record.Status)
	assert.Equal(t, "medium", record.Sensitivity)
	assert.NotEmpty(t, record.AuditID)
}

func TestAsyncValidation(t *testing.T) {
	svc := srvcap.NewRegisterService(&app.Deps{CapabilitiesManager: &fakeManager{}})
	input := sampleInput(false)
	input.AsyncMode = "async"
	input.AsyncConfig = srvcap.AsyncConfig{}

	result, err := svc.Validate(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Equal(t, "async_config.callback_url", result.Errors[0].Field)

	input.AsyncConfig.CallbackURL = "https://callback.example"
	input.AsyncConfig.StatusEndpoint = "https://status.example"
	result, err = svc.Validate(context.Background(), input)
	require.NoError(t, err)
	assert.True(t, result.Valid)
}

func sampleInput(draft bool) *srvcap.RegisterInput {
	return &srvcap.RegisterInput{
		Namespace:   "com.powerx.plugins.base",
		Resource:    "template",
		Action:      "create",
		Name:        srvcap.LocalizedField{Zh: "创建模板", En: "Create Template"},
		Summary:     srvcap.LocalizedField{Zh: "示例摘要", En: "Sample summary"},
		Description: srvcap.LocalizedField{Zh: "描述", En: "Description"},
		Scenario:    "demo",
		Sensitivity: "medium",
		Tags:        []string{"integration"},
		TenantScope: "global",
		Schemas: srvcap.SchemaPair{
			Input:  "contracts/schema/input/com.powerx.plugins.base.template.create.json",
			Output: "contracts/schema/output/com.powerx.plugins.base.template.create.json",
		},
		Protocols: srvcap.ProtocolMatrix{
			"rest": map[string]string{
				"path":   "/api/v1/templates",
				"method": "POST",
			},
		},
		Samples: srvcap.SampleBundle{
			Request:  map[string]any{"name": "demo"},
			Response: map[string]any{"id": "tpl-1"},
			Errors: []srvcap.SampleError{
				{Code: "INVALID_INPUT", Message: "参数错误"},
			},
		},
		Demo: srvcap.DemoInfo{
			URL:            "https://demo.powerx.cloud",
			CredentialHint: "使用测试租户",
		},
		Owner: srvcap.ContactInfo{
			Name:  "Matrix",
			Email: "matrix@example.com",
		},
		AsyncMode: "sync",
		Draft:     draft,
		Metadata:  map[string]string{"source": "test"},
	}
}
