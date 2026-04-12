package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	defaultTenantKey     = "00000000-0000-0000-0000-000000000001"
	defaultTenantName    = "Local Tenant"
	defaultAdminEmail    = "admin@local.test"
	defaultAdminPassword = "S3cret!!"
	defaultAdminName     = "Local Admin"
	defaultPolicyVersion = "local.v1"
)

type iamSeedFlags struct {
	Entry         string
	Config        string
	TenantKey     string
	TenantName    string
	AdminEmail    string
	AdminPassword string
	AdminName     string
	Force         bool
	Timeout       time.Duration
}

type iamSeedResult struct {
	TenantUUID string
	TenantKey  string
	TenantName string
	AdminEmail string
	RoleCode   string
}

func runIAMSeed(args []string) error {
	flags := &iamSeedFlags{Timeout: 15 * time.Second}
	fs := flag.NewFlagSet("iam seed", flag.ExitOnError)
	fs.StringVar(&flags.Entry, "entry", "", "Path to the plugin root (default: current directory)")
	fs.StringVar(&flags.Config, "config", "", "Path to backend/etc/config.yaml (default: auto-detect)")
	fs.StringVar(&flags.TenantKey, "tenant-key", "", "Tenant key/UUID for the default admin")
	fs.StringVar(&flags.TenantName, "tenant-name", "", "Tenant name for the default admin")
	fs.StringVar(&flags.AdminEmail, "admin-email", "", "Administrator email")
	fs.StringVar(&flags.AdminPassword, "admin-password", "", "Administrator password (min 6 chars)")
	fs.StringVar(&flags.AdminName, "admin-name", "", "Administrator display name")
	fs.BoolVar(&flags.Force, "force", false, "Force seeding even when Delegated IAM is enabled")
	fs.DurationVar(&flags.Timeout, "timeout", flags.Timeout, "Maximum duration for seeding transaction")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse iam seed flags: %w", err)
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

	if reason := detectDelegatedMode(cfg); reason != "" && !flags.Force {
		return fmt.Errorf("local IAM seeding is disabled (%s indicates delegated mode). Re-run with --force if you really want to override.", reason)
	}

	input := normalizeSeedInput(flags)
	if err := input.Validate(); err != nil {
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

	start := time.Now()
	result, err := seedLocalAdmin(ctx, db, input)
	if err != nil {
		return err
	}

	fmt.Printf("Local IAM admin ready in %s\n  Tenant: %s (%s)\n  Admin:  %s\n  Role:   %s\n",
		time.Since(start).Round(time.Millisecond), result.TenantName, result.TenantUUID, result.AdminEmail, result.RoleCode)
	return nil
}

type iamSeedInput struct {
	TenantKey     string
	TenantName    string
	AdminEmail    string
	AdminPassword string
	AdminName     string
}

func normalizeSeedInput(flags *iamSeedFlags) *iamSeedInput {
	input := &iamSeedInput{
		TenantKey:     strings.ToLower(strings.TrimSpace(flags.TenantKey)),
		TenantName:    strings.TrimSpace(flags.TenantName),
		AdminEmail:    strings.ToLower(strings.TrimSpace(flags.AdminEmail)),
		AdminPassword: strings.TrimSpace(flags.AdminPassword),
		AdminName:     strings.TrimSpace(flags.AdminName),
	}
	if input.TenantKey == "" {
		input.TenantKey = defaultTenantKey
	}
	if input.TenantName == "" {
		input.TenantName = defaultTenantName
	}
	if input.AdminEmail == "" {
		input.AdminEmail = defaultAdminEmail
	}
	if input.AdminPassword == "" {
		input.AdminPassword = defaultAdminPassword
	}
	if input.AdminName == "" {
		if idx := strings.Index(input.AdminEmail, "@"); idx > 0 {
			input.AdminName = input.AdminEmail[:idx]
		} else {
			input.AdminName = defaultAdminName
		}
	}
	return input
}

func (i *iamSeedInput) Validate() error {
	if len(i.AdminPassword) < 6 {
		return errors.New("admin password must be at least 6 characters")
	}
	return nil
}

func seedLocalAdmin(ctx context.Context, db *gorm.DB, input *iamSeedInput) (*iamSeedResult, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	result := &iamSeedResult{AdminEmail: input.AdminEmail}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tenant iamTenantRow
		lookupKey := strings.ToLower(strings.TrimSpace(input.TenantKey))
		q := tx.Table("iam_tenants").Where("LOWER(key) = ?", lookupKey)
		if err := q.First(&tenant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				tenant = iamTenantRow{
					UUID:   ensureTenantUUID(input.TenantKey),
					Key:    lookupKey,
					Name:   input.TenantName,
					Status: "active",
					Plan:   "free",
				}
				if err := tx.Table("iam_tenants").Create(&tenant).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			updates := map[string]any{
				"name":   input.TenantName,
				"status": "active",
			}
			if strings.TrimSpace(tenant.UUID) == "" {
				updates["uuid"] = ensureTenantUUID(input.TenantKey)
			}
			if err := tx.Table("iam_tenants").Where("id = ?", tenant.ID).Updates(updates).Error; err != nil {
				return err
			}
			if val, ok := updates["uuid"]; ok {
				tenant.UUID = val.(string)
			}
			tenant.Name = input.TenantName
		}

		result.TenantUUID = tenant.UUID
		result.TenantKey = tenant.Key
		result.TenantName = tenant.Name

		var account iamAccountRow
		if err := tx.Table("iam_users").Where("LOWER(email) = ?", input.AdminEmail).First(&account).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				account = iamAccountRow{
					Email:       input.AdminEmail,
					DisplayName: input.AdminName,
					Status:      "active",
					Phone:       "",
				}
				account.PasswordHash = string(hashed)
				if err := tx.Table("iam_users").Create(&account).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			update := map[string]any{
				"display_name":  input.AdminName,
				"status":        "active",
				"password_hash": string(hashed),
			}
			if err := tx.Table("iam_users").Where("id = ?", account.ID).Updates(update).Error; err != nil {
				return err
			}
		}

		username := input.AdminEmail
		if idx := strings.Index(username, "@"); idx > 0 {
			username = username[:idx]
		}
		if username == "" {
			username = fmt.Sprintf("admin-%d", tenant.ID)
		}

		tenantUUID := ensureTenantUUIDValue(tenant)
		var member iamMemberRow
		where := "tenant_uuid = ? AND user_id = ?"
		if err := tx.Table("iam_members").Where(where, tenantUUID, account.ID).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				member = iamMemberRow{
					TenantUUID:  tenantUUID,
					UserID:      account.ID,
					Username:    strings.ToLower(username),
					DisplayName: input.AdminName,
					Status:      "active",
				}
				if err := tx.Table("iam_members").Create(&member).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			updates := map[string]any{
				"display_name": input.AdminName,
				"status":       "active",
			}
			if err := tx.Table("iam_members").Where("id = ?", member.ID).Updates(updates).Error; err != nil {
				return err
			}
		}

		var role iamRoleRow
		roleHasPolicyColumn := hasColumn(tx, "iam_roles", "policy_version")
		roleHasScopeColumn := hasColumn(tx, "iam_roles", "scope_type")
		if err := tx.Table("iam_roles").Where("tenant_uuid = ? AND code = ?", tenantUUID, "system.admin").First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				roleData := map[string]any{
					"tenant_uuid": tenantUUID,
					"code":        "system.admin",
					"name":        "System Admin",
					"description": "Default administrator role",
				}
				if roleHasScopeColumn {
					roleData["scope_type"] = "system"
				}
				if roleHasPolicyColumn {
					roleData["policy_version"] = defaultPolicyVersion
				}
				if err := tx.Table("iam_roles").Create(roleData).Error; err != nil {
					return err
				}
				if err := tx.Table("iam_roles").Where("tenant_uuid = ? AND code = ?", tenantUUID, "system.admin").First(&role).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			updates := map[string]any{}
			if roleHasPolicyColumn && strings.TrimSpace(role.PolicyVersion) == "" {
				updates["policy_version"] = defaultPolicyVersion
			}
			if roleHasScopeColumn && strings.TrimSpace(role.ScopeType) == "" {
				updates["scope_type"] = "system"
			}
			if len(updates) > 0 {
				if err := tx.Table("iam_roles").Where("id = ?", role.ID).Updates(updates).Error; err != nil {
					return err
				}
			}
		}

		result.RoleCode = role.Code

		if err := tx.Table("iam_member_roles").Where("member_id = ? AND role_id = ?", member.ID, role.ID).
			Assign(map[string]any{"created_at": time.Now()}).
			FirstOrCreate(&iamMemberRoleRow{}).Error; err != nil {
			return err
		}

		deptID, err := ensureDefaultDepartmentTx(tx, tenantUUID)
		if err != nil {
			return err
		}
		if deptID != nil {
			if err := tx.Table("iam_members").Where("id = ?", member.ID).Update("department_id", *deptID).Error; err != nil {
				return err
			}
		}

		if err := seedDefaultPermissionsTx(tx, role.ID, tenantUUID); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func ensureTenantUUIDValue(t iamTenantRow) string {
	uuidVal := strings.ToLower(strings.TrimSpace(t.UUID))
	if uuidVal != "" {
		return uuidVal
	}
	return ensureTenantUUID(t.Key)
}

