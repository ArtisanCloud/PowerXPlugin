package customer

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	contactfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/contactfw"
	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	customermodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/customer"
	metadatamodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/metadata"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// FrameworkContactLocalStore is Skeleton's local Contact master-data adapter.
// Its only tenant input is the trusted request context set by the admin route.
type FrameworkContactLocalStore struct{ db *gorm.DB }

const contactIdentityChannelNamespace = "corex.customer.contact_identity_channel"

func NewFrameworkContactLocalStore(db *gorm.DB) *FrameworkContactLocalStore {
	return &FrameworkContactLocalStore{db: db}
}

func (s *FrameworkContactLocalStore) Create(ctx context.Context, in contactfw.CreateContactInput) (*contactfw.Contact, error) {
	if err := contactfw.ValidateCreateInput(in); err != nil {
		return nil, err
	}
	tenantUUID, customerUUID, err := s.activeScope(ctx, in.CustomerUUID)
	if err != nil {
		return nil, err
	}
	roles, tags, err := marshalRolesTags(in.Roles, in.Tags)
	if err != nil {
		return nil, err
	}
	metadata, _ := json.Marshal(map[string]string{"creation_intent": string(in.CreationIntent)})
	row := &customermodel.Contact{
		ContactUUID: uuid.NewString(), TenantUUID: tenantUUID, CustomerUUID: customerUUID,
		DisplayName: strings.TrimSpace(in.DisplayName), GivenName: strings.TrimSpace(in.GivenName), FamilyName: strings.TrimSpace(in.FamilyName),
		Status: string(in.Status), Roles: roles, Tags: tags, Metadata: datatypes.JSON(metadata),
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, localError(err)
	}
	return contactFromModel(row)
}

func (s *FrameworkContactLocalStore) Get(ctx context.Context, in contactfw.GetContactInput) (*contactfw.Contact, error) {
	if err := contactfw.ValidateGetInput(in); err != nil {
		return nil, err
	}
	tenantUUID, customerUUID, err := s.activeScope(ctx, in.CustomerUUID)
	if err != nil {
		return nil, err
	}
	row, err := s.scopedContact(ctx, tenantUUID, customerUUID, in.ContactUUID)
	if err != nil {
		return nil, err
	}
	return contactFromModel(row)
}

func (s *FrameworkContactLocalStore) Update(ctx context.Context, in contactfw.UpdateContactInput) (*contactfw.Contact, error) {
	if err := contactfw.ValidateUpdateInput(in); err != nil {
		return nil, err
	}
	tenantUUID, customerUUID, err := s.activeScope(ctx, in.CustomerUUID)
	if err != nil {
		return nil, err
	}
	var updated *customermodel.Contact
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := s.scopedContactWithDB(ctx, tx, tenantUUID, customerUUID, in.ContactUUID)
		if err != nil {
			return err
		}
		if in.DisplayName != nil {
			row.DisplayName = strings.TrimSpace(*in.DisplayName)
		}
		if in.GivenName != nil {
			row.GivenName = strings.TrimSpace(*in.GivenName)
		}
		if in.FamilyName != nil {
			row.FamilyName = strings.TrimSpace(*in.FamilyName)
		}
		if in.Status != nil {
			row.Status = string(*in.Status)
		}
		if in.Roles != nil {
			raw, err := json.Marshal(*in.Roles)
			if err != nil {
				return err
			}
			row.Roles = datatypes.JSON(raw)
		}
		if in.Tags != nil {
			raw, err := json.Marshal(*in.Tags)
			if err != nil {
				return err
			}
			row.Tags = datatypes.JSON(raw)
		}
		if err := tx.Save(row).Error; err != nil {
			return err
		}
		updated = row
		return nil
	})
	if err != nil {
		return nil, localError(err)
	}
	return contactFromModel(updated)
}

