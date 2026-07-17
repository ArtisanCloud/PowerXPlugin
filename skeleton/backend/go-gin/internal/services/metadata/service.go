package metadata

import (
	"context"
	"errors"
	"strings"

	fwmetadata "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/metadata"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/metadata"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

var ErrDuplicate = errors.New("metadata: duplicate record")

const (
	DuplicateFieldNamespace    = "namespace"
	DuplicateFieldCode         = "code"
	DuplicateFieldTag          = "tag"
	DuplicateFieldResourceType = "resource_type"
)

type DuplicateError struct {
	Field string
}

func (e *DuplicateError) Error() string {
	if e == nil || strings.TrimSpace(e.Field) == "" {
		return ErrDuplicate.Error()
	}
	return ErrDuplicate.Error() + ": " + strings.TrimSpace(e.Field)
}

func (e *DuplicateError) Unwrap() error {
	return ErrDuplicate
}

func NewService(db *gorm.DB) *Service {
	if db == nil {
		return nil
	}
	return &Service{db: db}
}

type ListOptions struct {
	TenantUUID   string
	Module       string
	Namespace    string
	ResourceType string
	Status       string
	Query        string
	Locale       string
	Page         int
	PageSize     int
}

type CreateDictionaryNamespaceInput struct {
	TenantUUID      string
	Namespace       string
	Module          string
	NameI18n        map[string]string
	DescriptionI18n map[string]string
}

type CreateDictionaryItemInput struct {
	TenantUUID      string
	NamespaceUUID   string
	Code            string
	LabelI18n       map[string]string
	DescriptionI18n map[string]string
	SortOrder       int
}

type CreateTaxonomyInput struct {
	TenantUUID      string
	Namespace       string
	Module          string
	NameI18n        map[string]string
	DescriptionI18n map[string]string
	MaxDepth        int
}

type CreateTaxonomyNodeInput struct {
	TenantUUID      string
	TaxonomyUUID    string
	ParentUUID      *string
	Code            string
	LabelI18n       map[string]string
	DescriptionI18n map[string]string
	SortOrder       int
}

type CreateTagInput struct {
	TenantUUID      string
	Namespace       string
	ResourceType    string
	Code            string
	Color           string
	LabelI18n       map[string]string
	DescriptionI18n map[string]string
}

type CreateResourceTypeInput struct {
	TenantUUID      string
	ResourceType    string
	Module          string
	NameI18n        map[string]string
	DescriptionI18n map[string]string
	ValidatorKey    string
	BindingEnabled  bool
}

func (s *Service) ListDictionaryNamespaces(ctx context.Context, opts ListOptions) (*fwmetadata.Page[fwmetadata.DictionaryNamespace], error) {
	var rows []model.DictionaryNamespace
	var total int64
	q := scoped(s.db.WithContext(ctx).Model(&model.DictionaryNamespace{}), opts).
		Scopes(filterModule(opts.Module), filterStatus(opts.Status), filterSearch(opts.Query, "namespace", "module"))
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := page(q.Order("module asc, namespace asc"), opts).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]fwmetadata.DictionaryNamespace, 0, len(rows))
	for _, row := range rows {
		items = append(items, fwmetadata.DictionaryNamespace{
			UUID: row.UUID, Namespace: row.Namespace, Module: row.Module, NameI18n: mapString(row.NameI18n),
			DescriptionI18n: mapString(row.DescriptionI18n), Status: row.Status, Display: display(row.NameI18n, row.DescriptionI18n, opts.Locale),
		})
	}
	return result(items, total, opts), nil
}

