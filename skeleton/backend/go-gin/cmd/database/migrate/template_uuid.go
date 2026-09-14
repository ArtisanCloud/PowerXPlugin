package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ensureTemplateUUIDs precedes AutoMigrate's NOT NULL/unique constraints. Only
// missing UUIDs are assigned; existing identities are never rewritten. It does
// not remove private numeric keys or reinterpret external numeric input.
func ensureTemplateUUIDs(ctx context.Context, db *gorm.DB) error {
	table := models.S(models.TableTemplate)
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
		// Include soft-deleted rows: UUIDs remain identities after deletion.
		if err := tx.Table(table).Select("id, uuid").Order("id").Scan(&rows).Error; err != nil {
			return err
		}
		seen := make(map[string]bool, len(rows))
		for _, row := range rows {
			value := row.UUID.String
			if row.UUID.Valid && value != "" {
				parsed, err := uuid.Parse(value)
				if err != nil || parsed == uuid.Nil || parsed.String() != value {
					return fmt.Errorf("TEMPLATE_UUID_MIGRATION_INVALID: row=%d", row.ID)
				}
				if seen[value] {
					return fmt.Errorf("TEMPLATE_UUID_MIGRATION_DUPLICATE: row=%d", row.ID)
				}
				seen[value] = true
			}
		}
		for _, row := range rows {
			if row.UUID.Valid && row.UUID.String != "" {
				continue
			}
			value := uuid.NewString()
			for seen[value] {
				value = uuid.NewString()
			}
			seen[value] = true
			if err := tx.Table(table).Where("id = ?", row.ID).Update("uuid", value).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
