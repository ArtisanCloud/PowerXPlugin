package regression

import (
	"context"
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	StoreGorm "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression/storegorm"
	Models "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	Record "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/regression"
	Repository "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository"
	"gorm.io/gorm"
)

type RegressionRepository struct {
	*Repository.BaseRepository[Record.RunRecord]
	*StoreGorm.Store
}

func (r *RegressionRepository) Create(ctx context.Context, a Regression.Actor, d Regression.RunDetail, e Regression.RegressionEvent) (Regression.RunDetail, bool, error) {
	return r.Store.Create(ctx, a, d, e)
}
func (r *RegressionRepository) Get(ctx context.Context, tenant, id string) (Regression.RunDetail, error) {
	return r.Store.Get(ctx, tenant, id)
}
func (r *RegressionRepository) List(ctx context.Context, tenant string, page, size uint64, profile string) ([]Regression.Run, uint64, error) {
	return r.Store.List(ctx, tenant, page, size, profile)
}

func NewRegressionRepository(db *gorm.DB) (*RegressionRepository, error) {
	if db == nil || (db.Dialector.Name() != "postgres" && db.Dialector.Name() != "sqlite") {
		return nil, Regression.NewError(Regression.StoreUnavailable)
	}
	base := Repository.NewBaseRepository[Record.RunRecord](db)
	repository := &RegressionRepository{BaseRepository: base}
	store, err := StoreGorm.New(StoreGorm.Tables{Runs: Models.S(Models.TableRegressionRuns), Cases: Models.S(Models.TableRegressionCaseExecutions), Events: Models.S(Models.TableRegressionEvents)}, func(ctx context.Context, tenant string, fn func(*gorm.DB) error) error {
		return repository.WithTenantTx(ctx, tenant, fn)
	})
	if err != nil {
		return nil, err
	}
	repository.Store = store
	return repository, nil
}

// set_config(..., true) binds tenant scope to the transaction and supports prepared statements.
func (r *RegressionRepository) WithTenantTx(ctx context.Context, tenant string, fn func(*gorm.DB) error) error {
	if !Regression.ValidUUID(tenant) || fn == nil {
		return Regression.NewError(Regression.InvalidRequest)
	}
	return r.BaseRepository.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT set_config('app.tenant_uuid', ?, true)", tenant).Error; err != nil {
				return err
			}
		}
		return fn(tx)
	})
}
