package host_contract

import (
	hostcontract "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
	dto "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/media"
	"github.com/gin-gonic/gin"
)

func (h *Handler) probeMediaTransfer(c *gin.Context, input map[string]any, result *hostcontract.ProbeResult) error {
	s, err := h.media.Media()
	if err != nil {
		return err
	}
	ctx := c.Request.Context()
	if result.Operation == hostcontract.OperationImageDerivativesStatus || result.Operation == hostcontract.OperationImageDerivativesEnsure {
		if err := ensureInputKeys(input, "asset_uuid", "retry_failed"); err != nil {
			return err
		}
		if result.Operation == hostcontract.OperationImageDerivativesStatus {
			if err := ensureInputKeys(input, "asset_uuid"); err != nil {
				return err
			}
		}
		asset, err := inputUUID(input, "asset_uuid")
		if err != nil {
			return err
		}
		retry, err := inputOptionalBool(input, "retry_failed")
		if err != nil {
			return err
		}
		service, err := h.media.ImageDerivatives()
		if err != nil {
			return err
		}
		var state *dto.ImageDerivatives
		if result.Operation == hostcontract.OperationImageDerivativesStatus {
			state, err = service.GetImageDerivatives(ctx, asset)
		} else {
			state, err = service.EnsureImageDerivatives(ctx, asset, dto.EnsureImageDerivativesInput{RetryFailed: retry})
		}
		if err != nil {
			return err
		}
		result.Result = map[string]any{"derivatives": state}
		return nil
	}
	fields := make(map[string]any, len(input))
	for k, v := range input {
		fields[k] = v
	}
	var asset, variant string
	if result.Operation != "asset.create" && result.Operation != "variant.get" {
		asset, err = inputUUID(input, "asset_uuid")
		if err != nil {
			return err
		}
		delete(fields, "asset_uuid")
	}
	if result.Operation == "variant.get" || result.Operation == "variant.presign_upload" || result.Operation == "variant.presign_download" || result.Operation == "variant.complete_upload" {
		variant, err = inputUUID(input, "variant_uuid")
		if err != nil {
			return err
		}
		delete(fields, "variant_uuid")
	}
	var output any
	switch result.Operation {
	case "asset.create":
		var v dto.CreateAssetInput
		if err = decodeStorage(fields, &v); err == nil {
			output, err = s.CreateAsset(ctx, v)
		}
	case "asset.update":
		var v dto.UpdateAssetInput
		if err = decodeStorage(fields, &v); err == nil {
			output, err = s.UpdateAsset(ctx, asset, v)
		}
	case "variant.create":
		var v dto.CreateVariantInput
		if err = decodeStorage(fields, &v); err == nil {
			output, err = s.CreateVariant(ctx, asset, v)
		}
	case "asset.complete_upload", "variant.complete_upload":
		var v dto.CompleteUploadInput
		if err = decodeStorage(fields, &v); err == nil {
			if variant != "" {
				output, err = s.CompleteVariantUpload(ctx, asset, variant, v)
			} else {
				output, err = s.CompleteUpload(ctx, asset, v)
			}
		}
	case "variant.presign_upload", "variant.presign_download":
		var v dto.VariantTicketInput
		if err = decodeStorage(fields, &v); err == nil {
			if result.Operation == "variant.presign_upload" {
				output, err = s.PresignVariantUpload(ctx, asset, variant, v)
			} else {
				output, err = s.PresignVariantDownload(ctx, asset, variant, v)
			}
		}
	default:
		if err = ensureInputKeys(fields); err != nil {
			return err
		}
		switch result.Operation {
		case "asset.get":
			output, err = s.GetAsset(ctx, asset)
		case "asset.delete":
			err = s.DeleteAsset(ctx, asset)
		case "asset.presign_upload":
			output, err = s.PresignUpload(ctx, asset)
		case "asset.presign_download":
			output, err = s.PresignDownload(ctx, asset)
		case "variant.get":
			output, err = s.GetVariant(ctx, variant)
		default:
			return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedOperation}
		}
	}
	if err == nil {
		result.Result = map[string]any{"output": output}
	}
	return err
}
