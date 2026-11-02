package cmd

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/yandex/perforator/perforator/internal/symbolizer/cli"
	"github.com/yandex/perforator/perforator/proto/lib/pagination"
	"github.com/yandex/perforator/perforator/proto/perforator"
	"github.com/yandex/perforator/perforator/symbolizer/pkg/client"
)

const clusterTopExperimental = "EXPERIMENTAL: Commands, flags, and output formats may change without backward compatibility guarantees."

type clusterTopOptions struct {
	url      string
	timeout  time.Duration
	logLevel string
	json     bool
	format   string
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
	common := &clusterTopOptions{}
	o := &clusterTopCommandOptions{filters: make(map[perforator.ClusterTopStringFilter_FieldName]*[]string)}
	cmd := &cobra.Command{
		Use:   "cluster-top",
		Short: "Inspect cluster-wide CPU hotspots (experimental)",
		Long: "Show the top functions, or filter exact function names and choose a breakdown.\n" +
			"Default grouping: function. Use --by or --group-by to change it.\n" +
			"All --by presets include function; --by lines also separates binaries and source positions.\n" +
			"Queries beyond the global function top require 1–10 exact --function values.\n" +
			"CPU is estimated average cores over the generation interval. Cumulative CPU may overlap.\n\n" + clusterTopExperimental,
		Example: "  perforator cluster-top\n" +
			"  perforator cluster-top --search 'foo'\n" +
			"  perforator cluster-top --function 'foo::bar' --by lines\n" +
			"  perforator cluster-top --function f --function g --by binaries --format json\n" +
			"  perforator cluster-top --function f --group-by function,service,build_id --order-by self_cpu:desc",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			plan, err := o.plan()
			if err != nil {
				return err
			}
			app, err := newClusterTopApp(cmd.Context(), common)
			if err != nil {
				return err
			}
			defer app.Shutdown()
			response, err := fetchClusterTop(app.Context(), app.Client().ClusterTop(), plan)
			if err != nil {
				return err
			}
			return writeClusterTopResult(cmd.OutOrStdout(), cmd.ErrOrStderr(), response, plan.groups, common.json)
		},
	}
	cmd.Flags().StringVar(&o.generation, "generation", "latest", "Generation ID or latest completed generation")
	cmd.Flags().StringVar(&o.by, "by", "functions", "Breakdown preset: lines, binaries, services, functions")
	cmd.Flags().StringSliceVar(&o.groups, "group-by", nil, "Group keys: function,service,build_id,commit_id,binary_path,filename,source_line,source_column")
	cmd.MarkFlagsMutuallyExclusive("by", "group-by")
	cmd.Flags().StringSliceVar(&o.order, "order-by", []string{"self_cpu:desc"}, "Sort by field:asc|desc; fields: self_cpu, cumulative_cpu or a key from --group-by/--by (default group key: function)")
	cmd.Flags().Uint64Var(&o.limit, "limit", 50, "Page size (1–1000)")
	cmd.Flags().Uint64Var(&o.offset, "offset", 0, "Number of rows to skip (up to 10000 for detailed queries); pin --generation across pages")
	for _, f := range []struct {
		name  string
		field perforator.ClusterTopStringFilter_FieldName
	}{
		{"function", perforator.ClusterTopStringFilter_Function}, {"service", perforator.ClusterTopStringFilter_Service},
		{"build-id", perforator.ClusterTopStringFilter_BuildID}, {"commit-id", perforator.ClusterTopStringFilter_CommitID},
		{"binary-path", perforator.ClusterTopStringFilter_BinaryPath}, {"filename", perforator.ClusterTopStringFilter_Filename},
	} {
		values := new([]string)
		o.filters[f.field] = values
		cmd.Flags().StringArrayVar(values, f.name, nil, fmt.Sprintf("Exact, case-sensitive %s filter; repeat for OR", f.name))
	}
	cmd.Flags().StringVar(&o.search, "search", "", "Substring in function names; global function top only")
	cmd.MarkFlagsMutuallyExclusive("search", "function")
	cmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		switch common.format {
		case "table":
		case "json":
			common.json = true
		default:
			return fmt.Errorf("--format must be table or json")
		}
		return nil
	}
	cmd.PersistentFlags().StringVar(&common.url, "url", "", "Perforator proxy URL")
	cmd.PersistentFlags().DurationVar(&common.timeout, "timeout", 60*time.Second, "Overall command timeout")
	cmd.PersistentFlags().StringVar(&common.logLevel, "log-level", "warn", "Logging level")
	cmd.PersistentFlags().BoolVar(&common.json, "json", false, "Write JSON instead of a human-readable table")
	cmd.PersistentFlags().StringVar(&common.format, "format", "table", "Output format: table or json")
	cmd.MarkFlagsMutuallyExclusive("json", "format")
	cmd.AddCommand(newClusterTopGenerationsCommand(common))
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

