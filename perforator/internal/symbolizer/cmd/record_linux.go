//go:build linux
// +build linux

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"time"

	pprof "github.com/google/pprof/profile"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profiler"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/storage/client"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider"
	"github.com/yandex/perforator/perforator/internal/symbolizer/cli"
	"github.com/yandex/perforator/perforator/internal/symbolizer/proxy/server"
	"github.com/yandex/perforator/perforator/internal/symbolizer/symbolize"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/debuginfod"
	"github.com/yandex/perforator/perforator/pkg/linux/perfevent"
	"github.com/yandex/perforator/perforator/pkg/profile/python"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	"github.com/yandex/perforator/perforator/pkg/xelf"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

type recordOptions struct {
	logLevel string
	duration time.Duration
	debug    bool

	pids        []int
	cgroups     []string
	wholeSystem bool

	freq     uint64
	interval uint64

	event    string
	signals  bool
	walltime bool

	upload    bool
	uploadURL string

	renderFormat                  string
	flamegraphOptions             perforator.FlamegraphOptions
	profileSinkOptions            sinkOptions
	enableSymbolization           bool
	enableInterpreterStackMerging bool
	disablePerfMap                bool
	disablePerfMapJVM             bool
}

func (o *recordOptions) Bind(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.logLevel, "log-level", "l", "info", "Set log level")
	cmd.Flags().IntSliceVarP(&o.pids, "pid", "p", nil, "Process id to profile")
	cmd.Flags().StringSliceVarP(&o.cgroups, "cgroup", "G", nil, "Paths of cgroups to profile")
	cmd.Flags().BoolVarP(&o.wholeSystem, "whole-system", "a", false, "Profile whole system")
	cmd.Flags().Uint64VarP(&o.freq, "freq", "F", 99, "Profiling frequency")
	cmd.Flags().Uint64VarP(&o.interval, "count", "c", 0, "Profiling interval")
	cmd.Flags().StringVarP(&o.event, "event", "e", "", "Perf event to profile")
	cmd.Flags().DurationVarP(&o.duration, "duration", "d", 0, "Profiling duration")
	cmd.Flags().StringVarP(&o.renderFormat, "format", "f", "flamegraph", "Profile visualization format")
	cmd.Flags().BoolVarP(&o.debug, "debug", "", false, "Run perforator in debug mode")
	cmd.Flags().BoolVarP(&o.signals, "record-signals", "", false, "Record fatal signals")
	cmd.Flags().BoolVarP(&o.walltime, "record-walltime", "", false, "Record wall time")
	cmd.Flags().BoolVarP(&o.upload, "upload", "", false, "Upload profile to the public perforator backend")
	cmd.Flags().StringVarP(&o.uploadURL, "upload-url", "", "", "URL of the perforator backend")
	cmd.Flags().BoolVar(&o.enableSymbolization, "symbolize", true, "Enable profile symbolization")
	cmd.Flags().BoolVar(&o.enableInterpreterStackMerging, "merge-native-interpreter-stacks", true, "Enable interpreter and native stack merging")
	cmd.Flags().BoolVar(&o.disablePerfMap, "disable-perf-maps", false, "Disable perf map")
	cmd.Flags().BoolVar(&o.disablePerfMapJVM, "disable-perf-maps-jvm", false, "Disable perf map for JVM")

	cmd.MarkFlagsMutuallyExclusive("freq", "count")

	bindFlamegraphRenderOptions(cmd.Flags(), &o.flamegraphOptions)
	addSinkOptions(cmd, &o.profileSinkOptions)
}

func (o *recordOptions) postprocess(args []string) error {
	o.profileSinkOptions.postprocess()

	if !o.wholeSystem && len(o.pids) == 0 && len(o.cgroups) == 0 && len(args) == 0 {
		return fmt.Errorf("no profiling target defined")
	}

	return nil
}