func (s *Service) CreateDictionaryNamespace(ctx context.Context, in CreateDictionaryNamespaceInput) (*fwmetadata.DictionaryNamespace, error) {
	if err := require(in.TenantUUID, in.Namespace, in.Module); err != nil {
		return nil, err
	}
	if err := ensureNotExists(s.db.WithContext(ctx).Model(&model.DictionaryNamespace{}).
		Where("tenant_uuid = ? AND namespace = ?", strings.TrimSpace(in.TenantUUID), strings.TrimSpace(in.Namespace))); err != nil {
		return nil, duplicate(DuplicateFieldNamespace)
	}
	row := model.DictionaryNamespace{
		TenantUUID:      strings.TrimSpace(in.TenantUUID),
		Namespace:       strings.TrimSpace(in.Namespace),
		Module:          strings.TrimSpace(in.Module),
		NameI18n:        jsonMap(in.NameI18n),
		DescriptionI18n: jsonMap(in.DescriptionI18n),
		Status:          "active",
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, duplicate(DuplicateFieldNamespace)
		}
		return nil, err
	}
	item := fwmetadata.DictionaryNamespace{
		UUID: row.UUID, Namespace: row.Namespace, Module: row.Module, NameI18n: mapString(row.NameI18n),
		DescriptionI18n: mapString(row.DescriptionI18n), Status: row.Status, Display: display(row.NameI18n, row.DescriptionI18n, ""),
	}
	return &item, nil
}

func (s *Service) ListDictionaryItems(ctx context.Context, namespaceUUID string, opts ListOptions) (*fwmetadata.Page[fwmetadata.DictionaryItem], error) {
	var rows []model.DictionaryItem
	var total int64
	q := scoped(s.db.WithContext(ctx).Model(&model.DictionaryItem{}), opts).
		Where("namespace_uuid = ?", strings.TrimSpace(namespaceUUID)).
		Scopes(filterStatus(opts.Status), filterSearch(opts.Query, "code"))
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := page(q.Order("sort_order asc, code asc"), opts).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]fwmetadata.DictionaryItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, fwmetadata.DictionaryItem{
			UUID: row.UUID, NamespaceUUID: row.NamespaceUUID, Code: row.Code, LabelI18n: mapString(row.LabelI18n),
			DescriptionI18n: mapString(row.DescriptionI18n), Status: row.Status, SortOrder: row.SortOrder, ReferenceCount: row.ReferenceCount,
			Display: display(row.LabelI18n, row.DescriptionI18n, opts.Locale),
		})
	}
	return result(items, total, opts), nil
}

func (s *Service) CreateDictionaryItem(ctx context.Context, in CreateDictionaryItemInput) (*fwmetadata.DictionaryItem, error) {
	if err := require(in.TenantUUID, in.NamespaceUUID, in.Code); err != nil {
		return nil, err
	}
	if err := ensureNotExists(s.db.WithContext(ctx).Model(&model.DictionaryItem{}).
		Where("namespace_uuid = ? AND code = ?", strings.TrimSpace(in.NamespaceUUID), strings.TrimSpace(in.Code))); err != nil {
		return nil, duplicate(DuplicateFieldCode)
	}
	row := model.DictionaryItem{
		TenantUUID:      strings.TrimSpace(in.TenantUUID),
		NamespaceUUID:   strings.TrimSpace(in.NamespaceUUID),
		Code:            strings.TrimSpace(in.Code),
		LabelI18n:       jsonMap(in.LabelI18n),
		DescriptionI18n: jsonMap(in.DescriptionI18n),
		Status:          "active",
		SortOrder:       in.SortOrder,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, duplicate(DuplicateFieldCode)
		}
		return nil, err
	}
	item := fwmetadata.DictionaryItem{
		UUID: row.UUID, NamespaceUUID: row.NamespaceUUID, Code: row.Code, LabelI18n: mapString(row.LabelI18n),
		DescriptionI18n: mapString(row.DescriptionI18n), Status: row.Status, SortOrder: row.SortOrder,
		ReferenceCount: row.ReferenceCount, Display: display(row.LabelI18n, row.DescriptionI18n, ""),
	}
	return &item, nil
}