var clusterTopDimensions = map[string]perforator.ClusterTopDimension{
	"function":      perforator.ClusterTopDimension_Function,
	"service":       perforator.ClusterTopDimension_Service,
	"build_id":      perforator.ClusterTopDimension_BuildID,
	"commit_id":     perforator.ClusterTopDimension_CommitID,
	"binary_path":   perforator.ClusterTopDimension_BinaryPath,
	"filename":      perforator.ClusterTopDimension_Filename,
	"source_line":   perforator.ClusterTopDimension_SourceLine,
	"source_column": perforator.ClusterTopDimension_SourceColumn,
}

var clusterTopBreakdowns = map[string][]string{
	"functions": {"function"},
	"lines":     {"function", "build_id", "commit_id", "binary_path", "filename", "source_line", "source_column"},
	"binaries":  {"function", "build_id", "commit_id", "binary_path"},
	"services":  {"function", "service"},
}

type clusterTopCommandOptions struct {
	generation    string
	by            string
	search        string
	groups        []string
	order         []string
	filters       map[perforator.ClusterTopStringFilter_FieldName]*[]string
	limit, offset uint64
}

func (o *clusterTopCommandOptions) request() (*perforator.ClusterTopQueryRequest, []string, error) {
	req := &perforator.ClusterTopQueryRequest{Query: &perforator.ClusterTopQuery{}, Pagination: &pagination.Paginated{Limit: o.limit, Offset: o.offset}}
	if o.generation != "latest" {
		id, err := strconv.ParseUint(o.generation, 10, 32)
		if err != nil || id == 0 {
			return nil, nil, fmt.Errorf("--generation must be latest or a positive uint32 ID")
		}
		req.Query.Generation = uint32(id)
	}
	if o.limit == 0 {
		return nil, nil, fmt.Errorf("--limit must be positive")
	}
	groups := o.groups
	if groups == nil {
		var ok bool
		groups, ok = clusterTopBreakdowns[o.by]
		if !ok {
			return nil, nil, fmt.Errorf("unknown --by %q: use lines, binaries, services or functions", o.by)
		}
	}
	for _, name := range groups {
		dimension, ok := clusterTopDimensions[name]
		if !ok {
			return nil, nil, fmt.Errorf("unknown group dimension %q", name)
		}
		req.GroupBy = append(req.GroupBy, dimension)
	}
	// Iterate enum order so requests are deterministic regardless of flag order.
	for field := perforator.ClusterTopStringFilter_Function; field <= perforator.ClusterTopStringFilter_Filename; field++ {
		values := o.filters[field]
		if values == nil || len(*values) == 0 {
			continue
		}
		filter := &perforator.ClusterTopStringFilter{Field: field}
		if len(*values) == 1 {
			filter.Condition = &perforator.ClusterTopStringFilter_Exact{Exact: (*values)[0]}
		} else {
			filter.Condition = &perforator.ClusterTopStringFilter_In{In: &perforator.ClusterTopStringValues{Values: *values}}
		}
		req.Query.Filters = append(req.Query.Filters, &perforator.ClusterTopFieldFilter{Filter: &perforator.ClusterTopFieldFilter_String_{String_: filter}})
	}
	for _, value := range o.order {
		name, direction, ok := strings.Cut(value, ":")
		if !ok || (direction != "asc" && direction != "desc") {
			return nil, nil, fmt.Errorf("invalid --order-by %q: use field:asc|desc pairs", value)
		}
		order := &perforator.ClusterTopQueryOrder{Direction: perforator.ClusterTopQueryOrder_Ascending}
		if direction == "desc" {
			order.Direction = perforator.ClusterTopQueryOrder_Descending
		}
		switch name {
		case "self_cpu":
			order.Key = &perforator.ClusterTopQueryOrder_Metric{Metric: perforator.ClusterTopMetric_SelfCPU}
		case "cumulative_cpu":
			order.Key = &perforator.ClusterTopQueryOrder_Metric{Metric: perforator.ClusterTopMetric_CumulativeCPU}
		default:
			dimension, ok := clusterTopDimensions[name]
			if !ok {
				return nil, nil, fmt.Errorf("unknown ordering field %q", name)
			}
			order.Key = &perforator.ClusterTopQueryOrder_Dimension{Dimension: dimension}
		}
		req.OrderBy = append(req.OrderBy, order)
	}
	return req, groups, nil
}

