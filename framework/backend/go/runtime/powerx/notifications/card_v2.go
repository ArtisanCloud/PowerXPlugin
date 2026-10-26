package notifications

import (
	"bytes"
	"encoding/json"
	"time"
	"unicode/utf8"
)

// BusinessCardPresentation remains the v1 DTO. V2 has a separate image shape;
// callers select it explicitly instead of changing a v1 image_key's meaning.
type BusinessCardPresentationV2 struct {
	Contract          string               `json:"contract"`
	Revision          int64                `json:"revision"`
	Title             string               `json:"title"`
	Summary           string               `json:"summary"`
	UpdatedAt         time.Time            `json:"updated_at"`
	UnreadChangeCount int64                `json:"unread_change_count"`
	Image             *BusinessCardImageV2 `json:"image,omitempty"`
}
type BusinessCardImageV2 struct {
	Alt       string        `json:"alt"`
	Thumbnail CardRendition `json:"thumbnail"`
	Preview   CardRendition `json:"preview"`
}
type CardRendition struct {
	Variant   string  `json:"variant"`
	Status    string  `json:"status"`
	ImageKey  *string `json:"image_key,omitempty"`
	ErrorCode string  `json:"error_code,omitempty"`
	Retryable *bool   `json:"retryable,omitempty"`
}

func (r *CardRendition) UnmarshalJSON(raw []byte) error {
	type rendition CardRendition
	var value rendition
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return contractError("NOTIFICATION_PRESENTATION_INVALID")
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	if _, present := fields["image_key"]; present && (value.Status != "ready" || value.ImageKey == nil) {
		return contractError("NOTIFICATION_PRESENTATION_INVALID")
	}
	*r = CardRendition(value)
	return nil
}

func (r CardRendition) validate(variant string) bool {
	if r.Variant != variant || len(r.ErrorCode) > 128 {
		return false
	}
	if r.Status == "ready" {
		return r.ImageKey != nil && cardToken.MatchString(*r.ImageKey)
	}
	switch r.Status {
	case "not_requested", "waiting_upload", "waiting_source", "queued", "processing", "failed", "scheduling_failed", "unsupported", "unavailable":
		return r.ImageKey == nil
	default:
		return false
	}
}

func (p BusinessCardPresentationV2) Validate() error {
	if p.Contract != BusinessCardContractV2 || !validRevision(p.Revision) || p.Title == "" || utf8.RuneCountInString(p.Title) > 255 || utf8.RuneCountInString(p.Summary) > 4000 || p.UpdatedAt.IsZero() || p.UnreadChangeCount < 0 || p.UnreadChangeCount > MaxRevision {
		return contractError("NOTIFICATION_PRESENTATION_INVALID")
	}
	if p.Image != nil && (utf8.RuneCountInString(p.Image.Alt) > 255 || !p.Image.Thumbnail.validate("thumbnail_64") || !p.Image.Preview.validate("preview")) {
		return contractError("NOTIFICATION_PRESENTATION_INVALID")
	}
	return nil
}
