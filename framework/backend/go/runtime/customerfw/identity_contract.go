package customerfw

import (
	"encoding/json"
	"errors"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"
)

func ValidExternalIdentitySubject(subject string) bool {
	parts := strings.SplitN(subject, ":", 4)
	if len(parts) != 4 || len(subject) > 255 || subject != strings.TrimSpace(subject) || strings.IndexFunc(subject, unicode.IsControl) >= 0 {
		return false
	}
	for _, p := range parts {
		if strings.TrimSpace(p) == "" || p != strings.TrimSpace(p) {
			return false
		}
	}
	return true
}
func ShopifyExternalIdentitySubject(domain, id string) string {
	return "shop:" + strings.TrimSpace(domain) + ":customer:" + strings.TrimSpace(id)
}
func IdentityContractError(code string, status int) *Error {
	return &Error{Code: ErrorCode(code), ReasonCode: code, StatusCode: status, Message: code}
}
func NormalizeIdentityError(err error) error {
	if err == nil {
		return nil
	}
	var typed *Error
	if errors.As(err, &typed) {
		return err
	}
	var upstream *gateway.InvocationError
	if errors.As(err, &upstream) {
		code := "CUSTOMER_IDENTITY_GATEWAY_FAILED"
		var body struct {
			Details struct {
				Reason string `json:"reason_code"`
			} `json:"details"`
			Error struct {
				Details struct {
					Reason string `json:"reason_code"`
				} `json:"details"`
			} `json:"error"`
		}
		if json.Unmarshal(upstream.Body, &body) == nil {
			if body.Details.Reason != "" {
				code = body.Details.Reason
			} else if body.Error.Details.Reason != "" {
				code = body.Error.Details.Reason
			}
		}
		out := IdentityContractError(code, upstream.StatusCode)
		out.Cause = err
		return out
	}
	out := IdentityContractError("CUSTOMER_IDENTITY_UNAVAILABLE", 503)
	out.Cause = err
	return out
}

func ValidateIdentityCustomer(p ExternalIdentityCustomer) error {
	n := NormalizeBasicAccountLabels(CreateBasicAccountRequest{Type: p.Type, DisplayName: p.DisplayName, Nickname: p.Nickname, GivenName: p.GivenName, FamilyName: p.FamilyName, PrimaryEmail: p.PrimaryEmail, PrimaryPhone: p.PrimaryPhone, PrimaryContact: p.PrimaryContact})
	invalid := func() error { return IdentityContractError("CUSTOMER_ACCOUNT_INVALID_ARGUMENT", 400) }
	if n.Type != "person" && n.Type != "company" || n.DisplayName == "" {
		return invalid()
	}
	for _, value := range []string{n.DisplayName, p.Nickname, p.GivenName, p.FamilyName} {
		if !utf8.ValidString(value) || utf8.RuneCountInString(strings.TrimSpace(value)) > 128 || strings.IndexFunc(value, unicode.IsControl) >= 0 {
			return invalid()
		}
	}
	checkChannels := func(email, phone string) bool {
		email = strings.TrimSpace(email)
		phone = strings.TrimSpace(phone)
		if utf8.RuneCountInString(email) > 255 || utf8.RuneCountInString(phone) > 32 {
			return false
		}
		if email != "" {
			a, e := mail.ParseAddress(email)
			if e != nil || a.Address != email || a.Name != "" {
				return false
			}
		}
		if phone != "" {
			digit := false
			for _, r := range phone {
				if r >= '0' && r <= '9' {
					digit = true
				} else if !strings.ContainsRune(" +-()", r) {
					return false
				}
			}
			if !digit {
				return false
			}
		}
		return true
	}
	if !checkChannels(p.PrimaryEmail, p.PrimaryPhone) {
		return invalid()
	}
	if n.Type == "company" && n.PrimaryContact == nil {
		return invalid()
	}
	if c := n.PrimaryContact; c != nil {
		if c.DisplayName == "" || !checkChannels(c.Email, c.Phone) {
			return invalid()
		}
	}
	return nil
}
