package media

import (
	"context"
	"github.com/google/uuid"
)

const (
	ImageRecipeV2                    = "standard-image-v2"
	VariantPreview                   = "preview"
	VariantThumbnail                 = "thumbnail"
	VariantThumbnail64               = "thumbnail_64"
	Thumbnail64MaxBytes              = 32768
	Thumbnail64MaxEdge               = 64
	ImageDerivativesReadCapability   = "com.corex.media.image_derivatives.read"
	ImageDerivativesEnsureCapability = "com.corex.media.image_derivatives.ensure"
)

// ImageDerivatives mirrors Core's optional standard-image processing response.
// Reading this snapshot does not schedule work, subscribe to changes, or imply
// that any variant is ready. Use a ready item's UUID with the variant APIs to
// retrieve its actual MIME type, size, and download ticket.
type ImageDerivatives struct {
	AssetUUID   string                `json:"asset_uuid"`
	UploadState string                `json:"upload_state"`
	Status      string                `json:"status"`
	Recipe      string                `json:"recipe"`
	JobUUID     string                `json:"job_uuid,omitempty"`
	Attempts    int                   `json:"attempts"`
	ErrorCode   string                `json:"error_code,omitempty"`
	Retryable   bool                  `json:"retryable"`
	Items       []ImageDerivativeItem `json:"items"`
}

type ImageDerivativeItem struct {
	Variant     string `json:"variant"`
	Status      string `json:"status"`
	VariantUUID string `json:"variant_uuid,omitempty"`
	ErrorCode   string `json:"error_code,omitempty"`
	Retryable   *bool  `json:"retryable,omitempty"`
}

func validMediaUUID(raw string) bool {
	id, err := uuid.Parse(raw)
	return err == nil && id != uuid.Nil && id.String() == raw
}
func validDerivativeStatus(status string) bool {
	switch status {
	case "not_requested", "waiting_upload", "waiting_source", "queued", "processing", "ready", "failed", "scheduling_failed", "unsupported", "unavailable":
		return true
	}
	return false
}
func (d ImageDerivatives) Validate(assetUUID string) error {
	if !validMediaUUID(d.AssetUUID) || (assetUUID != "" && d.AssetUUID != assetUUID) || (d.Recipe != ImageRecipeV2 && d.Recipe != "standard-image-v1") || d.UploadState == "" || !validDerivativeStatus(d.Status) || d.Attempts < 0 {
		return derivativeError(502, "MEDIA_DERIVATIVES_RESPONSE_INVALID")
	}
	if d.Recipe == ImageRecipeV2 && len(d.Items) < 3 {
		return derivativeError(502, "MEDIA_DERIVATIVES_RESPONSE_INVALID")
	}
	seen := map[string]bool{}
	for _, item := range d.Items {
		if item.Variant == "" || seen[item.Variant] || !validDerivativeStatus(item.Status) || (item.Status == "ready" && !validMediaUUID(item.VariantUUID)) || (item.Status != "ready" && item.VariantUUID != "") {
			return derivativeError(502, "MEDIA_DERIVATIVES_RESPONSE_INVALID")
		}
		seen[item.Variant] = true
	}
	if d.Status == "ready" {
		variants := []string{VariantPreview, VariantThumbnail}
		if d.Recipe == ImageRecipeV2 {
			variants = append(variants, VariantThumbnail64)
		}
		for _, name := range variants {
			found := false
			for _, item := range d.Items {
				if item.Variant == name && item.Status == "ready" {
					found = true
				}
			}
			if !found {
				return derivativeError(502, "MEDIA_DERIVATIVES_RESPONSE_INVALID")
			}
		}
	}
	return nil
}

// Item selects by name; the aggregate task status never gates a ready item.
func (d *ImageDerivatives) Item(variant string) (*ImageDerivativeItem, error) {
	if d == nil {
		return nil, derivativeError(503, "MEDIA_DERIVATIVES_UNAVAILABLE")
	}
	if err := d.Validate(""); err != nil {
		return nil, err
	}
	for _, item := range d.Items {
		if item.Variant == variant {
			copy := item
			return &copy, nil
		}
	}
	return nil, derivativeError(404, "MEDIA_DERIVATIVE_NOT_FOUND")
}

type ImageDerivativeService interface {
	GetImageDerivatives(context.Context, string) (*ImageDerivatives, error)
	EnsureImageDerivatives(context.Context, string, EnsureImageDerivativesInput) (*ImageDerivatives, error)
	SubscribeImageDerivatives(context.Context, string, ImageDerivativeSubscriptionOptions, func(ImageDerivativeEvent) error) error
}
type EnsureImageDerivativesInput struct {
	RetryFailed bool `json:"retry_failed,omitempty"`
}

// ReadImageDerivative uses the original parent and actual variant metadata.
// Missing/pending/failed specs never select a different size or the origin.
func (c *HostClient) ReadImageDerivative(ctx context.Context, d *ImageDerivatives, variant string) (*Variant, *TransferTicket, error) {
	item, err := d.Item(variant)
	if err != nil {
		return nil, nil, err
	}
	if item.Status != "ready" {
		return nil, nil, derivativeError(409, "MEDIA_DERIVATIVE_NOT_READY")
	}
	v, err := c.GetVariant(ctx, item.VariantUUID)
	if err != nil {
		return nil, nil, err
	}
	if v.VariantUUID != item.VariantUUID || v.AssetUUID != d.AssetUUID || v.Status != "ready" || (v.MimeType != "image/jpeg" && v.MimeType != "image/png") || v.SizeBytes < 1 || (variant == VariantThumbnail64 && v.SizeBytes > Thumbnail64MaxBytes) {
		return nil, nil, derivativeError(502, "MEDIA_VARIANT_RESPONSE_INVALID")
	}
	ticket, err := c.PresignVariantDownload(ctx, d.AssetUUID, v.VariantUUID, VariantTicketInput{})
	if err != nil {
		return nil, nil, err
	}
	return v, ticket, nil
}
