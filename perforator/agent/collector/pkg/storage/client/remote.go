package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profileformat"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profileresult"
	"github.com/yandex/perforator/perforator/internal/agent_gateway/client/storage"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/xelf"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

type remoteStorageClientMetrics struct {
	profilesUploaded         metrics.Counter
	profilesUploadedSizeHist metrics.Histogram
	profilesUploadedSize     metrics.Counter
	profileBuildIDCount      metrics.Counter

	binariesFailedUnseal      metrics.Counter
	binariesUploaded          metrics.Counter
	binariesUploadTimer       metrics.Timer
	binariesUploadedBytes     metrics.Counter
	binariesUploadsInProgress metrics.IntGauge
}

type RemoteStorage struct {
	client        *storage.Client
	logger        xlog.Logger
	metrics       remoteStorageClientMetrics
	profileFormat profileformat.ProfileFormat
}

func NewRemoteStorage(l xlog.Logger, r metrics.Registry, client *storage.Client, profileFormat profileformat.ProfileFormat) *RemoteStorage {
	return &RemoteStorage{
		client:        client,
		logger:        l,
		profileFormat: profileFormat,
		metrics: remoteStorageClientMetrics{
			profilesUploaded: r.WithTags(map[string]string{"kind": "uploaded"}).Counter("profiles.count"),
			profilesUploadedSizeHist: r.Histogram(
				"profiles.uploaded_size.hist",
				metrics.MakeLinearBuckets(0, 1024*100, 10),
			),
			profilesUploadedSize: r.Counter("profiles.uploaded.bytes"),
			profileBuildIDCount:  r.Counter("profile.buildid.count"),

			binariesFailedUnseal:      r.WithTags(map[string]string{"kind": "failed_unseal"}).Counter("binaries.count"),
			binariesUploaded:          r.WithTags(map[string]string{"kind": "uploaded"}).Counter("binaries.count"),
			binariesUploadTimer:       r.Timer("binaries.upload-timer"),
			binariesUploadedBytes:     r.Counter("binaries.uploaded_bytes"),
			binariesUploadsInProgress: r.IntGauge("binaries.uploads_in_progress"),
		},
	}
}

func getTimeInterval(meta *profileresult.Meta) (time.Time, time.Duration) {
	if meta.StartTimestamp.IsZero() {
		return time.Time{}, time.Duration(0)
	}

	return meta.StartTimestamp, meta.EndTimestamp.Sub(meta.StartTimestamp)
}

func (s *RemoteStorage) StoreProfile(ctx context.Context, profile LabeledProfile) error {
	if profile.Profile == nil || profile.Profile.Bundle == nil {
		return errors.New("profile has no serialized body")
	}
	meta := &profile.Profile.Meta
	cpoID := profile.Labels[profilequerylang.CPOIDLabel]
	buildIDs := meta.BuildIDs
	startTimestamp, duration := getTimeInterval(meta)

	pushProfile := &storage.Profile{
		Labels:                     profile.Labels,
		BuildIDs:                   buildIDs,
		Envs:                       meta.Envs,
		EventTypes:                 meta.EventTypes,
		SignalTypes:                meta.SignalTypes,
		CustomProfilingOperationID: cpoID,
		StartTimestamp:             startTimestamp,
		Duration:                   duration,
	}

	profileBundle := profile.Profile.Bundle

	switch s.profileFormat {
	case profileformat.Pprof:
		pprofBytes, err := profileBundle.GetOrConvertPprof()
		if err != nil {
			return fmt.Errorf("failed to get pprof profile: %w", err)
		}
		pushProfile.Raw = pprofBytes
	case profileformat.Yaprof:
		yaprofBytes, err := profileBundle.GetOrConvertYaprof()
		if err != nil {
			return fmt.Errorf("failed to get yaprof profile: %w", err)
		}
		pushProfile.YaprofRaw = yaprofBytes
	default:
		return fmt.Errorf("unsupported profile format %q", s.profileFormat)
	}

	sz, err := s.client.PushProfile(ctx, pushProfile)
	if err != nil {
		return err
	}

	s.metrics.profilesUploadedSize.Add(int64(sz))
	s.metrics.profilesUploadedSizeHist.RecordValue(float64(sz))
	s.metrics.profilesUploaded.Inc()
	s.metrics.profileBuildIDCount.Add(int64(len(buildIDs)))

	return nil
}

func unsealBinarySafe(binary binary.SealedFile, expectedBuildID string) (f binary.UnsealedFile, err error) {
	f, err = binary.Unseal()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			_ = f.Close()
		}
	}()

	// sanity check
	var buildID string
	buildID, err = xelf.ReadBuildID(f.GetFile())
	if err != nil {
		return
	}
	if buildID != expectedBuildID {
		err = fmt.Errorf("expected build id %s, found build id %s", expectedBuildID, buildID)
	}

	return
}

func (s *RemoteStorage) StoreBinary(ctx context.Context, buildID string, attributes *perforatorstorage.BinaryAttributes, binary binary.SealedFile) error {
	l := s.logger.WithContext(ctx)
	start := time.Now()
	s.metrics.binariesUploadsInProgress.Add(1)
	defer s.metrics.binariesUploadsInProgress.Add(-1)

	file, err := unsealBinarySafe(binary, buildID)
	if err != nil {
		l.Debug("Failed to unseal binary safe", log.String("buildID", buildID), log.Error(err))
		s.metrics.binariesFailedUnseal.Inc()
		return err
	}
	defer file.Close()

	fi, err := file.GetFile().Stat()
	if err != nil {
		l.Debug("Failed to stat file", log.String("buildID", buildID), log.Error(err))
		s.metrics.binariesFailedUnseal.Inc()
		return err
	}

	w, err := s.client.PushBinary(ctx, buildID,
		storage.WithUncompressedSize(uint64(fi.Size())),
		storage.WithBinaryMetadata(attributes),
	)
	if err != nil {
		l.Error("Failed to start binary writer", log.String("build_id", buildID), log.Error(err))
		return err
	}

	written, copyErr := io.Copy(w, file.GetFile())
	if copyErr != nil {
		w.Abort()
		l.Error(
			"Failed to upload binary",
			log.String("build_id", buildID),
			log.Int64("bytes_written", written),
			log.Error(copyErr),
		)
		return copyErr
	}

	if closeErr := w.Close(); closeErr != nil {
		l.Error(
			"Failed to close binary writer",
			log.String("build_id", buildID),
			log.Int64("bytes_written", written),
			log.Error(closeErr),
		)
		return closeErr
	}

	s.metrics.binariesUploadTimer.RecordDuration(time.Since(start))
	s.metrics.binariesUploadedBytes.Add(written)
	s.metrics.binariesUploaded.Inc()

	l.Info("Uploaded binary", log.String("build_id", buildID), log.Int64("size", written))

	return nil
}

func (s *RemoteStorage) HasBinary(ctx context.Context, buildID string) (bool, error) {
	unknownBuildIDs, err := s.client.AnnounceBinaries(ctx, []string{buildID})
	if err != nil {
		return false, err
	}

	return len(unknownBuildIDs) == 0, nil
}

func (s *RemoteStorage) AnnounceBinaries(ctx context.Context, buildIDs []string) ([]string, error) {
	return s.client.AnnounceBinaries(ctx, buildIDs)
}
