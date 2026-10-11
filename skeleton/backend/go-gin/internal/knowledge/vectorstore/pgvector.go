// Package vectorstore provisions the plugin-owned pgvector storage used by
// local knowledge spaces. It deliberately has no Core gateway dependency.
package vectorstore

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// PGVectorConfig is the local equivalent of the Core pgvector storage
// settings. A space activation may override only the table dimension.
type PGVectorConfig struct {
	Schema     string
	Table      string
	Dimensions int
	Lists      int
}

func (c PGVectorConfig) WithDefaults(defaultSchema string) PGVectorConfig {
	c.Schema = strings.TrimSpace(c.Schema)
	if c.Schema == "" {
		c.Schema = strings.TrimSpace(defaultSchema)
	}
	if c.Schema == "" {
		c.Schema = "public"
	}
	c.Table = strings.TrimSpace(c.Table)
	if c.Table == "" {
		c.Table = "local_knowledge_vectors_v1_1536"
	}
	if c.Dimensions <= 0 {
		c.Dimensions = 1536
	}
	if c.Lists <= 0 {
		c.Lists = 100
	}
	return c
}

func quoteIdentifier(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !identifierPattern.MatchString(value) {
		return "", fmt.Errorf("invalid PostgreSQL identifier")
	}
	return `"` + value + `"`, nil
}

// EnsurePGVectorTable provisions one dimension-specific table idempotently.
// It mirrors Core's migration contract: extension, schema, table, indexes,
// then extension/table visibility verification. It never installs the server
// pgvector package; PostgreSQL must provide it already.
func EnsurePGVectorTable(ctx context.Context, db *gorm.DB, cfg PGVectorConfig, defaultSchema string) error {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return fmt.Errorf("pgvector requires PostgreSQL")
	}
	cfg = cfg.WithDefaults(defaultSchema)
	schema, err := quoteIdentifier(cfg.Schema)
	if err != nil {
		return err
	}
	table, err := quoteIdentifier(cfg.Table)
	if err != nil {
		return err
	}
	spaceIndex, err := quoteIdentifier(cfg.Table + "_space_idx")
	if err != nil {
		return err
	}
	embeddingIndex, err := quoteIdentifier(cfg.Table + "_embedding_idx")
	if err != nil {
		return err
	}
	fullTable := schema + "." + table
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS vector`,
		`CREATE SCHEMA IF NOT EXISTS ` + schema,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
            space_uuid uuid NOT NULL,
            chunk_uuid uuid NOT NULL,
            embedding vector(%d) NOT NULL,
            metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
            updated_at timestamptz NOT NULL DEFAULT NOW(),
            PRIMARY KEY (space_uuid, chunk_uuid)
        )`, fullTable, cfg.Dimensions),
		`CREATE INDEX IF NOT EXISTS ` + spaceIndex + ` ON ` + fullTable + ` (space_uuid)`,
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s USING ivfflat (embedding vector_l2_ops) WITH (lists = %d)`, embeddingIndex, fullTable, cfg.Lists),
	}
	for _, statement := range statements {
		if err := db.WithContext(ctx).Exec(statement).Error; err != nil {
			if strings.Contains(statement, "CREATE EXTENSION") {
				return fmt.Errorf("pgvector extension unavailable: install pgvector on PostgreSQL and grant CREATE EXTENSION: %w", err)
			}
			return fmt.Errorf("pgvector provision failed: %w", err)
		}
	}
	var extension int
	if err := db.WithContext(ctx).Raw(`SELECT 1 FROM pg_extension WHERE extname = 'vector' LIMIT 1`).Scan(&extension).Error; err != nil || extension != 1 {
		return fmt.Errorf("pgvector extension was not found after provisioning: %w", err)
	}
	var regclass string
	if err := db.WithContext(ctx).Raw(`SELECT to_regclass(?)`, cfg.Schema+"."+cfg.Table).Scan(&regclass).Error; err != nil {
		return fmt.Errorf("pgvector table visibility check failed: %w", err)
	}
	if strings.TrimSpace(regclass) == "" {
		return fmt.Errorf("pgvector table is not visible after provisioning: %s.%s", cfg.Schema, cfg.Table)
	}
	return nil
}
