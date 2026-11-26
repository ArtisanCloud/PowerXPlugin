//go:build darwin || linux

package publisher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const ContainerUnavailable = "PUBLISHER_CONTAINER_UNAVAILABLE"
const ContainerStartFailed = "PUBLISHER_CONTAINER_START_FAILED"
const ContainerBuildFailed = "PUBLISHER_CONTAINER_BUILD_FAILED"

type BuildEnvironment struct {
	NativeAvailable    bool `json:"native_available"`
	ContainerAvailable bool `json:"container_available"`
	ContainerReady     bool `json:"container_ready"`
	CanStart           bool `json:"can_start"`
}

func dockerContext(ctx context.Context) (string, error) {
	data, err := exec.CommandContext(ctx, "docker", "context", "show").Output()
	name := strings.TrimSpace(string(data))
	if err != nil || name == "" || strings.HasPrefix(name, "-") {
		return "", failure(ContainerUnavailable, err)
	}
	data, err = exec.CommandContext(ctx, "docker", "context", "inspect", name, "--format", "{{.Endpoints.docker.Host}}").Output()
	// Publishing never redirects builds to a remote daemon or exposes the host socket.
	if err != nil || !strings.HasPrefix(strings.TrimSpace(string(data)), "unix:///") {
		return "", failure(ContainerUnavailable, err)
	}
	return name, nil
}
func (e *Engine) BuildEnvironment() BuildEnvironment {
	_, nativeErr := e.nativeBuildTools()
	result := BuildEnvironment{NativeAvailable: nativeErr == nil}
	if _, err := exec.LookPath("docker"); err != nil {
		return result
	}
	result.ContainerAvailable = true
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	name, err := dockerContext(ctx)
	if err != nil {
		return result
	}
	data, err := exec.CommandContext(ctx, "docker", "--context", name, "version", "--format", "{{.Server.Os}}").Output()
	result.ContainerReady = err == nil && strings.TrimSpace(string(data)) == "linux"
	if runtime.GOOS == "darwin" {
		if name != "orbstack" && name != "desktop-linux" && name != "default" {
			return result
		}
		app := "/Applications/Docker.app"
		if name == "orbstack" {
			app = "/Applications/OrbStack.app"
		}
		info, err := os.Stat(app)
		result.CanStart = err == nil && info.IsDir()
	}
	return result
}
func (e *Engine) StartBuildEnvironment(ctx context.Context) (BuildEnvironment, error) {
	result := e.BuildEnvironment()
	if result.ContainerReady {
		return result, nil
	}
	if !result.CanStart {
		return result, failure(ContainerUnavailable, nil)
	}
	name, err := dockerContext(ctx)
	if err != nil {
		return result, err
	}
	app := "Docker"
	if name == "orbstack" {
		app = "OrbStack"
	}
	if err = exec.CommandContext(ctx, "/usr/bin/open", "-a", app).Run(); err != nil {
		return result, failure(ContainerStartFailed, err)
	}
	return result, nil
}

const publisherDockerfile = `FROM golang:1.26-alpine AS go
FROM node:22-bookworm-slim
COPY --from=go /usr/local/go /usr/local/go
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends make ca-certificates ripgrep binutils && rm -rf /var/lib/apt/lists/*
ENV PATH=/workspace/.publisher-dependencies/bin:/usr/local/go/bin:$PATH CGO_ENABLED=0 GOTOOLCHAIN=local GOWORK=off GOENV=off NUXT_TELEMETRY_DISABLED=1 CI=true NODE_OPTIONS="--preserve-symlinks"
WORKDIR /workspace
`

func cleanSourceDigest(ctx context.Context, root, destination string) (string, error) {
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", failure(SourceUnsafe, err)
	}
	w := treeWalk{ctx: ctx, root: root, destination: destination, hash: sha256.New(), stack: map[string]bool{}, source: true, excludeModules: true}
	if link, err := os.Readlink(filepath.Join(root, "contracts")); err == nil && link == "../contracts" {
		plugin, pluginErr := readPlugin(root)
		module, moduleErr := os.ReadFile(filepath.Join(root, "../framework/backend/go/go.mod"))
		if pluginErr != nil || plugin.ID != "com.powerx.plugins.base" || moduleErr != nil || !strings.HasPrefix(string(module), "module github.com/ArtisanCloud/PowerXPlugin/framework/backend/go\n") {
			return "", failure(SourceUnsafe, nil)
		}
		approved, err := filepath.EvalSymlinks(filepath.Join(root, link))
		if err != nil {
			return "", failure(SourceUnsafe, err)
		}
		w.approvedRoots = map[string]string{"contracts": approved}
	}
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

