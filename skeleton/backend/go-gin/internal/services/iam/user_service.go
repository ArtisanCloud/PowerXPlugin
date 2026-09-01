package iam

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	iamm "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/iam"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserService struct {
	db    *gorm.DB
	audit *AuditService
}

func NewUserService(db *gorm.DB, audit *AuditService) *UserService {
	return &UserService{db: db, audit: audit}
}

type UserFilter struct {
	TenantUUID string
	Status     string
	Query      string
}

type UserView struct {
	ID              uint64            `json:"-"`
	UserID          uint64            `json:"-"`
	MemberUUID      string            `json:"member_uuid"`
	TenantUUID      string            `json:"tenant_uuid"`
	Email           string            `json:"email"`
	Phone           string            `json:"phone"`
	DisplayName     string            `json:"display_name"`
	Username        string            `json:"username"`
	Status          string            `json:"status"`
	DepartmentID    *uint64           `json:"-"`
	DepartmentIDs   []uint64          `json:"-" gorm:"-"`
	DepartmentUUIDs []string          `json:"department_uuids,omitempty" gorm:"-"`
	Meta            datatypes.JSONMap `json:"-"`
	LastLoginAt     *time.Time        `json:"last_login_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	Roles           []string          `json:"roles" gorm:"-"`
}

type UserBulkImportResult struct {
	Created []*UserView           `json:"created"`
	Failed  []UserBulkImportError `json:"failed"`
}

type UserBulkImportError struct {
	Index   int    `json:"index"`
	Email   string `json:"email"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (s *UserService) List(ctx context.Context, filter UserFilter) ([]UserView, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: user service unavailable")
	}
	tenantUUID := strings.TrimSpace(filter.TenantUUID)
	if tenantUUID == "" {
		return nil, errors.New("tenant_uuid required")
	}
	query := s.db.WithContext(ctx).
		Table(iamm.Member{}.TableName()+" u").
		Select(`u.id AS id, u.uuid AS member_uuid, u.user_id AS user_id, u.tenant_uuid, u.username, u.status, u.department_id,
            u.meta, u.last_login_at, u.created_at, COALESCE(u.display_name, a.display_name) AS display_name,
            a.email, a.phone`).
		Joins("JOIN "+iamm.User{}.TableName()+" a ON a.id = u.user_id").
		Where("u.tenant_uuid = ?", tenantUUID)
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("u.status = ?", status)
	}
	if search := strings.TrimSpace(filter.Query); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where("(lower(a.email) LIKE ? OR lower(u.username) LIKE ? OR lower(a.display_name) LIKE ?)", like, like, like)
	}
	query = query.Order("u.created_at DESC")
	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []UserView{}
	for rows.Next() {
		var view UserView
		if err := s.db.ScanRows(rows, &view); err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	if len(result) == 0 {
		return result, nil
	}
	ids := make([]uint64, 0, len(result))
	for _, view := range result {
		ids = append(ids, view.ID)
	}
	roleMap, err := s.userRolesMap(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range result {
		if roles, ok := roleMap[result[i].ID]; ok {
			result[i].Roles = roles
		}
		result[i].DepartmentIDs = extractDepartmentIDs(result[i].Meta, result[i].DepartmentID)
		if err := s.hydrateUUIDView(ctx, &result[i]); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *UserService) userRolesMap(ctx context.Context, userIDs []uint64) (map[uint64][]string, error) {
	roleMap := make(map[uint64][]string)
	if s == nil || s.db == nil || len(userIDs) == 0 {
		return roleMap, nil
	}
	dedup := make([]uint64, 0, len(userIDs))
	seen := make(map[uint64]struct{}, len(userIDs))
	for _, id := range userIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		dedup = append(dedup, id)
	}
	if len(dedup) == 0 {
		return roleMap, nil
	}
	rows := []struct {
		UserID uint64
		Code   string
	}{}
	if err := s.db.WithContext(ctx).
		Table(iamm.MemberRole{}.TableName()+" ur").
		Select("ur.member_id AS user_id, r.code").
		Joins("JOIN "+iamm.Role{}.TableName()+" r ON r.id = ur.role_id").
		Where("ur.member_id IN ?", dedup).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		roleMap[row.UserID] = append(roleMap[row.UserID], row.Code)
	}
	return roleMap, nil
}