type clusterTopReadPlan struct {
	request      *perforator.ClusterTopQueryRequest
	groups       []string
	aggregateRPC *clusterTopAggregateRPCOptions
}

func (o *clusterTopCommandOptions) plan() (*clusterTopReadPlan, error) {
	req, groups, err := o.request()
	if err != nil {
		return nil, err
	}
	plan := &clusterTopReadPlan{request: req, groups: groups}
	functions := o.filters[perforator.ClusterTopStringFilter_Function]
	hasFunctions := functions != nil && len(*functions) > 0
	// QueryClusterTop reads cluster_top_v3 and requires exact function names.
	// The function aggregate RPC reads the serving snapshot for global top and
	// substring search. The service aggregate RPC reads cluster_top_v3 but also
	// returns percentages. Use these RPCs only when they express the full query.
	aggregateRPC := &clusterTopAggregateRPCOptions{generation: o.generation, search: o.search, limit: o.limit, offset: o.offset}
	if len(req.OrderBy) == 1 && req.OrderBy[0].Direction == perforator.ClusterTopQueryOrder_Descending {
		switch req.OrderBy[0].GetMetric() {
		case perforator.ClusterTopMetric_SelfCPU:
			aggregateRPC.sort = "self"
		case perforator.ClusterTopMetric_CumulativeCPU:
			aggregateRPC.sort = "cumulative"
		}
	}
	if len(req.Query.Filters) == 0 && slices.Equal(groups, []string{"function"}) {
		if aggregateRPC.sort == "" {
			return nil, fmt.Errorf("global function top supports --order-by self_cpu:desc or cumulative_cpu:desc")
		}
		aggregateRPC.groupBy = "functions"
	} else if o.search != "" {
		return nil, fmt.Errorf("--search supports only the global function top without additional filters")
	} else if !hasFunctions {
		return nil, fmt.Errorf("this breakdown requires --function")
	} else if len(req.Query.Filters) == 1 && len(*functions) == 1 && strings.TrimSpace((*functions)[0]) != "" && aggregateRPC.sort != "" &&
		(slices.Equal(groups, []string{"service"}) || len(groups) == 2 && slices.Contains(groups, "service") && slices.Contains(groups, "function")) {
		aggregateRPC.groupBy, aggregateRPC.function = "services", (*functions)[0]
	}
	if aggregateRPC.groupBy != "" {
		if _, err := aggregateRPC.request(); err != nil {
			return nil, err
		}
		plan.aggregateRPC = aggregateRPC
	}
	return plan, nil
}

type clusterTopRow struct {
	Key           map[string]any `json:"key"`
	SelfCPU       float64        `json:"self_cpu"`
	CumulativeCPU float64        `json:"cumulative_cpu"`
	SelfPct       *float64       `json:"self_pct,omitempty"`
	CumulativePct *float64       `json:"cumulative_pct,omitempty"`
}

type clusterTopOutput struct {
	Generation      clusterTopGenerationOutput `json:"generation"`
	Rows            []clusterTopRow            `json:"rows"`
	Pagination      clusterTopPage             `json:"pagination"`
	PercentageBasis string                     `json:"percentage_basis,omitempty"`
}

func fetchClusterTop(ctx context.Context, api perforator.ClusterTopClient, p *clusterTopReadPlan) (*clusterTopOutput, error) {
	if p.aggregateRPC != nil {
		req, err := p.aggregateRPC.request()
		if err != nil {
			return nil, err
		}
		return fetchClusterTopAggregateRPC(ctx, api, p.aggregateRPC, req, p.groups)
	}
	response, err := queryClusterTopRows(ctx, api, p.request)
	if err != nil {
		return nil, err
	}
	generation, err := makeClusterTopGenerationOutput(response.GetGeneration())
	if err != nil {
		return nil, err
	}
	result := &clusterTopOutput{Generation: generation, Rows: make([]clusterTopRow, 0, len(response.GetRows())), Pagination: clusterTopPage{Offset: p.request.Pagination.Offset, Limit: p.request.Pagination.Limit, HasMore: response.GetHasMore()}}
	for _, row := range response.GetRows() {
		if row.GetMetrics() == nil {
			return nil, fmt.Errorf("server returned a row without metrics")
		}
		result.Rows = append(result.Rows, clusterTopRow{Key: clusterTopKeyValues(row.GetKey()), SelfCPU: row.Metrics.SelfCPU, CumulativeCPU: row.Metrics.CumulativeCPU})
	}
	if result.Pagination.HasMore {
		if len(result.Rows) == 0 {
			return nil, fmt.Errorf("server returned an empty page with has_more")
		}
		next := result.Pagination.Offset + uint64(len(result.Rows))
		if next <= 10000 {
			result.Pagination.NextOffset = &next
		}
	}
	return result, nil
}

