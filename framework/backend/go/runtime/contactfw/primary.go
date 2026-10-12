package contactfw

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

const CapabilityContactsPrimaryManage = "com.corex.customer.contacts.primary_manage"

// A non-nil empty expected reference explicitly asserts that no primary exists.
type SetPrimaryContactInput struct {
	CustomerUUID               string  `json:"customer_uuid"`
	ContactUUID                string  `json:"contact_uuid"`
	ExpectedPrimaryContactUUID *string `json:"expected_primary_contact_uuid"`
}

type PrimaryContactResult struct {
	CustomerUUID               string `json:"customer_uuid"`
	PrimaryContactUUID         string `json:"primary_contact_uuid"`
	PreviousPrimaryContactUUID string `json:"previous_primary_contact_uuid"`
}

// PrimaryContactManager is independent of the six-operation Store.
type PrimaryContactManager interface {
	SetPrimary(context.Context, SetPrimaryContactInput) (*PrimaryContactResult, error)
}

func (in *SetPrimaryContactInput) UnmarshalJSON(raw []byte) error {
	if !utf8.Valid(raw) {
		return NewError(CodeInvalidArgument, nil)
	}
	type wire SetPrimaryContactInput
	var out wire
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&out); err != nil {
		return NewError(CodeInvalidArgument, err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return NewError(CodeInvalidArgument, err)
	}
	if err := ValidateSetPrimaryInput(SetPrimaryContactInput(out)); err != nil {
		return err
	}
	*in = SetPrimaryContactInput(out)
	return nil
}

func ValidateSetPrimaryInput(in SetPrimaryContactInput) error {
	if in.ExpectedPrimaryContactUUID == nil {
		return NewError(CodePrimaryExpectedRequired, nil)
	}
	if err := ValidateGetInput(GetContactInput{CustomerUUID: in.CustomerUUID, ContactUUID: in.ContactUUID}); err != nil {
		return err
	}
	if *in.ExpectedPrimaryContactUUID != "" {
		return validateUUID(*in.ExpectedPrimaryContactUUID, "expected_primary_contact_uuid")
	}
	return nil
}

func ValidatePrimaryResult(in SetPrimaryContactInput, out *PrimaryContactResult) error {
	if out == nil || out.CustomerUUID != in.CustomerUUID || out.PrimaryContactUUID != in.ContactUUID || in.ExpectedPrimaryContactUUID == nil || out.PreviousPrimaryContactUUID != *in.ExpectedPrimaryContactUUID {
		return NewError(CodeDelegateUnavailable, errors.New("CONTACT_PRIMARY_RESPONSE_INVALID"))
	}
	return nil
}

func (c *CapabilityClient) SetPrimary(ctx context.Context, in SetPrimaryContactInput) (*PrimaryContactResult, error) {
	if err := ValidateSetPrimaryInput(in); err != nil {
		return nil, err
	}
	// Core's operation returns {payload: result} inside the invocation payload.
	var out struct {
		Payload *PrimaryContactResult `json:"payload"`
	}
	err := c.invoke(ctx, CapabilityContactsPrimaryManage, "set_primary", contactOperation{CustomerUUID: in.CustomerUUID, ContactUUID: in.ContactUUID, ExpectedPrimaryContactUUID: in.ExpectedPrimaryContactUUID}, &out)
	if err != nil {
		return nil, err
	}
	if err := ValidatePrimaryResult(in, out.Payload); err != nil {
		return nil, err
	}
	return out.Payload, nil
}

func (c *CoreDelegatedClient) SetPrimary(ctx context.Context, in SetPrimaryContactInput) (*PrimaryContactResult, error) {
	if err := ValidateSetPrimaryInput(in); err != nil {
		return nil, err
	}
	binding, err := c.requireBinding()
	if err != nil {
		return nil, err
	}
	manager, ok := binding.(PrimaryContactManager)
	if !ok {
		return nil, NewError(CodeDelegateUnavailable, nil)
	}
	out, err := manager.SetPrimary(ctx, in)
	if err != nil {
		return nil, normalizeDelegatedError(err)
	}
	if err := ValidatePrimaryResult(in, out); err != nil {
		return nil, err
	}
	return out, nil
}

var _ PrimaryContactManager = (*CapabilityClient)(nil)
var _ PrimaryContactManager = (*CoreDelegatedClient)(nil)