func (s *Service) ListTaxonomies(ctx context.Context, opts ListOptions) (*fwmetadata.Page[fwmetadata.Taxonomy], error) {
	var rows []model.Taxonomy
	var total int64
	q := scoped(s.db.WithContext(ctx).Model(&model.Taxonomy{}), opts).
		Scopes(filterModule(opts.Module), filterStatus(opts.Status), filterSearch(opts.Query, "namespace", "module"))
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := page(q.Order("module asc, namespace asc"), opts).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]fwmetadata.Taxonomy, 0, len(rows))
	for _, row := range rows {
		items = append(items, fwmetadata.Taxonomy{
			UUID: row.UUID, Namespace: row.Namespace, Module: row.Module, NameI18n: mapString(row.NameI18n),
			DescriptionI18n: mapString(row.DescriptionI18n), MaxDepth: row.MaxDepth, Status: row.Status,
			Display: display(row.NameI18n, row.DescriptionI18n, opts.Locale),
		})
	}
	return result(items, total, opts), nil
}

func (s *Service) CreateTaxonomy(ctx context.Context, in CreateTaxonomyInput) (*fwmetadata.Taxonomy, error) {
	if err := require(in.TenantUUID, in.Namespace, in.Module); err != nil {
		return nil, err
	}
	if err := ensureNotExists(s.db.WithContext(ctx).Model(&model.Taxonomy{}).
		Where("tenant_uuid = ? AND namespace = ?", strings.TrimSpace(in.TenantUUID), strings.TrimSpace(in.Namespace))); err != nil {
		return nil, duplicate(DuplicateFieldNamespace)
	}
	maxDepth := in.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 3
	}
	row := model.Taxonomy{
		TenantUUID:      strings.TrimSpace(in.TenantUUID),
		Namespace:       strings.TrimSpace(in.Namespace),
		Module:          strings.TrimSpace(in.Module),
		NameI18n:        jsonMap(in.NameI18n),
		DescriptionI18n: jsonMap(in.DescriptionI18n),
		MaxDepth:        maxDepth,
		Status:          "active",
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, duplicate(DuplicateFieldNamespace)
		}
		return nil, err
	}
	item := fwmetadata.Taxonomy{
		UUID: row.UUID, Namespace: row.Namespace, Module: row.Module, NameI18n: mapString(row.NameI18n),
		DescriptionI18n: mapString(row.DescriptionI18n), MaxDepth: row.MaxDepth, Status: row.Status,
		Display: display(row.NameI18n, row.DescriptionI18n, ""),
	}
	return &item, nil
}

func (s *Service) ListTaxonomyNodes(ctx context.Context, taxonomyUUID string, opts ListOptions) (*fwmetadata.Page[fwmetadata.TaxonomyNode], error) {
	var rows []model.TaxonomyNode
	var total int64
	q := scoped(s.db.WithContext(ctx).Model(&model.TaxonomyNode{}), opts).
		Where("taxonomy_uuid = ?", strings.TrimSpace(taxonomyUUID)).
		Scopes(filterStatus(opts.Status), filterSearch(opts.Query, "code", "path"))
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := page(q.Order("path asc, sort_order asc, code asc"), opts).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]fwmetadata.TaxonomyNode, 0, len(rows))
	for _, row := range rows {
		items = append(items, fwmetadata.TaxonomyNode{
			UUID: row.UUID, TaxonomyUUID: row.TaxonomyUUID, ParentUUID: row.ParentUUID, Code: row.Code,
			LabelI18n: mapString(row.LabelI18n), DescriptionI18n: mapString(row.DescriptionI18n), Path: row.Path,
			Depth: row.Depth, SortOrder: row.SortOrder, Status: row.Status, ReferenceCount: row.ReferenceCount, Version: row.Version,
			Display: display(row.LabelI18n, row.DescriptionI18n, opts.Locale),
		})
	}
	return result(items, total, opts), nil
}

