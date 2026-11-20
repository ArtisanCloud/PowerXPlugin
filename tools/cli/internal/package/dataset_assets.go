package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/powerx-plugin/cli/internal/manifest"
)

var datasetKey = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)

func packageDatasetAssets(entry string, result *Result, index map[string]manifest.DatasetAsset) ([]Artifact, error) {
	if len(index) == 0 {
		return nil, nil
	}
	root, err := os.OpenRoot(entry)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	keys := make([]string, 0, len(index))
	for key := range index {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	artifacts := make([]Artifact, 0, len(keys))
	for _, key := range keys {
		if !datasetKey.MatchString(key) {
			return nil, fmt.Errorf("DATASET_ASSET_KEY_INVALID: %q", key)
		}
		name := "contracts/storage-datasets/" + key + ".json"
		raw, err := readDatasetAsset(root, name)
		if err != nil {
			return nil, fmt.Errorf("DATASET_ASSET_INVALID %s: %w", name, err)
		}
		sum := sha256.Sum256(raw)
		hash := hex.EncodeToString(sum[:])
		if hash != index[key].SHA256 {
			return nil, fmt.Errorf("DATASET_ASSET_DIGEST_MISMATCH: %s", name)
		}

		target := filepath.Join(result.PayloadDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(target, raw, 0o644); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, Artifact{Name: name, Path: rel(target, result.BuildDir), Size: int64(len(raw)), Hash: hash})
	}
	return artifacts, nil
}

func readDatasetAsset(root *os.Root, name string) ([]byte, error) {
	const maxBytes = 16 << 10
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maxBytes {
		return nil, fmt.Errorf("DATASET_ASSET_NOT_REGULAR_OR_SIZE_INVALID")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err == nil && (len(raw) == 0 || len(raw) > maxBytes) {
		return nil, fmt.Errorf("DATASET_ASSET_SIZE_INVALID")
	}
	return raw, err
}
