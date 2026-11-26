//go:build darwin || linux

package publisher

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

// Image acquisition runs on the host, so the existing development network
// proxy also covers registry requests. Only public official images are loaded.
func prepareContainerImages(ctx context.Context, work string, args func(...string) []string, log io.Writer) error {
	for _, image := range []string{"docker.io/library/golang:1.26-alpine", "docker.io/library/node:22-bookworm-slim"} {
		data, err := exec.CommandContext(ctx, "docker", args("image", "inspect", image, "--format", "{{.Os}}/{{.Architecture}}")...).Output()
		if err == nil && strings.TrimSpace(string(data)) == "linux/amd64" {
			continue
		}
		reference, err := name.NewTag(image, name.StrictValidation)
		if err != nil {
			return failure(ContainerBuildFailed, err)
		}
		remoteImage, err := remote.Image(reference, remote.WithContext(ctx), remote.WithAuth(authn.Anonymous), remote.WithPlatform(v1.Platform{OS: "linux", Architecture: "amd64"}))
		if err != nil {
			return failure(ContainerBuildFailed, err)
		}
		archive := filepath.Join(work, "image-"+filepath.Base(strings.Split(image, ":")[0])+".tar")
		defer os.Remove(archive)
		if err = tarball.WriteToFile(archive, reference, remoteImage); err != nil {
			return failure(ContainerBuildFailed, err)
		}
		_ = os.Chmod(archive, 0600)
		err = runOwnedCommand(ctx, "docker", work, args("load", "--input", archive), nil, log, func(int) error { return nil })
		_ = os.Remove(archive)
		if err != nil {
			return failure(ContainerBuildFailed, err)
		}
	}
	return nil
}

// Forward an already configured, credential-free development proxy. A host
// loopback proxy is addressed through Docker's explicit host gateway.
func containerProxy() ([]string, error) {
	var result []string
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"} {
		value := os.Getenv(key)
		if value == "" {
			value = os.Getenv(strings.ToLower(key))
		}
		if value == "" {
			continue
		}
		if key != "NO_PROXY" {
			proxy, err := url.Parse(value)
			if err != nil || (proxy.Scheme != "http" && proxy.Scheme != "https") || proxy.Hostname() == "" || proxy.User != nil || proxy.RawQuery != "" || proxy.Fragment != "" || (proxy.Path != "" && proxy.Path != "/") {
				return nil, failure(ContainerUnavailable, nil)
			}
			if proxy.Hostname() == "127.0.0.1" || proxy.Hostname() == "localhost" || proxy.Hostname() == "::1" {
				host := "host.docker.internal"
				if proxy.Port() != "" {
					host += ":" + proxy.Port()
				}
				proxy.Host = host
			}
			value = proxy.String()
		}
		result = append(result, key+"="+value, strings.ToLower(key)+"="+value)
	}
	return result, nil
}

// Copy only download-cache objects named by the source modules' locked
// requirements. No home directory, VCS cache or credentials are mounted.
func prepareModuleProxy(ctx context.Context, root, run string, sdks []localSDK) error {
	cache, err := exec.CommandContext(ctx, "go", "env", "GOMODCACHE").Output()
	if err != nil {
		return failure(SourceUnsafe, err)
	}
	download := filepath.Join(strings.TrimSpace(string(cache)), "cache/download")
	backend := filepath.Join(root, "backend/go.mod")
	if _, err = os.Stat(backend); os.IsNotExist(err) {
		backend = filepath.Join(root, "backend/go-gin/go.mod")
	}
	files := []string{backend}
	for _, sdk := range sdks {
		if filepath.Base(sdk.destination) == "go" {
			files = append(files, filepath.Join(sdk.source, "go.mod"))
		}
	}
	var copied int64
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			return failure(SourceUnsafe, err)
		}
		mod, err := modfile.Parse(file, raw, nil)
		if err != nil {
			return failure(SourceUnsafe, err)
		}
		for _, require := range mod.Require {
			path, err := module.EscapePath(require.Mod.Path)
			if err != nil {
				return failure(SourceUnsafe, err)
			}
			version, err := module.EscapeVersion(require.Mod.Version)
			if err != nil {
				return failure(SourceUnsafe, err)
			}
			for _, suffix := range []string{".mod", ".info", ".zip", ".ziphash"} {
				relative := filepath.Join(path, "@v", version+suffix)
				source := filepath.Join(download, relative)
				info, err := os.Lstat(source)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil || !info.Mode().IsRegular() || info.Size() > 512<<20 {
					return failure(SourceUnsafe, err)
				}
				destination := filepath.Join(run, ".publisher-dependencies/module-proxy", relative)
				if _, err = os.Lstat(destination); err == nil {
					continue
				}
				copied += info.Size()
				if copied > 2<<30 {
					return failure(TooLarge, nil)
				}
				if err = os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
					return failure(StorageFailure, err)
				}
				from, err := os.Open(source)
				if err != nil {
					return failure(SourceUnsafe, err)
				}
				to, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if err != nil {
					from.Close()
					return failure(StorageFailure, err)
				}
				n, copyErr := io.Copy(to, &contextReader{ctx: ctx, reader: io.LimitReader(from, info.Size()+1)})
				from.Close()
				closeErr := to.Close()
				if copyErr != nil || closeErr != nil || n != info.Size() {
					return failure(SourceChanged, copyErr)
				}
			}
		}
	}
	return nil
}

// make dist consumes the same CLI validators as local development. Cross-build
// the current checkout's CLI into the private build kit; never install it into
// the user's PATH or include it in the resulting plugin package.
func prepareContainerCLI(ctx context.Context, root, run string, log io.Writer) (string, error) {
	repository := root
	found := false
	for depth := 0; depth < 8; depth++ {
		data, err := os.ReadFile(filepath.Join(repository, "tools/cli/go.mod"))
		if err == nil && strings.HasPrefix(string(data), "module github.com/powerx-plugin/cli\n") {
			found = true
			break
		}
		parent := filepath.Dir(repository)
		if parent == repository {
			break
		}
		repository = parent
	}
	if !found {
		return "", failure(SourceUnsafe, nil)
	}
	private := filepath.Join(run, ".publisher-dependencies/bin")
	if err := os.MkdirAll(private, 0700); err != nil {
		return "", failure(StorageFailure, err)
	}
	quote := func(value string) string { data, _ := json.Marshal(value); return string(data) }
	work := filepath.Join(filepath.Dir(run), "validator.go.work")
	content := "go 1.24.0\nuse (\n " + quote(filepath.Join(repository, "tools/cli")) + "\n " + quote(filepath.Join(repository, "framework/backend/go")) + "\n)\n"
	if err := os.WriteFile(work, []byte(content), 0600); err != nil {
		return "", failure(StorageFailure, err)
	}
	binary := filepath.Join(private, "px-plugin")
	err := runOwnedCommand(ctx, "env", filepath.Join(repository, "tools/cli"), []string{"GOWORK=" + work, "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0", "go", "build", "-o", binary, "./cmd/px-plugin"}, nil, log, func(int) error { return nil })
	if err != nil {
		return "", err
	}
	_, hash, err := digestFile(binary)
	return hash, err
}
