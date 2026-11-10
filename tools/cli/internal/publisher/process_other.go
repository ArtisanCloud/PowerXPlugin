//go:build !darwin && !linux

package publisher

import (
	"context"
	"io"
)

type buildLog struct{}

func buildEnv() []string { return nil }

func probeTools() (BuildTools, error) { return BuildTools{}, failure(Unsupported, nil) }
func runBuildCommand(context.Context, string, []string, io.Writer, func(int) error) error {
	return failure(Unsupported, nil)
}
func newBuildLog(string, ...string) (*buildLog, error) { return nil, failure(Unsupported, nil) }
func (*buildLog) Write([]byte) (int, error)            { return 0, failure(Unsupported, nil) }
func (*buildLog) Close() error                         { return nil }
func (*Engine) BuildLog(string) (string, error)        { return "", failure(Unsupported, nil) }