func (s *Service) CreateTaxonomyNode(ctx context.Context, in CreateTaxonomyNodeInput) (*fwmetadata.TaxonomyNode, error) {
	if err := require(in.TenantUUID, in.TaxonomyUUID, in.Code); err != nil {
		return nil, err
	}
	if err := ensureNotExists(s.db.WithContext(ctx).Model(&model.TaxonomyNode{}).
		Where("taxonomy_uuid = ? AND code = ?", strings.TrimSpace(in.TaxonomyUUID), strings.TrimSpace(in.Code))); err != nil {
		return nil, duplicate(DuplicateFieldCode)
	}
	depth := 0
	path := strings.TrimSpace(in.Code)
	if in.ParentUUID != nil && strings.TrimSpace(*in.ParentUUID) != "" {
		var parent model.TaxonomyNode
		if err := s.db.WithContext(ctx).
			Where("tenant_uuid = ? AND taxonomy_uuid = ? AND uuid = ?", strings.TrimSpace(in.TenantUUID), strings.TrimSpace(in.TaxonomyUUID), strings.TrimSpace(*in.ParentUUID)).
			First(&parent).Error; err != nil {
			return nil, err
		}
		depth = parent.Depth + 1
		path = strings.Trim(strings.TrimSpace(parent.Path)+"/"+strings.TrimSpace(in.Code), "/")
	}
	row := model.TaxonomyNode{
		TenantUUID:      strings.TrimSpace(in.TenantUUID),
		TaxonomyUUID:    strings.TrimSpace(in.TaxonomyUUID),
		ParentUUID:      in.ParentUUID,
		Code:            strings.TrimSpace(in.Code),
		LabelI18n:       jsonMap(in.LabelI18n),
		DescriptionI18n: jsonMap(in.DescriptionI18n),
		Path:            path,
		Depth:           depth,
		SortOrder:       in.SortOrder,
		Status:          "active",
		Version:         1,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, duplicate(DuplicateFieldCode)
		}
		return nil, err
	}
	item := fwmetadata.TaxonomyNode{
		UUID: row.UUID, TaxonomyUUID: row.TaxonomyUUID, ParentUUID: row.ParentUUID, Code: row.Code,
		LabelI18n: mapString(row.LabelI18n), DescriptionI18n: mapString(row.DescriptionI18n), Path: row.Path,
		Depth: row.Depth, SortOrder: row.SortOrder, Status: row.Status, ReferenceCount: row.ReferenceCount,
		Version: row.Version, Display: display(row.LabelI18n, row.DescriptionI18n, ""),
	}
	return &item, nil
}

func (s *Service) ListTags(ctx context.Context, opts ListOptions) (*fwmetadata.Page[fwmetadata.Tag], error) {
	var rows []model.Tag
	var total int64
	q := scoped(s.db.WithContext(ctx).Model(&model.Tag{}), opts).
		Scopes(filterNamespace(opts.Namespace), filterResourceType(opts.ResourceType), filterStatus(opts.Status), filterSearch(opts.Query, "namespace", "resource_type", "code"))
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := page(q.Order("namespace asc, resource_type asc, code asc"), opts).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]fwmetadata.Tag, 0, len(rows))
	for _, row := range rows {
		items = append(items, fwmetadata.Tag{
			UUID: row.UUID, Namespace: row.Namespace, ResourceType: row.ResourceType, Code: row.Code,
			LabelI18n: mapString(row.LabelI18n), DescriptionI18n: mapString(row.DescriptionI18n), Color: row.Color,
			Status: row.Status, UsageCount: row.UsageCount, Display: display(row.LabelI18n, row.DescriptionI18n, opts.Locale),
		})
	}
	return result(items, total, opts), nil
}

