package migrate

import (
	"context"
	customer "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/customer"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func ensureCustomerIdentityUUIDs(ctx context.Context, db *gorm.DB) error {
	m := &customer.CustomerAuthIdentity{}
	if !db.Migrator().HasTable(m) {
		return nil
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Add nullable first, backfill, then normal AutoMigrate enforces NOT NULL/unique.
		if !tx.Migrator().HasColumn(m, "identity_uuid") {
			if err := tx.Exec("ALTER TABLE " + m.TableName() + " ADD COLUMN identity_uuid UUID").Error; err != nil {
				return err
			}
		}
		var rows []struct{ ID uint64 }
		if err := tx.Unscoped().Model(m).Select("id").Where("identity_uuid IS NULL").Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if err := tx.Unscoped().Model(m).Where("id = ?", row.ID).UpdateColumn("identity_uuid", uuid.NewString()).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
