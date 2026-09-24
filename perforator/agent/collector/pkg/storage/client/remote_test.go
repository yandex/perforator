package client

import (
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profileformat"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profileresult"
	gateway "github.com/yandex/perforator/perforator/internal/agent_gateway/client/storage"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/profile/bundle"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	storagepb "github.com/yandex/perforator/perforator/proto/storage"
)

type recordingStorage struct {
	storagepb.UnimplementedPerforatorStorageServer
	requests chan *storagepb.PushProfileRequest
}

func (s *recordingStorage) PushProfile(ctx context.Context, req *storagepb.PushProfileRequest) (*storagepb.PushProfileResponse, error) {
	s.requests <- req
	if req.Labels["fail"] == "true" {
		return nil, status.Error(codes.Unavailable, "storage unavailable")
	}
	return &storagepb.PushProfileResponse{}, nil
}

func newRecordingStorageClient(t *testing.T) (*gateway.Client, *recordingStorage) {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	recorder := &recordingStorage{requests: make(chan *storagepb.PushProfileRequest, 8)}
	storagepb.RegisterPerforatorStorageServer(server, recorder)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///storage", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	logger := xlog.ForTest(t)
	gatewayClient, err := gateway.NewClient(&gateway.Config{}, logger, conn)
	require.NoError(t, err)
	return gatewayClient, recorder
}

func TestSerializedStorage(t *testing.T) {
	gatewayClient, recorder := newRecordingStorageClient(t)
	logger := xlog.ForTest(t)
	for _, input := range []profileformat.ProfileFormat{profileformat.Pprof, profileformat.Yaprof} {
		for _, output := range []profileformat.ProfileFormat{profileformat.Pprof, profileformat.Yaprof} {
			t.Run(string(input)+"/"+string(output), func(t *testing.T) {
				start := time.Unix(1700000000, 0).UTC()
				b := profile.NewBuilder().AddSampleType("signal", "count")
				b.AddTimestampedSample(linux.ProcessKey{Pid: 42}, start).AddValue(1).AddStringLabel("signal:name", "SIGINT").
					AddStringLabel("env:key", "value").AddIntLabel("pid", 42, "").AddNativeLocation(0x1000).
					SetMapping().SetBuildID("build-id").Finish().Finish().Finish()
				b.SetStartTime(start).SetEndTime(start.Add(time.Second))
				p := b.Finish()
				p.DurationNanos = int64(time.Second)
				r, err := p.ToResult(input)
				require.NoError(t, err)
				if input == profileformat.Yaprof {
					// Exercise storage with no Go/pprof representation at all.
					r.Bundle = bundle.NewYaprofBundle(r.Bundle.GetYaprof())
				}
				labeled := LabeledProfile{Profile: r, Labels: map[string]string{profilequerylang.CPOIDLabel: "operation", "service": "test"}}
				remote := NewRemoteStorage(logger, nop.Registry{}, gatewayClient, output)
				require.NoError(t, remote.StoreProfile(t.Context(), labeled))
				req := <-recorder.requests
				require.Equal(t, labeled.Labels, req.Labels)
				require.Equal(t, "operation", req.CPOID)
				require.Equal(t, []string{"build-id"}, req.BuildIDs)
				require.Equal(t, []string{"key=value"}, req.Envs)
				require.Equal(t, []string{"signal.count"}, req.EventTypes)
				require.Equal(t, []string{"SIGINT"}, req.SignalTypes)
				require.Equal(t, start, req.StartTimestamp.AsTime())
				require.Equal(t, time.Second, req.Duration.AsDuration())
				body := &profileresult.Result{Bundle: bundle.NewBundle(req.GetProfileBytes(), req.YaprofBytes)}
				decoded, err := body.ParsePprof()
				require.NoError(t, err)
				require.Len(t, decoded.Sample, 1)
				if output == profileformat.Pprof {
					require.Empty(t, req.YaprofBytes)
				} else {
					require.Empty(t, req.GetProfileBytes())
				}

				skip := true
				dir := t.TempDir()
				local, err := NewLocalStorage(&LocalStorageConfig{ProfileDir: dir, SkipBinaries: &skip}, logger.Logger())
				require.NoError(t, err)
				require.NoError(t, local.StoreProfile(t.Context(), labeled))
				data, err := os.ReadFile(filepath.Join(dir, "profile.signal.count.0.tar.gz"))
				require.NoError(t, err)
				decoded, err = (&profileresult.Result{Bundle: bundle.NewPprofBundle(data)}).ParsePprof()
				require.NoError(t, err)
				require.Len(t, decoded.Sample, 1)
			})
		}
	}
	failed := LabeledProfile{Profile: &profileresult.Result{Bundle: bundle.NewPprofBundle([]byte("body"))}, Labels: map[string]string{"fail": "true"}}
	remote := NewRemoteStorage(logger, nop.Registry{}, gatewayClient, profileformat.Pprof)
	require.Equal(t, codes.Unavailable, status.Code(remote.StoreProfile(t.Context(), failed)))
	<-recorder.requests
}

