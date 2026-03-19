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

const defaultMaxProfilesCount = 20

type LocalStorageConfig struct {
	ProfileDir       string `yaml:"profile_dir"`
	BinaryDir        string `yaml:"binary_dir"`
	MaxProfilesCount *int   `yaml:"max_profiles_count,omitempty"`
	SkipBinaries     *bool  `yaml:"skip_binaries,omitempty"`
}

type LocalStorage struct {
	conf           *LocalStorageConfig
	l              log.Logger
	ringBufferSize int
	counter        int
}

func (c *LocalStorageConfig) skipBinaries() bool {
	if c.SkipBinaries == nil {
		return false
	}
	return *c.SkipBinaries
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
	if !conf.skipBinaries() {
		err = checkDir(conf.BinaryDir)
		if err != nil {
			return nil, err
		}
	}

	ringBufferSize := defaultMaxProfilesCount
	if conf.MaxProfilesCount != nil {
		ringBufferSize = *conf.MaxProfilesCount
	}
	if ringBufferSize <= 0 {
		return nil, fmt.Errorf("max_profiles_count must be greater than 0")
	}

	return &LocalStorage{
		conf:           conf,
		l:              l,
		ringBufferSize: ringBufferSize,
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
	profileName := fmt.Sprintf("profile.%s.%d.tar.gz", samplesTypeString, s.counter%s.ringBufferSize)

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
	if s.conf.skipBinaries() {
		return nil
	}

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

func (s *LocalStorage) hasBinary(ctx context.Context, buildID string) (bool, error) {
	path := s.binaryPath(buildID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	}

	return true, nil
}

func (s *LocalStorage) AnnounceBinaries(ctx context.Context, buildIDs []string) ([]string, error) {
	if s.conf.skipBinaries() {
		return nil, nil
	}

	unknownBuildIDs := []string{}
	for _, buildID := range buildIDs {
		present, err := s.hasBinary(ctx, buildID)
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
