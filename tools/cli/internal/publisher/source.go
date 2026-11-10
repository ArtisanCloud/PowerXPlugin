package publisher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const MaxSnapshotBytes int64 = 8 << 30
const MaxSnapshotFiles = 200000

var makePath = regexp.MustCompile(`^[A-Za-z0-9/_.-]+$`)

func safeMakePath(s string) bool { return makePath.MatchString(s) }

var sourceRoots = []string{"Makefile", "plugin.yaml", "make-files", "scripts", "backend", "web-admin", "plugin.d", "contracts", "skills", "config/event_fabric.yaml", "README.md", "LICENSE"}
var excludedDirs = map[string]bool{".git": true, ".px-plugin": true, ".publisher-out": true, ".nuxt": true, ".output": true, ".cache": true, ".vite": true, ".vite-temp": true, "bin": true, "dist": true, "coverage": true, "test-results": true, "uploads": true, "logs": true, "__pycache__": true}

func excludedFile(name string) bool {
	lower := strings.ToLower(name)
	return lower == "config.yaml" || lower == "config.yml" || lower == ".npmrc" || lower == ".netrc" || lower == ".env" || strings.HasPrefix(lower, ".env.") || strings.HasSuffix(lower, ".db") || strings.HasSuffix(lower, ".sqlite") || strings.HasSuffix(lower, ".sqlite3") || strings.HasSuffix(lower, ".key") || strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".log") || strings.HasSuffix(lower, ".tsbuildinfo")
}
func validateDependencies(ctx context.Context, root string) error {
	backend := filepath.Join(root, "backend")
	if _, err := os.Stat(filepath.Join(backend, "go.mod")); os.IsNotExist(err) {
		backend = filepath.Join(backend, "go-gin")
	}
	cmd := exec.CommandContext(ctx, "go", "mod", "edit", "-json")
	cmd.Dir = backend
	cmd.Env = buildEnv()
	data, err := cmd.Output()
	if err != nil {
		return failure(SourceUnsafe, err)
	}
	var mod struct {
		Replace []struct {
			New struct{ Path, Version string }
		}
	}
	if json.Unmarshal(data, &mod) != nil {
		return failure(SourceUnsafe, nil)
	}
	for _, replace := range mod.Replace {
		if replace.New.Version == "" {
			return failure(SourceUnsafe, nil)
		}
	}
	frontend := filepath.Join(root, "web-admin")
	if _, err = os.Stat(filepath.Join(frontend, "package.json")); os.IsNotExist(err) {
		frontend = filepath.Join(frontend, "nuxt")
	}
	data, err = os.ReadFile(filepath.Join(frontend, "package.json"))
	if err != nil {
		return failure(SourceUnsafe, err)
	}
	var pkg map[string]json.RawMessage
	if json.Unmarshal(data, &pkg) != nil {
		return failure(SourceUnsafe, nil)
	}
	for _, key := range []string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"} {
		if data, ok := pkg[key]; ok {
			var deps map[string]string
			if json.Unmarshal(data, &deps) != nil {
				return failure(SourceUnsafe, nil)
			}
			for _, version := range deps {
				for _, prefix := range []string{"file:", "link:", "workspace:"} {
					if strings.HasPrefix(version, prefix) {
						return failure(SourceUnsafe, nil)
					}
				}
			}
		}
	}
	return nil
}

type treeWalk struct {
	ctx         context.Context
	root        string
	destination string
	hash        hash.Hash
	bytes       int64
	count       int
	stack       map[string]bool
	source      bool
}

