package manifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

const BusinessNotificationCardContract = "business_notification_card.v1"
const BusinessNotificationCardContractV2 = "business_notification_card.v2"

func IsBusinessNotificationCardContract(contract string) bool {
	return contract == BusinessNotificationCardContract || contract == BusinessNotificationCardContractV2
}

type NotificationPresentation struct {
	BusinessType     string `json:"business_type" yaml:"business_type"`
	Contract         string `json:"contract" yaml:"contract"`
	PresentationPath string `json:"presentation_path" yaml:"presentation_path"`
	PreviewPath      string `json:"preview_path" yaml:"preview_path"`
}

var presentationToken = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,256}$`)

func (p NotificationPresentation) Validate() error {
	invalid := func() error { return errors.New("NOTIFICATION_PRESENTATION_INVALID") }
	if !presentationToken.MatchString(p.BusinessType) || !IsBusinessNotificationCardContract(p.Contract) {
		return invalid()
	}
	for _, path := range []string{p.PresentationPath, p.PreviewPath} {
		if !strings.HasPrefix(path, "/admin/") || strings.ContainsAny(path, "?%#\\\r\n") || strings.Contains(path, "//") || strings.Count(path, "{delivery_key}") != 1 {
			return invalid()
		}
		resolved := strings.ReplaceAll(strings.ReplaceAll(path, "{delivery_key}", "key"), "{image_key}", "image")
		for _, part := range strings.Split(strings.TrimPrefix(resolved, "/"), "/") {
			if !presentationToken.MatchString(part) {
				return invalid()
			}
		}
	}
	if strings.Contains(p.PresentationPath, "{image_key}") || strings.Count(p.PreviewPath, "{image_key}") != 1 {
		return invalid()
	}
	return nil
}

func ValidateNotificationPresentations(bindings []NotificationPresentation) error {
	seen := map[string]bool{}
	for _, binding := range bindings {
		if err := binding.Validate(); err != nil {
			return err
		}
		key := binding.BusinessType + ":" + binding.Contract
		if seen[key] {
			return errors.New("NOTIFICATION_PRESENTATION_DUPLICATE")
		}
		seen[key] = true
	}
	return nil
}

// DecodeNotificationPresentations validates the same field in raw manifest
// maps, CLI loading, registration and generated plugin checks.
func DecodeNotificationPresentations(value any) ([]NotificationPresentation, error) {
	if value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var bindings []NotificationPresentation
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&bindings); err != nil {
		return nil, err
	}
	return bindings, ValidateNotificationPresentations(bindings)
}
