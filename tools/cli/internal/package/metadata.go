package pkg

import (
	"encoding/json"
	"os"
	"time"
)

// Artifact describes a packaged artefact that can be verified by downstream services.
type Artifact struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	Hash string `json:"sha256"`
}

// Metadata captures the package level manifest that accompanies each artefact tarball.
type Metadata struct {
	Version    string     `json:"version"`
	Channel    string     `json:"channel"`
	BuildTime  time.Time  `json:"buildTime"`
	CLIVersion string     `json:"cliVersion"`
	GitCommit  string     `json:"gitCommit,omitempty"`
	DistHash   string     `json:"distHash,omitempty"`
	Artifacts  []Artifact `json:"artifacts"`
	Signature  *string    `json:"signature"`
}

// MetadataOptions controls how metadata.json is rendered.
type MetadataOptions struct {
	Version    string
	Channel    string
	BuildTime  time.Time
	CLIVersion string
	GitCommit  string
	DistHash   string
	Artifacts  []Artifact
}

// WriteMetadata renders the metadata to the provided path using pretty JSON formatting.
func WriteMetadata(path string, opts MetadataOptions) error {
	meta := Metadata{
		Version:    opts.Version,
		Channel:    opts.Channel,
		BuildTime:  opts.BuildTime.UTC(),
		CLIVersion: opts.CLIVersion,
		GitCommit:  opts.GitCommit,
		DistHash:   opts.DistHash,
		Artifacts:  append([]Artifact(nil), opts.Artifacts...),
		Signature:  nil,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}

	return nil
}
