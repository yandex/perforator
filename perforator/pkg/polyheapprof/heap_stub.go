//go:build !cgo

package polyheapprof

import "github.com/google/pprof/profile"

func startCHeapProfileRecording() error {
	return nil
}

func readCHeapProfile() (*profile.Profile, error) {
	return nil, nil
}
