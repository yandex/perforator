package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/humantime"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope"
	"github.com/yandex/perforator/perforator/symbolizer/pkg/client"
)

var (
	// list microscopes
	allMicroscopes          bool
	user                    string
	startTimeListMicroscope string

	// create microscope
	podID                     string
	nodeID                    string
	duration                  time.Duration
	startTimeCreateMicroscope string
)

func makeSelectorForNewMicroscope(
	podID string,
	nodeID string,
	startTime time.Time,
	duration time.Duration,
) (string, error) {
	if duration > time.Hour {
		return "", fmt.Errorf("duration %s is more than 1 hour", duration.String())
	}
	builder := profilequerylang.NewBuilder().From(startTime).To(startTime.Add(duration))
	if podID != "" {
		builder.PodIDs(podID)
	}
	if nodeID != "" {
		builder.NodeIDs(nodeID)
	}

	return profilequerylang.SelectorToString(builder.Build())
}

var (
	microscopeCmd = &cobra.Command{
		Use:   "microscope {list | create} ...",
		Short: "Output existing microscopes or create a new one",
		Long:  "Microscope is a tool to save all profiles during some time interval from a given selector (e.g pod or node)",
	}

	listMicroscopesCmd = &cobra.Command{
		Use:   "list",
		Short: "List microscopes",
		RunE: func(_ *cobra.Command, _ []string) error {
			cli, err := makeCLI()
			if err != nil {
				return err
			}
			defer cli.Shutdown()
			ctx := cli.Context()

			if allMicroscopes {
				user = microscope.AllUsers
			}

			stTime, err := humantime.Parse(startTimeListMicroscope)
			if err != nil {
				return err
			}

			microscopes, err := cli.Client().ListMicroscopes(
				cli.Context(),
				&client.MicroscopesFilters{
					User:        user,
					StartsAfter: &stTime,
				},
				offset,
				limit,
			)
			if err != nil {
				return err
			}

			cli.Logger().Info(ctx, "Found microscopes",
				log.Int("count", len(microscopes)),
				log.UInt64("offset", offset),
				log.UInt64("limit", limit),
			)

			for _, scope := range microscopes {
				json, err := protojson.Marshal(scope)
				if err != nil {
					return fmt.Errorf("failed to format profile metainfo: %w", err)
				}
				fmt.Println(string(json))
			}

			if int(limit) <= len(microscopes) {
				cli.Logger().Info(ctx, "Use --limit or --offset to fetch more microscopes")
			}

			return nil
		},
	}

	createMicroscopeCmd = &cobra.Command{
		Use:   "create",
		Short: "Create microscope",
		RunE: func(_ *cobra.Command, _ []string) error {
			cli, err := makeCLI()
			if err != nil {
				return err
			}
			defer cli.Shutdown()

			stTime, err := humantime.Parse(startTimeCreateMicroscope)
			if err != nil {
				return err
			}

			selector, err := makeSelectorForNewMicroscope(podID, nodeID, stTime, duration)
			if err != nil {
				return err
			}

			id, err := cli.Client().CreateMicroscope(
				cli.Context(),
				selector,
			)
			if err != nil {
				return err
			}

			cli.Logger().Info(cli.Context(), "Created new microscope", log.String("id", id))

			return nil
		},
	}
)

func init() {
	commands := []*cobra.Command{listMicroscopesCmd, createMicroscopeCmd}

	for _, cmd := range commands {
		cmd.Flags().StringVar(&url, "url", "", "Perforator proxy URL")
		cmd.Flags().BoolVar(&insecure, "insecure", false, "Disable TLS")

		cmd.Flags().DurationVar(&timeout, "timeout", time.Minute*10, "Request timeout for proxy")
		cmd.Flags().StringVar(&logLevel, "log-level", "info", "Logging level, one of ('debug', 'info', 'warn', 'error')")
	}

	listMicroscopesCmd.Flags().Uint64VarP(&limit, "limit", "l", 500, "Limit for output")
	listMicroscopesCmd.Flags().Uint64VarP(&offset, "offset", "o", 0, "Offset for output")
	listMicroscopesCmd.Flags().BoolVarP(&allMicroscopes, "all", "a", false, "Output all microscopes (created by any user)")
	listMicroscopesCmd.Flags().StringVarP(&user, "user", "u", "", "Output some user microscopes (default is self)")
	listMicroscopesCmd.Flags().StringVarP(
		&startTimeListMicroscope,
		"start-time",
		"s",
		humantime.LongTimeAgo,
		`Start time to list microscopes from. Unix time in seconds, ISO8601, or HH:MM in the last 24 hours`,
	)

	createMicroscopeCmd.Flags().StringVarP(&podID, "pod-id", "p", "", "Pod id to microscope")
	createMicroscopeCmd.Flags().StringVarP(&nodeID, "node-id", "n", "", "Node id to microscope")
	createMicroscopeCmd.Flags().StringVarP(
		&startTimeCreateMicroscope,
		"start-time",
		"s",
		humantime.Now,
		`Start time to create microscopes from. Unix time in seconds, ISO8601, or HH:MM in the last 24 hours`,
	)
	createMicroscopeCmd.Flags().DurationVar(&duration, "duration", time.Hour, "Duration of microscope. Not more than 1 hour.")

	createMicroscopeCmd.MarkFlagsMutuallyExclusive("pod-id", "node-id")
	createMicroscopeCmd.MarkFlagsOneRequired("pod-id", "node-id")

	microscopeCmd.AddCommand(listMicroscopesCmd)
	microscopeCmd.AddCommand(createMicroscopeCmd)
	rootCmd.AddCommand(microscopeCmd)
}
