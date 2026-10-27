package cmd

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/perforator/proto/lib/pagination"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

type clusterTopQueryOptions struct {
	groupBy    string
	generation string
	sort       string
	search     string
	function   string
	limit      uint64
	offset     uint64
}

type clusterTopItem struct {
	Name          string  `json:"name"`
	SelfCPU       float64 `json:"self_cpu"`
	CumulativeCPU float64 `json:"cumulative_cpu"`
	SelfPct       float64 `json:"self_pct"`
	CumulativePct float64 `json:"cumulative_pct"`
}

type clusterTopPage struct {
	Offset     uint64  `json:"offset"`
	Limit      uint64  `json:"limit"`
	HasMore    bool    `json:"has_more"`
	NextOffset *uint64 `json:"next_offset,omitempty"`
}

type clusterTopQueryOutput struct {
	Generation clusterTopGenerationOutput `json:"generation"`
	Metric     string                     `json:"metric"`
	Sort       string                     `json:"sort"`
	Search     string                     `json:"search,omitempty"`
	Function   string                     `json:"function,omitempty"`
	GroupBy    string                     `json:"group_by"`
	Items      []clusterTopItem           `json:"items"`
	Pagination clusterTopPage             `json:"pagination"`
}

func newClusterTopFunctionsCommand(common *clusterTopOptions) *cobra.Command {
	opts := &clusterTopQueryOptions{groupBy: "functions"}
	cmd := &cobra.Command{
		Use:   "functions",
		Short: "Show the top functions or search their names",
		Long: "Show functions ordered by descending self or cumulative CPU. Search uses a substring of the function name.\n" +
			"Estimated average CPU over the generation interval.\n" +
			"Percentages are relative to the generation's total self cycles, even when searching; cumulative percentages need not sum to 100.\n\n" +
			clusterTopExperimental,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runClusterTopQuery(cmd, common, opts)
		},
	}
	cmd.Flags().StringVar(&opts.search, "search", "", "Substring to search in function names")
	addClusterTopQueryFlags(cmd, opts)
	return cmd
}

func newClusterTopServicesCommand(common *clusterTopOptions) *cobra.Command {
	opts := &clusterTopQueryOptions{groupBy: "services"}
	cmd := &cobra.Command{
		Use:   "services",
		Short: "Show services using an exact function",
		Long: "Show services ordered by descending self or cumulative CPU for an exact, full function name.\n" +
			"Estimated average CPU over the generation interval.\n" +
			"Self and cumulative percentages are relative to this function's respective totals across services.\n\n" +
			clusterTopExperimental,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runClusterTopQuery(cmd, common, opts)
		},
	}
	cmd.Flags().StringVar(&opts.function, "function", "", "Exact, full function name (required)")
	addClusterTopQueryFlags(cmd, opts)
	return cmd
}

func addClusterTopQueryFlags(cmd *cobra.Command, opts *clusterTopQueryOptions) {
	cmd.Flags().StringVar(&opts.generation, "generation", "latest", "Generation ID or latest (highest completed ID)")
	cmd.Flags().StringVar(&opts.sort, "sort", "self", "Sort descending by self or cumulative CPU")
	cmd.Flags().Uint64Var(&opts.limit, "limit", 50, "Page size (1–1000)")
	cmd.Flags().Uint64Var(&opts.offset, "offset", 0, "Number of entries to skip")
}

func (o *clusterTopQueryOptions) request() (*perforator.ClusterTopRequest, error) {
	req := &perforator.ClusterTopRequest{
		Pagination: &pagination.Paginated{Limit: o.limit, Offset: o.offset},
	}
	if o.generation != "latest" {
		id, err := strconv.ParseUint(o.generation, 10, 32)
		if err != nil || id == 0 {
			return nil, fmt.Errorf("--generation must be latest or a positive uint32 ID")
		}
		req.Generation = uint32(id)
	}
	switch o.sort {
	case "self":
		req.OrderBy = "self_time"
	case "cumulative":
		req.OrderBy = "cumulative_time"
	default:
		return nil, fmt.Errorf("--sort must be self or cumulative")
	}
	if o.limit == 0 || o.limit > 1000 {
		return nil, fmt.Errorf("--limit must be between 1 and 1000")
	}
	if o.offset > math.MaxInt64-o.limit {
		return nil, fmt.Errorf("--offset is too large")
	}
	if o.groupBy == "services" {
		if strings.TrimSpace(o.function) == "" {
			return nil, fmt.Errorf("--function must be an exact, non-empty function name")
		}
		req.FunctionPattern = &o.function
	} else if o.search != "" {
		req.FunctionPattern = &o.search
	}
	return req, nil
}

