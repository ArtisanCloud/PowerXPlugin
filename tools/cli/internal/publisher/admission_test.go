package publisher

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type admissionEntry struct {
	name   string
	data   []byte
	mode   os.FileMode
	method uint16
}

func admissionPlan() Plan { return Plan{PluginID: "com.example.publisher", CreatePlan: input()} }
func admissionManifest() []byte {
	return []byte("id: com.example.publisher\nversion: 1.2.0\ntarget: {os: linux, arch: amd64}\n")
}
func admissionZIP(t *testing.T, entries ...admissionEntry) []byte {
	t.Helper()
	var data bytes.Buffer
	w := zip.NewWriter(&data)
	for _, e := range entries {
		h := &zip.FileHeader{Name: e.name, Method: e.method}
		if e.mode == 0 {
			h.SetMode(0644)
		} else {
			h.SetMode(e.mode)
		}
		entry, err := w.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = entry.Write(e.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}
func admissionCheck(data []byte) error {
	return validatePackageZIP(context.Background(), bytes.NewReader(data), int64(len(data)), admissionPlan())
}
func manifestEntry(data []byte) admissionEntry {
	return admissionEntry{name: "1.2.0/plugin.yaml", data: data}
}

func TestPackageManifestSyntaxAndByteBudget(t *testing.T) {
	base := string(admissionManifest())
	for _, tc := range []struct{ name, data, code string }{
		{"valid", base, ""},
		{"standard_tag", base + "title: !!str example\n", ""},
		{"duplicate_key", base + "id: com.example.other\n", ArtifactInvalid},
		{"nested_duplicate", base + "nested: {key: first, key: second}\n", ArtifactInvalid},
		{"alias", base + "first: &name example\nsecond: *name\n", ArtifactInvalid},
		{"custom_scalar_tag", base + "title: !application example\n", ArtifactInvalid},
		{"custom_mapping_tag", "!application\n" + base, ArtifactInvalid},
		{"custom_standard_namespace", base + "title: !!unknown example\n", ArtifactInvalid},
		{"second_document", base + "---\nname: second\n", ArtifactInvalid},
		{"empty_second_document", base + "---\n", ArtifactInvalid},
		{"non_mapping", "[one, two]\n", ArtifactInvalid},
		{"invalid_utf8", base + "title: \xff\n", ArtifactInvalid},
		{"nul", base + "title: \x00\n", ArtifactInvalid},
		{"byte_limit", base + "#" + strings.Repeat("a", int(MaxPackageManifestBytes)-len(base)-1), ""},
		{"byte_limit_exceeded", base + "#" + strings.Repeat("a", int(MaxPackageManifestBytes)-len(base)), TooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeManifest([]byte(tc.data))
			if tc.code == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				assertCode(t, err, tc.code)
			}
		})
	}
}

