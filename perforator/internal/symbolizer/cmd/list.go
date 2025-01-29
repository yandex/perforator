package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/internal/symbolizer/cli"
	"github.com/yandex/perforator/perforator/pkg/humantime"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/symbolizer/pkg/client"
)

var (
	limit       uint64
	offset      uint64
	regex       string
	maxStaleAge string
	order       string

	url      string
	insecure bool

	timeout time.Duration
)

func makeCLI() (*cli.App, error) {
	app, err := cli.New(&cli.Config{
		LogLevel: logLevel,
		Timeout:  timeout,
		Client: &client.Config{
			URL:      url,
			Insecure: insecure,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CLI: %w", err)
	}
	return app, nil
}

var (
	listCmd = &cobra.Command{
		Use:   "list {profiles | services} ...",
		Short: "list profiles or services",
	}

	listProfilesCmd = &cobra.Command{
		Use:   "profiles",
		Short: "List service profiles meta information",
		RunE: func(_ *cobra.Command, _ []string) error {
			cli, err := makeCLI()
			if err != nil {
				return err
			}
			defer cli.Shutdown()

			from, to, err := humantime.ParseInterval(startTime, endTime)
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
					From(from).
					To(to)

				if service != "" {
					builder.Services(service)
				}

				selector, err = profilequerylang.SelectorToString(builder.Build())
				if err != nil {
					return fmt.Errorf("failed to construct selector from cli arguments: %w", err)
				}
			}

			metas, err := cli.Client().ListProfiles(
				cli.Context(),
				&client.ProfileFilters{
					FromTS:   from,
					ToTS:     to,
					Selector: selector,
				},
				offset,
				limit,
			)
			if err != nil {
				return err
			}

			cli.Logger().Info(cli.Context(),
				"Found profiles",
				log.String("selector", selector),
				log.Int("count", len(metas)),
				log.UInt64("offset", offset),
				log.UInt64("limit", limit),
			)

			for _, meta := range metas {
				json, err := protojson.Marshal(meta)
				if err != nil {
					return fmt.Errorf("failed to format profile metainfo: %w", err)
				}
				fmt.Println(string(json))
			}

			if int(limit) <= len(metas) {
				cli.Logger().Info(cli.Context(), "Use --limit or --offset to fetch more profiles")
			}

			return nil
		},
	}

	listServicesCmd = &cobra.Command{
		Use:   "services",
		Short: "List service names",
		RunE: func(_ *cobra.Command, _ []string) error {
			cli, err := makeCLI()
			if err != nil {
				return err
			}
			defer cli.Shutdown()

			var regexp *string
			if regex != "" {
				regexp = &regex
			}
			var pruneInterval *time.Duration
			if maxStaleAge != "" {
				interval, err := time.ParseDuration(maxStaleAge)
				if err != nil {
					return err
				}
				pruneInterval = &interval
			}

			services, err := cli.Client().ListServices(
				cli.Context(),
				offset,
				limit,
				regexp,
				pruneInterval,
				order,
			)
			if err != nil {
				return err
			}

			cli.Logger().Info(cli.Context(), "Found services",
				log.Int("count", len(services)),
				log.UInt64("offset", offset),
				log.UInt64("limit", limit),
				log.String("order", order),
			)

			for _, meta := range services {
				json, err := protojson.Marshal(meta)
				if err != nil {
					return fmt.Errorf("failed to format profile metainfo: %w", err)
				}
				fmt.Println(string(json))
			}

			if int(limit) <= len(services) {
				cli.Logger().Info(cli.Context(), "Use --limit or --offset to fetch more services")
			}

			return nil
		},
	}
)

func init() {
	commands := []*cobra.Command{listProfilesCmd, listServicesCmd}

	for _, cmd := range commands {
		cmd.Flags().Uint64VarP(&limit, "limit", "l", 500, "Limit for output")
		cmd.Flags().Uint64VarP(&offset, "offset", "o", 0, "Offset for output")

		cmd.Flags().StringVar(&url, "url", "", "Perforator proxy URL")
		cmd.Flags().BoolVar(&insecure, "insecure", false, "Disable TLS")

		cmd.Flags().DurationVar(&timeout, "timeout", time.Minute*10, "Request timeout for proxy")
		cmd.Flags().StringVar(&logLevel, "log-level", "info", "Logging level, one of ('debug', 'info', 'warn', 'error')")
	}

	listServicesCmd.Flags().StringVarP(&regex, "regex", "r", "", "Regular expression to filter service names by (RE2 syntax)")
	listServicesCmd.Flags().StringVar(&maxStaleAge, "max-stale-age", "", "Show services with max_timestamp > now() - max_stale_age")
	listServicesCmd.Flags().StringVar(&order, "order-by", "services", `Response will be ordered by services names or profiles count. One of ('services', 'profiles')`)

	// filters

	listProfilesCmd.Flags().StringVar(
		&service,
		"service",
		"",
		"List profiles of this service",
	)

	listProfilesCmd.Flags().StringSliceVar(
		&nodeIDs,
		"node-id",
		[]string{},
		"List profiles by host",
	)

	listProfilesCmd.Flags().StringSliceVar(
		&podIDs,
		"pod-id",
		[]string{},
		"List profiles by pod id",
	)

	listProfilesCmd.Flags().StringSliceVar(
		&buildIDs,
		"build-id",
		[]string{},
		"List profiles with locations from these build ids",
	)

	listProfilesCmd.Flags().StringSliceVar(
		&cpuModels,
		"cpu-model",
		[]string{},
		"List profiles by cpu model of host",
	)

	listProfilesCmd.Flags().StringSliceVar(
		&clusters,
		"dc",
		[]string{},
		"List profiles by dc",
	)

	listProfilesCmd.Flags().StringVar(
		&selector,
		"selector",
		"",
		"Selector (https://perforator.tech/docs/en/reference/querylang)",
	)

	listProfilesCmd.MarkFlagsMutuallyExclusive("selector", "service")
	listProfilesCmd.MarkFlagsMutuallyExclusive("selector", "build-id")
	listProfilesCmd.MarkFlagsMutuallyExclusive("selector", "node-id")
	listProfilesCmd.MarkFlagsMutuallyExclusive("selector", "pod-id")
	listProfilesCmd.MarkFlagsMutuallyExclusive("selector", "cpu-model")
	listProfilesCmd.MarkFlagsMutuallyExclusive("selector", "dc")

	listProfilesCmd.MarkFlagsOneRequired("service", "pod-id", "node-id", "build-id", "cpu-model", "dc", "selector")

	listProfilesCmd.Flags().StringSliceVar(
		&profilerVersions,
		"profiler-version",
		[]string{},
		"List profiles with specified profiler versions",
	)

	listProfilesCmd.Flags().StringVarP(
		&startTime,
		"start-time",
		"s",
		humantime.LongTimeAgo,
		`Start time to list profiles from. Unix time in seconds, ISO8601, or HH:MM in the last 24 hours`,
	)
	listProfilesCmd.Flags().StringVarP(
		&endTime,
		"end-time",
		"e",
		humantime.Now,
		`End time list profiles to. Unix time in seconds, ISO8601, or HH:MM in the last 24 hours`,
	)

	listCmd.AddCommand(listProfilesCmd)
	listCmd.AddCommand(listServicesCmd)
	rootCmd.AddCommand(listCmd)
}