func (s *Service) CreateTag(ctx context.Context, in CreateTagInput) (*fwmetadata.Tag, error) {
	if err := require(in.TenantUUID, in.Namespace, in.ResourceType, in.Code); err != nil {
		return nil, err
	}
	if err := ensureNotExists(s.db.WithContext(ctx).Model(&model.Tag{}).
		Where("tenant_uuid = ? AND namespace = ? AND resource_type = ? AND code = ?", strings.TrimSpace(in.TenantUUID), strings.TrimSpace(in.Namespace), strings.TrimSpace(in.ResourceType), strings.TrimSpace(in.Code))); err != nil {
		return nil, duplicate(DuplicateFieldTag)
	}
	row := model.Tag{
		TenantUUID:      strings.TrimSpace(in.TenantUUID),
		Namespace:       strings.TrimSpace(in.Namespace),
		ResourceType:    strings.TrimSpace(in.ResourceType),
		Code:            strings.TrimSpace(in.Code),
		Color:           strings.TrimSpace(in.Color),
		LabelI18n:       jsonMap(in.LabelI18n),
		DescriptionI18n: jsonMap(in.DescriptionI18n),
		Status:          "active",
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, duplicate(DuplicateFieldTag)
		}
		return nil, err
	}
	item := fwmetadata.Tag{
		UUID: row.UUID, Namespace: row.Namespace, ResourceType: row.ResourceType, Code: row.Code,
		LabelI18n: mapString(row.LabelI18n), DescriptionI18n: mapString(row.DescriptionI18n), Color: row.Color,
		Status: row.Status, UsageCount: row.UsageCount, Display: display(row.LabelI18n, row.DescriptionI18n, ""),
	}
	return &item, nil
}

func (s *Service) ListResourceTypes(ctx context.Context, opts ListOptions) (*fwmetadata.Page[fwmetadata.ResourceType], error) {
	var rows []model.ResourceType
	var total int64
	q := scoped(s.db.WithContext(ctx).Model(&model.ResourceType{}), opts).
		Scopes(filterModule(opts.Module), filterStatus(opts.Status), filterSearch(opts.Query, "resource_type", "module"))
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := page(q.Order("module asc, resource_type asc"), opts).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]fwmetadata.ResourceType, 0, len(rows))
	for _, row := range rows {
		items = append(items, fwmetadata.ResourceType{
			UUID: row.UUID, ResourceType: row.ResourceType, Module: row.Module, NameI18n: mapString(row.NameI18n),
			DescriptionI18n: mapString(row.DescriptionI18n), ValidatorKey: row.ValidatorKey, BindingEnabled: row.BindingEnabled,
			ValidatorStatus: row.ValidatorStatus, Status: row.Status, Display: display(row.NameI18n, row.DescriptionI18n, opts.Locale),
		})
	}
	return result(items, total, opts), nil
}

func (s *Service) CreateResourceType(ctx context.Context, in CreateResourceTypeInput) (*fwmetadata.ResourceType, error) {
	if err := require(in.TenantUUID, in.ResourceType, in.Module); err != nil {
		return nil, err
	}
	if err := ensureNotExists(s.db.WithContext(ctx).Model(&model.ResourceType{}).
		Where("tenant_uuid = ? AND resource_type = ?", strings.TrimSpace(in.TenantUUID), strings.TrimSpace(in.ResourceType))); err != nil {
		return nil, duplicate(DuplicateFieldResourceType)
	}
	row := model.ResourceType{
		TenantUUID:      strings.TrimSpace(in.TenantUUID),
		ResourceType:    strings.TrimSpace(in.ResourceType),
		Module:          strings.TrimSpace(in.Module),
		NameI18n:        jsonMap(in.NameI18n),
		DescriptionI18n: jsonMap(in.DescriptionI18n),
		ValidatorKey:    strings.TrimSpace(in.ValidatorKey),
		BindingEnabled:  in.BindingEnabled,
		ValidatorStatus: "ready",
		Status:          "active",
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, duplicate(DuplicateFieldResourceType)
		}
		return nil, err
	}
	item := fwmetadata.ResourceType{
		UUID: row.UUID, ResourceType: row.ResourceType, Module: row.Module, NameI18n: mapString(row.NameI18n),
		DescriptionI18n: mapString(row.DescriptionI18n), ValidatorKey: row.ValidatorKey, BindingEnabled: row.BindingEnabled,
		ValidatorStatus: row.ValidatorStatus, Status: row.Status, Display: display(row.NameI18n, row.DescriptionI18n, ""),
	}
	return &item, nil
}

