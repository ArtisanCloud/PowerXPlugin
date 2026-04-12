package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/powerx-plugin/cli/internal/manifest"
	"gorm.io/gorm"
)

type iamExportFlags struct {
	Entry   string
	Config  string
	Output  string
	Tenant  string
	Format  string
	Pretty  bool
	Timeout time.Duration
}

type iamTenantRow struct {
	ID        uint64    `json:"id"`
	UUID      string    `json:"uuid"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type iamAccountRow struct {
	ID           uint64    `json:"id"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	DisplayName  string    `json:"display_name" gorm:"column:display_name"`
	Status       string    `json:"status"`
	PasswordHash string    `json:"-" gorm:"column:password_hash"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type iamMemberRow struct {
	ID           uint64     `json:"id"`
	TenantUUID   string     `json:"tenant_uuid" gorm:"column:tenant_uuid"`
	UserID       uint64     `json:"user_id" gorm:"column:user_id"`
	Username     string     `json:"username"`
	DisplayName  string     `json:"display_name" gorm:"column:display_name"`
	Status       string     `json:"status"`
	DepartmentID *uint64    `json:"department_id" gorm:"column:department_id"`
	LastLoginAt  *time.Time `json:"last_login_at" gorm:"column:last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type iamRoleRow struct {
	ID            uint64    `json:"id"`
	TenantUUID    string    `json:"tenant_uuid" gorm:"column:tenant_uuid"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	ScopeType     string    `json:"scope_type" gorm:"column:scope_type"`
	PolicyVersion string    `json:"policy_version" gorm:"column:policy_version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type iamPermissionRow struct {
	ID          uint64    `json:"id"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type iamDepartmentRow struct {
	ID          uint64    `json:"id"`
	TenantUUID  string    `json:"tenant_uuid" gorm:"column:tenant_uuid"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	ParentID    *uint64   `json:"parent_id" gorm:"column:parent_id"`
	Description string    `json:"description"`
	Path        string    `json:"path"`
	SortOrder   int       `json:"sort_order" gorm:"column:sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type iamMemberRoleRow struct {
	MemberID  uint64    `json:"member_id" gorm:"column:member_id"`
	RoleID    uint64    `json:"role_id" gorm:"column:role_id"`
	CreatedAt time.Time `json:"created_at"`
}

type iamRolePermissionRow struct {
	RoleID        uint64    `json:"role_id" gorm:"column:role_id"`
	PermissionID  uint64    `json:"permission_id" gorm:"column:permission_id"`
	TenantUUID    string    `json:"tenant_uuid" gorm:"column:tenant_uuid"`
	PolicyVersion string    `json:"policy_version" gorm:"column:policy_version"`
	CreatedAt     time.Time `json:"created_at"`
}

type iamExportPayload struct {
	GeneratedAt     time.Time              `json:"generated_at"`
	PluginID        string                 `json:"plugin_id"`
	TenantFilter    string                 `json:"tenant_filter,omitempty"`
	Driver          string                 `json:"driver"`
	Schema          string                 `json:"schema,omitempty"`
	Tenants         []iamTenantRow         `json:"tenants"`
	Departments     []iamDepartmentRow     `json:"departments"`
	Accounts        []iamAccountRow        `json:"accounts"`
	Members         []iamMemberRow         `json:"members"`
	Roles           []iamRoleRow           `json:"roles"`
	Permissions     []iamPermissionRow     `json:"permissions"`
	MemberRoles     []iamMemberRoleRow     `json:"member_roles"`
	RolePermissions []iamRolePermissionRow `json:"role_permissions"`
}

func runIAMExport(args []string) error {
	flags := &iamExportFlags{Timeout: 10 * time.Second, Format: "json"}
	fs := flag.NewFlagSet("iam export", flag.ExitOnError)
	fs.StringVar(&flags.Entry, "entry", "", "Path to the plugin root (default: current directory)")
	fs.StringVar(&flags.Config, "config", "", "Path to backend/etc/config.yaml (default: auto-detect)")
	fs.StringVar(&flags.Output, "output", "", "Path to write export data (default: stdout)")
	fs.StringVar(&flags.Tenant, "tenant", "", "Filter by tenant key or UUID (case-insensitive)")
	fs.StringVar(&flags.Format, "format", "json", "Export format (currently only json)")
	fs.BoolVar(&flags.Pretty, "pretty", false, "Pretty-print JSON output")
	fs.DurationVar(&flags.Timeout, "timeout", flags.Timeout, "Maximum duration for DB queries")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse iam export flags: %w", err)
	}
	if flags.Format != "json" {
		return fmt.Errorf("unsupported export format: %s", flags.Format)
	}

	entryPath, err := resolveEntryPath(flags.Entry)
	if err != nil {
		return err
	}
	configPath, err := resolveIAMConfigPath(entryPath, flags.Config)
	if err != nil {
		return err
	}
	cfg, err := loadIAMConfig(configPath)
	if err != nil {
		return err
	}

	db, sqlDB, driver, err := connectIAMDatabase(cfg)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	if err := applySchemaSearchPath(db, driver, cfg.Database.Schema); err != nil {
		return fmt.Errorf("set schema search_path: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), flags.Timeout)
	defer cancel()

	var pluginID string
	if m, err := manifest.Load(entryPath); err == nil {
		pluginID = m.ID
	}

	start := time.Now()
	payload, err := collectIAMExport(ctx, db, flags.Tenant, driver, cfg.Database.Schema, pluginID)
	if err != nil {
		return err
	}
	payload.GeneratedAt = time.Now()

	var raw []byte
	if flags.Pretty {
		raw, err = json.MarshalIndent(payload, "", "  ")
	} else {
		raw, err = json.Marshal(payload)
	}
	if err != nil {
		return fmt.Errorf("encode export json: %w", err)
	}

	if flags.Output != "" {
		outputPath := flags.Output
		if !filepath.IsAbs(outputPath) {
			outputPath = filepath.Join(entryPath, outputPath)
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
		if err := os.WriteFile(outputPath, raw, 0o600); err != nil {
			return fmt.Errorf("write export file: %w", err)
		}
		fmt.Printf("IAM export finished in %s\n  Entry: %s\n  Config: %s\n  Driver: %s (%s)\n  Output: %s (%d bytes)\n  Tenants: %d  Members: %d  Roles: %d  Permissions: %d\n",
			time.Since(start).Round(time.Millisecond), entryPath, configPath, driver, maskDSN(cfg.Database.DSN), outputPath, len(raw), len(payload.Tenants), len(payload.Members), len(payload.Roles), len(payload.Permissions))
	} else {
		fmt.Println(string(raw))
		fmt.Printf("IAM export printed to stdout in %s (tenants=%d members=%d roles=%d permissions=%d)\n",
			time.Since(start).Round(time.Millisecond), len(payload.Tenants), len(payload.Members), len(payload.Roles), len(payload.Permissions))
	}
	return nil
}

func collectIAMExport(ctx context.Context, db *gorm.DB, tenantFilter, driver, schema, pluginID string) (*iamExportPayload, error) {
	payload := &iamExportPayload{PluginID: pluginID, Driver: driver, Schema: schema}
	payload.TenantFilter = strings.TrimSpace(tenantFilter)

	tenants, err := queryTenants(ctx, db, payload.TenantFilter)
	if err != nil {
		return nil, err
	}
	if payload.TenantFilter != "" && len(tenants) == 0 {
		return nil, fmt.Errorf("tenant %s not found", payload.TenantFilter)
	}
	payload.Tenants = tenants

	tenantUUIDs := collectTenantUUIDs(tenants)

	departments, err := queryDepartments(ctx, db, tenantUUIDs)
	if err != nil {
		return nil, err
	}
	payload.Departments = departments

	members, err := queryMembers(ctx, db, tenantUUIDs)
	if err != nil {
		return nil, err
	}
	payload.Members = members

	userIDs := uniqueUint64(func() []uint64 {
		ids := make([]uint64, 0, len(members))
		for _, m := range members {
			ids = append(ids, m.UserID)
		}
		return ids
	}())

	if len(userIDs) > 0 {
		accounts, err := queryAccounts(ctx, db, userIDs)
		if err != nil {
			return nil, err
		}
		payload.Accounts = accounts
	}

	roles, err := queryRoles(ctx, db, tenantUUIDs)
	if err != nil {
		return nil, err
	}
	payload.Roles = roles

	permissions, err := queryPermissions(ctx, db)
	if err != nil {
		return nil, err
	}
	payload.Permissions = permissions

	rolePermissions, err := queryRolePermissions(ctx, db, tenantUUIDs, roles)
	if err != nil {
		return nil, err
	}
	payload.RolePermissions = rolePermissions

	memberIDs := uniqueUint64(func() []uint64 {
		ids := make([]uint64, 0, len(members))
		for _, m := range members {
			ids = append(ids, m.ID)
		}
		return ids
	}())

	if len(memberIDs) > 0 {
		memberRoles, err := queryMemberRoles(ctx, db, memberIDs)
		if err != nil {
			return nil, err
		}
		payload.MemberRoles = memberRoles
	}

	return payload, nil
}

func queryTenants(ctx context.Context, db *gorm.DB, tenantFilter string) ([]iamTenantRow, error) {
	var rows []iamTenantRow
	query := db.WithContext(ctx).Table("iam_tenants").Order("id ASC")
	filter := strings.TrimSpace(tenantFilter)
	if filter != "" {
		keyFilter := strings.ToLower(filter)
		if parsed, err := uuid.Parse(filter); err == nil {
			canonical := strings.ToLower(parsed.String())
			query = query.Where("uuid = ? OR LOWER(key) = ?", canonical, keyFilter)
		} else {
			query = query.Where("LOWER(key) = ?", keyFilter)
		}
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query tenants: %w", err)
	}
	return rows, nil
}

func queryDepartments(ctx context.Context, db *gorm.DB, tenantUUIDs []string) ([]iamDepartmentRow, error) {
	var rows []iamDepartmentRow
	query := db.WithContext(ctx).Table("iam_departments").Order("tenant_uuid, id")
	if len(tenantUUIDs) > 0 {
		query = query.Where("tenant_uuid IN ?", tenantUUIDs)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query departments: %w", err)
	}
	return rows, nil
}

func queryMembers(ctx context.Context, db *gorm.DB, tenantUUIDs []string) ([]iamMemberRow, error) {
	var rows []iamMemberRow
	query := db.WithContext(ctx).Table("iam_members").Order("tenant_uuid, id")
	if len(tenantUUIDs) > 0 {
		query = query.Where("tenant_uuid IN ?", tenantUUIDs)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query members: %w", err)
	}
	return rows, nil
}

func queryAccounts(ctx context.Context, db *gorm.DB, ids []uint64) ([]iamAccountRow, error) {
	var rows []iamAccountRow
	query := db.WithContext(ctx).Table("iam_users").Order("id")
	query = query.Where("id IN ?", ids)
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	return rows, nil
}

func queryRoles(ctx context.Context, db *gorm.DB, tenantUUIDs []string) ([]iamRoleRow, error) {
	var rows []iamRoleRow
	query := db.WithContext(ctx).Table("iam_roles").Order("tenant_uuid, code")
	if len(tenantUUIDs) > 0 {
		query = query.Where("tenant_uuid IN ?", tenantUUIDs)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query roles: %w", err)
	}
	return rows, nil
}

func queryPermissions(ctx context.Context, db *gorm.DB) ([]iamPermissionRow, error) {
	var rows []iamPermissionRow
	if err := db.WithContext(ctx).Table("iam_permissions").Order("resource, action").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query permissions: %w", err)
	}
	return rows, nil
}

func queryRolePermissions(ctx context.Context, db *gorm.DB, tenantUUIDs []string, roles []iamRoleRow) ([]iamRolePermissionRow, error) {
	var rows []iamRolePermissionRow
	query := db.WithContext(ctx).Table("iam_role_permissions").Order("role_id, permission_id")
	if len(tenantUUIDs) > 0 {
		roleIDs := filterRoleIDsByTenant(roles, tenantUUIDs)
		if len(roleIDs) == 0 {
			return rows, nil
		}
		query = query.Where("role_id IN ?", roleIDs)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query role permissions: %w", err)
	}
	return rows, nil
}

func filterRoleIDsByTenant(roles []iamRoleRow, tenantUUIDs []string) []uint64 {
	if len(tenantUUIDs) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(tenantUUIDs))
	for _, t := range tenantUUIDs {
		set[strings.ToLower(t)] = struct{}{}
	}
	ids := make([]uint64, 0, len(roles))
	for _, role := range roles {
		if role.ID == 0 {
			continue
		}
		if len(set) > 0 {
			if _, ok := set[strings.ToLower(strings.TrimSpace(role.TenantUUID))]; !ok {
				continue
			}
		}
		ids = append(ids, role.ID)
	}
	return uniqueUint64(ids)
}

func queryMemberRoles(ctx context.Context, db *gorm.DB, memberIDs []uint64) ([]iamMemberRoleRow, error) {
	var rows []iamMemberRoleRow
	query := db.WithContext(ctx).Table("iam_member_roles").Order("member_id, role_id")
	query = query.Where("member_id IN ?", memberIDs)
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query member roles: %w", err)
	}
	return rows, nil
}

func collectTenantUUIDs(rows []iamTenantRow) []string {
	out := make([]string, 0, len(rows))
	seen := make(map[string]struct{})
	for _, t := range rows {
		u := strings.ToLower(strings.TrimSpace(t.UUID))
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	sort.Strings(out)
	return out
}

func uniqueUint64(values []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(values))
	result := make([]uint64, 0, len(values))
	for _, v := range values {
		if v == 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