func queryClusterTopRows(ctx context.Context, api perforator.ClusterTopClient, req *perforator.ClusterTopQueryRequest) (*perforator.ClusterTopQueryResponse, error) {
	if req.Query.Generation == 0 {
		generations, err := api.ListClusterTopGenerations(ctx, &perforator.ListClusterTopGenerationRequest{})
		if err != nil {
			return nil, err
		}
		generation, err := resolveClusterTopGeneration(generations.GetGenerations(), 0)
		if err != nil {
			return nil, err
		}
		req.Query.Generation = generation.ID
	}
	return api.QueryClusterTop(ctx, req)
}

func writeClusterTopResult(out, diagnostics io.Writer, result *clusterTopOutput, groups []string, jsonOutput bool) error {
	generation := result.Generation
	if generation.Status != "completed" {
		if _, err := fmt.Fprintf(diagnostics, "Generation %d is %s; data and pages may change.\n", generation.ID, generation.Status); err != nil {
			return err
		}
	}
	if jsonOutput {
		return writeClusterTopJSON(out, result)
	}
	if _, err := fmt.Fprintf(out, "Generation %d (%s), %s — %s\nEstimated average CPU (cores); cumulative CPU may overlap.\n",
		generation.ID, generation.Status, generation.From.Format(time.RFC3339), generation.To.Format(time.RFC3339)); err != nil {
		return err
	}
	var basis string
	switch result.PercentageBasis {
	case "generation_self_cpu":
		basis = "Percentages: both relative to the generation's total self CPU, including when searching."
	case "function_cpu_across_services":
		basis = "Percentages: relative to the function's respective self and cumulative totals across services."
	}
	if basis != "" {
		if _, err := fmt.Fprintln(out, basis); err != nil {
			return err
		}
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	headers := []string{"SELF CPU", "CUMULATIVE CPU"}
	if basis != "" {
		headers = append(headers, "SELF %", "CUMULATIVE %")
	}
	headers = append(headers, groups...)
	if _, err := fmt.Fprintln(w, strings.Join(headers, "\t")); err != nil {
		return err
	}
	for _, row := range result.Rows {
		values := []string{strconv.FormatFloat(row.SelfCPU, 'g', 6, 64), strconv.FormatFloat(row.CumulativeCPU, 'g', 6, 64)}
		if basis != "" {
			values = append(values, strconv.FormatFloat(*row.SelfPct, 'g', 6, 64), strconv.FormatFloat(*row.CumulativePct, 'g', 6, 64))
		}
		for _, group := range groups {
			value := "—"
			if v, ok := row.Key[group]; ok {
				if text, ok := v.(string); ok {
					value = strconv.Quote(text)
				} else {
					value = fmt.Sprint(v)
				}
			}
			values = append(values, value)
		}
		if _, err := fmt.Fprintln(w, strings.Join(values, "\t")); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if result.Pagination.HasMore {
		if result.Pagination.NextOffset == nil {
			_, err := fmt.Fprintln(out, "More results: narrow the filters to stay within the pagination limit.")
			return err
		}
		_, err := fmt.Fprintf(out, "More results: repeat with --generation %d --offset %d and the same filters, breakdown and order.\n", generation.ID, *result.Pagination.NextOffset)
		return err
	}
	return nil
}

func clusterTopKeyValues(key *perforator.ClusterTopQueryKey) map[string]any {
	values := make(map[string]any)
	if key == nil {
		return values
	}
	for name, value := range map[string]*string{
		"function": key.Function, "service": key.Service, "build_id": key.BuildID,
		"commit_id": key.CommitID, "binary_path": key.BinaryPath, "filename": key.Filename,
	} {
		if value != nil {
			values[name] = *value
		}
	}
	if key.SourceLine != nil {
		values["source_line"] = *key.SourceLine
	}
	if key.SourceColumn != nil {
		values["source_column"] = *key.SourceColumn
	}
	return values
}