func record(opts *recordOptions, args []string) error {
	startTime := time.Now()

	cliconf := &cli.Config{
		LogLevel: opts.logLevel,
		Timeout:  time.Hour * 24 * 365, // FIXME(sskvor): Allow to disable timeout
	}

	if opts.upload {
		cliconf.Client = &cli.ClientConfig{
			URL: opts.uploadURL,
		}
	}

	app, err := cli.New(cliconf)
	if err != nil {
		return fmt.Errorf("failed to setup CLI app: %w", err)
	}

	logger := app.Logger()
	ctx := app.Context()

	// let's validate the format before we run profiling
	format, err := makeRenderFormat(opts.renderFormat, &opts.flamegraphOptions, opts.enableSymbolization, opts.enableInterpreterStackMerging)
	if err != nil {
		return fmt.Errorf("failed to build render format: %w", err)
	}

	storage, err := runProfiler(ctx, logger, opts, args)
	if err != nil {
		return err
	}

	profile, err := symbolizeProfile(ctx, logger, storage, opts, format)
	if err != nil {
		return err
	}

	postProcessResults := python.PostprocessSymbolizedProfileWithPython(profile)
	if len(postProcessResults.Errors) > 0 {
		logger.Fmt().Debugf("Errors on merge python and native stacks: %v", errors.Join(postProcessResults.Errors...))
	}

	mergedStacksPercentage := 100 * float64(postProcessResults.MergedStacksCount) / float64(postProcessResults.MergedStacksCount+postProcessResults.UnmergedStacksCount)
	logger.Fmt().Debugf("Merged stacks percentage %.2f%%", mergedStacksPercentage)

	if opts.upload {
		profileID, taskID, err := uploadProfile(app, opts, profile, startTime)
		if err != nil {
			return err
		}

		buf, err := json.Marshal(map[string]string{
			"taskID":    taskID,
			"profileID": profileID,
		})
		if err != nil {
			return err
		}
		fmt.Print(string(buf))
	}

	err = renderProfile(ctx, logger, profile, opts, format)
	if err != nil {
		return err
	}

	return nil
}

func parsePerfEvents(opts *recordOptions) ([]config.PerfEventConfig, error) {
	if opts.event == "" {
		if opts.signals {
			// Do not setup default perf events in `perforator record --signals` mode.
			return nil, nil
		}
		opts.event = "CPUCycles"
	}

	cfg := config.PerfEventConfig{
		Type: perfevent.Type(opts.event),
	}

	if opts.interval != 0 {
		cfg.SampleRate = ptr.T(opts.interval)
	} else {
		cfg.Frequency = ptr.T(opts.freq)
	}

	return []config.PerfEventConfig{cfg}, nil
}

