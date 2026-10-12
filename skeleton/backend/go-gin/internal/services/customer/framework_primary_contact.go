package customer

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/contactfw"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	audit "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/admin_console"
	repository "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/customer"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// The service principal is bound by bootstrap, never supplied by the request.
type FrameworkPrimaryContactManager struct {
	repository *repository.ContactRepository
	pluginID   string
}

func NewFrameworkPrimaryContactManager(repo *repository.ContactRepository, pluginID string) *FrameworkPrimaryContactManager {
	return &FrameworkPrimaryContactManager{repository: repo, pluginID: pluginID}
}

func (s *FrameworkPrimaryContactManager) SetPrimary(ctx context.Context, in fw.SetPrimaryContactInput) (*fw.PrimaryContactResult, error) {
	if err := fw.ValidateSetPrimaryInput(in); err != nil {
		return nil, err
	}
	if s == nil || s.repository == nil || strings.TrimSpace(s.pluginID) == "" {
		return nil, fw.NewError(fw.CodeLocalUnavailable, nil)
	}
	tenant, ok := customerfw.TenantUUIDFromContext(ctx)
	id, err := uuid.Parse(tenant)
	if !ok || err != nil || id == uuid.Nil {
		return nil, fw.NewError(fw.CodeInvalidArgument, nil)
	}
	var result fw.PrimaryContactResult
	err = s.repository.Transaction(ctx, tenant, in.CustomerUUID, func(tx *repository.ContactTransaction) error {
		target, err := tx.Contact(in.ContactUUID)
		if err != nil {
			return err
		}
		if target.Status != string(fw.StatusActive) {
			return fw.NewError(fw.CodePrimaryNotActive, nil)
		}
		previous := tx.Membership.PrimaryContactUUID
		if previous != *in.ExpectedPrimaryContactUUID {
			return fw.NewError(fw.CodePrimaryChanged, nil)
		}
		result = fw.PrimaryContactResult{CustomerUUID: in.CustomerUUID, PrimaryContactUUID: in.ContactUUID, PreviousPrimaryContactUUID: previous}
		rows, err := tx.Contacts()
		if err != nil {
			return err
		}
		changed := previous != in.ContactUUID
		for _, row := range rows {
			var roles []fw.Role
			if err := json.Unmarshal(row.Roles, &roles); err != nil {
				return err
			}
			kept := []fw.Role{}
			for _, role := range roles {
				if role != fw.RolePrimary {
					kept = append(kept, role)
				}
			}
			if row.ContactUUID == in.ContactUUID {
				kept = append(kept, fw.RolePrimary)
			}
			raw, err := json.Marshal(kept)
			if err != nil {
				return err
			}
			if string(raw) == string(row.Roles) {
				continue
			}
			row.Roles = datatypes.JSON(raw)
			changed = true
			if err := tx.Save(&row); err != nil {
				return err
			}
		}
		if !changed {
			return nil
		}
		if err := tx.SetPrimaryReference(in.ContactUUID); err != nil {
			return err
		}
		diff, err := json.Marshal(result)
		if err != nil {
			return err
		}
		return tx.Audit(&audit.AuditEvent{ID: uuid.NewString(), PluginID: s.pluginID, TenantUuid: &tenant, ActorID: s.pluginID, PermissionCode: fw.CapabilityContactsPrimaryManage, Action: "customer.contact.primary_changed", ResourceType: "customer_contact", ResourceRef: &in.ContactUUID, Diff: datatypes.JSON(diff), OccurredAt: time.Now().UTC()})
	})
	if err != nil {
		if fw.CodeOf(err) != "" {
			return nil, err
		}
		return nil, fw.NewError(fw.CodePrimaryOperationFailed, err)
	}
	return &result, nil
}

var _ fw.PrimaryContactManager = (*FrameworkPrimaryContactManager)(nil)