func TestPackageArchivePathsAndModes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		entries []admissionEntry
		code    string
	}{
		{"valid_unicode", []admissionEntry{{name: "1.2.0/公开/资源.txt", data: []byte("fixture")}}, ""},
		{"explicit_parent_after_child", []admissionEntry{{name: "1.2.0/assets/a"}, {name: "1.2.0/assets/", mode: os.ModeDir | 0755}}, ""},
		{"absolute", []admissionEntry{{name: "/1.2.0/a"}}, ArtifactInvalid},
		{"dot", []admissionEntry{{name: "1.2.0/./a"}}, ArtifactInvalid},
		{"dotdot", []admissionEntry{{name: "1.2.0/../a"}}, ArtifactInvalid},
		{"backslash", []admissionEntry{{name: "1.2.0/a\\b"}}, ArtifactInvalid},
		{"colon", []admissionEntry{{name: "1.2.0/a:b"}}, ArtifactInvalid},
		{"percent", []admissionEntry{{name: "1.2.0/a%20b"}}, ArtifactInvalid},
		{"nul", []admissionEntry{{name: "1.2.0/a\x00b"}}, ArtifactInvalid},
		{"invalid_utf8", []admissionEntry{{name: "1.2.0/\xff"}}, ArtifactInvalid},
		{"empty_segment", []admissionEntry{{name: "1.2.0/a//b"}}, ArtifactInvalid},
		{"outside_wrapper", []admissionEntry{{name: "other/a"}}, ArtifactInvalid},
		{"duplicate", []admissionEntry{{name: "1.2.0/a"}, {name: "1.2.0/a"}}, ArtifactInvalid},
		{"case_collision", []admissionEntry{{name: "1.2.0/a"}, {name: "1.2.0/A"}}, ArtifactInvalid},
		{"unicode_case_collision", []admissionEntry{{name: "1.2.0/straße"}, {name: "1.2.0/STRASSE"}}, ArtifactInvalid},
		{"implicit_parent_case_collision", []admissionEntry{{name: "1.2.0/assets/a"}, {name: "1.2.0/Assets/b"}}, ArtifactInvalid},
		{"unordered_implicit_parent_case_collision", []admissionEntry{{name: "1.2.0/assets/a"}, {name: "1.2.0/unrelated/b"}, {name: "1.2.0/Assets/c"}}, ArtifactInvalid},
		{"parent_casefold_expansion", []admissionEntry{{name: "1.2.0/İ/a"}, {name: "1.2.0/i\u0307/b"}}, ArtifactInvalid},
		{"file_before_directory", []admissionEntry{{name: "1.2.0/a"}, {name: "1.2.0/a/b"}}, ArtifactInvalid},
		{"interleaved_file_directory_collision", []admissionEntry{{name: "1.2.0/a"}, {name: "1.2.0/a-b"}, {name: "1.2.0/a/b"}}, ArtifactInvalid},
		{"interleaved_parent_case_collision", []admissionEntry{{name: "1.2.0/Assets/", mode: os.ModeDir | 0755}, {name: "1.2.0/assets-b"}, {name: "1.2.0/assets/a"}}, ArtifactInvalid},
		{"interleaved_distinct_names", []admissionEntry{{name: "1.2.0/a/", mode: os.ModeDir | 0755}, {name: "1.2.0/a-b"}, {name: "1.2.0/a/b"}}, ""},
		{"directory_before_file", []admissionEntry{{name: "1.2.0/a/b"}, {name: "1.2.0/a"}}, ArtifactInvalid},
		{"duplicate_directory", []admissionEntry{{name: "1.2.0/a/", mode: os.ModeDir | 0755}, {name: "1.2.0/a/", mode: os.ModeDir | 0755}}, ArtifactInvalid},
		{"nested_manifest", []admissionEntry{{name: "1.2.0/nested/plugin.yaml", data: admissionManifest()}}, ArtifactInvalid},
		{"nested_manifest_case", []admissionEntry{{name: "1.2.0/nested/PLUGIN.YAML", data: admissionManifest()}}, ArtifactInvalid},
		{"symlink", []admissionEntry{{name: "1.2.0/link", mode: os.ModeSymlink | 0777, data: []byte("plugin.yaml")}}, ArtifactInvalid},
		{"device", []admissionEntry{{name: "1.2.0/device", mode: os.ModeDevice | 0600}}, ArtifactInvalid},
		{"setuid", []admissionEntry{{name: "1.2.0/executable", mode: os.ModeSetuid | 0755}}, ArtifactInvalid},
		{"setgid", []admissionEntry{{name: "1.2.0/executable", mode: os.ModeSetgid | 0755}}, ArtifactInvalid},
		{"path_limit", []admissionEntry{{name: "1.2.0/" + strings.Repeat("d/", 508) + "aa"}}, ""},
		{"path_limit_exceeded", []admissionEntry{{name: "1.2.0/" + strings.Repeat("d/", 508) + "aaa"}}, TooLarge},
		{"multibyte_path_limit_exceeded", []admissionEntry{{name: "1.2.0/" + strings.Repeat("界", 340)}}, TooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entries := append([]admissionEntry{manifestEntry(admissionManifest())}, tc.entries...)
			err := admissionCheck(admissionZIP(t, entries...))
			if tc.code == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				assertCode(t, err, tc.code)
			}
		})
	}
}

func TestPackageArchiveEntryCountBoundary(t *testing.T) {
	entries := []admissionEntry{manifestEntry(admissionManifest())}
	for i := 1; i < MaxPackageEntries; i++ {
		entries = append(entries, admissionEntry{name: fmt.Sprintf("1.2.0/assets/%d", i)})
	}
	if err := admissionCheck(admissionZIP(t, entries...)); err != nil {
		t.Fatal(err)
	}
	entries = append(entries, admissionEntry{name: "1.2.0/extra/", mode: os.ModeDir | 0755})
	assertCode(t, admissionCheck(admissionZIP(t, entries...)), TooLarge)
}

