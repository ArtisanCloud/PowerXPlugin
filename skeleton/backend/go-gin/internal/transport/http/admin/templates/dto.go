package templates

import "strings"

// Template domain HTTP DTOs.

const (
	TemplateErrCodeInvalidID          = "TEMPLATE_INVALID_ID"
	TemplateErrCodeValidationFailed   = "TEMPLATE_VALIDATION_FAILED"
	TemplateErrCodeSourceIDsRequired  = "TEMPLATE_SOURCE_IDS_REQUIRED"
	templateFieldName                 = "name"
	templateFieldDescription          = "description"
	templateFieldContent              = "content"
	templateFieldSourceIDs            = "source_ids"
	templateFieldID                   = "id"
	templateMaxNameLength             = 128
	templateMaxDescriptionLength      = 512
	templateMaxContentLength          = 10000
)

type TemplateValidationError struct {
	Code    string `json:"code"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *TemplateValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type TemplateListRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Q        string `form:"q"`
}

type CreateTemplateRequest struct {
	Name        string `json:"name"        binding:"required"`
	Description string `json:"description" binding:"required"`
	Content     string `json:"content"     binding:"required"`
}

func (r *CreateTemplateRequest) Normalize() {
	if r == nil {
		return
	}
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	r.Content = strings.TrimSpace(r.Content)
}

func (r *CreateTemplateRequest) Validate() *TemplateValidationError {
	if r == nil {
		return &TemplateValidationError{Code: TemplateErrCodeValidationFailed, Field: templateFieldContent, Message: "request body is required"}
	}
	if r.Name == "" {
		return &TemplateValidationError{Code: TemplateErrCodeValidationFailed, Field: templateFieldName, Message: "name is required"}
	}
	if len(r.Name) > templateMaxNameLength {
		return &TemplateValidationError{Code: TemplateErrCodeValidationFailed, Field: templateFieldName, Message: "name is too long"}
	}
	if r.Description == "" {
		return &TemplateValidationError{Code: TemplateErrCodeValidationFailed, Field: templateFieldDescription, Message: "description is required"}
	}
	if len(r.Description) > templateMaxDescriptionLength {
		return &TemplateValidationError{Code: TemplateErrCodeValidationFailed, Field: templateFieldDescription, Message: "description is too long"}
	}
	if r.Content == "" {
		return &TemplateValidationError{Code: TemplateErrCodeValidationFailed, Field: templateFieldContent, Message: "content is required"}
	}
	if len(r.Content) > templateMaxContentLength {
		return &TemplateValidationError{Code: TemplateErrCodeValidationFailed, Field: templateFieldContent, Message: "content is too long"}
	}
	return nil
}

type UpdateTemplateRequest struct {
	Name        string `json:"name"        binding:"required"`
	Description string `json:"description" binding:"required"`
	Content     string `json:"content"     binding:"required"`
}

func (r *UpdateTemplateRequest) Normalize() {
	if r == nil {
		return
	}
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	r.Content = strings.TrimSpace(r.Content)
}

func (r *UpdateTemplateRequest) Validate() *TemplateValidationError {
	if r == nil {
		return &TemplateValidationError{Code: TemplateErrCodeValidationFailed, Field: templateFieldContent, Message: "request body is required"}
	}
	createLike := CreateTemplateRequest{
		Name:        r.Name,
		Description: r.Description,
		Content:     r.Content,
	}
	return createLike.Validate()
}

type BatchCloneRequest struct {
	SourceIDs         []uint64 `json:"source_ids"        binding:"required"`
	Copies            int      `json:"copies"            binding:"omitempty,min=1,max=50"`
	NamePrefix        string   `json:"name_prefix"`
	DescriptionPrefix string   `json:"description_prefix"`
	Notes             string   `json:"notes"`
}

func (r *BatchCloneRequest) Normalize() {
	if r == nil {
		return
	}
	r.NamePrefix = strings.TrimSpace(r.NamePrefix)
	r.DescriptionPrefix = strings.TrimSpace(r.DescriptionPrefix)
	r.Notes = strings.TrimSpace(r.Notes)
}

func (r *BatchCloneRequest) Validate() *TemplateValidationError {
	if r == nil || len(r.SourceIDs) == 0 {
		return &TemplateValidationError{Code: TemplateErrCodeSourceIDsRequired, Field: templateFieldSourceIDs, Message: "source_ids is required"}
	}
	return nil
}

type ValidateTemplateRequest struct {
	Rules  []string `json:"rules"`
	Strict bool     `json:"strict"`
}

func newInvalidTemplateIDError() *TemplateValidationError {
	return &TemplateValidationError{
		Code:    TemplateErrCodeInvalidID,
		Field:   templateFieldID,
		Message: "invalid id",
	}
}
