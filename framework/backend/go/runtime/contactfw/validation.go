package contactfw

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

const (
	MaxTags         = 20
	MaxTagLength    = 64
	MaxTagsLength   = 1024
	DefaultPageSize = 20
	MaximumPageSize = 100
)

var tagPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// ValidateTags keeps tags a bounded classification field rather than a
// free-text extension slot. Its grammar matches the Core v1 contract.
func ValidateTags(tags []string) error {
	if len(tags) > MaxTags {
		return NewError(CodeInvalidArgument, fmt.Errorf("too many tags"))
	}
	totalBytes := 0
	for _, tag := range tags {
		if !tagPattern.MatchString(tag) {
			return NewError(CodeInvalidArgument, fmt.Errorf("invalid tag"))
		}
		totalBytes += len(tag)
	}
	if totalBytes > MaxTagsLength {
		return NewError(CodeInvalidArgument, fmt.Errorf("tags are too long"))
	}
	return nil
}

func ValidateRoles(roles []Role) error {
	for _, role := range roles {
		if role != RolePrimary && role != RoleLegalRepresentative {
			return NewError(CodeInvalidArgument, fmt.Errorf("unsupported contact role"))
		}
	}
	return nil
}

// ValidateCreateInput is shared validation for local adapters. Core remains
// authoritative for tenant scope, authorization, customer membership, and
// audit provenance.
func ValidateCreateInput(input CreateContactInput) error {
	if err := validateUUID(input.CustomerUUID, "customer_uuid"); err != nil {
		return err
	}
	if strings.TrimSpace(input.DisplayName) == "" {
		return NewError(CodeInvalidArgument, fmt.Errorf("customer_uuid and display_name are required"))
	}
	if input.Status != StatusActive && input.Status != StatusInactive && input.Status != StatusTemporary {
		return NewError(CodeInvalidArgument, fmt.Errorf("invalid contact status"))
	}
	if input.CreationIntent != CreationIntentExplicitCreate && input.CreationIntent != CreationIntentExplicitTemporary {
		return NewError(CodeInvalidArgument, fmt.Errorf("invalid creation intent"))
	}
	if input.Status == StatusTemporary && input.CreationIntent != CreationIntentExplicitTemporary {
		return NewError(CodeInvalidArgument, fmt.Errorf("temporary contact requires explicit temporary intent"))
	}
	if input.Status != StatusTemporary && input.CreationIntent != CreationIntentExplicitCreate {
		return NewError(CodeInvalidArgument, fmt.Errorf("non-temporary contact requires explicit create intent"))
	}
	if err := ValidateRoles(input.Roles); err != nil {
		return err
	}
	return ValidateTags(input.Tags)
}

func ValidateUpdateInput(input UpdateContactInput) error {
	if err := validateUUID(input.CustomerUUID, "customer_uuid"); err != nil {
		return err
	}
	if err := validateUUID(input.ContactUUID, "contact_uuid"); err != nil {
		return err
	}
	if strings.TrimSpace(input.CustomerUUID) == "" || strings.TrimSpace(input.ContactUUID) == "" {
		return NewError(CodeInvalidArgument, fmt.Errorf("customer_uuid and contact_uuid are required"))
	}
	if input.Status != nil && *input.Status != StatusActive && *input.Status != StatusInactive && *input.Status != StatusTemporary {
		return NewError(CodeInvalidArgument, fmt.Errorf("invalid contact status"))
	}
	if input.Roles != nil {
		if err := ValidateRoles(*input.Roles); err != nil {
			return err
		}
	}
	if input.Tags != nil {
		return ValidateTags(*input.Tags)
	}
	return nil
}

func NormalizeListInput(input ListByCustomerInput) (ListByCustomerInput, error) {
	if err := validateUUID(input.CustomerUUID, "customer_uuid"); err != nil {
		return input, err
	}
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = DefaultPageSize
	}
	if input.PageSize > MaximumPageSize {
		return input, NewError(CodeInvalidArgument, fmt.Errorf("page_size exceeds maximum"))
	}
	if input.Status != nil && *input.Status != StatusActive && *input.Status != StatusInactive && *input.Status != StatusTemporary {
		return input, NewError(CodeInvalidArgument, fmt.Errorf("invalid contact status"))
	}
	return input, nil
}

func ValidateGetInput(input GetContactInput) error {
	if err := validateUUID(input.CustomerUUID, "customer_uuid"); err != nil {
		return err
	}
	return validateUUID(input.ContactUUID, "contact_uuid")
}

func ValidateResolveIdentityInput(input ResolveContactIdentityInput) error {
	if err := validateUUID(input.CustomerUUID, "customer_uuid"); err != nil {
		return err
	}
	return validateIdentity(input.ChannelDictionaryItemUUID, input.ExternalSubject)
}

func ValidateBindIdentityInput(input BindContactIdentityInput) error {
	if err := ValidateGetInput(GetContactInput{CustomerUUID: input.CustomerUUID, ContactUUID: input.ContactUUID}); err != nil {
		return err
	}
	return validateIdentity(input.ChannelDictionaryItemUUID, input.ExternalSubject)
}

func ValidateMigrateIdentityChannelInput(input MigrateContactIdentityChannelInput) error {
	if err := ValidateGetInput(GetContactInput{CustomerUUID: input.CustomerUUID, ContactUUID: input.ContactUUID}); err != nil {
		return err
	}
	if err := validateUUID(input.IdentityUUID, "identity_uuid"); err != nil {
		return err
	}
	return validateUUID(input.ChannelDictionaryItemUUID, "channel_dictionary_item_uuid")
}

func validateUUID(value, field string) error {
	if _, err := uuid.Parse(strings.TrimSpace(value)); err != nil {
		return NewError(CodeInvalidArgument, fmt.Errorf("%s must be a UUID", field))
	}
	return nil
}

func validateIdentity(channelDictionaryItemUUID, subject string) error {
	if err := validateUUID(channelDictionaryItemUUID, "channel_dictionary_item_uuid"); err != nil {
		return err
	}
	subject = strings.TrimSpace(subject)
	if subject == "" || len(subject) > 255 {
		return NewError(CodeInvalidArgument, fmt.Errorf("channel_dictionary_item_uuid and external_subject are required"))
	}
	return nil
}
