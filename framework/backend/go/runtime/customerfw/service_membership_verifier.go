package customerfw

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/google/uuid"
)

const CapabilityCustomerMembershipsServiceRead = "com.corex.customer.memberships.service_read"

const (
	CodeServiceMembershipGatewayRequired  = "CUSTOMER_MEMBERSHIP_SERVICE_GATEWAY_REQUIRED"
	CodeServiceMembershipTenantRequired   = "CUSTOMER_MEMBERSHIP_SERVICE_TENANT_REQUIRED"
	CodeServiceMembershipInvalidArgument  = "CUSTOMER_MEMBERSHIP_SERVICE_INVALID_ARGUMENT"
	CodeServiceMembershipResponseInvalid  = "CUSTOMER_MEMBERSHIP_SERVICE_RESPONSE_INVALID"
	CodeServiceMembershipTenantMismatch   = "CUSTOMER_MEMBERSHIP_SERVICE_TENANT_MISMATCH"
	CodeServiceMembershipCustomerMismatch = "CUSTOMER_MEMBERSHIP_SERVICE_CUSTOMER_MISMATCH"
	CodeServiceMembershipExpired          = "CUSTOMER_MEMBERSHIP_SERVICE_EXPIRED"
)

// ServiceMembershipGateway preserves the exact Gateway data bytes and exposes
// its configured tenant assertion. Core remains authoritative for credential
// identity, tenant scope and grants. The standard gateway.Client implements it.
type ServiceMembershipGateway interface {
	AdminGatewayInvoker
	TenantUUID() string
}

// VerifiedCustomerMembership is one service verification snapshot, not a
// customer login, transaction authorization or cacheable grant.
type VerifiedCustomerMembership struct {
	SchemaVersion      int        `json:"schema_version"`
	TenantUUID         string     `json:"tenant_uuid"`
	CustomerUUID       string     `json:"customer_uuid"`
	MembershipUUID     string     `json:"membership_uuid"`
	PrimaryContactUUID string     `json:"primary_contact_uuid"`
	Status             string     `json:"status"`
	ExpiresAt          *time.Time `json:"expires_at"`
	CheckedAt          time.Time  `json:"checked_at"`
	Revision           string     `json:"revision"`
	TraceID            string     `json:"trace_id"`
}

// ServiceMembershipVerificationError describes local input or response
// validation. Code is a machine code for the caller's locale boundary. Core
// failures are returned unchanged as gateway.InvocationError, not remapped here.
type ServiceMembershipVerificationError struct {
	Code       string
	StatusCode int
	TraceID    string
	Cause      error
}

func (e *ServiceMembershipVerificationError) Error() string { return e.Code }
func (e *ServiceMembershipVerificationError) Unwrap() error { return e.Cause }

type ServiceMembershipVerifier struct {
	invoker    ServiceMembershipGateway
	tenantUUID string
}

func NewServiceMembershipVerifier(invoker ServiceMembershipGateway) (*ServiceMembershipVerifier, error) {
	if invoker == nil {
		return nil, verificationError(CodeServiceMembershipGatewayRequired, http.StatusServiceUnavailable, "", nil)
	}
	tenant := invoker.TenantUUID()
	if !serviceMembershipUUID(tenant) {
		return nil, verificationError(CodeServiceMembershipTenantRequired, http.StatusBadRequest, "", nil)
	}
	return &ServiceMembershipVerifier{invoker: invoker, tenantUUID: tenant}, nil
}

