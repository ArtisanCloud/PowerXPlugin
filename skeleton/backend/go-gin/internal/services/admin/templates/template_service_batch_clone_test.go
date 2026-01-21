package templates

import (
	"context"
	"testing"

	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	entmodels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	dbtemplate "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTemplateServiceForTest(t *testing.T) (*TemplateService, context.Context) {
	t.Helper()
	entmodels.ForceSchemaForTests("")
	db, err := gorm.Open(dbx.SQLiteDialector("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dbtemplate.Template{}))
	svc := NewTemplateService(db)
	ctx := authx.ContextWithTenantUUID(context.Background(), "tenant-test")
	return svc, ctx
}

func TestTemplateServiceBatchClone(t *testing.T) {
	svc, ctx := newTemplateServiceForTest(t)

	tpl, err := svc.Create(ctx, "Demo", "Desc", "Hello world")
	require.NoError(t, err)

	result, err := svc.BatchClone(ctx, []uint64{tpl.ID}, 2, BatchCloneOptions{NamePrefix: "Clone"})
	require.NoError(t, err)
	require.Len(t, result.CreatedIDs, 2)
	require.Empty(t, result.Failed)

	// invalid source appended -> failure recorded but success preserved
	badResult, err := svc.BatchClone(ctx, []uint64{tpl.ID, 9999}, 1, BatchCloneOptions{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(badResult.CreatedIDs), 1)
	require.Len(t, badResult.Failed, 1)
	require.Equal(t, uint64(9999), badResult.Failed[0].SourceID)
}

func TestTemplateServiceValidate(t *testing.T) {
	svc, ctx := newTemplateServiceForTest(t)

	tpl, err := svc.Create(ctx, "", "", "short")
	require.NoError(t, err)

	res, err := svc.Validate(ctx, tpl.ID, []string{"name_not_empty", "content_min_length"}, false)
	require.NoError(t, err)
	require.False(t, res.Valid)
	require.Equal(t, tpl.ID, res.TemplateID)
	require.Greater(t, len(res.Violations), 0)

	// fix template
	_, err = svc.Update(ctx, tpl.ID, "Fixed", "long desc", "this is a long enough content body")
	require.NoError(t, err)

	res, err = svc.Validate(ctx, tpl.ID, nil, false)
	require.NoError(t, err)
	require.True(t, res.Valid)
}