func (s *UserService) assignRoles(ctx context.Context, tx *gorm.DB, tenantUUID string, userID uint64, roleIDs []uint64) error {
	if tx == nil {
		return errors.New("iam: db is nil")
	}
	cleaned := uniqueUint64(roleIDs)
	if len(cleaned) == 0 {
		return tx.WithContext(ctx).Where("member_id = ?", userID).Delete(&iamm.MemberRole{}).Error
	}
	var roles []iamm.Role
	if err := tx.WithContext(ctx).Model(&iamm.Role{}).
		Where("tenant_uuid = ?", tenantUUID).
		Where("id IN ?", cleaned).
		Find(&roles).Error; err != nil {
		return err
	}
	if len(roles) != len(cleaned) {
		return fmt.Errorf("role ids do not belong to tenant")
	}
	roleUUIDs := make(map[uint64]string, len(roles))
	for _, role := range roles {
		roleUUIDs[role.ID] = role.UUID
	}
	if err := tx.WithContext(ctx).Where("member_id = ?", userID).Delete(&iamm.MemberRole{}).Error; err != nil {
		return err
	}
	for _, roleID := range cleaned {
		var member iamm.Member
		if err := tx.WithContext(ctx).Where("id = ? AND tenant_uuid = ?", userID, tenantUUID).First(&member).Error; err != nil {
			return err
		}
		rel := &iamm.MemberRole{MemberUUID: member.UUID, RoleUUID: roleUUIDs[roleID], UserID: userID, RoleID: roleID}
		if err := tx.WithContext(ctx).Create(rel).Error; err != nil {
			return err
		}
	}
	return nil
}

type CreateUserInput struct {
	TenantUUID    string
	Email         string
	DisplayName   string
	Username      string
	Phone         string
	DepartmentID  *uint64
	DepartmentIDs []uint64
	Status        string
	ActorID       *uint64
	Roles         []uint64
}

type CreateUserUUIDInput struct {
	TenantUUID      string
	Email           string
	DisplayName     string
	Username        string
	Phone           string
	DepartmentUUIDs []string
	Status          string
	ActorID         *uint64
	RoleUUIDs       []string
}

func (s *UserService) CreateByUUID(ctx context.Context, input CreateUserUUIDInput) (*UserView, error) {
	departmentIDs, err := s.departmentIDsByUUID(ctx, input.TenantUUID, input.DepartmentUUIDs)
	if err != nil {
		return nil, err
	}
	roleIDs, err := s.roleIDsByUUID(ctx, input.TenantUUID, input.RoleUUIDs)
	if err != nil {
		return nil, err
	}
	return s.Create(ctx, CreateUserInput{TenantUUID: input.TenantUUID, Email: input.Email, DisplayName: input.DisplayName, Username: input.Username, Phone: input.Phone, DepartmentIDs: departmentIDs, Status: input.Status, ActorID: input.ActorID, Roles: roleIDs})
}

