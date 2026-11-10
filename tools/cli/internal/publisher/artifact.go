package publisher

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"debug/elf"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func digestFile(file string) (int64, string, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, "", failure(ArtifactInvalid, err)
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return 0, "", failure(ArtifactInvalid, err)
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}
func safePackagePath(root, rel string) (string, error) {
	rel = filepath.FromSlash(rel)
	if strings.HasPrefix(rel, "./") {
		rel = strings.TrimPrefix(rel, "./")
	}
	if !filepath.IsLocal(rel) || strings.Contains(rel, "\\") {
		return "", failure(ArtifactInvalid, nil)
	}
	file := filepath.Join(root, rel)
	resolved, err := filepath.EvalSymlinks(file)
	if err != nil {
		return "", failure(ArtifactInvalid, err)
	}
	inside, err := filepath.Rel(root, resolved)
	if err != nil || !filepath.IsLocal(inside) {
		return "", failure(ArtifactInvalid, err)
	}
	return file, nil
}
func copyPackageTree(ctx context.Context, root, rel, dest string) error {
	w := treeWalk{ctx: ctx, root: root, destination: dest, stack: map[string]bool{}}
	if err := w.visit(rel); err != nil {
		return err
	}
	return nil
}
func validateGoBinary(file, version string) error {
	info, err := buildinfo.ReadFile(file)
	if err != nil {
		return failure(ArtifactInvalid, err)
	}
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	if settings["GOOS"] != "linux" || settings["GOARCH"] != "amd64" {
		return failure(ArtifactInvalid, nil)
	}
	if settings["CGO_ENABLED"] != "0" {
		return failure(NativeUnsupported, nil)
	}
	if settings["-trimpath"] != "true" {
		return failure(ArtifactInvalid, nil)
	}
	f, err := elf.Open(file)
	if err != nil {
		return failure(ArtifactInvalid, err)
	}
	defer f.Close()
	if f.Class != elf.ELFCLASS64 || f.Machine != elf.EM_X86_64 || f.OSABI != elf.ELFOSABI_NONE && f.OSABI != elf.ELFOSABI_LINUX {
		return failure(ArtifactInvalid, nil)
	}
	if f.Type != elf.ET_EXEC || f.Data != elf.ELFDATA2LSB {
		return failure(ArtifactInvalid, nil)
	}
	if version != "" {
		symbols, err := f.Symbols()
		if err != nil {
			return failure(ArtifactInvalid, err)
		}
		matched := false
		for _, symbol := range symbols {
			if symbol.Name != info.Main.Path+"/internal/shared/app.PluginVersion" {
				continue
			}
			if symbol.Size != 16 {
				return failure(ArtifactInvalid, nil)
			}
			header, err := elfBytes(f, symbol.Value, 16)
			if err != nil {
				return err
			}
			address, length := f.ByteOrder.Uint64(header[:8]), f.ByteOrder.Uint64(header[8:])
			if length != uint64(len(version)) {
				return failure(ArtifactInvalid, nil)
			}
			data, err := elfBytes(f, address, length)
			if err != nil {
				return err
			}
			if string(data) != version {
				return failure(ArtifactInvalid, nil)
			}
			matched = true
			break
		}
		if !matched {
			return failure(ArtifactInvalid, nil)
		}
	}
	stat, err := os.Stat(file)
	if err != nil || stat.Mode().Perm()&0111 == 0 {
		return failure(ArtifactInvalid, err)
	}
	return nil
}
func inspectPublicRuntime(root string) error {
	return filepath.WalkDir(root, func(file string, entry os.DirEntry, err error) error {
		if err != nil {
			return failure(ArtifactInvalid, err)
		}
		if entry.IsDir() {
			return nil
		}
		if excludedFile(entry.Name()) {
			return failure(ArtifactInvalid, nil)
		}
		if strings.HasSuffix(strings.ToLower(file), ".node") {
			return failure(NativeUnsupported, nil)
		}
		f, err := os.Open(file)
		if err != nil {
			return failure(ArtifactInvalid, err)
		}
		var b [4]byte
		n, _ := f.Read(b[:])
		f.Close()
		if n >= 2 && b[0] == 'M' && b[1] == 'Z' || n == 4 && (string(b[:]) == "\x7fELF" || string(b[:]) == "\xcf\xfa\xed\xfe" || string(b[:]) == "\xfe\xed\xfa\xcf" || string(b[:]) == "\xca\xfe\xba\xbe") {
			return failure(NativeUnsupported, nil)
		}
		return nil
	})
}
func rejectSecrets(node any) error {
	switch value := node.(type) {
	case map[string]any:
		for key, v := range value {
			upper := strings.ToUpper(key)
			s, scalar := v.(string)
			if scalar && s != "" {
				for _, part := range []string{"TOKEN", "SECRET", "PASSWORD", "API_KEY", "PRIVATE_KEY"} {
					if upper == part || strings.HasSuffix(upper, "_"+part) {
						return failure(ArtifactInvalid, nil)
					}
				}
			}
			if err := rejectSecrets(v); err != nil {
				return err
			}
		}
	case []any:
		for _, v := range value {
			if err := rejectSecrets(v); err != nil {
				return err
			}
		}
	case string:
		if strings.Contains(value, "PRIVATE KEY-----") {
			return failure(ArtifactInvalid, nil)
		}
		u, err := url.Parse(value)
		if err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.User != nil {
			return failure(ArtifactInvalid, nil)
		}
	}
	return nil
}
func decodeYAML(file string) (map[string]any, error) {
	info, err := os.Stat(file)
	if err != nil || info.Size() > 2<<20 {
		return nil, failure(ArtifactInvalid, err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, failure(ArtifactInvalid, err)
	}
	var result map[string]any
	if yaml.Unmarshal(data, &result) != nil {
		return nil, failure(ArtifactInvalid, nil)
	}
	return result, nil
}
func freezePackage(ctx context.Context, snapshot, dist, stage string, p Plan, tools BuildTools) (int, error) {
	if err := os.Mkdir(stage, 0700); err != nil {
		return 0, failure(StorageFailure, err)
	}
	for _, rel := range []string{"plugin.yaml", "backend/bin", "web-admin/.output"} {
		if err := copyPackageTree(ctx, dist, rel, stage); err != nil {
			return 0, err
		}
	}
	for _, rel := range []string{"plugin.d", "contracts", "skills", "config/event_fabric.yaml", "web-admin/i18n", "web-admin/public", "backend/etc/ai/providers.d", "README.md", "LICENSE"} {
		if _, err := os.Lstat(filepath.Join(snapshot, rel)); os.IsNotExist(err) {
			continue
		}
		if err := copyPackageTree(ctx, snapshot, rel, stage); err != nil {
			return 0, err
		}
	}
	doc, err := decodeYAML(filepath.Join(stage, "plugin.yaml"))
	if err != nil {
		return 0, err
	}
	if doc["id"] != p.PluginID || doc["version"] != p.Version {
		return 0, failure(ArtifactInvalid, nil)
	}
	target, ok := doc["target"].(map[string]any)
	if !ok || len(target) != 2 || target["os"] != p.Target.OS || target["arch"] != p.Target.Arch {
		return 0, failure(ArtifactInvalid, nil)
	}
	if err = rejectSecrets(doc); err != nil {
		return 0, err
	}
	runtime, ok := doc["runtime"].(map[string]any)
	if !ok || runtime["kind"] != "process" {
		return 0, failure(ArtifactInvalid, nil)
	}
	entry, ok := runtime["entry"].(string)
	if !ok || entry != "backend/bin/plugin" {
		return 0, failure(ArtifactInvalid, nil)
	}
	if err = validateGoBinary(filepath.Join(stage, entry), p.Version); err != nil {
		return 0, err
	}
	binaries, err := os.ReadDir(filepath.Join(stage, "backend/bin"))
	if err != nil {
		return 0, failure(ArtifactInvalid, err)
	}
	for _, binary := range binaries {
		if binary.Name() != "plugin" && binary.Name() != "migrate" {
			return 0, failure(ArtifactInvalid, nil)
		}
		version := p.Version
		if binary.Name() == "migrate" {
			version = ""
		}
		if err = validateGoBinary(filepath.Join(stage, "backend/bin", binary.Name()), version); err != nil {
			return 0, err
		}
	}
	if migrations, ok := doc["migrations"].(map[string]any); ok {
		entry, ok := migrations["entry"].(string)
		if !ok || entry != "backend/bin/migrate" {
			return 0, failure(ArtifactInvalid, nil)
		}
		if err = validateGoBinary(filepath.Join(stage, entry), ""); err != nil {
			return 0, err
		}
	}
	if _, err = os.Stat(filepath.Join(stage, "web-admin/.output/server/index.mjs")); err != nil {
		return 0, failure(ArtifactInvalid, err)
	}
	if info, err := os.Stat(filepath.Join(stage, "web-admin/.output/public")); err != nil || !info.IsDir() {
		return 0, failure(ArtifactInvalid, err)
	}
	if err = inspectPublicRuntime(filepath.Join(stage, "web-admin/.output")); err != nil {
		return 0, err
	}
	front, ok := doc["frontend"].(map[string]any)
	if !ok {
		return 0, failure(ArtifactInvalid, nil)
	}
	admin, ok := front["admin"].(map[string]any)
	if !ok || admin["kind"] != "process" {
		return 0, failure(ArtifactInvalid, nil)
	}
	process, ok := admin["process"].(map[string]any)
	if !ok || process["entry"] != "node" {
		return 0, failure(ArtifactInvalid, nil)
	}
	args, ok := process["args"].([]any)
	if !ok || len(args) != 1 || args[0] != "./web-admin/.output/server/index.mjs" {
		return 0, failure(ArtifactInvalid, nil)
	}
	if backend, ok := doc["backend"].(map[string]any); ok && backend["entry"] != entry {
		return 0, failure(ArtifactInvalid, nil)
	}
	if assets, ok := doc["assets"].(map[string]any); ok {
		for _, key := range []string{"public_dir", "webAdminPath"} {
			if value, ok := assets[key]; ok {
				rel, ok := value.(string)
				if !ok {
					return 0, failure(ArtifactInvalid, nil)
				}
				if _, err = safePackagePath(stage, rel); err != nil {
					return 0, err
				}
			}
		}
	}
	if catalogs, ok := doc["catalogs"].(map[string]any); ok {
		for _, value := range catalogs {
			rel, ok := value.(string)
			if !ok {
				return 0, failure(ArtifactInvalid, nil)
			}
			file, err := safePackagePath(stage, rel)
			if err != nil {
				return 0, err
			}
			if _, err = os.Stat(file); err != nil {
				return 0, failure(ArtifactInvalid, err)
			}
		}
	}
	meta := struct {
		Version string     `json:"version"`
		Target  Target     `json:"target"`
		Tools   BuildTools `json:"tools"`
		Profile string     `json:"profile"`
	}{p.Version, p.Target, tools, "make-dist-pure-go-js/v1"}
	data, _ := json.Marshal(meta)
	if err = os.WriteFile(filepath.Join(stage, "web-admin/.output/powerx-publisher.json"), append(data, '\n'), 0444); err != nil {
		return 0, failure(StorageFailure, err)
	}
	count := 0
	err = filepath.WalkDir(stage, func(file string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return failure(ArtifactInvalid, walkErr)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return failure(ArtifactInvalid, nil)
		}
		if entry.IsDir() {
			return nil
		}
		if strings.Contains(file, "\\") {
			return failure(ArtifactInvalid, nil)
		}
		if excludedFile(entry.Name()) {
			return failure(ArtifactInvalid, nil)
		}
		if entry.Name() == "plugin.yaml" && file != filepath.Join(stage, "plugin.yaml") {
			return failure(ArtifactInvalid, nil)
		}
		if strings.HasPrefix(file, filepath.Join(stage, "config")+string(os.PathSeparator)) || strings.HasPrefix(file, filepath.Join(stage, "backend/etc/ai/providers.d")+string(os.PathSeparator)) {
			doc, err := decodeYAML(file)
			if err != nil {
				return err
			}
			if err = rejectSecrets(doc); err != nil {
				return err
			}
		}
		if strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml") {
			doc, err := decodeYAML(file)
			if err != nil {
				return err
			}
			if err = checkReferences(stage, doc); err != nil {
				return err
			}
		}
		count++
		mode := os.FileMode(0444)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().Perm()&0111 != 0 {
			mode = 0555
		}
		return os.Chmod(file, mode)
	})
	return count, err
}
func checkReferences(root string, node any) error {
	switch value := node.(type) {
	case map[string]any:
		for _, v := range value {
			if err := checkReferences(root, v); err != nil {
				return err
			}
		}
	case []any:
		for _, v := range value {
			if err := checkReferences(root, v); err != nil {
				return err
			}
		}
	case string:
		if strings.HasPrefix(value, "contracts/") || strings.HasPrefix(value, "schema/") {
			rel := value
			if strings.HasPrefix(rel, "schema/") {
				rel = "contracts/" + rel
			}
			if _, err := safePackagePath(root, rel); err != nil {
				return err
			}
		}
	}
	return nil
}
func packZIP(ctx context.Context, stage, file, version string) error {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return failure(StorageFailure, err)
	}
	success := false
	defer func() {
		f.Close()
		if !success {
			os.Remove(file)
		}
	}()
	w := zip.NewWriter(f)
	paths := []string{}
	if err = filepath.WalkDir(stage, func(path string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !e.IsDir() {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return failure(ArtifactInvalid, err)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if ctx.Err() != nil {
			w.Close()
			return ctx.Err()
		}
		rel, err := filepath.Rel(stage, path)
		if err != nil || !filepath.IsLocal(rel) {
			w.Close()
			return failure(ArtifactInvalid, err)
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			w.Close()
			return failure(ArtifactInvalid, err)
		}
		header := &zip.FileHeader{Name: version + "/" + filepath.ToSlash(rel), Method: zip.Deflate}
		header.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		mode := os.FileMode(0644)
		if info.Mode().Perm()&0111 != 0 {
			mode = 0755
		}
		header.SetMode(mode)
		entry, err := w.CreateHeader(header)
		if err != nil {
			w.Close()
			return failure(ArtifactInvalid, err)
		}
		input, err := os.Open(path)
		if err != nil {
			w.Close()
			return failure(ArtifactInvalid, err)
		}
		_, copyErr := io.Copy(entry, &contextReader{ctx: ctx, reader: input})
		input.Close()
		if copyErr != nil {
			w.Close()
			return failure(ArtifactInvalid, copyErr)
		}
	}
	if err = w.Close(); err != nil {
		return failure(ArtifactInvalid, err)
	}
	if err = f.Sync(); err != nil {
		return failure(StorageFailure, err)
	}
	success = true
	return nil
}

func elfBytes(file *elf.File, address, length uint64) ([]byte, error) {
	for _, section := range file.Sections {
		if address < section.Addr || length > section.Size || address-section.Addr > section.Size-length {
			continue
		}
		r := section.Open()
		if _, err := r.Seek(int64(address-section.Addr), io.SeekStart); err != nil {
			return nil, failure(ArtifactInvalid, err)
		}
		data := make([]byte, int(length))
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, failure(ArtifactInvalid, err)
		}
		return data, nil
	}
	return nil, failure(ArtifactInvalid, nil)
}
