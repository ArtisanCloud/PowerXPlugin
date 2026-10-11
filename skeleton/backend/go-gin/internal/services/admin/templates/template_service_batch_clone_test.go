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

	result, err := svc.BatchClone(ctx, []string{tpl.UUID}, 2, BatchCloneOptions{NamePrefix: "Clone"})
	require.NoError(t, err)
	require.Len(t, result.CreatedUUIDs, 2)
	require.Empty(t, result.Failed)
	for _, ids := range [][]string{{tpl.UUID, "123"}, {tpl.UUID, tpl.UUID}} {
		before, queryErr := svc.List(ctx, "", 1, 100)
		require.NoError(t, queryErr)
		_, invalidErr := svc.BatchClone(ctx, ids, 1, BatchCloneOptions{})
		require.ErrorIs(t, invalidErr, gorm.ErrInvalidData)
		after, queryErr := svc.List(ctx, "", 1, 100)
		require.NoError(t, queryErr)
		require.Equal(t, before.Total, after.Total)
	}

	// invalid source appended -> failure recorded but success preserved
	badResult, err := svc.BatchClone(ctx, []string{tpl.UUID, "11111111-1111-4111-8111-111111111111"}, 1, BatchCloneOptions{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(badResult.CreatedUUIDs), 1)
	require.Len(t, badResult.Failed, 1)
	require.Equal(t, "11111111-1111-4111-8111-111111111111", badResult.Failed[0].SourceUUID)
}

func TestTemplateServiceValidate(t *testing.T) {
	svc, ctx := newTemplateServiceForTest(t)

	tpl, err := svc.Create(ctx, "", "", "short")
	require.NoError(t, err)

	res, err := svc.Validate(ctx, tpl.UUID, []string{"name_not_empty", "content_min_length"}, false)
	require.NoError(t, err)
	require.False(t, res.Valid)
	require.Equal(t, tpl.UUID, res.TemplateUUID)
	require.Greater(t, len(res.Violations), 0)

	// fix template
	_, err = svc.Update(ctx, tpl.UUID, "Fixed", "long desc", "this is a long enough content body")
	require.NoError(t, err)

	res, err = svc.Validate(ctx, tpl.UUID, nil, false)
	require.NoError(t, err)
	require.True(t, res.Valid)
}