func TestPackageArchiveManifestIdentityAndBudget(t *testing.T) {
	for _, tc := range []struct{ name, data, code string }{
		{"wrong_id", strings.ReplaceAll(string(admissionManifest()), "com.example.publisher", "com.example.other"), ArtifactInvalid},
		{"wrong_version", strings.ReplaceAll(string(admissionManifest()), "1.2.0", "1.2.1"), ArtifactInvalid},
		{"wrong_target", strings.ReplaceAll(string(admissionManifest()), "amd64", "arm64"), ArtifactInvalid},
		{"missing_target", "id: com.example.publisher\nversion: 1.2.0\n", ArtifactInvalid},
		{"oversized_manifest", string(admissionManifest()) + "#" + strings.Repeat("a", int(MaxPackageManifestBytes)), TooLarge},
		{"alias", string(admissionManifest()) + "first: &value example\nsecond: *value\n", ArtifactInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertCode(t, admissionCheck(admissionZIP(t, manifestEntry([]byte(tc.data)))), tc.code)
		})
	}
	assertCode(t, admissionCheck(admissionZIP(t, admissionEntry{name: "1.2.0/readme"})), ArtifactInvalid)
}

func TestPackageArchiveCRCAndActualSize(t *testing.T) {
	data := admissionZIP(t, manifestEntry(admissionManifest()), admissionEntry{name: "1.2.0/payload", data: []byte("fixture")})
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	offset, err := r.File[1].DataOffset()
	if err != nil {
		t.Fatal(err)
	}
	data[offset] ^= 1
	assertCode(t, admissionCheck(data), ArtifactInvalid)
	for _, declared := range []uint64{2, 20, MaxPackageUncompressedBytes + 1, ^uint64(0)} {
		t.Run(fmt.Sprint(declared), func(t *testing.T) {
			var data bytes.Buffer
			w := zip.NewWriter(&data)
			for _, e := range []struct {
				name string
				data []byte
				size uint64
			}{{"1.2.0/plugin.yaml", admissionManifest(), uint64(len(admissionManifest()))}, {"1.2.0/payload", []byte("fixture"), declared}} {
				entry, err := w.CreateRaw(&zip.FileHeader{Name: e.name, Method: zip.Store, CRC32: crc32.ChecksumIEEE(e.data), CompressedSize64: uint64(len(e.data)), UncompressedSize64: e.size})
				if err != nil {
					t.Fatal(err)
				}
				if _, err = entry.Write(e.data); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			code := ArtifactInvalid
			if declared > MaxPackageUncompressedBytes {
				code = TooLarge
			}
			assertCode(t, admissionCheck(data.Bytes()), code)
		})
	}
}

func TestPackageArchiveRejectsEncryptedAndNonemptyDirectoryMetadata(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header zip.FileHeader
	}{
		{"encrypted", zip.FileHeader{Name: "1.2.0/payload", Flags: 1}},
		{"strong_encrypted", zip.FileHeader{Name: "1.2.0/payload", Flags: 0x40}},
		{"masked_headers", zip.FileHeader{Name: "1.2.0/payload", Flags: 0x2000}},
		{"directory_compressed_payload", zip.FileHeader{Name: "1.2.0/assets/", CompressedSize64: 1}},
		{"directory_crc", zip.FileHeader{Name: "1.2.0/assets/", CRC32: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var data bytes.Buffer
			w := zip.NewWriter(&data)
			entry, err := w.CreateRaw(&tc.header)
			if err != nil {
				t.Fatal(err)
			}
			if tc.header.CompressedSize64 > 0 && !strings.HasSuffix(tc.header.Name, "/") {
				if _, err = entry.Write([]byte{'x'}); err != nil {
					t.Fatal(err)
				}
			}
			manifest := admissionManifest()
			entry, err = w.CreateHeader(&zip.FileHeader{Name: "1.2.0/plugin.yaml"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = entry.Write(manifest); err != nil {
				t.Fatal(err)
			}
			if err = w.Close(); err != nil {
				t.Fatal(err)
			}
			assertCode(t, admissionCheck(data.Bytes()), ArtifactInvalid)
		})
	}
}

func TestPackageArchiveCompressionAndCompressedSizeBudget(t *testing.T) {
	data := admissionZIP(t, manifestEntry(admissionManifest()), admissionEntry{name: "1.2.0/payload", data: bytes.Repeat([]byte{'a'}, 1<<20), method: zip.Deflate})
	assertCode(t, admissionCheck(data), TooLarge)
	assertCode(t, validatePackageZIP(context.Background(), bytes.NewReader(data), MaxPackageBytes+1, admissionPlan()), TooLarge)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := validatePackageZIP(ctx, bytes.NewReader(data), int64(len(data)), admissionPlan()); err != context.Canceled {
		t.Fatalf("cancellation=%v", err)
	}
}

func TestPackageArchiveCompressionRatioBoundary(t *testing.T) {
	const total = 1000000
	manifest := admissionManifest()
	data := admissionZIP(t, manifestEntry(manifest), admissionEntry{name: "1.2.0/payload", data: bytes.Repeat([]byte{'a'}, total-len(manifest)), method: zip.Deflate})
	// A valid ZIP comment changes compressed file bytes, not unpacked bytes.
	// Exercise exactly 200, and the first byte beyond that ratio.
	for _, size := range []int{total / 200, total/200 - 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			padding := size - len(data)
			if padding < 0 || padding > 65535 {
				t.Fatal("invalid_comment_fixture")
			}
			archive := append([]byte{}, data...)
			binary.LittleEndian.PutUint16(archive[len(archive)-2:], uint16(padding))
			archive = append(archive, bytes.Repeat([]byte{'x'}, padding)...)
			err := admissionCheck(archive)
			if size == total/200 {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				assertCode(t, err, TooLarge)
			}
		})
	}
}

