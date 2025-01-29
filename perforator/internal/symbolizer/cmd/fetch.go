package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/pkg/humantime"
	"github.com/yandex/perforator/perforator/pkg/must"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/render"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/pkg/xpflag"
	proto "github.com/yandex/perforator/perforator/proto/perforator"
	"github.com/yandex/perforator/perforator/symbolizer/pkg/client"
)

var (
	logLevel   string
	startTime  string
	endTime    string
	maxSamples uint32
	profileID  string

	format                        string
	pgoFormat                     string
	flamegraphOptions             client.FlamegraphOptions
	profileSinkOptions            sinkOptions
	enableSymbolization           bool
	enableInterpreterStackMerging bool

	selector         string
	service          string
	podIDs           = []string{}
	nodeIDs          = []string{}
	buildIDs         = []string{}
	cpuModels        = []string{}
	clusters         = []string{}
	profilerVersions = []string{}
)

func makeRenderFormat(format string, options *client.FlamegraphOptions, enableSymbolization, enableStackMerge bool) (*proto.RenderFormat, error) {
	switch format {
	case "flamegraph", "flame", "fg":
		return &proto.RenderFormat{
			Symbolize: &proto.SymbolizeOptions{
				Symbolize: ptr.Bool(enableSymbolization),
			},
			Postprocessing: &proto.PostprocessOptions{
				MergePythonAndNativeStacks: ptr.Bool(enableStackMerge),
			},
			Format: &proto.RenderFormat_Flamegraph{
				Flamegraph: options,
			},
		}, nil

	case "pprof":
		return &proto.RenderFormat{
			Symbolize: &proto.SymbolizeOptions{
				Symbolize: ptr.Bool(enableSymbolization),
			},
			Postprocessing: &proto.PostprocessOptions{
				MergePythonAndNativeStacks: ptr.Bool(enableStackMerge),
			},
			Format: &proto.RenderFormat_RawProfile{
				RawProfile: &proto.RawProfileOptions{},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsuppported format %s", format)
	}
}

type sinkOptions struct {
	outputPath     string
	serveAddress   string
	disableBrowser bool
}

func (o *sinkOptions) postprocess() {
	if o.outputPath == "" && o.serveAddress == "" {
		o.serveAddress = ":0"
	}
}

func makeProfileSink(log log.Logger, options *sinkOptions, format *proto.RenderFormat) (ProfileSink, error) {
	var sinkLog = log.WithName("sink")

	if options.outputPath != "" {
		return MakeFileSink(sinkLog, options.outputPath)
	}

	if options.serveAddress != "" {
		if format.GetRawProfile() != nil {
			return MakePProfSink(sinkLog, options.serveAddress, !options.disableBrowser)
		} else {
			return MakeHTTPSink(sinkLog, options.serveAddress, !options.disableBrowser)
		}
	}

	return nil, errors.New("unsupported render format")
}

func profileLogFields(meta *proto.ProfileMeta) []log.Field {
	host := "<unknown>"
	if actualHost, ok := meta.Attributes["host"]; ok {
		host = actualHost
	}
	pod := "<unknown>"
	if podLabel, ok := meta.Attributes["pod"]; ok {
		pod = podLabel
	}

	res := []log.Field{}

	if id := meta.ProfileID; id != "" {
		res = append(res, log.String("id", id))
	} else if profileID != "" {
		res = append(res, log.String("id", profileID))
	}

	res = append(res,
		log.String("service", meta.Service),
		log.String("pod", pod),
		log.String("host", host),
		log.Time("timestamp", meta.Timestamp.AsTime().Local()),
	)

	return res
}

func getProfile(
	ctx context.Context,
	log xlog.Logger,
	client *client.Client,
	id string,
	format *client.RenderFormat,
) ([]byte, error) {
	profile, meta, err := client.GetProfile(ctx, profileID, format)
	if err != nil {
		return nil, err
	}

	log.Info(ctx, "Fetched profile", profileLogFields(meta)...)

	return profile, err
}

func mergeProfiles(
	ctx context.Context,
	logger xlog.Logger,
	proxyClient *client.Client,
	filters client.ProfileFilters,
	maxSamples uint32,
	format *client.RenderFormat,
) ([]byte, error) {
	profile, metas, err := proxyClient.MergeProfiles(
		ctx,
		&client.MergeProfilesRequest{
			ProfileFilters: filters,
			MaxSamples:     maxSamples,
			Format:         format,
		},
		false,
	)
	if err != nil {
		return nil, err
	}

	logger.Info(ctx,
		"Fetched profile",
		log.Time("start", filters.FromTS),
		log.Time("end", filters.ToTS),
		log.Int("nprofiles", len(metas)),
		log.String("selector", filters.Selector),
	)

	for _, meta := range metas {
		logger.Debug(ctx, "Raw profile", profileLogFields(meta)...)
	}

	return profile, nil
}

func fetchProfile() error {
	cli, err := makeCLI()
	if err != nil {
		return err
	}
	defer cli.Shutdown()

	format, err := makeRenderFormat(format, &flamegraphOptions, enableSymbolization, enableInterpreterStackMerging)
	if err != nil {
		return err
	}

	var profile []byte
	if profileID == "" {
		startTime, endTime, err := humantime.ParseInterval(startTime, endTime)
		if err != nil {
			return err
		}

		if selector == "" {
			builder := profilequerylang.NewBuilder().
				BuildIDs(buildIDs...).
				NodeIDs(nodeIDs...).
				PodIDs(podIDs...).
				CPUs(cpuModels...).
				Clusters(clusters...).
				ProfilerVersions(profilerVersions...).
				From(startTime).
				To(endTime)

			if service != "" {
				builder.Services(service)
			}

			var err error
			selector, err = profilequerylang.SelectorToString(builder.Build())
			if err != nil {
				return fmt.Errorf("failed to construct selector from cli arguments: %w", err)
			}
		}

		profile, err = mergeProfiles(
			cli.Context(),
			cli.Logger(),
			cli.Client(),
			client.ProfileFilters{
				Selector: selector,
				FromTS:   startTime,
				ToTS:     endTime,
			},
			maxSamples,
			format,
		)
		if err != nil {
			return err
		}
	} else {
		profile, err = getProfile(cli.Context(), cli.Logger(), cli.Client(), profileID, format)
		if err != nil {
			return err
		}
	}

	sink, err := makeProfileSink(cli.Logger().WithContext(cli.Context()), &profileSinkOptions, format)
	if err != nil {
		return err
	}

	err = sink.Store(profile)
	if err != nil {
		return err
	}

	return nil
}

func fetchDiffProfile(args []string) error {
	cli, err := makeCLI()
	if err != nil {
		return err
	}
	defer cli.Shutdown()

	format, err := makeRenderFormat(format, &flamegraphOptions, enableSymbolization, enableInterpreterStackMerging)
	if err != nil {
		return err
	}

	sink, err := makeProfileSink(cli.ContextLogger(), &profileSinkOptions, format)
	if err != nil {
		return err
	}

	var interval *proto.TimeInterval
	if startTime != "" || endTime != "" {
		startTime, endTime, err := humantime.ParseInterval(startTime, endTime)
		if err != nil {
			return err
		}

		interval = &proto.TimeInterval{
			From: timestamppb.New(startTime),
			To:   timestamppb.New(endTime),
		}
	}

	profile, err := cli.Client().DiffProfiles(cli.Context(), &proto.DiffProfilesRequest{
		BaselineQuery: &proto.ProfileQuery{
			Selector:     args[0],
			MaxSamples:   maxSamples,
			TimeInterval: interval,
		},
		DiffQuery: &proto.ProfileQuery{
			Selector:     args[1],
			MaxSamples:   maxSamples,
			TimeInterval: interval,
		},
		SymbolizeOptions: format.Symbolize,
		RenderFormat:     format,
	}, false)
	if err != nil {
		return err
	}

	err = sink.Store(profile)
	if err != nil {
		return err
	}

	return nil
}

func makePGORenderFormat(format string) (*proto.PGOProfileFormat, error) {
	switch format {
	case "autofdo":
		return &proto.PGOProfileFormat{
			Format: &proto.PGOProfileFormat_AutoFDO{},
		}, nil

	case "bolt":
		return &proto.PGOProfileFormat{
			Format: &proto.PGOProfileFormat_Bolt{},
		}, nil

	default:
		return nil, fmt.Errorf("unsuppported pgo format %s", format)
	}
}

func fetchPGOProfile(args []string) error {
	cli, err := makeCLI()
	if err != nil {
		return err
	}
	defer cli.Shutdown()

	sink, err := makeProfileSink(cli.ContextLogger(), &profileSinkOptions, nil)
	if err != nil {
		return err
	}

	format, err := makePGORenderFormat(pgoFormat)
	if err != nil {
		return err
	}

	profile, PGOMeta, err := cli.Client().GetPGOProfile(
		cli.Context(),
		args[0],
		format,
		false,
	)
	if err != nil {
		return err
	}

	if PGOMeta == nil {
		return fmt.Errorf("failed to parse spgo-profile metadata")
	}
	cli.Logger().Info(cli.Context(),
		"Fetched PGO profile",
		log.String("Service", args[0]),
		log.String("GuessedBuildId", PGOMeta.GetGuessedBuildID()),
		log.UInt64("TotalProfiles", PGOMeta.GetTotalProfiles()),
		log.UInt64("TotalSamples", PGOMeta.GetTotalSamples()),
		log.UInt64("TotalBranches", PGOMeta.GetTotalBranches()),
		log.UInt64("BogusLBREntries", PGOMeta.GetBogusLbrEntries()),
		log.UInt64("AddressCountMapSize", PGOMeta.GetAddressCountMapSize()),
		log.UInt64("BranchCountMapSize", PGOMeta.GetBranchCountMapSize()),
		log.UInt64("RangeCountMapsize", PGOMeta.GetRangeCountMapSize()),
		log.Float32("TakenBranchesToExecutableBytesRatio", PGOMeta.GetTakenBranchesToExecutableBytesRatio()),
	)

	err = sink.Store(profile)
	if err != nil {
		return err
	}

	return nil
}

func postprocessArgs() {
	profileSinkOptions.postprocess()
}

var (
	fetchCmd = &cobra.Command{
		Use:   "fetch [selector]",
		Short: "Fetch aggregated profile",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			postprocessArgs()
			return fetchProfile()
		},
	}

	diffCmd = &cobra.Command{
		Use:   "diff [old-selector] [new-selector]",
		Short: "Build differential profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			postprocessArgs()
			return fetchDiffProfile(args)
		},
	}

	pgoCmd = &cobra.Command{
		Use:   "pgo [Service]",
		Short: "Fetch PGO-profile for the Service",
		Long: `Perforator supports creating sampling-PGO profile for binaries, which one might feed into
subsequent compilation via '-fprofile-sample-use=<path-to-spgo-profile>'.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			postprocessArgs()
			return fetchPGOProfile(args)
		},
	}
)

func addCommonProxyFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&url,
		"url",
		"",
		"Perforator proxy URL",
	)

	cmd.Flags().BoolVar(
		&insecure,
		"insecure",
		false,
		"Disable TLS",
	)

	cmd.Flags().DurationVar(
		&timeout,
		"timeout",
		time.Minute*10,
		"Request timeout for proxy",
	)
}

func addLoggingFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&logLevel,
		"log-level",
		"info",
		"Logging level, one of ('debug', 'info', 'warn', 'error')",
	)
}

func addSinkOptions(cmd *cobra.Command, opts *sinkOptions) {
	cmd.Flags().StringVarP(
		&opts.outputPath,
		"output",
		"o",
		"",
		"Output path",
	)
	cmd.Flags().StringVarP(
		&opts.serveAddress,
		"serve",
		"S",
		"",
		"Address to serve rendered profile at",
	)
	cmd.Flags().BoolVar(
		&opts.disableBrowser,
		"no-browser",
		false,
		"Do not try to open served profile in browser",
	)
	must.Must(cmd.MarkFlagFilename("output"))
	cmd.MarkFlagsMutuallyExclusive("serve", "output")
}

func addCommonSelectorOptions(cmd *cobra.Command) {
	cmd.Flags().Uint32VarP(
		&maxSamples,
		"max-profiles",
		"m",
		10,
		"Maximum amount of aggregated profiles (approximate, sampling is deterministic because of hashes)",
	)

	cmd.Flags().StringVarP(
		&startTime,
		"start",
		"s",
		humantime.LongTimeAgo,
		`Start time to aggregate from. Unix time in seconds, ISO8601, HH:MM in the last 24 hours, or "now - 1d2h3m4s"`,
	)

	cmd.Flags().StringVarP(
		&endTime,
		"end",
		"e",
		humantime.Now,
		`End time to aggregate to. Unix time in seconds, ISO8601, HH:MM in the last 24 hours, or "now - 1d2h3m4s"`,
	)
}

func addFlamegraphRenderOptions(cmd *cobra.Command) {
	bindFlamegraphRenderOptions(cmd.Flags(), &flamegraphOptions)

	cmd.Flags().BoolVar(
		&enableSymbolization,
		"symbolize",
		true,
		"Enable profile symbolization",
	)
	cmd.Flags().BoolVar(
		&enableInterpreterStackMerging,
		"merge-native-interpreter-stacks",
		true,
		"Enable native and interpreter stack merging",
	)
}

func bindFlamegraphRenderOptions(flags *pflag.FlagSet, options *client.FlamegraphOptions) {
	flags.Float64Var(
		maybe(&options.MinWeight),
		"flamegraph-min-weight",
		0.00001,
		"Minimum relative sample weight to render",
	)
	flags.Uint32Var(
		maybe(&options.MaxDepth),
		"flamegraph-max-depth",
		256,
		"Maximum stack depth",
	)
	flags.BoolVar(
		maybe(&options.ShowLineNumbers),
		"flamegraph-line-numbers",
		false,
		"Show line numbers in the flamegraph",
	)
	flags.BoolVar(
		maybe(&options.ShowFileNames),
		"flamegraph-show-file-names",
		true,
		"Show file names in the flamegraph",
	)

	addressRenderPolicies := "[" + strings.Join([]string{
		string(render.RenderAddressesNever),
		string(render.RenderAddressesUnsymbolized),
		string(render.RenderAddressesAlways),
	}, ", ") + "]"
	addressRenderPolicy := xpflag.NewFunc(func(val string) error {
		switch render.AddressRenderPolicy(val) {
		case render.RenderAddressesNever:
			options.RenderAddresses = ptr.T(proto.FlamegraphOptions_RenderAddressesNever)
			return nil
		case render.RenderAddressesUnsymbolized:
			options.RenderAddresses = ptr.T(proto.FlamegraphOptions_RenderAddressesUnsymbolized)
			return nil
		case render.RenderAddressesAlways:
			options.RenderAddresses = ptr.T(proto.FlamegraphOptions_RenderAddressesAlways)
			return nil
		default:
			return fmt.Errorf("unexpected address render policy %s, expected one of %s", val, addressRenderPolicies)
		}
	})
	flags.Var(
		addressRenderPolicy,
		"flamegraph-show-addresses",
		"Show addresses inside flamegraph, one of "+addressRenderPolicies,
	)
}

func maybe[T any, P **T](ptr P) *T {
	if *ptr == nil {
		*ptr = new(T)
	}
	return *ptr
}

func setupFetchCmd() *cobra.Command {
	addCommonProxyFlags(fetchCmd)
	addLoggingFlags(fetchCmd)
	addCommonSelectorOptions(fetchCmd)
	addFlamegraphRenderOptions(fetchCmd)
	addSinkOptions(fetchCmd, &profileSinkOptions)

	// Profile aggregation options
	fetchCmd.Flags().StringVar(
		&profileID,
		"id",
		"",
		"Profile ID to fetch",
	)

	fetchCmd.Flags().StringVar(
		&service,
		"service",
		"",
		"Aggregate profiles of this service",
	)

	fetchCmd.Flags().StringSliceVar(
		&nodeIDs,
		"node-id",
		[]string{},
		"Aggregate profiles by host",
	)

	fetchCmd.Flags().StringSliceVar(
		&podIDs,
		"pod-id",
		[]string{},
		"Aggregate profiles by pod id",
	)

	fetchCmd.Flags().StringSliceVar(
		&buildIDs,
		"build-id",
		[]string{},
		"Aggregate profiles with locations from these build ids",
	)

	fetchCmd.Flags().StringSliceVar(
		&cpuModels,
		"cpu-model",
		[]string{},
		"Aggregate profiles by cpu model of host",
	)

	fetchCmd.Flags().StringSliceVar(
		&clusters,
		"dc",
		[]string{},
		"Aggregate profiles by dc",
	)

	fetchCmd.Flags().StringVar(
		&selector,
		"selector",
		"",
		"Selector (https://perforator.tech/docs/en/reference/querylang)",
	)

	fetchCmd.MarkFlagsMutuallyExclusive("id", "service")
	fetchCmd.MarkFlagsMutuallyExclusive("id", "build-id")
	fetchCmd.MarkFlagsMutuallyExclusive("id", "node-id")
	fetchCmd.MarkFlagsMutuallyExclusive("id", "pod-id")
	fetchCmd.MarkFlagsMutuallyExclusive("id", "cpu-model")
	fetchCmd.MarkFlagsMutuallyExclusive("id", "selector")
	fetchCmd.MarkFlagsMutuallyExclusive("id", "dc")
	fetchCmd.MarkFlagsMutuallyExclusive("selector", "service")
	fetchCmd.MarkFlagsMutuallyExclusive("selector", "build-id")
	fetchCmd.MarkFlagsMutuallyExclusive("selector", "node-id")
	fetchCmd.MarkFlagsMutuallyExclusive("selector", "pod-id")
	fetchCmd.MarkFlagsMutuallyExclusive("selector", "cpu-model")
	fetchCmd.MarkFlagsMutuallyExclusive("selector", "dc")
	fetchCmd.MarkFlagsOneRequired("id", "service", "pod-id", "node-id", "build-id", "cpu-model", "dc", "selector")

	fetchCmd.Flags().StringSliceVar(
		&profilerVersions,
		"profiler-version",
		[]string{},
		"Aggregate profiles by profiler version",
	)

	// profile format
	fetchCmd.Flags().StringVarP(
		&format,
		"format",
		"f",
		"flamegraph",
		"Format of the profile (pprof, flamegraph or pgo)",
	)

	return fetchCmd
}

func setupDiffCmd() *cobra.Command {
	addCommonProxyFlags(diffCmd)
	addLoggingFlags(diffCmd)
	addCommonSelectorOptions(diffCmd)
	addFlamegraphRenderOptions(diffCmd)
	addSinkOptions(diffCmd, &profileSinkOptions)
	return diffCmd
}

func setupPGOCmd() *cobra.Command {
	addCommonProxyFlags(pgoCmd)
	addLoggingFlags(pgoCmd)
	addSinkOptions(pgoCmd, &profileSinkOptions)

	// profile format
	pgoCmd.Flags().StringVarP(
		&pgoFormat,
		"format",
		"f",
		"autofdo",
		"Format of the profile (autofdo or bolt)",
	)

	return pgoCmd
}

func init() {
	rootCmd.AddCommand(setupFetchCmd())
	rootCmd.AddCommand(setupDiffCmd())
	rootCmd.AddCommand(setupPGOCmd())
}
