package publisher

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"gopkg.in/yaml.v3"
)

// These publication limits are separate from source snapshot/copy budgets.
// Market P01 defines them independently and must also enforce them server-side.
const (
	MaxPackageBytes             int64  = 2 << 30
	MaxPackageManifestBytes     int64  = 256 << 10
	MaxPackageUncompressedBytes uint64 = 8 << 30
	MaxPackageEntries                  = 50000
	MaxPackagePathBytes                = 1024
	MaxPackageCompressionRatio  uint64 = 200
)

type packageOutput struct {
	writer io.Writer
	wrote  int64
}

func (w *packageOutput) Write(data []byte) (int, error) {
	if int64(len(data)) > MaxPackageBytes-w.wrote {
		return 0, failure(TooLarge, nil)
	}
	n, err := w.writer.Write(data)
	w.wrote += int64(n)
	return n, err
}

func decodeManifest(data []byte) (map[string]any, error) {
	if int64(len(data)) > MaxPackageManifestBytes {
		return nil, failure(TooLarge, nil)
	}
	if !utf8.Valid(data) {
		return nil, failure(ArtifactInvalid, nil)
	}
	d := yaml.NewDecoder(bytes.NewReader(data))
	var doc, extra yaml.Node
	if d.Decode(&doc) != nil || d.Decode(&extra) != io.EOF || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, failure(ArtifactInvalid, nil)
	}
	// Inspect syntax before decoding: ordinary YAML decoding resolves aliases
	// and silently discards custom tags. Neither is a publication manifest.
	pending := []*yaml.Node{doc.Content[0]}
	for len(pending) > 0 {
		n := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if n.Kind == yaml.AliasNode {
			return nil, failure(ArtifactInvalid, nil)
		}
		switch n.ShortTag() {
		case "!!map", "!!seq", "!!str", "!!bool", "!!int", "!!float", "!!null", "!!timestamp", "!!binary", "!!merge", "!!set", "!!omap", "!!pairs":
		default:
			return nil, failure(ArtifactInvalid, nil)
		}
		if n.Kind == yaml.MappingNode {
			keys := map[string]bool{}
			for i := 0; i < len(n.Content); i += 2 {
				key := n.Content[i]
				if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" || keys[key.Value] {
					return nil, failure(ArtifactInvalid, nil)
				}
				keys[key.Value] = true
			}
		}
		pending = append(pending, n.Content...)
	}
	var result map[string]any
	if doc.Decode(&result) != nil {
		return nil, failure(ArtifactInvalid, nil)
	}
	return result, nil
}

func readManifest(file string) (map[string]any, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, failure(ArtifactInvalid, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxPackageManifestBytes+1))
	if err != nil {
		return nil, failure(ArtifactInvalid, err)
	}
	return decodeManifest(data)
}

type packagePath struct {
	name string
	dir  bool
}
type packagePaths map[string]packagePath

func (paths packagePaths) add(name string, dir bool) error {
	if len(name) > MaxPackagePathBytes {
		return failure(TooLarge, nil)
	}
	if name == "" || !utf8.ValidString(name) || strings.ContainsAny(name, "\\\x00:%") || strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") != dir {
		return failure(ArtifactInvalid, nil)
	}
	parts := strings.Split(strings.TrimSuffix(name, "/"), "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return failure(ArtifactInvalid, nil)
		}
	}
	name = strings.TrimSuffix(name, "/")
	// NUL is forbidden in input. As an internal component separator it sorts
	// before every allowed path byte, keeping a parent next to its children
	// even when names such as "a-b" would otherwise sort between "a" and "a/b".
	key := strings.ReplaceAll(cases.Fold().String(name), "/", "\x00")
	if _, exists := paths[key]; exists {
		return failure(ArtifactInvalid, nil)
	}
	paths[key] = packagePath{name: name, dir: dir}
	return nil
}