type localSDK struct{ source, destination, digest string }

func copySDK(ctx context.Context, source, destination string) (localSDK, error) {
	source, err := filepath.EvalSymlinks(source)
	if err != nil {
		return localSDK{}, failure(SourceUnsafe, err)
	}
	w := treeWalk{ctx: ctx, root: source, destination: destination, hash: sha256.New(), stack: map[string]bool{}, source: true, excludeModules: true}
	if err := w.visit("."); err != nil {
		return localSDK{}, err
	}
	return localSDK{source, destination, hex.EncodeToString(w.hash.Sum(nil))}, nil
}
func verifySDK(ctx context.Context, sdk localSDK) error {
	for _, root := range []string{sdk.source, sdk.destination} {
		var err error
		root, err = filepath.EvalSymlinks(root)
		if err != nil {
			return failure(SourceUnsafe, err)
		}
		w := treeWalk{ctx: ctx, root: root, hash: sha256.New(), stack: map[string]bool{}, source: true, excludeModules: true}
		if err := w.visit("."); err != nil {
			return err
		}
		if hex.EncodeToString(w.hash.Sum(nil)) != sdk.digest {
			return failure(SourceChanged, nil)
		}
	}
	return nil
}

// Only the three named Framework SDK dependencies may be local. Other local
// dependencies fail explicitly; source configuration is never rewritten.
func prepareCleanDependencies(ctx context.Context, root, run string) ([]localSDK, error) {
	var sdks []localSDK
	backend := "backend"
	if _, err := os.Stat(filepath.Join(root, backend, "go.mod")); os.IsNotExist(err) {
		backend = "backend/go-gin"
	}
	cmd := exec.CommandContext(ctx, "go", "mod", "edit", "-json")
	cmd.Dir = filepath.Join(root, backend)
	cmd.Env = buildEnv()
	raw, err := cmd.Output()
	if err != nil {
		return nil, failure(SourceUnsafe, err)
	}
	var mod struct {
		Replace []struct {
			Old struct{ Path string }
			New struct{ Path, Version string }
		}
	}
	if json.Unmarshal(raw, &mod) != nil {
		return nil, failure(SourceUnsafe, nil)
	}
	for _, replacement := range mod.Replace {
		if replacement.New.Version != "" {
			continue
		}
		if replacement.Old.Path != "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go" || filepath.IsAbs(replacement.New.Path) {
			return nil, failure(SourceUnsafe, nil)
		}
		source := filepath.Join(root, backend, replacement.New.Path)
		metadata, err := os.ReadFile(filepath.Join(source, "go.mod"))
		if err != nil || !strings.HasPrefix(string(metadata), "module "+replacement.Old.Path+"\n") {
			return nil, failure(SourceUnsafe, err)
		}
		sdk, err := copySDK(ctx, source, filepath.Join(run, ".publisher-dependencies/go"))
		if err != nil {
			return nil, err
		}
		sdks = append(sdks, sdk)
		relative, err := filepath.Rel(filepath.Join(run, backend), sdk.destination)
		if err != nil {
			return nil, failure(SourceUnsafe, err)
		}
		edit := exec.CommandContext(ctx, "go", "mod", "edit", "-replace="+replacement.Old.Path+"="+filepath.ToSlash(relative))
		edit.Dir = filepath.Join(run, backend)
		edit.Env = buildEnv()
		if err = edit.Run(); err != nil {
			return nil, failure(SourceUnsafe, err)
		}
	}
	frontend := frontendBuildDir(run)
	packageFile := filepath.Join(run, frontend, "package.json")
	raw, err = os.ReadFile(packageFile)
	if err != nil {
		return nil, failure(SourceUnsafe, err)
	}
	var pkg map[string]any
	if json.Unmarshal(raw, &pkg) != nil {
		return nil, failure(SourceUnsafe, nil)
	}
	replacements := map[string]string{}
	for _, kind := range []string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"} {
		deps, _ := pkg[kind].(map[string]any)
		for name, value := range deps {
			version, ok := value.(string)
			if !ok {
				return nil, failure(SourceUnsafe, nil)
			}
			if !strings.HasPrefix(version, "file:") && !strings.HasPrefix(version, "link:") && !strings.HasPrefix(version, "workspace:") {
				continue
			}
			short := ""
			switch name {
			case "@artisan-cloud/plugin-framework-admin":
				short = "admin"
			case "@artisan-cloud/plugin-framework-client":
				short = "client"
			}
			if short == "" || !strings.HasPrefix(version, "file:") || filepath.IsAbs(strings.TrimPrefix(version, "file:")) {
				return nil, failure(SourceUnsafe, nil)
			}
			old := strings.TrimPrefix(version, "file:")
			source := filepath.Join(root, frontend, old)
			data, err := os.ReadFile(filepath.Join(source, "package.json"))
			if err != nil {
				return nil, failure(SourceUnsafe, err)
			}
			var identity struct{ Name string }
			if json.Unmarshal(data, &identity) != nil || identity.Name != name {
				return nil, failure(SourceUnsafe, nil)
			}
			sdk, err := copySDK(ctx, source, filepath.Join(run, ".publisher-dependencies", short))
			if err != nil {
				return nil, err
			}
			sdks = append(sdks, sdk)
			relative, err := filepath.Rel(filepath.Join(run, frontend), sdk.destination)
			if err != nil {
				return nil, failure(SourceUnsafe, err)
			}
			replacements[old] = filepath.ToSlash(relative)
		}
	}
	var transform func(any) any
	transform = func(input any) any {
		switch value := input.(type) {
		case string:
			for old, next := range replacements {
				if value == "file:"+old {
					return "file:" + next
				}
				if value == old || strings.HasPrefix(value, old+"/") {
					return next + strings.TrimPrefix(value, old)
				}
			}
			return value
		case map[string]any:
			out := map[string]any{}
			for key, child := range value {
				out[transform(key).(string)] = transform(child)
			}
			return out
		case []any:
			for i, item := range value {
				value[i] = transform(item)
			}
			return value
		default:
			return input
		}
	}
	if len(replacements) == 0 {
		return sdks, nil
	}
	for _, file := range []string{"package.json", "package-lock.json"} {
		target := filepath.Join(run, frontend, file)
		raw, err = os.ReadFile(target)
		if err != nil {
			return nil, failure(SourceUnsafe, err)
		}
		var document any
		if json.Unmarshal(raw, &document) != nil {
			return nil, failure(SourceUnsafe, nil)
		}
		raw, err = json.MarshalIndent(transform(document), "", "  ")
		if err != nil {
			return nil, err
		}
		if err = os.WriteFile(target, raw, 0600); err != nil {
			return nil, failure(StorageFailure, err)
		}
	}
	return sdks, nil
}