func runProfiler(ctx context.Context, logger xlog.Logger, opts *recordOptions, args []string) (*binaryStorage, error) {
	storage, err := newBinaryStorage(ctx, logger)
	if err != nil {
		return nil, err
	}

	registry := xmetrics.NewRegistry(xmetrics.WithFormat(xmetrics.FormatText))

	perfevents, err := parsePerfEvents(opts)
	if err != nil {
		return nil, err
	}

	prof, err := profiler.NewProfiler(&config.Config{
		Debug: opts.debug,
		BPF: machine.Config{
			Debug:         opts.debug,
			TraceLBR:      ptr.Bool(false),
			TraceSignals:  ptr.Bool(opts.signals),
			TraceWallTime: ptr.Bool(opts.walltime),
		},
		ProcessDiscovery: config.ProcessDiscoveryConfig{
			IgnoreUnrelatedProcesses: true,
		},
		Egress: config.EgressConfig{
			Interval: time.Second * 10,
		},
		SampleConsumer: config.SampleConsumerConfig{
			PerfBufferWatermark: ptr.Int(0),
		},
		PerfEvents:        perfevents,
		EnablePerfMaps:    ptr.Bool(!opts.disablePerfMap),
		EnablePerfMapsJVM: ptr.Bool(!opts.disablePerfMapJVM),
	}, logger.WithContext(ctx), registry, profiler.WithStorage(storage))

	if err != nil {
		return nil, fmt.Errorf("failed to initialize profiler: %w", err)
	}
	defer prof.Close()

	for _, pid := range opts.pids {
		err = prof.TracePid(pid, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to trace pid %d: %w", pid, err)
		}
	}
	for _, cgroup := range opts.cgroups {
		err = prof.AddCgroup(&profiler.CgroupConfig{
			Name: cgroup,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to trace cgroup %s: %w", cgroup, err)
		}
	}
	if opts.wholeSystem {
		err = prof.TraceWholeSystem(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to trace whole system: %w", err)
		}
	}

	// This context is cancelled when profiler is done.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	childdone := make(chan bool)

	if len(args) > 0 {
		g.Go(func() error {
			err := runSubProcess(ctx, args, func(pid int) error {
				return prof.TracePid(pid, map[string]string{"pid": fmt.Sprint(pid)})
			})
			if err != nil {
				logger.Error(ctx, "Subprocess failed", log.Error(err))
			}
			close(childdone)
			return nil
		})
	}

	g.Go(func() error {
		defer cancel()

		err = prof.Run(ctx)
		if err != nil {
			if !errors.Is(context.Cause(ctx), profiler.ErrStopped) {
				return fmt.Errorf("profiler failed: %w, cause: %w", err, context.Cause(ctx))
			}
		}

		return nil
	})

	g.Go(func() error {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)

		var timeout <-chan time.Time
		if opts.duration > 0 {
			timeout = time.After(opts.duration)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-signals:
			logger.Warn(ctx, "Stopping the profiler because SIGINT received")
		case <-timeout:
			logger.Warn(ctx, "Stopping the profiler because timeout reached")
		case <-childdone:
			logger.Warn(ctx, "Stopping the profiler because child subprocess finished")
		}

		signal.Stop(signals)

		// Stop our profiler gracefully.
		return prof.Stop(ctx)
	})

	if err = g.Wait(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	_ = registry.StreamMetrics(context.Background(), buf)
	logger.Debug(ctx, "Collected profiler metrics", log.ByteString("metrics", buf.Bytes()))

	return storage, nil
}

func symbolizeProfile(ctx context.Context, logger xlog.Logger, storage *binaryStorage, opts *recordOptions, format *perforator.RenderFormat) (*pprof.Profile, error) {
	sampleType := deduceProfileSampleType(opts)
	logger.Debug(ctx, "Deduced profile sample type", log.String("type", sampleType))

	profiles := make([]*pprof.Profile, 0, len(storage.profiles))
	for i, profile := range storage.profiles {
		_, err := profile.Profile.SampleIndexByName(sampleType)
		if err != nil {
			logger.Debug(ctx, "Skipped profile",
				log.Int("index", i),
				log.Any("labels", profile.Labels),
				log.Any("header", profile.Profile.SampleType),
			)
			continue
		}

		profile.Profile.PeriodType = &pprof.ValueType{}
		profiles = append(profiles, profile.Profile)
		logger.Debug(ctx, "Collected profile",
			log.Int("index", i),
			log.Any("labels", profile.Labels),
			log.Any("header", profile.Profile.SampleType),
		)
	}

	logger.Debug(ctx, "Merging profiles", log.Int("count", len(profiles)))
	profile, err := pprof.Merge(profiles)
	if err != nil {
		return nil, fmt.Errorf("failed to merge profiles: %w", err)
	}
	profile.DefaultSampleType = sampleType

	if !format.GetSymbolize().GetSymbolize() {
		return profile, nil
	}

	sym, err := symbolize.NewSymbolizer(logger, &nop.Registry{}, storage, storage, symbolize.SymbolizationModeDWARF)
	if err != nil {
		return nil, fmt.Errorf("failed to create symbolizer: %w", err)
	}

	profile, err = sym.SymbolizeStorageProfile(ctx, profile, format.GetSymbolize())
	if err != nil {
		return nil, fmt.Errorf("failed to symbolize profile: %w", err)
	}

	return profile, nil
}

func uploadProfile(app *cli.App, opts *recordOptions, profile *pprof.Profile, startTime time.Time) (profileID string, taskID string, err error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve self hostname: %w", err)
	}

	meta := &perforator.ProfileMeta{
		NodeID:    hostname,
		Timestamp: timestamppb.New(startTime),
	}

	profileID, taskID, err = app.Client().UploadRenderedProfile(app.Context(), meta, profile)
	if err != nil {
		return "", "", fmt.Errorf("failed to upload profile: %w", err)
	}

	return
}

func renderProfile(ctx context.Context, logger xlog.Logger, profile *pprof.Profile, opts *recordOptions, format *perforator.RenderFormat) error {
	sink, err := makeProfileSink(logger.Logger(), &opts.profileSinkOptions, format)
	if err != nil {
		return fmt.Errorf("failed to build profile sink: %w", err)
	}

	buf, err := server.RenderProfile(ctx, profile, format)
	if err != nil {
		return fmt.Errorf("failed to render profile: %w", err)
	}

	err = sink.Store(buf)
	if err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	return nil
}

func deduceProfileSampleType(opts *recordOptions) string {
	if opts.walltime {
		return sampletype.SampleTypeWall
	}
	if opts.signals {
		return sampletype.SampleTypeSignal
	}
	return sampletype.SampleTypeCPU
}

////////////////////////////////////////////////////////////////////////////////

type binaryStorage struct {
	logger     xlog.Logger
	binariesmu sync.Mutex
	profilesmu sync.Mutex
	binaries   map[string]*binary.SealedMultiHandle
	profiles   []client.LabeledProfile

	debuginfodClient *debuginfod.CachedClient
}

func newBinaryStorage(ctx context.Context, logger xlog.Logger) (*binaryStorage, error) {
	client, err := debuginfod.NewCachedClient(
		debuginfod.WithEnvConfig(),
		debuginfod.WithLogger(logger),
	)
	if err != nil {
		if errors.Is(err, debuginfod.ErrNoEndpoints) {
			client = nil
			logger.Debug(ctx, "No debuginfod endpoint defined, will not try to fetch binaries from debuginfod servers",
				log.NamedError("debuginfod_error", err),
			)
		} else {
			return nil, fmt.Errorf("failed to setup debuginfod client: %w", err)
		}
	}

	return &binaryStorage{
		logger:           logger.WithName("binarystorage"),
		binaries:         make(map[string]*binary.SealedMultiHandle),
		debuginfodClient: client,
	}, nil
}