func (s *FrameworkContactLocalStore) ListByCustomer(ctx context.Context, in contactfw.ListByCustomerInput) (contactfw.ContactPage, error) {
	in, err := contactfw.NormalizeListInput(in)
	if err != nil {
		return contactfw.ContactPage{}, err
	}
	tenantUUID, customerUUID, err := s.activeScope(ctx, in.CustomerUUID)
	if err != nil {
		return contactfw.ContactPage{}, err
	}
	q := s.db.WithContext(ctx).Model(&customermodel.Contact{}).Where("tenant_uuid = ? AND customer_uuid = ?", tenantUUID, customerUUID)
	if in.Status != nil {
		q = q.Where("status = ?", string(*in.Status))
	}
	if text := strings.TrimSpace(in.Query); text != "" {
		like := "%" + text + "%"
		q = q.Where("display_name LIKE ? OR given_name LIKE ? OR family_name LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return contactfw.ContactPage{}, localError(err)
	}
	var rows []customermodel.Contact
	if err := q.Order("updated_at DESC").Offset((in.Page - 1) * in.PageSize).Limit(in.PageSize).Find(&rows).Error; err != nil {
		return contactfw.ContactPage{}, localError(err)
	}
	items := make([]contactfw.Contact, 0, len(rows))
	for i := range rows {
		item, err := contactFromModel(&rows[i])
		if err != nil {
			return contactfw.ContactPage{}, err
		}
		items = append(items, *item)
	}
	return contactfw.ContactPage{Items: items, Page: in.Page, PageSize: in.PageSize, Total: total}, nil
}

func (s *FrameworkContactLocalStore) ResolveIdentity(ctx context.Context, in contactfw.ResolveContactIdentityInput) (*contactfw.ContactIdentityResolution, error) {
	if err := contactfw.ValidateResolveIdentityInput(in); err != nil {
		return nil, err
	}
	tenantUUID, customerUUID, err := s.activeScope(ctx, in.CustomerUUID)
	if err != nil {
		return nil, err
	}
	var identity customermodel.ContactIdentity
	err = s.db.WithContext(ctx).Where("tenant_uuid = ? AND channel_dictionary_item_uuid = ? AND external_subject = ?", tenantUUID, in.ChannelDictionaryItemUUID, strings.TrimSpace(in.ExternalSubject)).First(&identity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		var legacyCount int64
		if legacyErr := s.db.WithContext(ctx).Model(&customermodel.ContactIdentity{}).Where("tenant_uuid = ? AND external_subject = ? AND channel_dictionary_item_uuid IS NULL", tenantUUID, strings.TrimSpace(in.ExternalSubject)).Count(&legacyCount).Error; legacyErr != nil {
			return nil, localError(legacyErr)
		}
		if legacyCount > 0 {
			return nil, contactfw.NewError(contactfw.CodeIdentityChannelMigrationRequired, nil)
		}
		return nil, contactfw.NewError(contactfw.CodeIdentityNotFound, err)
	}
	if err != nil {
		return nil, localError(err)
	}
	if identity.CustomerUUID != customerUUID {
		return nil, contactfw.NewError(contactfw.CodeCustomerMismatch, nil)
	}
	if identity.Status != string(contactfw.IdentityStatusActive) {
		return nil, contactfw.NewError(contactfw.CodeIdentityNotFound, nil)
	}
	contact, err := s.scopedContact(ctx, tenantUUID, customerUUID, identity.ContactUUID)
	if err != nil {
		return nil, err
	}
	if contact.Status != string(contactfw.StatusActive) {
		return nil, contactfw.NewError(contactfw.CodeIdentityNotFound, nil)
	}
	contactOut, err := contactFromModel(contact)
	if err != nil {
		return nil, err
	}
	identityOut, err := identityFromModel(&identity)
	if err != nil {
		return nil, err
	}
	return &contactfw.ContactIdentityResolution{Contact: *contactOut, Identity: *identityOut}, nil
}

func (s *FrameworkContactLocalStore) BindIdentity(ctx context.Context, in contactfw.BindContactIdentityInput) (*contactfw.ContactIdentity, error) {
	if err := contactfw.ValidateBindIdentityInput(in); err != nil {
		return nil, err
	}
	tenantUUID, customerUUID, err := s.activeScope(ctx, in.CustomerUUID)
	if err != nil {
		return nil, err
	}
	var identity *customermodel.ContactIdentity
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := s.scopedContactWithDB(ctx, tx, tenantUUID, customerUUID, in.ContactUUID); err != nil {
			return err
		}
		if err := s.validateChannelDictionaryItem(ctx, tx, tenantUUID, in.ChannelDictionaryItemUUID); err != nil {
			return err
		}
		subject := strings.TrimSpace(in.ExternalSubject)
		var existing customermodel.ContactIdentity
		if err := tx.Where("tenant_uuid = ? AND channel_dictionary_item_uuid = ? AND external_subject = ?", tenantUUID, in.ChannelDictionaryItemUUID, subject).First(&existing).Error; err == nil {
			return contactfw.NewError(contactfw.CodeIdentityConflict, nil)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		channelDictionaryItemUUID := strings.TrimSpace(in.ChannelDictionaryItemUUID)
		identity = &customermodel.ContactIdentity{IdentityUUID: uuid.NewString(), TenantUUID: tenantUUID, CustomerUUID: customerUUID, ContactUUID: in.ContactUUID, ChannelDictionaryItemUUID: &channelDictionaryItemUUID, ExternalSubject: subject, Status: string(contactfw.IdentityStatusActive), Metadata: datatypes.JSON([]byte("{}"))}
		return tx.Create(identity).Error
	})
	if err != nil {
		return nil, localError(err)
	}
	return identityFromModel(identity)
}