func (s *UserService) Create(ctx context.Context, input CreateUserInput) (*UserView, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: user service unavailable")
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(input.TenantUUID))
	if tenantUUID == "" {
		return nil, errors.New("tenant_uuid required")
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		return nil, errors.New("email required")
	}
	username := strings.TrimSpace(input.Username)
	if username == "" {
		username = slugify(strings.Split(email, "@")[0])
	}
	status := normalizeUserStatus(input.Status)
	if status == "" {
		status = iamm.StatusActive
	}

	departmentIDs := normalizeDepartmentIDs(input.DepartmentIDs)
	if len(departmentIDs) == 0 && input.DepartmentID != nil && *input.DepartmentID > 0 {
		departmentIDs = []uint64{*input.DepartmentID}
	}
	if len(departmentIDs) > 0 {
		if _, err := s.resolveDepartments(ctx, tenantUUID, departmentIDs); err != nil {
			return nil, err
		}
	}

	var created *UserView
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		account, err := ensureAccount(ctx, tx, email, input.DisplayName, input.Phone)
		if err != nil {
			return err
		}
		var exists int64
		if err := tx.Model(&iamm.Member{}).Where("tenant_uuid = ? AND user_id = ?", tenantUUID, account.ID).Count(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			return fmt.Errorf("user already exists for tenant")
		}
		var usernameExists int64
		if err := tx.Model(&iamm.Member{}).
			Where("tenant_uuid = ? AND lower(username) = ?", tenantUUID, strings.ToLower(username)).
			Count(&usernameExists).Error; err != nil {
			return err
		}
		if usernameExists > 0 {
			return fmt.Errorf("username already exists")
		}
		memberMeta := datatypes.JSONMap{}
		if len(departmentIDs) > 0 {
			memberMeta["department_ids"] = departmentIDs
		}
		record := &iamm.Member{
			BaseModel:   basemodels.BaseModel{TenantUuid: tenantUUID},
			UserID:      account.ID,
			Username:    username,
			DisplayName: strings.TrimSpace(input.DisplayName),
			Status:      status,
			Meta:        memberMeta,
		}
		if len(departmentIDs) > 0 {
			primaryDeptID := departmentIDs[0]
			record.DepartmentID = &primaryDeptID
			var department iamm.Department
			if err := tx.WithContext(ctx).Where("id = ? AND tenant_uuid = ?", primaryDeptID, tenantUUID).First(&department).Error; err != nil {
				return err
			}
			departmentUUID := department.UUID
			record.DepartmentUUID = &departmentUUID
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		if len(input.Roles) > 0 {
			if err := s.assignRoles(ctx, tx, tenantUUID, record.ID, input.Roles); err != nil {
				return err
			}
		}
		created = &UserView{
			ID:            record.ID,
			UserID:        account.ID,
			TenantUUID:    tenantUUID,
			Email:         account.Email,
			Phone:         account.Phone,
			DisplayName:   firstNonEmpty(record.DisplayName, account.DisplayName),
			Username:      record.Username,
			Status:        record.Status,
			DepartmentID:  record.DepartmentID,
			DepartmentIDs: extractDepartmentIDs(record.Meta, record.DepartmentID),
			CreatedAt:     record.CreatedAt,
			MemberUUID:    record.UUID,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if created != nil {
		roleMap, err := s.userRolesMap(ctx, []uint64{created.ID})
		if err == nil {
			created.Roles = roleMap[created.ID]
		}
		if err := s.hydrateUUIDView(ctx, created); err != nil {
			return nil, err
		}
	}
	if s.audit != nil && created != nil {
		diff := map[string]any{
			"member_id": created.ID,
			"user_id":   created.UserID,
			"username":  created.Username,
			"status":    created.Status,
		}
		if len(created.Roles) > 0 {
			diff["roles"] = created.Roles
		}
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    tenantUUID,
			ActorMemberID: input.ActorID,
			Action:        "create",
			Resource:      "iam.user",
			Diff:          diff,
		})
	}
	return created, nil
}

type UpdateUserInput struct {
	TenantUUID    string
	DisplayName   string
	Status        string
	DepartmentID  *uint64
	DepartmentIDs []uint64
	ActorID       *uint64
	Roles         []uint64
	ReplaceRoles  bool
}

type UpdateUserUUIDInput struct {
	TenantUUID      string
	DisplayName     string
	Status          string
	DepartmentUUIDs []string
	ActorID         *uint64
	RoleUUIDs       []string
	ReplaceRoles    bool
}