func TestPackageOutputStopsAtCompressedBudget(t *testing.T) {
	w := &packageOutput{writer: io.Discard, wrote: MaxPackageBytes - 1}
	if n, err := w.Write([]byte{'a'}); n != 1 || err != nil {
		t.Fatalf("boundary_write=%d error=%v", n, err)
	}
	n, err := w.Write([]byte{'a'})
	assertCode(t, err, TooLarge)
	if n != 0 || w.wrote != MaxPackageBytes {
		t.Fatal("compressed_budget_exceeded")
	}
}

func TestPackingRemovesRejectedArchive(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload []byte
		code    string
	}{
		{"path", []byte("fixture"), ArtifactInvalid},
		{"ratio", bytes.Repeat([]byte{'a'}, 1<<20), TooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stage := t.TempDir()
			if err := os.WriteFile(filepath.Join(stage, "plugin.yaml"), admissionManifest(), 0600); err != nil {
				t.Fatal(err)
			}
			name := "payload"
			if tc.name == "path" {
				name = "payload%20"
			}
			if err := os.WriteFile(filepath.Join(stage, name), tc.payload, 0600); err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(t.TempDir(), "artifact.zip")
			assertCode(t, packZIP(context.Background(), stage, file, admissionPlan()), tc.code)
			if _, err := os.Stat(file); !os.IsNotExist(err) {
				t.Fatal("rejected_archive_retained")
			}
		})
	}
}

func TestVerifyArtifactRechecksAdmissionForMatchingStoredDigest(t *testing.T) {
	e := openTest(t, workspace(t))
	p, err := e.Create(input())
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	dir := filepath.Join(e.work.path, id)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "artifact.zip")
	data := admissionZIP(t, manifestEntry(admissionManifest()), admissionEntry{name: "1.2.0/asset%20"})
	if err := os.WriteFile(file, data, 0400); err != nil {
		t.Fatal(err)
	}
	size, sha, err := digestFile(file)
	if err != nil {
		t.Fatal(err)
	}
	b := Build{Schema: BuildSchema, BuildID: id, ReleasePlanID: p.ReleasePlanID, PluginID: p.PluginID, Version: p.Version, Target: p.Target, Stage: "ready", CreatedAt: time.Now().UTC(), Source: &BuildSource{SHA256: strings.Repeat("a", 64), VCS: "filesystem"}, Artifact: &Artifact{Path: "work/" + id + "/artifact.zip", SHA256: sha, Size: size, FileCount: 2}}
	if err := e.saveBuild(&b, false); err != nil {
		t.Fatal(err)
	}
	_, err = e.VerifyArtifact(id)
	assertCode(t, err, ArtifactInvalid)
	current, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(current, data) {
		t.Fatal("verification_changed_rejected_archive")
	}
}
