package aggregated

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/perforator/proto/lib/pagination"
	pb "github.com/yandex/perforator/perforator/proto/perforator"
)

func request() *pb.ClusterTopQueryRequest {
	return &pb.ClusterTopQueryRequest{
		Query: &pb.ClusterTopQuery{Generation: 42,
			Filters: []*pb.ClusterTopFieldFilter{{Filter: &pb.ClusterTopFieldFilter_String_{String_: &pb.ClusterTopStringFilter{Field: pb.ClusterTopStringFilter_Function, Condition: &pb.ClusterTopStringFilter_Exact{Exact: "worker"}}}}},
		},
		GroupBy: []dimension{pb.ClusterTopDimension_Service, pb.ClusterTopDimension_Function},
	}
}

func TestPrepareSupportedShapes(t *testing.T) {
	for _, group := range [][]dimension{
		{1, 2}, {1, 3, 4, 5}, {1, 3, 4, 5, 6, 7, 8},
		{2}, {7}, {8, 6}, {1, 2, 3, 4, 5, 6, 7, 8},
	} {
		req := request()
		req.GroupBy = group
		builder, err := prepareQuery(req)
		require.NoError(t, err)
		sql, args, err := builder.ToSql()
		require.NoError(t, err)
		require.Contains(t, sql, "FROM cluster_top_v3 AS src")
		require.NotContains(t, sql, "FINAL")
		require.Contains(t, sql, "sum(src.self_cycles)")
		require.Contains(t, sql, "LIMIT 101")
		require.Contains(t, args, "worker")
		require.Equal(t, uint64(100), queryLimit(req))
	}
}

func TestPrepareDefaultOrderAndSecondaryStringFilters(t *testing.T) {
	req := request()
	req.GroupBy = []dimension{pb.ClusterTopDimension_SourceLine, pb.ClusterTopDimension_Filename}
	req.Query.Filters = append(req.Query.Filters,
		&pb.ClusterTopFieldFilter{Filter: &pb.ClusterTopFieldFilter_String_{String_: &pb.ClusterTopStringFilter{Field: pb.ClusterTopStringFilter_Service, Condition: &pb.ClusterTopStringFilter_Prefix{Prefix: "svc'%_"}}}},
		&pb.ClusterTopFieldFilter{Filter: &pb.ClusterTopFieldFilter_String_{String_: &pb.ClusterTopStringFilter{Field: pb.ClusterTopStringFilter_Filename, Condition: &pb.ClusterTopStringFilter_Contains{Contains: "%_.cc"}}}},
	)
	builder, err := prepareQuery(req)
	require.NoError(t, err)
	sql, args, err := builder.ToSql()
	require.NoError(t, err)
	require.Contains(t, sql, "startsWith(src.service, ?)")
	require.Contains(t, sql, "position(src.source_file, ?) > 0")
	require.NotContains(t, sql, "svc'%_")
	require.Contains(t, args, "svc'%_")
	require.Contains(t, args, "%_.cc")
	require.Contains(t, sql, "ORDER BY source_file ASC, source_line ASC")
}