func (s *UserService) UpdateByUUID(ctx context.Context, memberUUID string, input UpdateUserUUIDInput) (*UserView, error) {
	var member iamm.Member
	if err := s.db.WithContext(ctx).Where("tenant_uuid = ? AND uuid = ?", strings.TrimSpace(input.TenantUUID), strings.TrimSpace(memberUUID)).First(&member).Error; err != nil {
		return nil, err
	}
	departmentIDs, err := s.departmentIDsByUUID(ctx, input.TenantUUID, input.DepartmentUUIDs)
	if err != nil {
		return nil, err
	}
	roleIDs, err := s.roleIDsByUUID(ctx, input.TenantUUID, input.RoleUUIDs)
	if err != nil {
		return nil, err
	}
	return s.Update(ctx, member.ID, UpdateUserInput{TenantUUID: input.TenantUUID, DisplayName: input.DisplayName, Status: input.Status, DepartmentIDs: departmentIDs, ActorID: input.ActorID, Roles: roleIDs, ReplaceRoles: input.ReplaceRoles})
}

func (s *UserService) BulkImport(ctx context.Context, inputs []CreateUserInput) (*UserBulkImportResult, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: user service unavailable")
	}
	result := &UserBulkImportResult{Created: []*UserView{}, Failed: []UserBulkImportError{}}
	var firstErr error
	for idx, payload := range inputs {
		view, err := s.Create(ctx, payload)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			result.Failed = append(result.Failed, UserBulkImportError{
				Index:   idx,
				Email:   payload.Email,
				Message: err.Error(),
				Err:     err,
			})
			continue
		}
		result.Created = append(result.Created, view)
	}
	return result, firstErr
}

func (s *UserService) Update(ctx context.Context, id uint64, input UpdateUserInput) (*UserView, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: user service unavailable")
	}
	tenantUUID := strings.TrimSpace(input.TenantUUID)
	if tenantUUID == "" {
		return nil, errors.New("tenant_uuid required")
	}
	var user iamm.Member
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_uuid = ?", id, tenantUUID).First(&user).Error; err != nil {
		return nil, err
	}
	departmentIDs := normalizeDepartmentIDs(input.DepartmentIDs)
	if len(departmentIDs) == 0 && input.DepartmentID != nil {
		if *input.DepartmentID > 0 {
			departmentIDs = []uint64{*input.DepartmentID}
		}
	}
	if len(departmentIDs) > 0 {
		if _, err := s.resolveDepartments(ctx, user.TenantUuid, departmentIDs); err != nil {
			return nil, err
		}
	}

	updates := map[string]any{}
	if name := strings.TrimSpace(input.DisplayName); name != "" && name != user.DisplayName {
		updates["display_name"] = name
	}
	var statusChanged bool
	if status := normalizeUserStatus(input.Status); status != "" && status != user.Status {
		updates["status"] = status
		statusChanged = true
	}
	if input.DepartmentIDs != nil {
		if len(departmentIDs) == 0 {
			updates["department_id"] = nil
			updates["department_uuid"] = nil
			updates["meta"] = mergeDepartmentIDsMeta(user.Meta, nil)
		} else {
			primary := departmentIDs[0]
			if user.DepartmentID == nil || *user.DepartmentID != primary {
				updates["department_id"] = primary
				var department iamm.Department
				if err := s.db.WithContext(ctx).Where("id = ? AND tenant_uuid = ?", primary, user.TenantUuid).First(&department).Error; err != nil {
					return nil, err
				}
				updates["department_uuid"] = department.UUID
			}
			updates["meta"] = mergeDepartmentIDsMeta(user.Meta, departmentIDs)
		}
	} else if input.DepartmentID != nil {
		if *input.DepartmentID == 0 {
			updates["department_id"] = nil
			updates["department_uuid"] = nil
			updates["meta"] = mergeDepartmentIDsMeta(user.Meta, nil)
		} else if user.DepartmentID == nil || *user.DepartmentID != *input.DepartmentID {
			updates["department_id"] = *input.DepartmentID
			var department iamm.Department
			if err := s.db.WithContext(ctx).Where("id = ? AND tenant_uuid = ?", *input.DepartmentID, user.TenantUuid).First(&department).Error; err != nil {
				return nil, err
			}
			updates["department_uuid"] = department.UUID
			updates["meta"] = mergeDepartmentIDsMeta(user.Meta, []uint64{*input.DepartmentID})
		}
	}

	var rolesChanged bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&user).Updates(updates).Error; err != nil {
				return err
			}
		}
		if input.ReplaceRoles {
			if err := s.assignRoles(ctx, tx, user.TenantUuid, user.ID, input.Roles); err != nil {
				return err
			}
			rolesChanged = true
		}
		return tx.Where("id = ?", user.ID).First(&user).Error
	})
	if err != nil {
		return nil, err
	}

	var account iamm.User
	if err := s.db.WithContext(ctx).Where("id = ?", user.UserID).First(&account).Error; err != nil {
		return nil, err
	}
	roleMap, err := s.userRolesMap(ctx, []uint64{user.ID})
	if err != nil {
		return nil, err
	}
	view := &UserView{
		ID:            user.ID,
		UserID:        account.ID,
		TenantUUID:    user.TenantUuid,
		Email:         account.Email,
		Phone:         account.Phone,
		DisplayName:   firstNonEmpty(user.DisplayName, account.DisplayName),
		Username:      user.Username,
		Status:        user.Status,
		DepartmentID:  user.DepartmentID,
		DepartmentIDs: extractDepartmentIDs(user.Meta, user.DepartmentID),
		LastLoginAt:   user.LastLoginAt,
		CreatedAt:     user.CreatedAt,
		Roles:         roleMap[user.ID],
	}
	if err := s.hydrateUUIDView(ctx, view); err != nil {
		return nil, err
	}
	if s.audit != nil && (len(updates) > 0 || rolesChanged) {
		diff := make(map[string]any, len(updates))
		for k, v := range updates {
			diff[k] = v
		}
		if rolesChanged {
			diff["roles"] = view.Roles
		}
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    user.TenantUuid,
			ActorMemberID: input.ActorID,
			Action:        "update",
			Resource:      "iam.user",
			Diff:          diff,
		})
	}
	if (statusChanged && user.Status != iamm.StatusActive) || rolesChanged {
		_ = revokeMemberSession(ctx, s.db, user.ID)
	}
	return view, nil
}

