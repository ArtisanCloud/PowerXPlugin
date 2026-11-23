package publisher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const BuildSchema = "powerx.plugin-publisher-build/v1"
const BuildTimeout = 30 * time.Minute
const BuildPlatform = "PUBLISHER_BUILD_PLATFORM_REQUIRED"
const BuildBusy = "PUBLISHER_BUILD_BUSY"
const BuildFailed = "PUBLISHER_BUILD_FAILED"
const SourceChanged = "PUBLISHER_BUILD_SOURCE_CHANGED"
const SourceUnsafe = "PUBLISHER_BUILD_SOURCE_UNSAFE"
const ArtifactInvalid = "PUBLISHER_ARTIFACT_INVALID"
const ArtifactChanged = "PUBLISHER_ARTIFACT_CHANGED"
const NativeUnsupported = "PUBLISHER_NATIVE_RUNTIME_UNVERIFIED"
const BuildCancelled = "PUBLISHER_BUILD_CANCELLED"

type BuildTools struct {
	Go       string `json:"go"`
	Node     string `json:"node"`
	NodeOS   string `json:"node_os"`
	NodeArch string `json:"node_arch"`
	CLI      string `json:"cli_sha256,omitempty"`
}
type BuildSource struct {
	SHA256           string            `json:"sha256"`
	Commit           string            `json:"commit,omitempty"`
	Dirty            *bool             `json:"dirty"`
	VCS              string            `json:"vcs"`
	DependencySHA256 map[string]string `json:"dependency_sha256,omitempty"`
}
type Artifact struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	Size      int64  `json:"size"`
	FileCount int    `json:"file_count"`
}
type BuildProcess struct {
	ParentPID        int        `json:"parent_pid"`
	LeaderPID        int        `json:"leader_pid"`
	GroupID          int        `json:"group_id"`
	WorkingDirectory string     `json:"working_directory"`
	StartedAt        time.Time  `json:"started_at"`
	ExitedAt         *time.Time `json:"exited_at,omitempty"`
}
type Build struct {
	Schema         string        `json:"schema"`
	BuildID        string        `json:"build_id"`
	ReleasePlanID  string        `json:"release_plan_id"`
	PluginID       string        `json:"plugin_id"`
	Version        string        `json:"version"`
	Target         Target        `json:"target"`
	Stage          string        `json:"stage"`
	Executor       string        `json:"executor,omitempty"`
	ContainerImage string        `json:"container_image,omitempty"`
	Source         *BuildSource  `json:"source,omitempty"`
	Tools          *BuildTools   `json:"tools,omitempty"`
	Artifact       *Artifact     `json:"artifact,omitempty"`
	Error          *Error        `json:"error,omitempty"`
	PID            int           `json:"pid,omitempty"`
	Process        *BuildProcess `json:"process,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}
type BuildPage struct {
	Items []Build `json:"items"`
	Total int     `json:"total"`
	Page  int     `json:"page"`
	Size  int     `json:"size"`
}
type buildJob struct {
	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
	err    error
}

func nativeTarget() Target { return Target{OS: runtime.GOOS, Arch: runtime.GOARCH} }

func (e *Engine) nativeBuildTools() (BuildTools, error) {
	tools, err := e.availableTools()
	if err != nil {
		return BuildTools{}, err
	}
	switch e.host {
	case Target{OS: "linux", Arch: "amd64"}:
		if tools.NodeOS == "linux" && tools.NodeArch == "x64" {
			return tools, nil
		}
	case Target{OS: "darwin", Arch: "arm64"}:
		if tools.NodeOS == "darwin" && tools.NodeArch == "arm64" {
			for _, name := range []string{"make", "npm", "rg", "strings"} {
				if _, err := exec.LookPath(name); err != nil {
					return BuildTools{}, failure(BuildPlatform, err)
				}
			}
			return tools, nil
		}
	}
	return BuildTools{}, failure(BuildPlatform, nil)
}

func (e *Engine) BuildTargets() []Target {
	if _, err := e.nativeBuildTools(); err == nil {
		return []Target{{OS: "linux", Arch: "amd64"}}
	}
	if _, err := exec.LookPath("docker"); err == nil {
		return []Target{{OS: "linux", Arch: "amd64"}}
	}
	return []Target{}
}
func (e *Engine) availableTools() (BuildTools, error) {
	e.toolsOnce.Do(func() { e.tools, e.toolsErr = e.toolProbe() })
	return e.tools, e.toolsErr
}
func active(stage string) bool {
	switch stage {
	case "queued", "prechecking", "building", "packing", "verifying":
		return true
	}
	return false
}
func (e *Engine) saveBuild(b *Build, replace bool) error {
	b.UpdatedAt = time.Now().UTC()
	data, err := json.Marshal(b)
	if err != nil {
		return failure(StorageFailure, err)
	}
	return e.builds.write(b.BuildID+".json", append(data, '\n'), replace)
}
func (e *Engine) GetBuild(id string) (Build, error) {
	if !canonicalUUID(id) {
		return Build{}, invalid("build_id")
	}
	data, err := e.builds.read(id+".json", MaxRecordBytes)
	if err != nil {
		return Build{}, err
	}
	var b Build
	if decodeStrict(data, &b) != nil || b.Schema != BuildSchema || b.BuildID != id || !canonicalUUID(b.ReleasePlanID) || b.PluginID != e.plugin.ID || !stableVersion.MatchString(b.Version) || b.Target != (Target{OS: "linux", Arch: "amd64"}) || b.CreatedAt.IsZero() || b.UpdatedAt.IsZero() {
		return Build{}, failure(InvalidRecord, nil)
	}
	if !active(b.Stage) && b.Stage != "ready" && b.Stage != "failed" && b.Stage != "cancelled" && b.Stage != "interrupted" {
		return Build{}, failure(InvalidRecord, nil)
	}
	if b.Artifact != nil && (b.Artifact.Path != "work/"+id+"/artifact.zip" || !validSHA(b.Artifact.SHA256) || b.Artifact.Size <= 0) {
		return Build{}, failure(InvalidRecord, nil)
	}
	if b.Source != nil && (!validSHA(b.Source.SHA256) || b.Source.VCS != "git" && b.Source.VCS != "filesystem") {
		return Build{}, failure(InvalidRecord, nil)
	}
	if b.Executor != "" && b.Executor != "native" && b.Executor != "container" {
		return Build{}, failure(InvalidRecord, nil)
	}
	workingDirectory := "work/" + id + "/source"
	if b.Executor == "container" {
		workingDirectory = "work/" + id + "/run"
	}
	if b.Process != nil && (b.Process.ParentPID < 1 || b.Process.LeaderPID < 1 || b.Process.GroupID != b.Process.LeaderPID || b.Process.WorkingDirectory != workingDirectory) {
		return Build{}, failure(InvalidRecord, nil)
	}
	if !active(b.Stage) && b.Stage != "ready" && b.Error == nil {
		return Build{}, failure(InvalidRecord, nil)
	}
	if b.Stage == "ready" && (b.Artifact == nil || b.Source == nil || !validSHA(b.Source.SHA256)) {
		return Build{}, failure(InvalidRecord, nil)
	}
	return b, nil
}
func validSHA(s string) bool {
	data, err := hex.DecodeString(s)
	return err == nil && len(data) == sha256.Size && strings.ToLower(s) == s
}
func (e *Engine) ListBuilds(page, size int) (BuildPage, error) {
	if page < 1 || size < 1 || size > MaxPageSize {
		return BuildPage{}, invalid("pagination")
	}
	names, err := e.builds.names()
	if err != nil {
		return BuildPage{}, err
	}
	items := []Build{}
	for _, name := range names {
		if strings.HasPrefix(name, ".pending-") {
			continue
		}
		if !strings.HasSuffix(name, ".json") || !canonicalUUID(strings.TrimSuffix(name, ".json")) {
			return BuildPage{}, failure(InvalidRecord, nil)
		}
		b, err := e.GetBuild(strings.TrimSuffix(name, ".json"))
		if err != nil {
			return BuildPage{}, err
		}
		items = append(items, b)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].BuildID < items[j].BuildID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	out := BuildPage{Items: []Build{}, Total: len(items), Page: page, Size: size}
	if page-1 > len(items)/size {
		return out, nil
	}
	start := (page - 1) * size
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	out.Items = items[start:end]
	return out, nil
}
func (e *Engine) recoverBuilds() error {
	release, err := e.state.lockFile("build.lock")
	if err != nil {
		if AsError(err).Code == Conflict {
			return nil
		}
		return err
	}
	defer release()
	page, err := e.ListBuilds(1, MaxPageSize)
	if err != nil {
		return err
	}
	for i := 1; (i-1)*MaxPageSize < page.Total; i++ {
		if i > 1 {
			page, err = e.ListBuilds(i, MaxPageSize)
			if err != nil {
				return err
			}
		}
		for _, b := range page.Items {
			if active(b.Stage) {
				b.Stage = "interrupted"
				b.PID = 0
				b.Error = failure(BuildFailed, nil)
				if err = e.saveBuild(&b, true); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func (e *Engine) StartBuild(ctx context.Context, planID string) (Build, error) {
	return e.StartBuildWithExecutor(ctx, planID, "native")
}
func (e *Engine) StartBuildWithExecutor(ctx context.Context, planID, executor string) (Build, error) {
	p, err := e.Get(planID)
	if err != nil {
		return Build{}, err
	}
	if executor != "native" && executor != "container" {
		return Build{}, invalid("executor")
	}
	if executor == "native" {
		if _, err := e.nativeBuildTools(); err != nil {
			return Build{}, err
		}
	}
	if !safeMakePath(e.root) || !e.work.current() {
		return Build{}, failure(SourceUnsafe, nil)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closing {
		return Build{}, failure(BuildCancelled, nil)
	}
	release, err := e.state.lockFile("build.lock")
	if err != nil {
		if AsError(err).Code == Conflict {
			return Build{}, failure(BuildBusy, nil)
		}
		return Build{}, err
	}
	b := Build{Schema: BuildSchema, BuildID: uuid.NewString(), ReleasePlanID: p.ReleasePlanID, PluginID: p.PluginID, Version: p.Version, Target: p.Target, Executor: executor, Stage: "queued", CreatedAt: time.Now().UTC()}
	if err = e.saveBuild(&b, false); err != nil {
		release()
		return Build{}, err
	}
	result := b
	worker := b
	jobCtx, cancel := context.WithTimeout(ctx, BuildTimeout)
	job := &buildJob{cancel: cancel, done: make(chan struct{})}
	e.jobs[b.BuildID] = job
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer close(job.done)
		defer cancel()
		defer release()
		b := worker
		err := e.executeBuild(jobCtx, &b, p)
		if err != nil {
			b.Artifact = nil
			b.PID = 0
			b.Stage = "failed"
			b.Error = AsError(err)
			if jobCtx.Err() != nil {
				b.Stage = "cancelled"
				b.Error = failure(BuildCancelled, nil)
			}
			if saveErr := e.saveBuild(&b, true); saveErr != nil {
				err = saveErr
			}
		}
		job.mu.Lock()
		job.err = err
		job.mu.Unlock()
	}()
	return result, nil
}
func (e *Engine) WaitBuild(id string) (Build, error) {
	e.mu.Lock()
	job := e.jobs[id]
	e.mu.Unlock()
	if job != nil {
		<-job.done
		job.mu.Lock()
		err := job.err
		job.mu.Unlock()
		if err != nil && AsError(err).Code == StorageFailure {
			return Build{}, err
		}
	}
	return e.GetBuild(id)
}
func (e *Engine) CancelBuild(id string) (Build, error) {
	b, err := e.GetBuild(id)
	if err != nil {
		return Build{}, err
	}
	e.mu.Lock()
	job := e.jobs[id]
	e.mu.Unlock()
	if job == nil {
		if active(b.Stage) {
			return Build{}, failure(BuildBusy, nil)
		}
		return b, nil
	}
	job.cancel()
	return e.WaitBuild(id)
}
func (e *Engine) VerifyArtifact(id string) (Artifact, error) {
	return e.verifyArtifact(context.Background(), id)
}
func (e *Engine) verifyArtifact(ctx context.Context, id string) (Artifact, error) {
	b, err := e.GetBuild(id)
	if err != nil {
		return Artifact{}, err
	}
	if b.Stage != "ready" || b.Artifact == nil {
		return Artifact{}, failure(ArtifactInvalid, nil)
	}
	dir, err := e.work.existing(id, true)
	if err != nil {
		return Artifact{}, failure(ArtifactChanged, err)
	}
	defer dir.close()
	f, err := dir.openRead("artifact.zip")
	if err != nil {
		return Artifact{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.Size() != b.Artifact.Size {
		return Artifact{}, failure(ArtifactChanged, err)
	}
	if info.Size() > MaxPackageBytes {
		return Artifact{}, failure(TooLarge, nil)
	}
	h := sha256.New()
	size, err := io.Copy(h, &contextReader{ctx: ctx, reader: io.LimitReader(f, MaxPackageBytes+1)})
	sha := hex.EncodeToString(h.Sum(nil))
	if err != nil || size != b.Artifact.Size || sha != b.Artifact.SHA256 {
		return Artifact{}, failure(ArtifactChanged, err)
	}
	if err = validatePackageZIP(ctx, f, size, Plan{PluginID: b.PluginID, CreatePlan: CreatePlan{Version: b.Version, Target: b.Target}}); err != nil {
		return Artifact{}, err
	}
	return *b.Artifact, nil
}
func (e *Engine) executeBuild(ctx context.Context, b *Build, p Plan) error {
	if b.Executor == "container" {
		return e.executeContainerBuild(ctx, b, p)
	}
	b.Stage = "prechecking"
	if err := e.saveBuild(b, true); err != nil {
		return err
	}
	tools, err := e.nativeBuildTools()
	if err != nil {
		return err
	}
	b.Tools = &tools
	current, err := readPlugin(e.root)
	if err != nil {
		return err
	}
	if current.ID != e.plugin.ID || current.SourceVersion != e.plugin.SourceVersion {
		return failure(WorkspaceChanged, nil)
	}
	work := filepath.Join(e.work.path, b.BuildID)
	if err = os.Mkdir(work, 0700); err != nil {
		return failure(StorageFailure, err)
	}
	if e.host.OS == "darwin" {
		return e.executeDarwinBuild(ctx, b, p, tools, work)
	}
	snapshot := filepath.Join(work, "source")
	source, err := snapshotSource(ctx, e.root, snapshot)
	if err != nil {
		return err
	}
	b.Source = &source
	if err = validateDependencies(ctx, snapshot); err != nil {
		return err
	}
	b.Stage = "building"
	if err = e.saveBuild(b, true); err != nil {
		return err
	}
	log, err := newBuildLog(filepath.Join(work, "build.log"), e.root, work)
	if err != nil {
		return err
	}
	err = e.runNativeBuildProcess(ctx, b, snapshot, "make", makeDistArgs(snapshot, p), log)
	closeErr := log.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return failure(StorageFailure, closeErr)
	}
	if err = unchangedSource(ctx, e.root, snapshot, source); err != nil {
		return err
	}
	return e.finishBuild(ctx, b, p, tools, snapshot, work)
}

func makeDistArgs(snapshot string, p Plan) []string {
	dist := filepath.Join(snapshot, ".publisher-out/dist")
	return []string{"dist", "VERSION=" + p.Version, "PLATFORM=linux", "TARGET_ARCH=amd64",
		"BUILD_DIR=" + filepath.Join(snapshot, ".publisher-out/bin"), "DIST_ROOT=" + dist, "DIST_DIR=" + filepath.Join(dist, p.Version)}
}

func (e *Engine) runNativeBuildProcess(ctx context.Context, b *Build, snapshot, program string, args []string, log io.Writer) error {
	err := runOwnedCommand(ctx, program, snapshot, args, nil, log, func(pid int) error {
		b.PID = pid
		b.Process = &BuildProcess{ParentPID: os.Getpid(), LeaderPID: pid, GroupID: pid, WorkingDirectory: "work/" + b.BuildID + "/source", StartedAt: time.Now().UTC()}
		return e.saveBuild(b, true)
	})
	if b.Process != nil {
		now := time.Now().UTC()
		b.Process.ExitedAt = &now
	}
	b.PID = 0
	return err
}
func (e *Engine) finishBuild(ctx context.Context, b *Build, p Plan, tools BuildTools, snapshot, work string) error {
	b.Stage = "packing"
	if err := e.saveBuild(b, true); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(work, "staging-")
	if err != nil {
		return failure(StorageFailure, err)
	}
	defer os.RemoveAll(stage)
	count, err := freezePackage(ctx, snapshot, filepath.Join(snapshot, ".publisher-out/dist", p.Version), stage, p, tools)
	if err != nil {
		return err
	}
	b.Stage = "verifying"
	if err = e.saveBuild(b, true); err != nil {
		return err
	}
	artifact := filepath.Join(work, "artifact.zip")
	if err = packZIP(ctx, stage, artifact, p); err != nil {
		return err
	}
	size, sha, err := digestFile(artifact)
	if err != nil {
		return err
	}
	if size > MaxPackageBytes {
		return failure(TooLarge, nil)
	}
	if err = os.Chmod(artifact, 0400); err != nil {
		return failure(StorageFailure, err)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	b.Artifact = &Artifact{Path: "work/" + b.BuildID + "/artifact.zip", SHA256: sha, Size: size, FileCount: count}
	b.Stage = "ready"
	return e.saveBuild(b, true)
}
