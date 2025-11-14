package iam

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	iamm "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type SeedOptions struct {
	TenantKey  string
	TenantName string
	AdminEmail string
	AdminPwd   string
	AdminName  string
}

func SeedLocalAdmin(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	if db == nil {
		return errors.New("iam: db is nil")
	}

	opts := SeedOptions{
		TenantKey:  strings.TrimSpace(os.Getenv("PLUGIN_IAM_TENANT_KEY")),
		TenantName: strings.TrimSpace(os.Getenv("PLUGIN_IAM_TENANT_NAME")),
		AdminEmail: strings.TrimSpace(os.Getenv("PLUGIN_IAM_ADMIN_EMAIL")),
		AdminPwd:   os.Getenv("PLUGIN_IAM_ADMIN_PASSWORD"),
		AdminName:  strings.TrimSpace(os.Getenv("PLUGIN_IAM_ADMIN_NAME")),
	}
	if opts.TenantKey == "" {
		opts.TenantKey = "px_local"
	}
	if opts.TenantName == "" {
		opts.TenantName = "Local Tenant"
	}
	if opts.AdminEmail == "" || strings.TrimSpace(opts.AdminPwd) == "" {
		return fmt.Errorf("PLUGIN_IAM_ADMIN_EMAIL and PLUGIN_IAM_ADMIN_PASSWORD must be provided when IAM mode is local")
	}
	if len(opts.AdminPwd) < 6 {
		return fmt.Errorf("PLUGIN_IAM_ADMIN_PASSWORD must be at least 6 characters")
	}
	if opts.AdminName == "" {
		if idx := strings.Index(opts.AdminEmail, "@"); idx > 0 {
			opts.AdminName = opts.AdminEmail[:idx]
		} else {
			opts.AdminName = "Local Admin"
		}
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(opts.AdminPwd), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tenant iamm.Tenant
		if err := tx.Where("key = ?", strings.ToLower(opts.TenantKey)).First(&tenant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				tenant = iamm.Tenant{
					Key:    strings.ToLower(opts.TenantKey),
					Name:   opts.TenantName,
					Status: iamm.StatusActive,
					Plan:   "free",
				}
				if err := tx.Create(&tenant).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			updates := map[string]any{"name": opts.TenantName}
			if err := tx.Model(&tenant).Updates(updates).Error; err != nil {
				return err
			}
		}

		var user iamm.User
		if err := tx.Where("email = ?", opts.AdminEmail).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				user = iamm.User{
					Email:        strings.ToLower(opts.AdminEmail),
					DisplayName:  opts.AdminName,
					Status:       iamm.StatusActive,
					PasswordHash: string(hashed),
				}
				if err := tx.Create(&user).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			update := map[string]any{
				"display_name":  opts.AdminName,
				"status":        iamm.StatusActive,
				"password_hash": string(hashed),
			}
			if err := tx.Model(&user).Updates(update).Error; err != nil {
				return err
			}
		}

		username := strings.Split(opts.AdminEmail, "@")[0]
		username = strings.ToLower(username)
		if username == "" {
			username = fmt.Sprintf("admin-%d", tenant.ID)
		}

		var member iamm.Member
		memberWhere := "tenant_id = ? AND user_id = ?"
		if err := tx.Where(memberWhere, tenant.ID, user.ID).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				member = iamm.Member{
					BaseModel:   basemodels.BaseModel{TenantID: tenant.ID},
					UserID:      user.ID,
					Username:    username,
					DisplayName: opts.AdminName,
					Status:      iamm.StatusActive,
				}
				if err := tx.Create(&member).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			updates := map[string]any{
				"status":       iamm.StatusActive,
				"display_name": opts.AdminName,
			}
			if err := tx.Model(&member).Updates(updates).Error; err != nil {
				return err
			}
		}

		var role iamm.Role
		if err := tx.Where("tenant_id = ? AND code = ?", tenant.ID, "system.admin").First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				role = iamm.Role{
					BaseModel:   basemodels.BaseModel{TenantID: tenant.ID},
					Code:        "system.admin",
					Name:        "System Admin",
					Description: "Default administrator role",
				}
				if err := tx.Create(&role).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}

		var rel iamm.MemberRole
		if err := tx.Where("member_id = ? AND role_id = ?", member.ID, role.ID).First(&rel).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				rel = iamm.MemberRole{MemberID: member.ID, RoleID: role.ID}
				if err := tx.Create(&rel).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}

		return nil
	})
}
