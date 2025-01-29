package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	pprof "github.com/google/pprof/profile"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
	"github.com/yandex/perforator/perforator/pkg/atomicfs"
)

////////////////////////////////////////////////////////////////////////////////

var _ Storage = (*LocalStorage)(nil)

////////////////////////////////////////////////////////////////////////////////

type LocalStorageConfig struct {
	ProfileDir string `yaml:"profile_dir"`
	BinaryDir  string `yaml:"binary_dir"`
}

type LocalStorage struct {
	conf           *LocalStorageConfig
	l              log.Logger
	ringBufferSize int
	counter        int
}

func checkDir(path string) error {
	fileinfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !fileinfo.Mode().IsDir() {
		return fmt.Errorf("path `%s` is not directory", path)
	}

	return nil
}

func NewLocalStorage(conf *LocalStorageConfig, l log.Logger) (*LocalStorage, error) {
	err := checkDir(conf.ProfileDir)
	if err != nil {
		return nil, err
	}
	err = checkDir(conf.BinaryDir)
	if err != nil {
		return nil, err
	}

	return &LocalStorage{
		conf:           conf,
		l:              l,
		ringBufferSize: 20,
		counter:        0,
	}, nil
}

func sampleTypesToString(sampleTypes []*pprof.ValueType) string {
	strs := make([]string, 0, len(sampleTypes))
	for _, sampleType := range sampleTypes {
		strs = append(strs, sampleType.Type+"."+sampleType.Unit)
	}

	sort.Slice(strs, func(i, j int) bool {
		return strs[i] < strs[j]
	})

	return strings.Join(strs, ".")
}

func (s *LocalStorage) StoreProfile(ctx context.Context, profile LabeledProfile) error {
	addProfileComments(profile.Profile, profile.Labels)

	err := profile.Profile.CheckValid()
	if err != nil {
		return err
	}

	samplesTypeString := sampleTypesToString(profile.Profile.SampleType)
	profileName := fmt.Sprintf("profile.%s.%d.tar.gz", samplesTypeString, s.counter%20)

	f, err := atomicfs.Create(filepath.Join(s.conf.ProfileDir, profileName))
	if err != nil {
		return err
	}
	defer f.Close()
	s.counter++
	return profile.Profile.WriteUncompressed(f)
}

func (s *LocalStorage) binaryPath(buildID string) string {
	return filepath.Join(s.conf.BinaryDir, fmt.Sprintf("binary_%s", strings.ReplaceAll(buildID, "/", "%")))
}

func (s *LocalStorage) StoreBinary(ctx context.Context, buildID string, binary binary.SealedFile) error {
	src, err := binary.Unseal()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(s.binaryPath(buildID))
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src.GetFile())
	if err != nil {
		return err
	}

	return nil
}

func (s *LocalStorage) HasBinary(ctx context.Context, buildID string) (bool, error) {
	path := s.binaryPath(buildID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	}

	return true, nil
}

func (s *LocalStorage) AnnounceBinaries(ctx context.Context, buildIDs []string) ([]string, error) {
	unknownBuildIDs := []string{}
	for _, buildID := range buildIDs {
		present, err := s.HasBinary(ctx, buildID)
		if err != nil {
			return nil, err
		}
		if !present {
			unknownBuildIDs = append(unknownBuildIDs, buildID)
		}
	}

	return unknownBuildIDs, nil
}

////////////////////////////////////////////////////////////////////////////////