// MigrateIdentityChannel repairs one legacy local identity after an operator
// explicitly selects an approved dictionary item. It never derives that UUID
// from the historic free-text channel column.
func (s *FrameworkContactLocalStore) MigrateIdentityChannel(ctx context.Context, in contactfw.MigrateContactIdentityChannelInput) (*contactfw.ContactIdentity, error) {
	if err := contactfw.ValidateMigrateIdentityChannelInput(in); err != nil {
		return nil, err
	}
	tenantUUID, customerUUID, err := s.activeScope(ctx, in.CustomerUUID)
	if err != nil {
		return nil, err
	}
	var migrated *customermodel.ContactIdentity
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := s.scopedContactWithDB(ctx, tx, tenantUUID, customerUUID, in.ContactUUID); err != nil {
			return err
		}
		if err := s.validateChannelDictionaryItem(ctx, tx, tenantUUID, in.ChannelDictionaryItemUUID); err != nil {
			return err
		}
		var identity customermodel.ContactIdentity
		if err := tx.Where("tenant_uuid = ? AND customer_uuid = ? AND contact_uuid = ? AND identity_uuid = ?", tenantUUID, customerUUID, in.ContactUUID, in.IdentityUUID).First(&identity).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return contactfw.NewError(contactfw.CodeIdentityNotFound, err)
			}
			return err
		}
		if identity.ChannelDictionaryItemUUID != nil && strings.TrimSpace(*identity.ChannelDictionaryItemUUID) != "" {
			return contactfw.NewError(contactfw.CodeInvalidArgument, nil)
		}
		var conflicting customermodel.ContactIdentity
		if err := tx.Where("tenant_uuid = ? AND channel_dictionary_item_uuid = ? AND external_subject = ?", tenantUUID, in.ChannelDictionaryItemUUID, identity.ExternalSubject).First(&conflicting).Error; err == nil {
			return contactfw.NewError(contactfw.CodeIdentityConflict, nil)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		channelDictionaryItemUUID := strings.TrimSpace(in.ChannelDictionaryItemUUID)
		identity.ChannelDictionaryItemUUID = &channelDictionaryItemUUID
		if err := tx.Save(&identity).Error; err != nil {
			return err
		}
		migrated = &identity
		return nil
	})
	if err != nil {
		return nil, localError(err)
	}
	return identityFromModel(migrated)
}

func (s *FrameworkContactLocalStore) validateChannelDictionaryItem(ctx context.Context, db *gorm.DB, tenantUUID, itemUUID string) error {
	var count int64
	itemsTable := metadatamodel.DictionaryItem{}.TableName()
	namespacesTable := metadatamodel.DictionaryNamespace{}.TableName()
	err := db.WithContext(ctx).Table(itemsTable+" AS di").
		Joins("JOIN "+namespacesTable+" AS dn ON dn.uuid = di.namespace_uuid").
		Where("di.tenant_uuid = ? AND di.uuid = ? AND di.status = ?", tenantUUID, strings.TrimSpace(itemUUID), "enabled").
		Where("dn.tenant_uuid = ? AND dn.namespace = ? AND dn.status = ?", tenantUUID, contactIdentityChannelNamespace, "enabled").
		Count(&count).Error
	if err != nil {
		return localError(err)
	}
	if count != 1 {
		return contactfw.NewError(contactfw.CodeChannelDictionaryInvalid, nil)
	}
	return nil
}