func (s *binaryStorage) StoreBinary(ctx context.Context, buildID string, file binary.SealedFile) error {
	s.binariesmu.Lock()
	defer s.binariesmu.Unlock()

	handle, ok := s.binaries[buildID]
	if !ok {
		handle = &binary.SealedMultiHandle{}
		s.binaries[buildID] = handle
	}

	handle.AddHandles(file)
	return nil
}

func (s *binaryStorage) AnnounceBinaries(ctx context.Context, buildIDs []string) ([]string, error) {
	return buildIDs, nil
}

func (s *binaryStorage) StoreProfile(ctx context.Context, profile client.LabeledProfile) error {
	s.profilesmu.Lock()
	defer s.profilesmu.Unlock()

	s.profiles = append(s.profiles, profile)
	return nil
}

func (s *binaryStorage) Acquire(ctx context.Context, buildID string) (binaryprovider.FileHandle, error) {
	handle, ok := s.binaries[buildID]
	if !ok {
		return s.fetchSeparateDebugInfo(ctx, buildID)
	}

	file, err := handle.Unseal()
	if err != nil {
		return nil, err
	}

	buildInfo, err := xelf.ReadBuildInfo(file.GetFile())
	if err != nil || !buildInfo.HasDebugInfo {
		di, err := s.fetchSeparateDebugInfo(ctx, buildID)
		if err == nil {
			return di, nil
		}
	}

	return &dsoFileHandle{file, buildID}, nil
}

func (s *binaryStorage) fetchSeparateDebugInfo(ctx context.Context, buildID string) (h binaryprovider.FileHandle, err error) {
	s.logger.Debug(ctx, "Trying to fetch separate debug info", log.String("buildID", buildID))
	defer func() {
		if err == nil {
			s.logger.Info(ctx, "Fetched separate debug info",
				log.String("build_id", buildID),
				log.String("path", h.Path()),
			)
		} else {
			s.logger.Warn(ctx, "Failed to find separate debug info",
				log.String("build_id", buildID),
				log.Error(err),
			)
		}
	}()

	if s.debuginfodClient == nil {
		return nil, fmt.Errorf("no handle found")
	}

	file, err := s.debuginfodClient.OpenDebugInfo(ctx, buildID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s debug info: %w", buildID, err)
	}

	return &osFileHandle{file}, nil
}

func (s *binaryStorage) AcquireGSYM(ctx context.Context, buildID string) (binaryprovider.FileHandle, error) {
	return nil, fmt.Errorf("Not implemented")
}

////////////////////////////////////////////////////////////////////////////////

type dsoFileHandle struct {
	handle  binary.UnsealedFile
	buildID string
}

func (h *dsoFileHandle) Path() string {
	return fmt.Sprintf("/proc/self/fd/%d", h.handle.GetFile().Fd())
}

func (h *dsoFileHandle) WaitStored(ctx context.Context) error {
	return nil
}

func (h *dsoFileHandle) Close() {
	_ = h.handle.Close()
}

////////////////////////////////////////////////////////////////////////////////

type osFileHandle struct {
	file *os.File
}

func (h *osFileHandle) Path() string {
	return h.file.Name()
}

func (h *osFileHandle) WaitStored(ctx context.Context) error {
	return nil
}

func (h *osFileHandle) Close() {
	_ = h.file.Close()
}

////////////////////////////////////////////////////////////////////////////////

func runSubProcess(ctx context.Context, args []string, register func(int) error) error {
	child := exec.CommandContext(ctx, args[0], args[1:]...)
	child.Stderr = os.Stderr
	child.Stdout = os.Stdout
	child.Stdin = os.Stdin

	err := child.Start()
	if err != nil {
		return fmt.Errorf("failed to run subprocess: %w", err)
	}

	err = register(child.Process.Pid)
	if err != nil {
		return err
	}

	err = child.Wait()
	if err != nil {
		return fmt.Errorf("subprocess failed: %w", err)
	}

	return nil
}

func makeRecordCommand() *cobra.Command {
	opts := &recordOptions{}

	cmd := &cobra.Command{
		Use:   "record",
		Short: "record local profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := opts.postprocess(args)
			if err != nil {
				return err
			}
			return record(opts, args)
		},
	}

	opts.Bind(cmd)

	return cmd
}

func init() {
	rootCmd.AddCommand(makeRecordCommand())
}
