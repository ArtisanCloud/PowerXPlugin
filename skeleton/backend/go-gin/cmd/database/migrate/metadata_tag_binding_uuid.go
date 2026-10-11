package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ensureMetadataTagBindingUUIDs gives every historical binding a stable public
// identity before AutoMigrate enforces the non-null/unique model contract.
func ensureMetadataTagBindingUUIDs(ctx context.Context, db *gorm.DB) error {
	table := models.S(models.TableMetadataTagBindings)
	exists, err := iamTableExists(ctx, db, table)
	if err != nil || !exists {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureUUIDColumn(ctx, tx, table); err != nil {
			return err
		}
		var rows []struct {
			ID   uint64
			UUID sql.NullString
		}
		if err := tx.Table(table).Select("id, uuid").Order("id").Scan(&rows).Error; err != nil {
			return err
		}
		seen := make(map[string]struct{}, len(rows))
		for _, row := range rows {
			if !row.UUID.Valid || row.UUID.String == "" {
				continue
			}
			parsed, parseErr := uuid.Parse(row.UUID.String)
			if parseErr != nil || parsed == uuid.Nil || parsed.String() != row.UUID.String {
				return fmt.Errorf("METADATA_TAG_BINDING_UUID_MIGRATION_INVALID: row=%d", row.ID)
			}
			if _, duplicate := seen[row.UUID.String]; duplicate {
				return fmt.Errorf("METADATA_TAG_BINDING_UUID_MIGRATION_DUPLICATE: row=%d", row.ID)
			}
			seen[row.UUID.String] = struct{}{}
		}
		for _, row := range rows {
			if row.UUID.Valid && row.UUID.String != "" {
				continue
			}
			value := uuid.NewString()
			for {
				if _, exists := seen[value]; !exists {
					break
				}
				value = uuid.NewString()
			}
			seen[value] = struct{}{}
			if err := tx.Table(table).Where("id = ?", row.ID).Update("uuid", value).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