func TestPrepareRejectsBeforeRead(t *testing.T) {
	cases := []struct {
		name        string
		change      func(*pb.ClusterTopQueryRequest)
		unsupported bool
	}{
		{"zero generation", func(r *pb.ClusterTopQueryRequest) { r.Query.Generation = 0 }, false},
		{"limit", func(r *pb.ClusterTopQueryRequest) { r.Pagination = &pagination.Paginated{Limit: 1001} }, false},
		{"offset", func(r *pb.ClusterTopQueryRequest) { r.Pagination = &pagination.Paginated{Offset: 10001} }, false},
		{"unknown group", func(r *pb.ClusterTopQueryRequest) { r.GroupBy = []dimension{99} }, false},
		{"duplicate group", func(r *pb.ClusterTopQueryRequest) { r.GroupBy = []dimension{1, 1} }, false},
		{"empty grouping", func(r *pb.ClusterTopQueryRequest) { r.GroupBy = nil }, false},
		{"file only", func(r *pb.ClusterTopQueryRequest) {
			r.Query.Filters[0].GetString_().Field = pb.ClusterTopStringFilter_Filename
		}, true},
		{"missing function", func(r *pb.ClusterTopQueryRequest) { r.Query.Filters = nil }, true},
		{"duplicate function filter", func(r *pb.ClusterTopQueryRequest) { r.Query.Filters = append(r.Query.Filters, r.Query.Filters[0]) }, true},
		{"substring", func(r *pb.ClusterTopQueryRequest) {
			r.Query.Filters[0].GetString_().Condition = &pb.ClusterTopStringFilter_Contains{Contains: "work"}
		}, true},
		{"prefix", func(r *pb.ClusterTopQueryRequest) {
			r.Query.Filters[0].GetString_().Condition = &pb.ClusterTopStringFilter_Prefix{Prefix: "work"}
		}, true},
		{"empty name", func(r *pb.ClusterTopQueryRequest) {
			r.Query.Filters[0].GetString_().Condition = &pb.ClusterTopStringFilter_Exact{}
		}, false},
		{"no condition", func(r *pb.ClusterTopQueryRequest) { r.Query.Filters[0].GetString_().Condition = nil }, false},
		{"missing filter type", func(r *pb.ClusterTopQueryRequest) { r.Query.Filters[0] = &pb.ClusterTopFieldFilter{} }, false},
		{"missing string filter", func(r *pb.ClusterTopQueryRequest) {
			r.Query.Filters[0] = &pb.ClusterTopFieldFilter{Filter: &pb.ClusterTopFieldFilter_String_{}}
		}, false},
		{"unknown string field", func(r *pb.ClusterTopQueryRequest) {
			r.Query.Filters[0].GetString_().Field = pb.ClusterTopStringFilter_UnknownField
		}, false},
		{"missing integer filter", func(r *pb.ClusterTopQueryRequest) {
			r.Query.Filters[0] = &pb.ClusterTopFieldFilter{Filter: &pb.ClusterTopFieldFilter_Integer{}}
		}, false},
		{"missing integer condition", func(r *pb.ClusterTopQueryRequest) {
			r.Query.Filters[0] = &pb.ClusterTopFieldFilter{Filter: &pb.ClusterTopFieldFilter_Integer{Integer: &pb.ClusterTopIntegerFieldFilter{Field: pb.ClusterTopIntegerFieldFilter_SourceLine}}}
		}, false},
		{"nil filter", func(r *pb.ClusterTopQueryRequest) { r.Query.Filters[0] = nil }, false},
		{"non-string field value", func(r *pb.ClusterTopQueryRequest) {
			r.Query.Filters[0].GetString_().Field = pb.ClusterTopStringFilter_FieldName(7)
		}, false},
		{"unknown filter", func(r *pb.ClusterTopQueryRequest) { r.Query.Filters[0].GetString_().Field = 99 }, false},
		{"large string", func(r *pb.ClusterTopQueryRequest) {
			r.Query.Filters[0].GetString_().Condition = &pb.ClusterTopStringFilter_Exact{Exact: strings.Repeat("x", 4097)}
		}, false},
		{"empty In", func(r *pb.ClusterTopQueryRequest) {
			r.Query.Filters[0].GetString_().Condition = &pb.ClusterTopStringFilter_In{}
		}, false},
		{"duplicate names", func(r *pb.ClusterTopQueryRequest) {
			r.Query.Filters[0].GetString_().Condition = &pb.ClusterTopStringFilter_In{In: &pb.ClusterTopStringValues{Values: []string{"f", "f"}}}
		}, false},
		{"order outside group", func(r *pb.ClusterTopQueryRequest) {
			r.OrderBy = []*pb.ClusterTopQueryOrder{{Key: &pb.ClusterTopQueryOrder_Dimension{Dimension: pb.ClusterTopDimension_BuildID}, Direction: pb.ClusterTopQueryOrder_Ascending}}
		}, false},
		{"no order direction", func(r *pb.ClusterTopQueryRequest) {
			r.OrderBy = []*pb.ClusterTopQueryOrder{{Key: &pb.ClusterTopQueryOrder_Metric{Metric: pb.ClusterTopMetric_SelfCPU}}}
		}, false},
		{"unknown metric", func(r *pb.ClusterTopQueryRequest) {
			r.OrderBy = []*pb.ClusterTopQueryOrder{{Key: &pb.ClusterTopQueryOrder_Metric{Metric: 99}, Direction: pb.ClusterTopQueryOrder_Ascending}}
		}, false},
		{"nil order", func(r *pb.ClusterTopQueryRequest) { r.OrderBy = []*pb.ClusterTopQueryOrder{nil} }, false},
	}
	_, err := prepareQuery(nil)
	require.Error(t, err)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := request()
			tc.change(req)
			builder, err := prepareQuery(req)
			require.Error(t, err)
			require.Empty(t, builder)
			require.Equal(t, tc.unsupported, errors.Is(err, ErrUnsupportedShape))
		})
	}
}

func TestPrepareInLimits(t *testing.T) {
	values := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}
	for _, tc := range []struct {
		name    string
		field   pb.ClusterTopStringFilter_FieldName
		count   int
		wantErr bool
	}{
		{"ten functions", pb.ClusterTopStringFilter_Function, 10, false},
		{"eleven functions", pb.ClusterTopStringFilter_Function, 11, true},
		{"five services", pb.ClusterTopStringFilter_Service, 5, false},
		{"six services", pb.ClusterTopStringFilter_Service, 6, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := request()
			filter := &pb.ClusterTopFieldFilter{Filter: &pb.ClusterTopFieldFilter_String_{String_: &pb.ClusterTopStringFilter{
				Field: tc.field, Condition: &pb.ClusterTopStringFilter_In{In: &pb.ClusterTopStringValues{Values: values[:tc.count]}},
			}}}
			if tc.field == pb.ClusterTopStringFilter_Function {
				req.Query.Filters[0] = filter
			} else {
				req.Query.Filters = append(req.Query.Filters, filter)
			}
			builder, err := prepareQuery(req)
			if tc.wantErr {
				require.Error(t, err)
				require.NotErrorIs(t, err, ErrUnsupportedShape)
				return
			}
			require.NoError(t, err)
			_, args, err := builder.ToSql()
			require.NoError(t, err)
			for _, value := range values[:tc.count] {
				require.Contains(t, args, value)
			}
		})
	}
}