// Folded lexical order groups every implicit parent contiguously. Comparing
// neighbors detects parent collisions without retaining every full prefix of
// every path (which would multiply memory use for deeply nested archives).
func (paths packagePaths) check() error {
	keys := make([]string, 0, len(paths))
	for key := range paths {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for i := 1; i < len(keys); i++ {
		previous, current := paths[keys[i-1]], paths[keys[i]]
		if !previous.dir && strings.HasPrefix(keys[i], keys[i-1]+"\x00") {
			return failure(ArtifactInvalid, nil)
		}
		left, right := strings.Split(keys[i-1], "\x00"), strings.Split(keys[i], "\x00")
		leftNames, rightNames := strings.Split(previous.name, "/"), strings.Split(current.name, "/")
		for j := 0; j < len(left) && j < len(right) && left[j] == right[j]; j++ {
			if leftNames[j] != rightNames[j] {
				return failure(ArtifactInvalid, nil)
			}
		}
	}
	return nil
}

// Validate the bytes actually being frozen, without extracting or executing
// them. The same reader is used for directory checks and bounded CRC reads.
func validatePackageZIP(ctx context.Context, input io.ReaderAt, size int64, p Plan) error {
	if size <= 0 || size > MaxPackageBytes {
		return failure(TooLarge, nil)
	}
	r, err := zip.NewReader(input, size)
	if err != nil {
		return failure(ArtifactInvalid, err)
	}
	if len(r.File) > MaxPackageEntries {
		return failure(TooLarge, nil)
	}
	paths := packagePaths{}
	var declared uint64
	var manifest *zip.File
	root := p.Version + "/"
	for _, f := range r.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		mode := f.Mode()
		if mode.Type() != 0 && mode.Type() != os.ModeDir || mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 || f.Flags&0x2041 != 0 {
			return failure(ArtifactInvalid, nil)
		}
		if err := paths.add(f.Name, mode.IsDir()); err != nil {
			return err
		}
		if !strings.HasPrefix(f.Name, root) || f.CompressedSize64 > uint64(size) || mode.IsDir() && (f.UncompressedSize64 != 0 || f.CompressedSize64 != 0 || f.CRC32 != 0) {
			return failure(ArtifactInvalid, nil)
		}
		if f.UncompressedSize64 > MaxPackageUncompressedBytes-declared {
			return failure(TooLarge, nil)
		}
		declared += f.UncompressedSize64
		if cases.Fold().String(path.Base(strings.TrimSuffix(f.Name, "/"))) == "plugin.yaml" {
			if f.Name != root+"plugin.yaml" || mode.IsDir() || manifest != nil {
				return failure(ArtifactInvalid, nil)
			}
			if f.UncompressedSize64 > uint64(MaxPackageManifestBytes) {
				return failure(TooLarge, nil)
			}
			manifest = f
		}
	}
	if err := paths.check(); err != nil {
		return err
	}
	// The denominator is the actual complete compressed ZIP byte size.
	if declared > uint64(size)*MaxPackageCompressionRatio {
		return failure(TooLarge, nil)
	}
	if manifest == nil {
		return failure(ArtifactInvalid, nil)
	}
	for _, f := range r.File {
		if f.Mode().IsDir() {
			continue
		}
		reader, err := f.Open()
		if err != nil {
			return failure(ArtifactInvalid, err)
		}
		var data bytes.Buffer
		var out io.Writer = io.Discard
		if f == manifest {
			out = &data
		}
		// The extra byte detects size lies and permits the final EOF/CRC check.
		n, readErr := io.Copy(out, &contextReader{ctx: ctx, reader: io.LimitReader(reader, int64(f.UncompressedSize64)+1)})
		closeErr := reader.Close()
		if err := ctx.Err(); err != nil {
			return err
		}
		if readErr != nil || closeErr != nil || uint64(n) != f.UncompressedSize64 {
			return failure(ArtifactInvalid, readErr)
		}
		if f == manifest {
			doc, err := decodeManifest(data.Bytes())
			if err != nil {
				return err
			}
			target, ok := doc["target"].(map[string]any)
			if doc["id"] != p.PluginID || doc["version"] != p.Version || !ok || len(target) != 2 || target["os"] != p.Target.OS || target["arch"] != p.Target.Arch {
				return failure(ArtifactInvalid, nil)
			}
		}
	}
	return nil
}