func ensureAccount(ctx context.Context, tx *gorm.DB, email, displayName, phone string) (*iamm.User, error) {
	var account iamm.User
	if err := tx.WithContext(ctx).Where("lower(email) = ?", strings.ToLower(email)).First(&account).Error; err == nil {
		return &account, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	passwordHash, err := randomPasswordHash()
	if err != nil {
		return nil, err
	}
	account = iamm.User{
		Email:        strings.ToLower(email),
		DisplayName:  strings.TrimSpace(displayName),
		Status:       iamm.StatusActive,
		PasswordHash: passwordHash,
		Phone:        strings.TrimSpace(phone),
		Meta:         datatypes.JSONMap{},
	}
	if account.DisplayName == "" {
		account.DisplayName = account.Email
	}
	if err := tx.WithContext(ctx).Create(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func randomPasswordHash() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	raw := hex.EncodeToString(buf)
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func normalizeUserStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", iamm.StatusActive:
		return iamm.StatusActive
	case iamm.StatusDisabled:
		return iamm.StatusDisabled
	case iamm.StatusLocked:
		return iamm.StatusLocked
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func slugify(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "\\", "-")
	return value
}

func (s *UserService) resolveDepartments(ctx context.Context, tenantUUID string, deptIDs []uint64) ([]iamm.Department, error) {
	cleaned := uniqueUint64(deptIDs)
	if len(cleaned) == 0 {
		return nil, nil
	}
	rows := make([]iamm.Department, 0, len(cleaned))
	if err := s.db.WithContext(ctx).
		Where("tenant_uuid = ?", strings.TrimSpace(tenantUUID)).
		Where("id IN ?", cleaned).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) != len(cleaned) {
		return nil, fmt.Errorf("department ids do not belong to tenant")
	}
	return rows, nil
}

func (s *UserService) departmentIDsByUUID(ctx context.Context, tenantUUID string, departmentUUIDs []string) ([]uint64, error) {
	uuids := uniqueStrings(departmentUUIDs)
	if len(uuids) == 0 {
		return nil, nil
	}
	var rows []iamm.Department
	if err := s.db.WithContext(ctx).Where("tenant_uuid = ? AND uuid IN ?", strings.TrimSpace(tenantUUID), uuids).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) != len(uuids) {
		return nil, fmt.Errorf("department uuids do not belong to tenant")
	}
	byUUID := make(map[string]uint64, len(rows))
	for _, row := range rows {
		byUUID[row.UUID] = row.ID
	}
	ids := make([]uint64, 0, len(uuids))
	for _, value := range uuids {
		ids = append(ids, byUUID[value])
	}
	return ids, nil
}

