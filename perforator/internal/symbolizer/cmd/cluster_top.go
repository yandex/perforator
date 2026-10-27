package cmd

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/perforator/internal/symbolizer/cli"
	"github.com/yandex/perforator/perforator/proto/perforator"
	"github.com/yandex/perforator/perforator/symbolizer/pkg/client"
)

const clusterTopExperimental = "EXPERIMENTAL: Commands, flags, and output formats may change without backward compatibility guarantees."

type clusterTopOptions struct {
	url      string
	timeout  time.Duration
	logLevel string
	json     bool
}

type clusterTopGenerationOutput struct {
	ID     uint32    `json:"id"`
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
	Status string    `json:"status"`
}

func init() {
	rootCmd.AddCommand(newClusterTopCommand())
}

func newClusterTopCommand() *cobra.Command {
	opts := &clusterTopOptions{}
	cmd := &cobra.Command{
		Use:   "cluster-top",
		Short: "Inspect cluster-wide CPU hotspots (experimental)",
		Long:  "Inspect cluster-wide CPU hotspots.\n\n" + clusterTopExperimental,
	}
	cmd.PersistentFlags().StringVar(&opts.url, "url", "", "Perforator proxy URL")
	cmd.PersistentFlags().DurationVar(&opts.timeout, "timeout", 60*time.Second, "Overall command timeout")
	cmd.PersistentFlags().StringVar(&opts.logLevel, "log-level", "warn", "Logging level")
	cmd.PersistentFlags().BoolVar(&opts.json, "json", false, "Write JSON instead of a human-readable table")
	cmd.AddCommand(newClusterTopGenerationsCommand(opts))
	cmd.AddCommand(newClusterTopFunctionsCommand(opts))
	return cmd
}

func newClusterTopApp(ctx context.Context, o *clusterTopOptions) (*cli.App, error) {
	if o.timeout <= 0 {
		return nil, fmt.Errorf("--timeout must be positive")
	}
	return cli.NewWithContext(ctx, &cli.Config{
		LogLevel: o.logLevel,
		Timeout:  o.timeout,
		Client:   &client.Config{URL: o.url},
	})
}

func makeClusterTopGenerationOutput(g *perforator.ClusterTopGeneration) (clusterTopGenerationOutput, error) {
	if g == nil || g.GetID() == 0 {
		return clusterTopGenerationOutput{}, fmt.Errorf("server returned an invalid generation")
	}
	if err := g.GetFrom().CheckValid(); err != nil {
		return clusterTopGenerationOutput{}, fmt.Errorf("generation %d: invalid start time: %w", g.GetID(), err)
	}
	if err := g.GetTo().CheckValid(); err != nil {
		return clusterTopGenerationOutput{}, fmt.Errorf("generation %d: invalid end time: %w", g.GetID(), err)
	}
	from, to := g.GetFrom().AsTime(), g.GetTo().AsTime()
	if !to.After(from) {
		return clusterTopGenerationOutput{}, fmt.Errorf("generation %d: interval must be positive", g.GetID())
	}
	return clusterTopGenerationOutput{
		ID: g.GetID(), From: from, To: to,
		Status: strings.ToLower(g.GetGenerationStatus().String()),
	}, nil
}

func writeClusterTopJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func newClusterTopGenerationsCommand(opts *clusterTopOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "generations",
		Short: "List cluster top generations",
		Long:  "List generation IDs, data intervals and statuses.\n\n" + clusterTopExperimental,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newClusterTopApp(cmd.Context(), opts)
			if err != nil {
				return err
			}
			defer app.Shutdown()
			response, err := app.Client().ClusterTop().ListClusterTopGenerations(app.Context(), &perforator.ListClusterTopGenerationRequest{})
			if err != nil {
				return err
			}
			generations := make([]clusterTopGenerationOutput, 0, len(response.GetGenerations()))
			for _, g := range response.GetGenerations() {
				item, err := makeClusterTopGenerationOutput(g)
				if err != nil {
					return err
				}
				generations = append(generations, item)
			}
			slices.SortFunc(generations, func(a, b clusterTopGenerationOutput) int {
				return cmp.Compare(b.ID, a.ID)
			})
			if opts.json {
				return writeClusterTopJSON(cmd.OutOrStdout(), struct {
					Generations []clusterTopGenerationOutput `json:"generations"`
				}{generations})
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			if _, err := fmt.Fprintln(w, "ID\tFROM (UTC)\tTO (UTC)\tSTATUS"); err != nil {
				return err
			}
			for _, g := range generations {
				if _, err := fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", g.ID, g.From.Format(time.RFC3339), g.To.Format(time.RFC3339), g.Status); err != nil {
					return err
				}
			}
			return w.Flush()
		},
	}
}
