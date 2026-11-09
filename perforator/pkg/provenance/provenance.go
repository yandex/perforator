package provenance

import "io"

type Info struct {
	CommitID        string `json:"commit_id,omitempty"`
	BuildTargetPath string `json:"build_target_path,omitempty"`
}

type Reader interface {
	Read(io.ReaderAt) (*Info, error)
}