// Verify performs exactly one fresh invocation. It never sends a tenant,
// customer JWT, arbitrary binding, headers, actor or plugin identity override.
func (v *ServiceMembershipVerifier) Verify(ctx context.Context, customerUUID string) (*VerifiedCustomerMembership, error) {
	if v == nil || v.invoker == nil {
		return nil, verificationError(CodeServiceMembershipGatewayRequired, http.StatusServiceUnavailable, "", nil)
	}
	if ctx == nil || !serviceMembershipUUID(customerUUID) {
		return nil, verificationError(CodeServiceMembershipInvalidArgument, http.StatusBadRequest, "", nil)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if v.invoker.TenantUUID() != v.tenantUUID {
		return nil, verificationError(CodeServiceMembershipTenantMismatch, http.StatusBadGateway, "", nil)
	}
	resp, err := v.invoker.Invoke(ctx, gateway.InvokeRequest{
		CapabilityID: CapabilityCustomerMembershipsServiceRead, PreferredProtocol: "core_internal",
		Payload: map[string]any{"body": struct {
			Operation     string `json:"operation"`
			SchemaVersion int    `json:"schema_version"`
			CustomerUUID  string `json:"customer_uuid"`
		}{Operation: "verify_customer", SchemaVersion: 1, CustomerUUID: customerUUID}},
	})
	if err != nil {
		return nil, err
	}
	return decodeServiceMembership(resp, v.tenantUUID, customerUUID)
}

var serviceMembershipRevision = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var serviceMembershipTimestamp = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]+)?(Z|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$`)

var serviceMembershipFields = []string{
	"schema_version", "tenant_uuid", "customer_uuid", "membership_uuid", "primary_contact_uuid",
	"status", "expires_at", "checked_at", "revision",
}

func decodeServiceMembership(resp *gateway.Response, tenant, customer string) (*VerifiedCustomerMembership, error) {
	trace := ""
	invalid := func(cause error) (*VerifiedCustomerMembership, error) {
		return nil, verificationError(CodeServiceMembershipResponseInvalid, http.StatusBadGateway, trace, cause)
	}
	if resp == nil || len(resp.RawData) == 0 {
		return invalid(nil)
	}
	// RawData is the exact data object from the standard Gateway. Re-encoding
	// Data would erase duplicate fields and prevent strict wire validation.
	data, err := serviceMembershipObject(resp.RawData, nil, []string{"trace_id", "payload"})
	if err != nil {
		return invalid(err)
	}
	if json.Unmarshal(data["trace_id"], &trace) != nil || !validMembershipTrace(trace) {
		trace = ""
		return invalid(nil)
	}
	payload, err := serviceMembershipObject(data["payload"], []string{"item"}, []string{"item"})
	if err != nil {
		return invalid(err)
	}
	item, err := serviceMembershipObject(payload["item"], serviceMembershipFields, serviceMembershipFields)
	if err != nil {
		return invalid(err)
	}
	var out VerifiedCustomerMembership
	if json.Unmarshal(item["schema_version"], &out.SchemaVersion) != nil || out.SchemaVersion != 1 {
		return invalid(nil)
	}
	for key, target := range map[string]*string{
		"tenant_uuid": &out.TenantUUID, "customer_uuid": &out.CustomerUUID,
		"membership_uuid": &out.MembershipUUID, "primary_contact_uuid": &out.PrimaryContactUUID,
	} {
		if json.Unmarshal(item[key], target) != nil || !serviceMembershipUUID(*target) {
			return invalid(nil)
		}
	}
	if out.TenantUUID != tenant {
		return nil, verificationError(CodeServiceMembershipTenantMismatch, http.StatusBadGateway, trace, nil)
	}
	if out.CustomerUUID != customer {
		return nil, verificationError(CodeServiceMembershipCustomerMismatch, http.StatusBadGateway, trace, nil)
	}
	if json.Unmarshal(item["status"], &out.Status) != nil || out.Status != "active" {
		return invalid(nil)
	}
	checked, err := serviceMembershipTime(item["checked_at"])
	if err != nil {
		return invalid(err)
	}
	_, offset := checked.Zone()
	if offset != 0 {
		return invalid(nil)
	}
	out.CheckedAt = checked
	if !bytes.Equal(bytes.TrimSpace(item["expires_at"]), []byte("null")) {
		expires, err := serviceMembershipTime(item["expires_at"])
		if err != nil {
			return invalid(err)
		}
		if !expires.After(checked) {
			return nil, verificationError(CodeServiceMembershipExpired, http.StatusConflict, trace, nil)
		}
		out.ExpiresAt = &expires
	}
	if json.Unmarshal(item["revision"], &out.Revision) != nil || !serviceMembershipRevision.MatchString(out.Revision) {
		return invalid(nil)
	}
	out.TraceID = trace
	return &out, nil
}

func serviceMembershipUUID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id != uuid.Nil && id.String() == value
}

func serviceMembershipTime(raw json.RawMessage) (time.Time, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return time.Time{}, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.IsZero() || !serviceMembershipTimestamp.MatchString(value) {
		return time.Time{}, errors.New(CodeServiceMembershipResponseInvalid)
	}
	return parsed, nil
}

func validMembershipTrace(value string) bool {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// serviceMembershipObject rejects duplicate, noncanonical and unknown keys in
// closed contract objects. A nil allowed list leaves Gateway metadata open.
func serviceMembershipObject(raw json.RawMessage, allowed, required []string) (map[string]json.RawMessage, error) {
	fail := func() (map[string]json.RawMessage, error) {
		return nil, errors.New(CodeServiceMembershipResponseInvalid)
	}
	if !utf8.Valid(raw) {
		return fail()
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return fail()
	}
	keys := make(map[string]bool, len(allowed))
	for _, key := range allowed {
		keys[key] = true
	}
	object := make(map[string]json.RawMessage)
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return fail()
		}
		key, ok := token.(string)
		if !ok || allowed != nil && !keys[key] {
			return fail()
		}
		if _, duplicate := object[key]; duplicate {
			return fail()
		}
		var value json.RawMessage
		if d.Decode(&value) != nil {
			return fail()
		}
		object[key] = value
	}
	if token, err = d.Token(); err != nil || token != json.Delim('}') {
		return fail()
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return fail()
	}
	for _, key := range required {
		if _, ok := object[key]; !ok {
			return fail()
		}
	}
	return object, nil
}

func verificationError(code string, status int, trace string, cause error) error {
	return &ServiceMembershipVerificationError{Code: code, StatusCode: status, TraceID: trace, Cause: cause}
}
