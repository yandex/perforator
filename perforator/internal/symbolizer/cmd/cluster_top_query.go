package cmd

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/yandex/perforator/perforator/proto/lib/pagination"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

type clusterTopAggregateRPCOptions struct {
	groupBy    string
	generation string
	sort       string
	search     string
	function   string
	limit      uint64
	offset     uint64
}

type clusterTopPage struct {
	Offset     uint64  `json:"offset"`
	Limit      uint64  `json:"limit"`
	HasMore    bool    `json:"has_more"`
	NextOffset *uint64 `json:"next_offset,omitempty"`
}

func (o *clusterTopAggregateRPCOptions) request() (*perforator.ClusterTopRequest, error) {
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

func fetchClusterTopAggregateRPC(ctx context.Context, api perforator.ClusterTopClient, opts *clusterTopAggregateRPCOptions, req *perforator.ClusterTopRequest, groups []string) (*clusterTopOutput, error) {
	generations, err := api.ListClusterTopGenerations(ctx, &perforator.ListClusterTopGenerationRequest{})
	if err != nil {
		return nil, err
	}
	generation, err := resolveClusterTopGeneration(generations.GetGenerations(), req.Generation)
	if err != nil {
		return nil, err
	}
	req.Generation = generation.ID
	var response *perforator.ClusterTopResponse
	if opts.groupBy == "services" {
		response, err = api.GetClusterTopAggregatedByService(ctx, req)
	} else {
		response, err = api.GetClusterTopAggregatedByFunction(ctx, req)
	}
	if err != nil {
		return nil, err
	}
	result := clusterTopOutput{
		Generation:      generation,
		Rows:            make([]clusterTopRow, 0, len(response.GetInstances())),
		Pagination:      clusterTopPage{Offset: opts.offset, Limit: opts.limit, HasMore: response.GetHasMore()},
		PercentageBasis: "generation_self_cpu",
	}
	if opts.groupBy == "services" {
		result.PercentageBasis = "function_cpu_across_services"
	}
	hours := generation.To.Sub(generation.From).Hours()
	for _, entry := range response.GetInstances() {
		if entry == nil || entry.Count == nil {
			return nil, fmt.Errorf("server returned an entry without counts")
		}
		count := entry.GetCount()
		key := make(map[string]any)
		for _, group := range groups {
			value := entry.GetName()
			if opts.groupBy == "services" && group == "function" {
				value = opts.function
			}
			if value != "" {
				key[group] = value
			}
		}
		selfPct, cumulativePct := count.GetSelfPct(), count.GetCumulativePct()
		row := clusterTopRow{
			Key: key, SelfCPU: count.GetSelf() / hours, CumulativeCPU: count.GetCumulative() / hours,
			SelfPct: &selfPct, CumulativePct: &cumulativePct,
		}
		for _, value := range []float64{row.SelfCPU, row.CumulativeCPU, selfPct, cumulativePct} {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("server returned non-finite counts for %q", entry.GetName())
			}
		}
		result.Rows = append(result.Rows, row)
	}
	if result.Pagination.HasMore {
		if len(result.Rows) == 0 {
			return nil, fmt.Errorf("server returned an empty page with has_more")
		}
		next := opts.offset + uint64(len(result.Rows))
		result.Pagination.NextOffset = &next
	}
	return &result, nil
}
