package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/google/pprof/profile"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
	"github.com/yandex/perforator/perforator/pkg/env"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	"github.com/yandex/perforator/perforator/pkg/storage/client"
	"github.com/yandex/perforator/perforator/pkg/xelf"
	"github.com/yandex/perforator/perforator/pkg/xlog"
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
	client  *client.Client
	logger  xlog.Logger
	metrics remoteStorageClientMetrics
}

func NewRemoteStorage(conf *client.Config, l xlog.Logger, r metrics.Registry) (*RemoteStorage, error) {
	client, err := client.NewStorageClient(conf, l)
	if err != nil {
		return nil, err
	}

	return &RemoteStorage{
		client: client,
		logger: l,
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
	}, nil
}

func addProfileComments(profile *profile.Profile, labels map[string]string) {
	for k, v := range labels {
		profile.Comments = append(profile.Comments, fmt.Sprintf("%s:%s", k, v))
	}
}

func getProfileBuildIDs(profile *profile.Profile) []string {
	ids := make([]string, 0, len(profile.Mapping))
	known := make(map[string]bool)
	for _, m := range profile.Mapping {
		if m == nil || m.BuildID == "" {
			continue
		}

		id := m.BuildID
		if known[id] {
			continue
		}

		known[id] = true
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func getProfileEnvs(profile *profile.Profile) []string {
	res := make([]string, 0)
	seenEnvs := make(map[string]struct{})
	for _, s := range profile.Sample {
		if s == nil {
			continue
		}
		for key, values := range s.Label {
			if len(values) == 0 {
				continue
			}
			value := values[0]
			if envKey, ok := env.BuildEnvKeyFromLabelKey(key); ok {
				concatenatedEnv := env.BuildConcatenatedEnv(envKey, value)
				if _, seen := seenEnvs[concatenatedEnv]; !seen {
					seenEnvs[concatenatedEnv] = struct{}{}
					res = append(res, concatenatedEnv)
				}
			}
		}
	}
	return res
}

func getProfileEventTypes(profile *profile.Profile) []string {
	if len(profile.SampleType) == 0 {
		return []string{sampletype.SampleTypeCPUCycles}
	}

	res := make([]string, 0, len(profile.SampleType))
	for _, sampleType := range profile.SampleType {
		res = append(res, sampletype.SampleTypeToString(sampleType))
	}

	return res
}

func (s *RemoteStorage) StoreProfile(ctx context.Context, profile LabeledProfile) error {
	addProfileComments(profile.Profile, profile.Labels)

	err := profile.Profile.CheckValid()
	if err != nil {
		return err
	}

	profileBytes := bytes.NewBuffer([]byte{})
	err = profile.Profile.WriteUncompressed(profileBytes)
	if err != nil {
		return nil
	}

	buildIDs := getProfileBuildIDs(profile.Profile)

	envs := getProfileEnvs(profile.Profile)

	eventTypes := getProfileEventTypes(profile.Profile)

	sz, err := s.client.PushProfile(
		ctx,
		profileBytes.Bytes(),
		profile.Labels,
		buildIDs,
		envs,
		eventTypes,
	)
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

func (s *RemoteStorage) StoreBinary(ctx context.Context, buildID string, binary binary.SealedFile) error {
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

	w, cancel, err := s.client.PushBinary(ctx, buildID)
	if cancel != nil {
		defer cancel()
	}
	if err != nil {
		l.Error("Failed to start binary writer", log.String("build_id", buildID), log.Error(err))
		return err
	}

	written, err := io.Copy(w, file.GetFile())
	if err != nil {
		// If send failed with EOF this means that the rpc was terminated by server,
		// so we must try to obtain error from Close()
		closeErr := w.Close()
		if closeErr != nil && err == io.EOF {
			err = closeErr
		}
		l.Error("Failed to upload binary", log.String("build_id", buildID), log.Error(err))
		return err
	}

	err = w.Close()
	if err != nil {
		l.Error("Failed to close binary writer", log.String("build_id", buildID), log.Error(err))
		return err
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