func TestStoragePreservesOriginalPprof(t *testing.T) {
	gatewayClient, recorder := newRecordingStorageClient(t)
	logger := xlog.ForTest(t)
	b := profile.NewBuilder().AddSampleType("cpu", "cycles")
	for range 2 {
		b.Add(linux.ProcessKey{Pid: 42}).AddValue(7).AddIntLabel("size", 16, "custom-unit").
			AddNativeLocation(0x1000).Finish().Finish()
	}
	p := b.FinishRaw()
	var original bytes.Buffer
	require.NoError(t, p.WriteUncompressed(&original))
	r, err := p.ToResult(profileformat.Yaprof)
	require.NoError(t, err)
	require.Equal(t, original.Bytes(), r.Bundle.GetPprof())
	require.NotEmpty(t, r.Bundle.GetYaprof())
	labeled := LabeledProfile{Profile: r}

	t.Run("remote", func(t *testing.T) {
		remote := NewRemoteStorage(logger, nop.Registry{}, gatewayClient, profileformat.Pprof)
		require.NoError(t, remote.StoreProfile(t.Context(), labeled))
		req := <-recorder.requests
		require.Equal(t, original.Bytes(), req.GetProfileBytes())
		require.Empty(t, req.YaprofBytes)
	})
	t.Run("local", func(t *testing.T) {
		skip := true
		dir := t.TempDir()
		local, err := NewLocalStorage(&LocalStorageConfig{ProfileDir: dir, SkipBinaries: &skip}, logger.Logger())
		require.NoError(t, err)
		require.NoError(t, local.StoreProfile(t.Context(), labeled))
		data, err := os.ReadFile(filepath.Join(dir, "profile.cpu.cycles.0.tar.gz"))
		require.NoError(t, err)
		require.Equal(t, original.Bytes(), data)
	})
}

func TestStorageConversionErrors(t *testing.T) {
	logger := xlog.ForTest(t)
	local := &LocalStorage{conf: &LocalStorageConfig{ProfileDir: t.TempDir()}, ringBufferSize: 1}
	for _, r := range []*profileresult.Result{nil, {}, {Bundle: bundle.NewBundle(nil, nil)}} {
		labeled := LabeledProfile{Profile: r}
		require.Error(t, local.StoreProfile(t.Context(), labeled))
		for _, format := range []profileformat.ProfileFormat{profileformat.Pprof, profileformat.Yaprof} {
			remote := NewRemoteStorage(logger, nop.Registry{}, nil, format)
			require.Error(t, remote.StoreProfile(t.Context(), labeled))
		}
	}
	for _, test := range []struct {
		format profileformat.ProfileFormat
		body   *bundle.ProfileBundle
	}{
		{profileformat.Pprof, bundle.NewYaprofBundle([]byte("invalid"))},
		{profileformat.Yaprof, bundle.NewPprofBundle([]byte("invalid"))},
	} {
		remote := NewRemoteStorage(logger, nop.Registry{}, nil, test.format)
		require.Error(t, remote.StoreProfile(t.Context(), LabeledProfile{Profile: &profileresult.Result{Bundle: test.body}}))
	}
	require.Equal(t, 0, local.counter)
}

func TestGetTimeInterval(t *testing.T) {
	start, duration := getTimeInterval(&profileresult.Meta{})
	require.True(t, start.IsZero())
	require.Zero(t, duration)
	first := time.Unix(1700000000, 0)
	start, duration = getTimeInterval(&profileresult.Meta{StartTimestamp: first, EndTimestamp: first.Add(time.Second)})
	require.Equal(t, first, start)
	require.Equal(t, time.Second, duration)
}
