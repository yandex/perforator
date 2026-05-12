package cluster_top

import (
	"context"
	_ "net/http/pprof"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yandex/perforator/perforator/internal/symbolizer/proxy/services"
	clickhouse "github.com/yandex/perforator/perforator/pkg/storage/cluster_top"
	"github.com/yandex/perforator/perforator/pkg/storage/cluster_top/aggregated"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

var generationArgumentError = status.Errorf(codes.InvalidArgument, "Must provide non-zero generation")

var (
	_ services.GRPCService = (*APIService)(nil)
)

type APIService struct {
	l                           xlog.Logger
	clusterTopGenerationStorage clickhouse.Storage
}

func NewService(l xlog.Logger, s clickhouse.Storage) *APIService {

	return &APIService{
		l:                           l,
		clusterTopGenerationStorage: s,
	}
}

// GetClusterTopAggregatedByFunction implements perforator.GetClusterTopAggregatedByFunction
func (s *APIService) GetClusterTopAggregatedByFunction(ctx context.Context, req *perforator.ClusterTopRequest) (*perforator.ClusterTopResponse, error) {
	filter := &aggregated.Filter{
		FunctionFilter:          req.GetFunctionPattern(),
		FunctionFilterMatchMode: aggregated.SubstringMatch,
	}

	return s.getClusterTop(ctx, req, aggregated.GroupByFunction, filter)
}

func (s *APIService) getClusterTop(ctx context.Context, req *perforator.ClusterTopRequest, groupBy aggregated.GroupByMode, filter *aggregated.Filter) (*perforator.ClusterTopResponse, error) {
	generation := req.GetGeneration()
	if generation == 0 {
		return nil, generationArgumentError
	}

	limit := req.GetPagination().GetLimit()

	if limit == 0 {
		limit = aggregated.DefaultPageSize
	}

	var sortOrder aggregated.SortOrder

	switch req.GetOrderBy() {
	case "self_time":
		sortOrder = aggregated.SelfTimeSortOrder
	case "":
		sortOrder = aggregated.SelfTimeSortOrder
	case "cumulative_time":
		sortOrder = aggregated.CumulativeTimeSortOrder
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Unknown order by: %s", req.GetOrderBy())
	}

	offset := req.GetPagination().GetOffset()

	g, ctx := errgroup.WithContext(ctx)

	var entries []*aggregated.AggregationValue
	var total *aggregated.TotalCycles

	g.Go(func() error {
		var err error
		entries, err = s.clusterTopGenerationStorage.AggregateClusterTop(ctx, generation, filter, groupBy, util.Pagination{
			Offset: offset,
			Limit:  limit + 1,
		}, sortOrder)
		return err
	})

	g.Go(func() error {
		totalFunctionFilter := ""
		if filter != nil && filter.FunctionFilterMatchMode == aggregated.ExactMatch && filter.FunctionFilter != "" && groupBy == aggregated.GroupByService {
			totalFunctionFilter = filter.FunctionFilter
		}

		var err error
		total, err = s.clusterTopGenerationStorage.CountTotalCycles(ctx, generation, totalFunctionFilter)
		return err
	})

	err := g.Wait()

	if err != nil {
		return nil, err
	}

	hasMore := len(entries) > int(limit)

	if hasMore {
		entries = entries[0 : len(entries)-1]
	}

	res := aggregated.MapEntries(total, entries)

	return &perforator.ClusterTopResponse{
		Instances: res,
		HasMore:   hasMore,
	}, err
}

// GetClusterTopAggregatedByService implements perforator.GetClusterTopAggregatedByService
func (s *APIService) GetClusterTopAggregatedByService(ctx context.Context, req *perforator.ClusterTopRequest) (*perforator.ClusterTopResponse, error) {
	if req.FunctionPattern == nil {
		return nil, status.Errorf(codes.InvalidArgument, "For service aggregation must provide non-empty function search pattern")
	}

	filter := &aggregated.Filter{
		FunctionFilter:          req.GetFunctionPattern(),
		FunctionFilterMatchMode: aggregated.ExactMatch,
	}

	return s.getClusterTop(ctx, req, aggregated.GroupByService, filter)
}

// ListClusterTopGenerations implements perforator.ListClusterTopGenerations
func (s *APIService) ListClusterTopGenerations(ctx context.Context, req *perforator.ListClusterTopGenerationRequest) (*perforator.ListClusterTopGenerationResponse, error) {
	generations, err := s.clusterTopGenerationStorage.ListGenerations(ctx)
	return &perforator.ListClusterTopGenerationResponse{
		Generations: generations,
	}, err
}

func (s *APIService) Register(server *grpc.Server) error {
	perforator.RegisterClusterTopServer(server, s)
	return nil
}

func (s *APIService) RegisterHandler(ctx context.Context, mux *runtime.ServeMux) error {
	return perforator.RegisterClusterTopHandlerServer(ctx, mux, s)
}