func TestPrepareBindsValuesAndKeepsStableOrder(t *testing.T) {
	req := request()
	literal := "worker' OR 1=1 -- %_"
	req.Query.Filters[0].GetString_().Condition = &pb.ClusterTopStringFilter_In{In: &pb.ClusterTopStringValues{Values: []string{literal, "other"}}}
	// A filter on an unselected column must address the source, not its empty SELECT alias.
	for field := pb.ClusterTopStringFilter_BuildID; field <= pb.ClusterTopStringFilter_Filename; field++ {
		req.Query.Filters = append(req.Query.Filters, &pb.ClusterTopFieldFilter{Filter: &pb.ClusterTopFieldFilter_String_{String_: &pb.ClusterTopStringFilter{Field: field, Condition: &pb.ClusterTopStringFilter_Exact{Exact: ""}}}})
	}
	req.OrderBy = []*pb.ClusterTopQueryOrder{{Key: &pb.ClusterTopQueryOrder_Metric{Metric: pb.ClusterTopMetric_CumulativeCPU}, Direction: pb.ClusterTopQueryOrder_Ascending}}
	req.Pagination = &pagination.Paginated{Offset: 3, Limit: 2}
	before := proto.Clone(req)
	builder, err := prepareQuery(req)
	require.NoError(t, err)
	require.True(t, proto.Equal(before, req))
	sql, args, err := builder.ToSql()
	require.NoError(t, err)
	require.NotContains(t, sql, literal)
	require.Contains(t, args, literal)
	require.Contains(t, sql, "src.build_id IN (?)")
	require.Contains(t, sql, "src.commit_id IN (?)")
	require.Contains(t, sql, "src.binary_path IN (?)")
	require.Contains(t, sql, "src.source_file IN (?)")
	require.Contains(t, sql, "ORDER BY cumulative_cycles ASC, function ASC, service ASC LIMIT 3 OFFSET 3")
}

func TestPrepareNumericRanges(t *testing.T) {
	cases := []struct {
		name        string
		field       pb.ClusterTopIntegerFieldFilter_FieldName
		bounds      *pb.ClusterTopUInt32Range
		unsupported bool
	}{
		{"closed", pb.ClusterTopIntegerFieldFilter_SourceLine, &pb.ClusterTopUInt32Range{Min: proto.Uint32(10), Max: proto.Uint32(20)}, true},
		{"lower only", pb.ClusterTopIntegerFieldFilter_SourceColumn, &pb.ClusterTopUInt32Range{Min: proto.Uint32(1)}, true},
		{"upper only", pb.ClusterTopIntegerFieldFilter_SourceLine, &pb.ClusterTopUInt32Range{Max: proto.Uint32(20)}, true},
		{"zero singleton", pb.ClusterTopIntegerFieldFilter_SourceLine, &pb.ClusterTopUInt32Range{Min: proto.Uint32(0), Max: proto.Uint32(0)}, true},
		{"no range", pb.ClusterTopIntegerFieldFilter_SourceLine, nil, false},
		{"no bounds", pb.ClusterTopIntegerFieldFilter_SourceLine, &pb.ClusterTopUInt32Range{}, false},
		{"reversed", pb.ClusterTopIntegerFieldFilter_SourceLine, &pb.ClusterTopUInt32Range{Min: proto.Uint32(20), Max: proto.Uint32(10)}, false},
		{"unset field", pb.ClusterTopIntegerFieldFilter_UnknownField, &pb.ClusterTopUInt32Range{Min: proto.Uint32(1)}, false},
		{"unknown field", pb.ClusterTopIntegerFieldFilter_FieldName(99), &pb.ClusterTopUInt32Range{Min: proto.Uint32(1)}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := request()
			req.Query.Filters = append(req.Query.Filters, &pb.ClusterTopFieldFilter{Filter: &pb.ClusterTopFieldFilter_Integer{Integer: &pb.ClusterTopIntegerFieldFilter{
				Field: tc.field, Condition: &pb.ClusterTopIntegerFieldFilter_Range{Range: tc.bounds},
			}}})
			builder, err := prepareQuery(req)
			require.Error(t, err)
			require.Empty(t, builder)
			require.Equal(t, tc.unsupported, errors.Is(err, ErrUnsupportedShape))
		})
	}
}
