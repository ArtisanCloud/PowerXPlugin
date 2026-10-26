package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
)

const (
	BusinessCardContract           = "business_notification_card.v1"
	BusinessCardContractV2         = "business_notification_card.v2"
	InvalidateCardCapability       = "com.corex.notifications.invalidate_card"
	CardProtocol                   = "core_internal"
	CardEndpoint                   = "core://notifications"
	CardMethod                     = "INVOKE"
	CardActor                      = "service_actor"
	ContentChanged                 = "content_changed"
	AccessChanged                  = "access_changed"
	PresentationChanged            = "presentation_changed"
	MaxRevision              int64 = 9007199254740991
)

var cardToken = regexp.MustCompile(`^[A-Za-z0-9_-]{1,256}$`)
var pluginID = regexp.MustCompile(`^[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)*$`)

type BusinessCardReference struct {
	Contract       string `json:"contract"`
	SourcePlugin   string `json:"source_plugin"`
	BusinessType   string `json:"business_type"`
	DeliveryKey    string `json:"delivery_key"`
	SourceRevision int64  `json:"source_revision"`
}

// Presentation is a member-visible snapshot supplied by the source plugin.
// Its revision and unread count are independent of the Core notification revision.
type BusinessCardPresentation struct {
	Contract          string             `json:"contract"`
	Revision          int64              `json:"revision"`
	Title             string             `json:"title"`
	Summary           string             `json:"summary"`
	UpdatedAt         time.Time          `json:"updated_at"`
	UnreadChangeCount int64              `json:"unread_change_count"`
	Image             *BusinessCardImage `json:"image,omitempty"`
}
type BusinessCardImage struct {
	ImageKey string `json:"image_key"`
	Alt      string `json:"alt"`
}
type BusinessCardPresentationSource struct {
	Contract           string `json:"contract"`
	SourcePlugin       string `json:"source_plugin"`
	PresentationURL    string `json:"presentation_url"`
	PreviewURLTemplate string `json:"preview_url_template"`
}
type NotificationUpdatedEvent struct {
	Notification
	EventType      string `json:"eventType"`
	PayloadVersion int    `json:"payload_version"`
	TenantUUID     string `json:"tenant_uuid"`
}

func validRevision(revision int64) bool { return revision > 0 && revision <= MaxRevision }
func validUUID(raw string) bool {
	id, err := uuid.Parse(raw)
	return err == nil && id != uuid.Nil && id.String() == raw
}
func contractError(code string) *HTTPError {
	return &HTTPError{StatusCode: http.StatusBadGateway, ReasonCode: code, ErrorCode: code}
}
func invalidCard() *HTTPError {
	return &HTTPError{StatusCode: http.StatusBadRequest, ReasonCode: "NOTIFICATION_CARD_INVALID", ErrorCode: "NOTIFICATION_CARD_INVALID"}
}

func (r BusinessCardReference) Validate() error {
	if (r.Contract != BusinessCardContract && r.Contract != BusinessCardContractV2) || len(r.SourcePlugin) > 128 || !pluginID.MatchString(r.SourcePlugin) || !cardToken.MatchString(r.BusinessType) || !cardToken.MatchString(r.DeliveryKey) || !validRevision(r.SourceRevision) {
		return invalidCard()
	}
	return nil
}

// CardReference decodes the reserved metadata object strictly. Other metadata
// belongs to the caller and is preserved without modification.
func CardReference(metadata map[string]any) (*BusinessCardReference, error) {
	value, present := metadata["business_card"]
	if !present {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, invalidCard()
	}
	var ref BusinessCardReference
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ref); err != nil {
		return nil, invalidCard()
	}
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	return &ref, nil
}

// WithBusinessCard returns a copy; adding the reserved reference never replaces
// unrelated metadata or modifies a caller-owned map.
func (in CreateInput) WithBusinessCard(ref BusinessCardReference) (CreateInput, error) {
	if err := ref.Validate(); err != nil {
		return in, err
	}
	metadata := make(map[string]any, len(in.Metadata)+1)
	for k, v := range in.Metadata {
		metadata[k] = v
	}
	if _, exists := metadata["business_card"]; exists {
		return in, invalidCard()
	}
	metadata["business_card"] = ref
	in.Metadata = metadata
	return in, nil
}

type InvalidateCardInput struct {
	MemberUUID       string                 `json:"member_uuid"`
	NotificationUUID string                 `json:"notification_uuid"`
	SourceRevision   int64                  `json:"source_revision"`
	Reason           string                 `json:"reason,omitempty"`
	CardReference    *BusinessCardReference `json:"card_reference,omitempty"`
}
type InvalidateCardResult struct {
	NotificationUUID string `json:"notification_uuid"`
	Revision         int64  `json:"revision"`
	SourceRevision   int64  `json:"source_revision"`
	Changed          bool   `json:"changed"`
}
type CardInvalidator interface {
	InvalidateCard(context.Context, InvalidateCardInput) (*InvalidateCardResult, error)
}

func (c *Client) InvalidateCard(ctx context.Context, in InvalidateCardInput) (*InvalidateCardResult, error) {
	if c == nil || c.http == nil || c.tokens == nil {
		return nil, &HTTPError{StatusCode: 503, ReasonCode: "NOTIFICATION_CLIENT_UNAVAILABLE"}
	}
	if !validUUID(in.MemberUUID) || !validUUID(in.NotificationUUID) || !validRevision(in.SourceRevision) || (in.Reason != "" && in.Reason != ContentChanged && in.Reason != AccessChanged && in.Reason != PresentationChanged) {
		return nil, invalidCard()
	}
	if in.CardReference != nil {
		if err := in.CardReference.Validate(); err != nil {
			return nil, err
		}
		if in.CardReference.SourceRevision != in.SourceRevision || (c.pluginID != "" && c.pluginID != in.CardReference.SourcePlugin) {
			return nil, invalidCard()
		}
	}
	request := struct {
		CapabilityID      string `json:"capability_id"`
		PreferredProtocol string `json:"preferred_protocol"`
		Payload           struct {
			Body any `json:"body"`
		} `json:"payload"`
	}{CapabilityID: InvalidateCardCapability, PreferredProtocol: CardProtocol}
	request.Payload.Body = struct {
		Operation string `json:"operation"`
		InvalidateCardInput
	}{"invalidate_card", in}
	var data struct {
		ProtocolUsed string `json:"protocol_used"`
		FallbackUsed *bool  `json:"fallback_used"`
		Payload      struct {
			Result json.RawMessage `json:"result"`
		} `json:"payload"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/tenant/invocations", request, &data); err != nil {
		return nil, err
	}
	if data.ProtocolUsed != CardProtocol || data.FallbackUsed == nil || *data.FallbackUsed {
		return nil, contractError("NOTIFICATION_CARD_BINDING_RESPONSE_INVALID")
	}
	var out struct {
		NotificationUUID string `json:"notification_uuid"`
		Revision         int64  `json:"revision"`
		SourceRevision   int64  `json:"source_revision"`
		Changed          *bool  `json:"changed"`
	}
	if err := json.Unmarshal(data.Payload.Result, &out); err != nil || out.NotificationUUID != in.NotificationUUID || !validRevision(out.Revision) || !validRevision(out.SourceRevision) || out.SourceRevision < in.SourceRevision || out.Changed == nil {
		return nil, contractError("NOTIFICATION_CARD_RESPONSE_INVALID")
	}
	return &InvalidateCardResult{out.NotificationUUID, out.Revision, out.SourceRevision, *out.Changed}, nil
}