func frontendBuildDir(root string) string {
	if _, err := os.Stat(filepath.Join(root, "web-admin/package.json")); os.IsNotExist(err) {
		return "web-admin/nuxt"
	}
	return "web-admin"
}

func (e *Engine) executeContainerBuild(ctx context.Context, b *Build, p Plan) error {
	b.Stage = "prechecking"
	if err := e.saveBuild(b, true); err != nil {
		return err
	}
	if !e.BuildEnvironment().ContainerReady {
		return failure(ContainerUnavailable, nil)
	}
	contextName, err := dockerContext(ctx)
	if err != nil {
		return err
	}
	current, err := readPlugin(e.root)
	if err != nil {
		return err
	}
	if current != e.plugin {
		return failure(WorkspaceChanged, nil)
	}
	work := filepath.Join(e.work.path, b.BuildID)
	if err = os.Mkdir(work, 0700); err != nil {
		return failure(StorageFailure, err)
	}
	log, err := newBuildLog(filepath.Join(work, "build.log"), e.root, work)
	if err != nil {
		return err
	}
	defer log.Close()
	source := filepath.Join(work, "source")
	if err = os.Mkdir(source, 0700); err != nil {
		return err
	}
	provenance, err := provenance(ctx, e.root)
	if err != nil {
		return err
	}
	provenance.SHA256, err = cleanSourceDigest(ctx, e.root, source)
	if err != nil {
		return err
	}
	b.Source = &provenance
	run := filepath.Join(work, "run")
	if err = os.Mkdir(run, 0700); err != nil {
		return err
	}
	if _, err = cleanSourceDigest(ctx, source, run); err != nil {
		return err
	}
	sdks, err := prepareCleanDependencies(ctx, e.root, run)
	if err != nil {
		return err
	}
	if err = prepareModuleProxy(ctx, e.root, run, sdks); err != nil {
		return err
	}
	cliHash, err := prepareContainerCLI(ctx, e.root, run, log)
	if err != nil {
		return err
	}
	if len(sdks) > 0 {
		b.Source.DependencySHA256 = map[string]string{}
		for _, sdk := range sdks {
			b.Source.DependencySHA256[filepath.Base(sdk.destination)] = sdk.digest
		}
	}
	imageHash := sha256.Sum256([]byte(publisherDockerfile))
	image := "powerx-publisher-builder:" + hex.EncodeToString(imageHash[:])[:16]
	dockerArgs := func(args ...string) []string { return append([]string{"--context", contextName}, args...) }
	noop := func(int) error { return nil }
	proxy, err := containerProxy()
	if err != nil {
		return err
	}
	inspect := exec.CommandContext(ctx, "docker", dockerArgs("image", "inspect", image, "--format", "{{.Os}}/{{.Architecture}}")...)
	if data, checkErr := inspect.Output(); checkErr != nil || strings.TrimSpace(string(data)) != "linux/amd64" {
		if err = prepareContainerImages(ctx, work, dockerArgs, log); err != nil {
			_, _ = io.WriteString(log, err.Error()+"\n")
			if cause := errors.Unwrap(err); cause != nil {
				_, _ = io.WriteString(log, cause.Error()+"\n")
			}
			return err
		}
		buildArgs := dockerArgs("build", "--platform=linux/amd64", "--tag", image)
		for _, value := range proxy {
			buildArgs = append(buildArgs, "--build-arg", value)
		}
		buildArgs = append(buildArgs, "-")
		if err = runOwnedCommand(ctx, "docker", run, buildArgs, strings.NewReader(publisherDockerfile), log, noop); err != nil {
			return failure(ContainerBuildFailed, err)
		}
	}
	imageID, err := exec.CommandContext(ctx, "docker", dockerArgs("image", "inspect", image, "--format", "{{.Id}}")...).Output()
	if err != nil {
		return failure(ContainerBuildFailed, err)
	}
	b.ContainerImage = strings.TrimSpace(string(imageID))
	output, err := exec.CommandContext(ctx, "docker", dockerArgs("run", "--rm", "--platform=linux/amd64", image, "sh", "-c", `go env GOVERSION; node -p 'JSON.stringify({version:process.version,os:process.platform,arch:process.arch})'`)...).Output()
	if err != nil {
		return failure(ContainerBuildFailed, err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 2 {
		return failure(ContainerBuildFailed, nil)
	}
	var node struct{ Version, OS, Arch string }
	if json.Unmarshal([]byte(lines[1]), &node) != nil || node.OS != "linux" || node.Arch != "x64" {
		return failure(ContainerBuildFailed, nil)
	}
	tools := BuildTools{Go: lines[0], Node: node.Version, NodeOS: node.OS, NodeArch: node.Arch, CLI: cliHash}
	b.Tools = &tools
	frontend := frontendBuildDir(run)
	container := "powerx-publisher-" + b.BuildID
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = exec.CommandContext(cleanup, "docker", dockerArgs("rm", "-f", container)...).Run()
	}()
	// Only this build's sanitized snapshot is mounted. No Docker socket, home,
	// source workspace, tokens, dotenv or system keychain enters the container.
	args := dockerArgs("run", "--name", container, "--label", "com.powerx.publisher.build="+b.BuildID, "--platform=linux/amd64", "--init", "--mount", "type=bind,source="+run+",target=/workspace")
	for _, value := range proxy {
		args = append(args, "--env", value)
	}
	args = append(args, "--env", "GOPROXY=file:///workspace/.publisher-dependencies/module-proxy,https://proxy.golang.org", "--env", "NODE_PATH=/workspace/"+frontend+"/node_modules", image, "sh", "-eu", "-c", `npm --prefix "$1" ci --no-audit --no-fund; make dist "VERSION=$2" PLATFORM=linux TARGET_ARCH=amd64 BUILD_DIR=/workspace/.publisher-out/bin DIST_ROOT=/workspace/.publisher-out/dist "DIST_DIR=/workspace/.publisher-out/dist/$2"`, "publisher-build", frontend, p.Version)
	b.Stage = "building"
	if err = e.saveBuild(b, true); err != nil {
		return err
	}
	err = runOwnedCommand(ctx, "docker", run, args, nil, log, func(pid int) error {
		b.PID = pid
		b.Process = &BuildProcess{ParentPID: os.Getpid(), LeaderPID: pid, GroupID: pid, WorkingDirectory: "work/" + b.BuildID + "/run", StartedAt: time.Now().UTC()}
		return e.saveBuild(b, true)
	})
	b.PID = 0
	if b.Process != nil {
		now := time.Now().UTC()
		b.Process.ExitedAt = &now
	}
	if err != nil {
		return failure(ContainerBuildFailed, err)
	}
	for _, root := range []string{e.root, source} {
		hash, err := cleanSourceDigest(ctx, root, "")
		if err != nil {
			return err
		}
		if hash != provenance.SHA256 {
			return failure(SourceChanged, nil)
		}
	}
	for _, sdk := range sdks {
		if err = verifySDK(ctx, sdk); err != nil {
			return err
		}
	}
	if err = log.Close(); err != nil {
		return failure(StorageFailure, err)
	}
	return e.finishBuild(ctx, b, p, tools, run, work)
}