func ensureTenantUUID(candidate string) string {
	trimmed := strings.ToLower(strings.TrimSpace(candidate))
	if trimmed != "" {
		if _, err := uuid.Parse(trimmed); err == nil {
			return trimmed
		}
	}
	return strings.ToLower(uuid.NewString())
}

func ensureDefaultDepartmentTx(tx *gorm.DB, tenantUUID string) (*uint64, error) {
	var dept iamDepartmentRow
	cond := tx.Table("iam_departments").Where("tenant_uuid = ? AND code = ?", tenantUUID, "general")
	if err := cond.First(&dept).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dept = iamDepartmentRow{
				TenantUUID:  tenantUUID,
				Name:        "General",
				Code:        "general",
				Description: "Default department",
				Path:        "general",
			}
			if err := tx.Table("iam_departments").Create(&dept).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &dept.ID, nil
}

func seedDefaultPermissionsTx(tx *gorm.DB, roleID uint64, tenantUUID string) error {
	perms := []struct {
		Resource string
		Action   string
		Desc     string
	}{
		{"*", "*", "Full access"},
		{"iam.user", "read", "Read IAM users"},
		{"iam.role", "read", "Read IAM roles"},
		{"iam.department", "read", "Read IAM departments"},
	}
	for _, p := range perms {
		var perm iamPermissionRow
		if err := tx.Table("iam_permissions").Where("resource = ? AND action = ?", p.Resource, p.Action).First(&perm).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				perm = iamPermissionRow{
					Resource:    p.Resource,
					Action:      p.Action,
					Description: p.Desc,
				}
				if err := tx.Table("iam_permissions").Create(&perm).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
		assign := map[string]any{}
		if hasColumn(tx, "iam_role_permissions", "tenant_uuid") {
			assign["tenant_uuid"] = strings.ToLower(tenantUUID)
		}
		if hasColumn(tx, "iam_role_permissions", "policy_version") {
			assign["policy_version"] = defaultPolicyVersion
		}
		if err := tx.Table("iam_role_permissions").Where("role_id = ? AND permission_id = ?", roleID, perm.ID).
			Assign(assign).FirstOrCreate(&iamRolePermissionRow{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func hasColumn(db *gorm.DB, table, column string) bool {
	if db == nil || table == "" || column == "" {
		return false
	}
	return db.Migrator().HasColumn(table, column)
}
