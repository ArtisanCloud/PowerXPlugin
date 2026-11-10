//go:build !darwin && !linux

package publisher

import "os"

// Unsupported platforms fail explicitly. They never use a less restrictive
// filesystem implementation or a different credential/state format.
type directory struct{ path string }

func openDirectory(string) (*directory, error)            { return nil, failure(Unsupported, nil) }
func (*directory) child(string, bool) (*directory, error) { return nil, failure(Unsupported, nil) }
func (*directory) close() error                           { return nil }
func (*directory) read(string, int64) ([]byte, error)     { return nil, failure(Unsupported, nil) }
func (*directory) names() ([]string, error)               { return nil, failure(Unsupported, nil) }
func (*directory) write(string, []byte, bool) error       { return failure(Unsupported, nil) }
func (*directory) remove(string) error                    { return failure(Unsupported, nil) }
func (*directory) lock() (func(), error)                  { return nil, failure(Unsupported, nil) }
func (*directory) lockFile(string) (func(), error)        { return nil, failure(Unsupported, nil) }

func (*directory) existing(string, bool) (*directory, error) { return nil, failure(Unsupported, nil) }
func (*directory) openRead(string) (*os.File, error)         { return nil, failure(Unsupported, nil) }
func (*directory) current() bool                             { return false }
