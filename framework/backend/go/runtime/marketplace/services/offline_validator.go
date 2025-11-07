package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// OfflineUploadPayload describes `.pxp` + integrity + signature metadata.
type OfflineUploadPayload struct {
	PublishID      string   `json:"publishId"`
	PluginID       string   `json:"pluginId"`
	Version        string   `json:"version"`
	IntegrityFile  []byte   `json:"integrityFile"`
	ManifestHash   string   `json:"manifestHash"`
	AllowedTenants []string `json:"allowedTenants"`
}

// OfflineValidator performs basic validation before offline review queue.
type OfflineValidator struct{}

func (OfflineValidator) Validate(payload OfflineUploadPayload) error {
	if payload.PublishID == "" {
		return errors.New("publishId is required")
	}
	if payload.PluginID == "" {
		return errors.New("pluginId is required")
	}
	if len(payload.IntegrityFile) == 0 {
		return errors.New("integrity file missing")
	}
	if err := validateManifestHash(payload.IntegrityFile, payload.ManifestHash); err != nil {
		return err
	}
	return nil
}

func validateManifestHash(integrity []byte, manifestHash string) error {
	if manifestHash == "" {
		return errors.New("manifest hash missing")
	}
	sum := sha256.Sum256(integrity)
	if manifestHash != hex.EncodeToString(sum[:]) {
		return errors.New("manifest hash mismatch")
	}
	return nil
}
