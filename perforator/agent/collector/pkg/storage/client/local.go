package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
	"github.com/yandex/perforator/perforator/pkg/atomicfs"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
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

// eventTypesToString joins sorted event types with dots without modifying the input.
func eventTypesToString(eventTypes []string) string {
	strs := append([]string(nil), eventTypes...)
	sort.Strings(strs)
	return strings.Join(strs, ".")
}

func (s *LocalStorage) StoreProfile(ctx context.Context, profile LabeledProfile) error {
	if profile.Profile == nil || profile.Profile.Bundle == nil {
		return errors.New("profile has no serialized body")
	}
	// Local profiles use pprof for compatibility with debugging tools.
	body, err := profile.Profile.Bundle.GetOrConvertPprof()
	if err != nil {
		return err
	}
	types := eventTypesToString(profile.Profile.Meta.EventTypes)
	name := fmt.Sprintf("profile.%s.%d.tar.gz", types, s.counter%s.ringBufferSize)
	if err := atomicfs.WriteFile(filepath.Join(s.conf.ProfileDir, name), body); err != nil {
		return err
	}
	s.counter++
	return nil
}

func (s *LocalStorage) binaryPath(buildID string) string {
	return filepath.Join(s.conf.BinaryDir, fmt.Sprintf("binary_%s", strings.ReplaceAll(buildID, "/", "%")))
}

func (s *LocalStorage) StoreBinary(ctx context.Context, buildID string, attributes *perforatorstorage.BinaryAttributes, binary binary.SealedFile) error {
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
