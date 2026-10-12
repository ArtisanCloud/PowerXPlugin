package contactfw

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	common "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common"
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
	if !common.ValidText(input.DisplayName, 128, true) || !common.ValidText(input.GivenName, 128, false) || !common.ValidText(input.FamilyName, 128, false) {
		return NewError(CodeInvalidArgument, fmt.Errorf("CONTACT_TEXT_INVALID"))
	}
	if err := validateContactChannels(input.Email, input.Phone); err != nil {
		return err
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
	for _, role := range input.Roles {
		if role == RolePrimary {
			return NewError(CodePrimaryOperationRequired, nil)
		}
	}
	return ValidateTags(input.Tags)
}

func ValidateUpdateInput(input UpdateContactInput) error {
	for _, field := range []struct {
		value    *string
		required bool
	}{{input.DisplayName, true}, {input.GivenName, false}, {input.FamilyName, false}} {
		if field.value != nil && !common.ValidText(*field.value, 128, field.required) {
			return NewError(CodeInvalidArgument, nil)
		}
	}
	if err := validateUUID(input.CustomerUUID, "customer_uuid"); err != nil {
		return err
	}
	if err := validateUUID(input.ContactUUID, "contact_uuid"); err != nil {
		return err
	}
	if strings.TrimSpace(input.CustomerUUID) == "" || strings.TrimSpace(input.ContactUUID) == "" {
		return NewError(CodeInvalidArgument, fmt.Errorf("customer_uuid and contact_uuid are required"))
	}
	if input.Email != nil || input.Phone != nil {
		email, phone := "", ""
		if input.Email != nil {
			email = *input.Email
		}
		if input.Phone != nil {
			phone = *input.Phone
		}
		if err := validateContactChannels(email, phone); err != nil {
			return err
		}
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

func validateContactChannels(email, phone string) error {
	if !common.ValidText(email, 255, false) || !common.ValidText(phone, 32, false) {
		return NewError(CodeInvalidArgument, fmt.Errorf("CONTACT_CHANNEL_INVALID"))
	}
	email, phone = strings.TrimSpace(email), strings.TrimSpace(phone)
	if email != "" {
		parsed, err := mail.ParseAddress(email)
		if err != nil || parsed.Address != email || parsed.Name != "" {
			return NewError(CodeInvalidArgument, fmt.Errorf("invalid contact email"))
		}
	}
	digit := false
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			digit = true
		}
		if !(char >= '0' && char <= '9') && char != '+' && char != '-' && char != ' ' && char != '(' && char != ')' {
			return NewError(CodeInvalidArgument, fmt.Errorf("invalid contact phone"))
		}
	}
	if phone != "" && !digit {
		return NewError(CodeInvalidArgument, nil)
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
	if id, err := uuid.Parse(strings.TrimSpace(value)); err != nil || id == uuid.Nil {
		return NewError(CodeInvalidArgument, fmt.Errorf("%s must be a UUID", field))
	}
	return nil
}

func validateIdentity(channelDictionaryItemUUID, subject string) error {
	if err := validateUUID(channelDictionaryItemUUID, "channel_dictionary_item_uuid"); err != nil {
		return err
	}
	subject = strings.TrimSpace(subject)
	if !common.ValidText(subject, 255, true) {
		return NewError(CodeInvalidArgument, fmt.Errorf("channel_dictionary_item_uuid and external_subject are required"))
	}
	return nil
}
