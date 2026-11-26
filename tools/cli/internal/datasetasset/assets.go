package datasetasset

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

const Directory = "contracts/storage-datasets"

type Index map[string]string

type InvalidDeclaration struct {
	Code string
	Path string
	Err  error
}

func (e *InvalidDeclaration) Error() string {
	message := e.Code
	if e.Path != "" {
		message += ": " + e.Path
	}
	if e.Err != nil {
		message += ": " + e.Err.Error()
	}
	return message
}

func (e *InvalidDeclaration) Unwrap() error { return e.Err }

// DecodeIndex reads only the data_lifecycle fields needed to package declarations.
func DecodeIndex(raw any) (Index, error) {
	if raw == nil {
		return nil, nil
	}
	entries, ok := raw.(map[string]any)
	if !ok {
		return nil, &InvalidDeclaration{Code: "DATASET_ASSET_INDEX_INVALID"}
	}
	index := make(Index, len(entries))
	for key, entry := range entries {
		fields, ok := entry.(map[string]any)
		if !ok {
			return nil, &InvalidDeclaration{Code: "DATASET_ASSET_INDEX_INVALID", Path: fmt.Sprintf("%q", key)}
		}
		sum, _ := fields["sha256"].(string)
		index[key] = sum
	}
	return index, nil
}

// File records the validated original bytes copied into the package.
type File struct {
	Path   string
	Size   int64
	SHA256 string
}

var datasetKey = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)

// Copy validates and copies only the declarations referenced by the package manifest.
func Copy(entry, destination string, index Index) ([]File, error) {
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

	files := make([]File, 0, len(keys))
	for _, key := range keys {
		if !datasetKey.MatchString(key) {
			return nil, &InvalidDeclaration{Code: "DATASET_ASSET_KEY_INVALID", Path: fmt.Sprintf("%q", key)}
		}
		name := Directory + "/" + key + ".json"
		raw, err := readDatasetAsset(root, name)
		if err != nil {
			return nil, &InvalidDeclaration{Code: "DATASET_ASSET_INVALID", Path: name, Err: err}
		}
		sum := sha256.Sum256(raw)
		hash := hex.EncodeToString(sum[:])
		if hash != index[key] {
			return nil, &InvalidDeclaration{Code: "DATASET_ASSET_DIGEST_MISMATCH", Path: name}
		}

		target := filepath.Join(destination, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(target, raw, 0o644); err != nil {
			return nil, err
		}
		files = append(files, File{Path: name, Size: int64(len(raw)), SHA256: hash})
	}
	return files, nil
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