func (s *UserService) roleIDsByUUID(ctx context.Context, tenantUUID string, roleUUIDs []string) ([]uint64, error) {
	uuids := uniqueStrings(roleUUIDs)
	if len(uuids) == 0 {
		return nil, nil
	}
	var rows []iamm.Role
	if err := s.db.WithContext(ctx).Where("tenant_uuid = ? AND uuid IN ?", strings.TrimSpace(tenantUUID), uuids).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) != len(uuids) {
		return nil, fmt.Errorf("role uuids do not belong to tenant")
	}
	byUUID := make(map[string]uint64, len(rows))
	for _, row := range rows {
		byUUID[row.UUID] = row.ID
	}
	ids := make([]uint64, 0, len(uuids))
	for _, value := range uuids {
		ids = append(ids, byUUID[value])
	}
	return ids, nil
}

func (s *UserService) hydrateUUIDView(ctx context.Context, view *UserView) error {
	if view == nil {
		return nil
	}
	var member iamm.Member
	if err := s.db.WithContext(ctx).Where("id = ?", view.ID).First(&member).Error; err != nil {
		return err
	}
	view.MemberUUID = member.UUID
	ids := extractDepartmentIDs(member.Meta, member.DepartmentID)
	if len(ids) == 0 {
		return nil
	}
	var departments []iamm.Department
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&departments).Error; err != nil {
		return err
	}
	byID := make(map[uint64]string, len(departments))
	for _, department := range departments {
		byID[department.ID] = department.UUID
	}
	for _, id := range ids {
		if value := byID[id]; value != "" {
			view.DepartmentUUIDs = append(view.DepartmentUUIDs, value)
		}
	}
	return nil
}

func normalizeDepartmentIDs(values []uint64) []uint64 {
	return uniqueUint64(values)
}

func mergeDepartmentIDsMeta(meta datatypes.JSONMap, departmentIDs []uint64) datatypes.JSONMap {
	next := datatypes.JSONMap{}
	for k, v := range meta {
		next[k] = v
	}
	if len(departmentIDs) == 0 {
		delete(next, "department_ids")
		return next
	}
	next["department_ids"] = departmentIDs
	return next
}

func extractDepartmentIDs(meta datatypes.JSONMap, primary *uint64) []uint64 {
	ids := make([]uint64, 0, 4)
	appendID := func(v uint64) {
		if v == 0 {
			return
		}
		for _, existing := range ids {
			if existing == v {
				return
			}
		}
		ids = append(ids, v)
	}
	if raw, ok := meta["department_ids"]; ok {
		switch arr := raw.(type) {
		case []uint64:
			for _, v := range arr {
				appendID(v)
			}
		case []any:
			for _, item := range arr {
				switch n := item.(type) {
				case uint64:
					appendID(n)
				case uint32:
					appendID(uint64(n))
				case int:
					if n > 0 {
						appendID(uint64(n))
					}
				case int64:
					if n > 0 {
						appendID(uint64(n))
					}
				case float64:
					if n > 0 {
						appendID(uint64(n))
					}
				}
			}
		}
	}
	if primary != nil {
		appendID(*primary)
	}
	return ids
}

func uniqueUint64(values []uint64) []uint64 {
	if len(values) == 0 {
		return values
	}
	result := make([]uint64, 0, len(values))
	seen := make(map[uint64]struct{}, len(values))
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
	return result
}
