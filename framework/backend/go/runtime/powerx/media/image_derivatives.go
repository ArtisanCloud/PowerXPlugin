package media

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
}
