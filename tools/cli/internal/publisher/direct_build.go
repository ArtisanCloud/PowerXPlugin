//go:build darwin || linux

package publisher

import (
	"context"
	"os"
	"path/filepath"
)

func (e *Engine) executeDarwinBuild(ctx context.Context, b *Build, p Plan, tools BuildTools, work string) error {
	input := filepath.Join(work, "input")
	snapshot := filepath.Join(work, "source")
	defer os.RemoveAll(input)
	defer os.RemoveAll(snapshot)

	source, err := provenance(ctx, e.root)
	if err != nil {
		return err
	}
	source.SHA256, err = cleanSourceDigest(ctx, e.root, input)
	if err != nil {
		return err
	}
	copied, err := cleanSourceDigest(ctx, input, snapshot)
	if err != nil {
		return err
	}
	if copied != source.SHA256 {
		return failure(SourceChanged, nil)
	}
	sdks, err := prepareCleanDependencies(ctx, e.root, snapshot)
	if err != nil {
		return err
	}
	if len(sdks) > 0 {
		source.DependencySHA256 = make(map[string]string, len(sdks))
		for _, sdk := range sdks {
			source.DependencySHA256[filepath.Base(sdk.destination)] = sdk.digest
		}
	}
	b.Source = &source

	executable, err := os.Executable()
	if err == nil {
		executable, err = filepath.EvalSymlinks(executable)
	}
	if err != nil {
		return failure(BuildFailed, err)
	}
	private := filepath.Join(snapshot, ".publisher-dependencies")
	validator := filepath.Join(private, filepath.Base(executable))
	if !safeMakePath(validator) {
		return failure(SourceUnsafe, nil)
	}
	if err = copyPackageTree(ctx, filepath.Dir(executable), filepath.Base(executable), private); err != nil {
		return err
	}
	if _, tools.CLI, err = digestFile(validator); err != nil {
		return err
	}
	b.Tools = &tools

	npmConfig := filepath.Join(private, "npmrc")
	if err = os.WriteFile(npmConfig, nil, 0600); err != nil {
		return failure(StorageFailure, err)
	}
	log, err := newBuildLog(filepath.Join(work, "build.log"), e.root, work)
	if err != nil {
		return err
	}
	b.Stage = "building"
	if err = e.saveBuild(b, true); err != nil {
		_ = log.Close()
		return err
	}
	const command = `npm --prefix "$1" ci --include=dev --prefer-offline --no-audit --no-fund --workspaces=false --userconfig "$2" --globalconfig /dev/null
shift 2
exec env GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org make "$@"`
	args := []string{"-eu", "-c", command, "publisher-build", frontendBuildDir(snapshot), npmConfig}
	args = append(args, makeDistArgs(snapshot, p)...)
	args = append(args, "PX_PLUGIN="+validator)
	err = e.runNativeBuildProcess(ctx, b, snapshot, "/bin/sh", args, log)
	closeErr := log.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return failure(StorageFailure, closeErr)
	}

	for _, root := range []string{e.root, input} {
		digest, err := cleanSourceDigest(ctx, root, "")
		if err != nil {
			return err
		}
		if digest != source.SHA256 {
			return failure(SourceChanged, nil)
		}
	}
	for _, sdk := range sdks {
		if err = verifySDK(ctx, sdk); err != nil {
			return err
		}
	}
	return e.finishBuild(ctx, b, p, tools, snapshot, work)
}