func resolveClusterTopGeneration(generations []*perforator.ClusterTopGeneration, id uint32) (clusterTopGenerationOutput, error) {
	var selected *perforator.ClusterTopGeneration
	for _, g := range generations {
		if g == nil {
			continue
		}
		if id != 0 && g.GetID() != id {
			continue
		}
		if id == 0 && g.GetGenerationStatus() != perforator.ClusterTopGenerationStatus_COMPLETED {
			continue
		}
		if selected == nil || g.GetID() > selected.GetID() {
			selected = g
		}
	}
	if selected == nil {
		if id == 0 {
			return clusterTopGenerationOutput{}, fmt.Errorf("no completed cluster top generation; use generations to list available IDs")
		}
		return clusterTopGenerationOutput{}, fmt.Errorf("cluster top generation %d is unavailable", id)
	}
	return makeClusterTopGenerationOutput(selected)
}

func runClusterTopQuery(cmd *cobra.Command, common *clusterTopOptions, opts *clusterTopQueryOptions) error {
	req, err := opts.request()
	if err != nil {
		return err
	}
	app, err := newClusterTopApp(cmd.Context(), common)
	if err != nil {
		return err
	}
	defer app.Shutdown()
	ctx, api := app.Context(), app.Client().ClusterTop()
	generations, err := api.ListClusterTopGenerations(ctx, &perforator.ListClusterTopGenerationRequest{})
	if err != nil {
		return err
	}
	generation, err := resolveClusterTopGeneration(generations.GetGenerations(), req.Generation)
	if err != nil {
		return err
	}
	req.Generation = generation.ID
	var response *perforator.ClusterTopResponse
	if opts.groupBy == "services" {
		response, err = api.GetClusterTopAggregatedByService(ctx, req)
	} else {
		response, err = api.GetClusterTopAggregatedByFunction(ctx, req)
	}
	if err != nil {
		return err
	}
	result := clusterTopQueryOutput{
		Generation: generation,
		Metric:     "estimated_average_cpu",
		Sort:       opts.sort,
		Search:     opts.search,
		Function:   opts.function,
		GroupBy:    opts.groupBy,
		Items:      make([]clusterTopItem, 0, len(response.GetInstances())),
		Pagination: clusterTopPage{Offset: opts.offset, Limit: opts.limit, HasMore: response.GetHasMore()},
	}
	hours := generation.To.Sub(generation.From).Hours()
	for _, entry := range response.GetInstances() {
		if entry == nil || entry.Count == nil {
			return fmt.Errorf("server returned an entry without counts")
		}
		count := entry.GetCount()
		item := clusterTopItem{
			Name: entry.GetName(), SelfCPU: count.GetSelf() / hours, CumulativeCPU: count.GetCumulative() / hours,
			SelfPct: count.GetSelfPct(), CumulativePct: count.GetCumulativePct(),
		}
		for _, value := range []float64{item.SelfCPU, item.CumulativeCPU, item.SelfPct, item.CumulativePct} {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return fmt.Errorf("server returned non-finite counts for %q", item.Name)
			}
		}
		result.Items = append(result.Items, item)
	}
	if result.Pagination.HasMore {
		if len(result.Items) == 0 {
			return fmt.Errorf("server returned an empty page with has_more")
		}
		next := opts.offset + uint64(len(result.Items))
		result.Pagination.NextOffset = &next
	}
	if generation.Status != "completed" {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Warning: generation %d is %s; snapshots and pages may change.\n", generation.ID, generation.Status); err != nil {
			return err
		}
	}
	if common.json {
		return writeClusterTopJSON(cmd.OutOrStdout(), result)
	}
	return writeClusterTopTable(cmd, result)
}

func writeClusterTopTable(cmd *cobra.Command, result clusterTopQueryOutput) error {
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(out, "Generation %d (%s), %s — %s\nEstimated average CPU over the generation interval.\n",
		result.Generation.ID, result.Generation.Status,
		result.Generation.From.Format(time.RFC3339), result.Generation.To.Format(time.RFC3339)); err != nil {
		return err
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	if result.Function != "" {
		if _, err := fmt.Fprintf(out, "Function: %q\n", result.Function); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "SELF CPU\tCUMULATIVE CPU\tSELF %\tCUMULATIVE %\tNAME"); err != nil {
		return err
	}
	for _, item := range result.Items {
		if _, err := fmt.Fprintf(w, "%.2f\t%.2f\t%.2f\t%.2f\t%q\n", item.SelfCPU, item.CumulativeCPU, item.SelfPct, item.CumulativePct, item.Name); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if result.Pagination.NextOffset != nil {
		_, err := fmt.Fprintf(out, "More results: repeat with --generation %d --offset %d and the same filters and sort.\n", result.Generation.ID, *result.Pagination.NextOffset)
		return err
	}
	return nil
}