func scoped(q *gorm.DB, opts ListOptions) *gorm.DB {
	return q.Where("tenant_uuid = ?", strings.TrimSpace(opts.TenantUUID))
}

func filterModule(module string) func(*gorm.DB) *gorm.DB {
	return func(q *gorm.DB) *gorm.DB {
		if module = strings.TrimSpace(module); module != "" {
			return q.Where("module = ?", module)
		}
		return q
	}
}

func filterNamespace(namespace string) func(*gorm.DB) *gorm.DB {
	return func(q *gorm.DB) *gorm.DB {
		if namespace = strings.TrimSpace(namespace); namespace != "" {
			return q.Where("namespace = ?", namespace)
		}
		return q
	}
}

func filterResourceType(resourceType string) func(*gorm.DB) *gorm.DB {
	return func(q *gorm.DB) *gorm.DB {
		if resourceType = strings.TrimSpace(resourceType); resourceType != "" {
			return q.Where("resource_type = ?", resourceType)
		}
		return q
	}
}

func filterStatus(status string) func(*gorm.DB) *gorm.DB {
	return func(q *gorm.DB) *gorm.DB {
		if status = strings.TrimSpace(status); status != "" {
			return q.Where("status = ?", status)
		}
		return q
	}
}

func filterSearch(query string, columns ...string) func(*gorm.DB) *gorm.DB {
	return func(q *gorm.DB) *gorm.DB {
		query = strings.TrimSpace(query)
		if query == "" || len(columns) == 0 {
			return q
		}
		like := "%" + query + "%"
		cond := make([]string, 0, len(columns))
		args := make([]any, 0, len(columns))
		for _, col := range columns {
			cond = append(cond, "LOWER("+col+") LIKE LOWER(?)")
			args = append(args, like)
		}
		return q.Where(strings.Join(cond, " OR "), args...)
	}
}

func page(q *gorm.DB, opts ListOptions) *gorm.DB {
	p := opts.Page
	if p <= 0 {
		p = 1
	}
	size := opts.PageSize
	if size <= 0 {
		size = fwmetadata.DefaultPageSize
	}
	if size > 200 {
		size = 200
	}
	return q.Offset((p - 1) * size).Limit(size)
}

func result[T any](items []T, total int64, opts ListOptions) *fwmetadata.Page[T] {
	p := opts.Page
	if p <= 0 {
		p = 1
	}
	size := opts.PageSize
	if size <= 0 {
		size = fwmetadata.DefaultPageSize
	}
	if size > 200 {
		size = 200
	}
	return &fwmetadata.Page[T]{Items: items, Pagination: fwmetadata.Pagination{Total: total, Page: p, PageSize: size}}
}

func mapString(in map[string]any) fwmetadata.I18nMap {
	out := fwmetadata.I18nMap{}
	for key, value := range in {
		if text, ok := value.(string); ok {
			out[key] = text
		}
	}
	return out
}

func display(name, description map[string]any, locale string) fwmetadata.Display {
	return fwmetadata.Display{
		DisplayName:        localized(name, locale),
		DisplayDescription: localized(description, locale),
	}
}

func localized(values map[string]any, locale string) string {
	keys := []string{strings.TrimSpace(locale), "zh", "zh-CN", "en"}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if text, ok := values[key].(string); ok && strings.TrimSpace(text) != "" {
			return text
		}
	}
	for _, value := range values {
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
}

func jsonMap(in map[string]string) map[string]any {
	out := map[string]any{}
	for key, value := range in {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			out[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return out
}

func require(values ...string) error {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return errors.New("metadata: required field is missing")
		}
	}
	return nil
}

func ensureNotExists(q *gorm.DB) error {
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return err
	}
	if total > 0 {
		return ErrDuplicate
	}
	return nil
}

func duplicate(field string) error {
	return &DuplicateError{Field: strings.TrimSpace(field)}
}

func DuplicateField(err error) string {
	var duplicateErr *DuplicateError
	if errors.As(err, &duplicateErr) {
		return strings.TrimSpace(duplicateErr.Field)
	}
	return ""
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