func (w *treeWalk) visit(rel string) error {
	if strings.Contains(rel, "\\") {
		return failure(SourceUnsafe, nil)
	}
	if err := w.ctx.Err(); err != nil {
		return err
	}
	name := filepath.Join(w.root, rel)
	info, err := os.Lstat(name)
	if err != nil {
		return failure(SourceUnsafe, err)
	}
	if !w.source && (info.Name() == ".git" || info.Name() == ".px-plugin") {
		return failure(ArtifactInvalid, nil)
	}
	if w.source && (excludedFile(info.Name()) || info.IsDir() && excludedDirs[info.Name()] && !strings.Contains(filepath.ToSlash(rel), "node_modules/")) {
		return nil
	}
	resolved, err := filepath.EvalSymlinks(name)
	if err != nil {
		return failure(SourceUnsafe, err)
	}
	within, err := filepath.Rel(w.root, resolved)
	if err != nil || !filepath.IsLocal(within) {
		return failure(SourceUnsafe, err)
	}
	info, err = os.Stat(resolved)
	if err != nil {
		return failure(SourceUnsafe, err)
	}
	if info.IsDir() {
		if w.source && excludedDirs[info.Name()] && info.Name() != "bin" && info.Name() != "dist" {
			return nil
		}
		if w.stack[resolved] {
			return failure(SourceUnsafe, nil)
		}
		w.stack[resolved] = true
		defer delete(w.stack, resolved)
		if w.destination != "" {
			if err = os.MkdirAll(filepath.Join(w.destination, rel), 0700); err != nil {
				return failure(StorageFailure, err)
			}
		}
		entries, err := os.ReadDir(resolved)
		if err != nil {
			return failure(SourceUnsafe, err)
		}
		for _, entry := range entries {
			if err = w.visit(filepath.Join(rel, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return failure(SourceUnsafe, nil)
	}
	w.count++
	w.bytes += info.Size()
	if w.count > MaxSnapshotFiles || w.bytes > MaxSnapshotBytes {
		return failure(TooLarge, nil)
	}
	if w.source && excludedFile(info.Name()) {
		return nil
	}
	if w.hash != nil {
		fmt.Fprintf(w.hash, "%s\x00%d\x00%d\x00", filepath.ToSlash(rel), info.Mode().Perm()&0111, info.Size())
	}
	f, err := os.Open(resolved)
	if err != nil {
		return failure(SourceUnsafe, err)
	}
	defer f.Close()
	out := io.Writer(io.Discard)
	if w.hash != nil {
		out = w.hash
	}
	if w.destination != "" {
		target := filepath.Join(w.destination, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return failure(StorageFailure, err)
		}
		mode := os.FileMode(0600) | info.Mode().Perm()&0111
		dest, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			return failure(StorageFailure, err)
		}
		n, copyErr := io.Copy(io.MultiWriter(out, dest), &contextReader{ctx: w.ctx, reader: io.LimitReader(f, info.Size()+1)})
		closeErr := dest.Close()
		if copyErr != nil {
			return failure(StorageFailure, copyErr)
		}
		if n != info.Size() {
			return failure(SourceChanged, nil)
		}
		if closeErr != nil {
			return failure(StorageFailure, closeErr)
		}
	} else {
		n, copyErr := io.Copy(out, &contextReader{ctx: w.ctx, reader: io.LimitReader(f, info.Size()+1)})
		if copyErr != nil {
			return failure(SourceUnsafe, copyErr)
		}
		if n != info.Size() {
			return failure(SourceChanged, nil)
		}
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
func sourceDigest(ctx context.Context, root, dest string) (string, error) {
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", failure(SourceUnsafe, err)
	}
	root = canonical
	w := treeWalk{ctx: ctx, root: root, destination: dest, hash: sha256.New(), stack: map[string]bool{}, source: true}
	for _, rel := range sourceRoots {
		if _, err := os.Lstat(filepath.Join(root, rel)); os.IsNotExist(err) {
			continue
		}
		if err := w.visit(rel); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(w.hash.Sum(nil)), nil
}
func provenance(ctx context.Context, root string) (BuildSource, error) {
	found := false
	for p := root; ; p = filepath.Dir(p) {
		if _, err := os.Lstat(filepath.Join(p, ".git")); err == nil {
			found = true
			break
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	if !found {
		return BuildSource{VCS: "filesystem"}, nil
	}
	query := func(args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
		return cmd.Output()
	}
	commit, err := query("rev-parse", "HEAD")
	if err != nil {
		return BuildSource{}, failure(SourceUnsafe, err)
	}
	status, err := query("status", "--porcelain", "--untracked-files=all", "--", ".")
	if err != nil {
		return BuildSource{}, failure(SourceUnsafe, err)
	}
	dirty := len(status) > 0
	return BuildSource{VCS: "git", Commit: strings.TrimSpace(string(commit)), Dirty: &dirty}, nil
}
func snapshotSource(ctx context.Context, root, dest string) (BuildSource, error) {
	p, err := provenance(ctx, root)
	if err != nil {
		return BuildSource{}, err
	}
	if err = os.Mkdir(dest, 0700); err != nil {
		return BuildSource{}, failure(StorageFailure, err)
	}
	hash, err := sourceDigest(ctx, root, dest)
	if err != nil {
		return BuildSource{}, err
	}
	p.SHA256 = hash
	if err = unchangedSource(ctx, root, dest, p); err != nil {
		return BuildSource{}, err
	}
	return p, nil
}
func unchangedSource(ctx context.Context, root, snapshot string, want BuildSource) error {
	for _, dir := range []string{root, snapshot} {
		hash, err := sourceDigest(ctx, dir, "")
		if err != nil {
			return err
		}
		if hash != want.SHA256 {
			return failure(SourceChanged, nil)
		}
	}
	p, err := provenance(ctx, root)
	if err != nil {
		return err
	}
	if p.VCS != want.VCS || p.Commit != want.Commit || (p.Dirty == nil) != (want.Dirty == nil) {
		return failure(SourceChanged, nil)
	}
	return nil
}