func (s *FrameworkContactLocalStore) activeScope(ctx context.Context, customerUUID string) (string, string, error) {
	if s == nil || s.db == nil {
		return "", "", contactfw.NewError(contactfw.CodeLocalUnavailable, nil)
	}
	tenantUUID, ok := customerfw.TenantUUIDFromContext(ctx)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		return "", "", contactfw.NewError(contactfw.CodeInvalidArgument, nil)
	}
	if _, err := uuid.Parse(strings.TrimSpace(customerUUID)); err != nil {
		return "", "", contactfw.NewError(contactfw.CodeInvalidArgument, err)
	}
	var membership customermodel.CustomerTenantMembership
	err := s.db.WithContext(ctx).Where("tenant_uuid = ? AND customer_uuid = ? AND status = ?", tenantUUID, customerUUID, customermodel.StatusActive).First(&membership).Error
	if err != nil {
		return "", "", contactfw.NewError(contactfw.CodeCustomerMembershipInactive, err)
	}
	var account customermodel.CustomerAccount
	err = s.db.WithContext(ctx).Where("tenant_uuid = ? AND customer_uuid = ? AND status = ?", tenantUUID, customerUUID, customermodel.StatusActive).First(&account).Error
	if err != nil {
		return "", "", contactfw.NewError(contactfw.CodeCustomerMembershipInactive, err)
	}
	return strings.TrimSpace(tenantUUID), strings.TrimSpace(customerUUID), nil
}

func (s *FrameworkContactLocalStore) scopedContact(ctx context.Context, tenantUUID, customerUUID, contactUUID string) (*customermodel.Contact, error) {
	return s.scopedContactWithDB(ctx, s.db, tenantUUID, customerUUID, contactUUID)
}
func (s *FrameworkContactLocalStore) scopedContactWithDB(ctx context.Context, db *gorm.DB, tenantUUID, customerUUID, contactUUID string) (*customermodel.Contact, error) {
	var row customermodel.Contact
	err := db.WithContext(ctx).Where("tenant_uuid = ? AND customer_uuid = ? AND contact_uuid = ?", tenantUUID, customerUUID, contactUUID).First(&row).Error
	if err == nil {
		return &row, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, localError(err)
	}
	var another customermodel.Contact
	if err := db.WithContext(ctx).Where("tenant_uuid = ? AND contact_uuid = ?", tenantUUID, contactUUID).First(&another).Error; err == nil {
		return nil, contactfw.NewError(contactfw.CodeCustomerMismatch, nil)
	}
	return nil, contactfw.NewError(contactfw.CodeNotFound, err)
}

func marshalRolesTags(roles []contactfw.Role, tags []string) (datatypes.JSON, datatypes.JSON, error) {
	rawRoles, err := json.Marshal(roles)
	if err != nil {
		return nil, nil, err
	}
	rawTags, err := json.Marshal(tags)
	if err != nil {
		return nil, nil, err
	}
	return datatypes.JSON(rawRoles), datatypes.JSON(rawTags), nil
}
func contactFromModel(row *customermodel.Contact) (*contactfw.Contact, error) {
	if row == nil {
		return nil, contactfw.NewError(contactfw.CodeNotFound, nil)
	}
	var roles []contactfw.Role
	var tags []string
	var metadata map[string]any
	if err := json.Unmarshal(row.Roles, &roles); err != nil {
		return nil, localError(err)
	}
	if err := json.Unmarshal(row.Tags, &tags); err != nil {
		return nil, localError(err)
	}
	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}
	return &contactfw.Contact{ContactUUID: row.ContactUUID, TenantUUID: row.TenantUUID, CustomerUUID: row.CustomerUUID, DisplayName: row.DisplayName, GivenName: row.GivenName, FamilyName: row.FamilyName, Status: contactfw.Status(row.Status), Roles: roles, Tags: tags, Metadata: metadata, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}
func identityFromModel(row *customermodel.ContactIdentity) (*contactfw.ContactIdentity, error) {
	if row == nil {
		return nil, contactfw.NewError(contactfw.CodeIdentityNotFound, nil)
	}
	if row.ChannelDictionaryItemUUID == nil || strings.TrimSpace(*row.ChannelDictionaryItemUUID) == "" {
		return nil, contactfw.NewError(contactfw.CodeIdentityChannelMigrationRequired, nil)
	}
	var metadata map[string]any
	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}
	return &contactfw.ContactIdentity{IdentityUUID: row.IdentityUUID, TenantUUID: row.TenantUUID, CustomerUUID: row.CustomerUUID, ContactUUID: row.ContactUUID, ChannelDictionaryItemUUID: *row.ChannelDictionaryItemUUID, ExternalSubject: row.ExternalSubject, Status: contactfw.IdentityStatus(row.Status), VerifiedAt: row.VerifiedAt, Metadata: metadata}, nil
}
func localError(err error) error {
	var typed *contactfw.Error
	if errors.As(err, &typed) {
		return err
	}
	return contactfw.NewError(contactfw.CodeLocalUnavailable, err)
}

var _ contactfw.LocalStore = (*FrameworkContactLocalStore)(nil)
var _ contactfw.IdentityChannelMigrator = (*FrameworkContactLocalStore)(nil)
