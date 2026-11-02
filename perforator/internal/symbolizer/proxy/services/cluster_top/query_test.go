package cluster_top

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/perforator/internal/symbolizer/proxy/services/cluster_top/mocks"
	"github.com/yandex/perforator/perforator/pkg/storage/cluster_top/aggregated"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/lib/pagination"
	pb "github.com/yandex/perforator/perforator/proto/perforator"
)

func queryRequest() *pb.ClusterTopQueryRequest {
	return &pb.ClusterTopQueryRequest{
		Query: &pb.ClusterTopQuery{Generation: 42,
			Filters: []*pb.ClusterTopFieldFilter{{Filter: &pb.ClusterTopFieldFilter_String_{String_: &pb.ClusterTopStringFilter{Field: pb.ClusterTopStringFilter_Function, Condition: &pb.ClusterTopStringFilter_Exact{Exact: "worker"}}}}},
		},
		GroupBy: []pb.ClusterTopDimension{pb.ClusterTopDimension_Function, pb.ClusterTopDimension_BuildID, pb.ClusterTopDimension_CommitID, pb.ClusterTopDimension_BinaryPath},
	}
}

func queryGeneration() *pb.ClusterTopGeneration {
	start := time.Unix(1700000000, 0)
	return &pb.ClusterTopGeneration{ID: 42, From: timestamppb.New(start), To: timestamppb.New(start.Add(2 * time.Hour)), GenerationStatus: pb.ClusterTopGenerationStatus_COMPLETED}
}

func TestQueryRejectsBeforeStorage(t *testing.T) {
	cases := []struct {
		name   string
		change func(*pb.ClusterTopQueryRequest)
		code   codes.Code
	}{
		{"malformed filter", func(r *pb.ClusterTopQueryRequest) { r.Query.Filters[0] = &pb.ClusterTopFieldFilter{} }, codes.InvalidArgument},
		{"missing function", func(r *pb.ClusterTopQueryRequest) { r.Query.Filters = nil }, codes.Unimplemented},
		{"numeric range", func(r *pb.ClusterTopQueryRequest) {
			r.Query.Filters = append(r.Query.Filters, &pb.ClusterTopFieldFilter{Filter: &pb.ClusterTopFieldFilter_Integer{Integer: &pb.ClusterTopIntegerFieldFilter{
				Field:     pb.ClusterTopIntegerFieldFilter_SourceLine,
				Condition: &pb.ClusterTopIntegerFieldFilter_Range{Range: &pb.ClusterTopUInt32Range{Min: proto.Uint32(10), Max: proto.Uint32(20)}},
			}}})
		}, codes.Unimplemented},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// No expectations: both error classes must be rejected before any storage call.
			storage := mocks.NewMockStorage(gomock.NewController(t))
			req := queryRequest()
			tc.change(req)
			resp, err := NewService(xlog.ForTest(t), storage, Config{}).QueryClusterTop(t.Context(), req)
			require.Nil(t, resp)
			require.Equal(t, tc.code, status.Code(err))
			if tc.code == codes.Unimplemented {
				details := status.Convert(err).Details()
				require.Len(t, details, 1)
				require.Equal(t, "UNSUPPORTED_QUERY_SHAPE", details[0].(*errdetails.ErrorInfo).Reason)
			}
		})
	}
}

func TestQueryMissingGeneration(t *testing.T) {
	storage := mocks.NewMockStorage(gomock.NewController(t))
	storage.EXPECT().GetGeneration(gomock.Any(), uint32(42)).Return(nil, nil)
	_, err := NewService(xlog.ForTest(t), storage, Config{}).QueryClusterTop(t.Context(), queryRequest())
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestQueryResultAndAverageCPU(t *testing.T) {
	for _, generationStatus := range []pb.ClusterTopGenerationStatus{pb.ClusterTopGenerationStatus_COMPLETED, pb.ClusterTopGenerationStatus_IN_PROGRESS} {
		t.Run(generationStatus.String(), func(t *testing.T) {
			storage := mocks.NewMockStorage(gomock.NewController(t))
			generation := queryGeneration()
			generation.GenerationStatus = generationStatus
			storage.EXPECT().GetGeneration(gomock.Any(), uint32(42)).Return(generation, nil)
			req := queryRequest()
			req.GroupBy = []pb.ClusterTopDimension{pb.ClusterTopDimension_BuildID, pb.ClusterTopDimension_Function}
			req.Pagination = &pagination.Paginated{Offset: 2, Limit: 1}
			conf := Config{QueryTimeout: 200 * time.Millisecond}
			storage.EXPECT().QueryClusterTop(gomock.Any(), req).DoAndReturn(func(ctx context.Context, _ *pb.ClusterTopQueryRequest) (*aggregated.QueryResult, error) {
				deadline, ok := ctx.Deadline()
				require.True(t, ok)
				require.LessOrEqual(t, time.Until(deadline), conf.QueryTimeout)
				return &aggregated.QueryResult{Rows: []*aggregated.QueryRow{{Function: "worker", BuildID: "build", SelfCycles: *big.NewInt(oneCpuHourCycles / 2), CumulativeCycles: *big.NewInt(oneCpuHourCycles)}}, HasMore: true}, nil
			})
			resp, err := NewService(xlog.ForTest(t), storage, conf).QueryClusterTop(t.Context(), req)
			require.NoError(t, err)
			require.Len(t, resp.Rows, 1)
			require.True(t, resp.HasMore)
			require.Equal(t, generationStatus, resp.Generation.GenerationStatus)
			row := resp.Rows[0]
			require.Equal(t, "worker", row.Key.GetFunction())
			require.Equal(t, "build", row.Key.GetBuildID())
			require.Nil(t, row.Key.CommitID)
			require.Nil(t, row.Key.BinaryPath)
			require.Nil(t, row.Key.Filename)
			require.Equal(t, 0.25, row.Metrics.SelfCPU)
			require.Equal(t, 0.5, row.Metrics.CumulativeCPU)
		})
	}
}

func TestQueryPreservesMetadataAndUInt128(t *testing.T) {
	n, _ := new(big.Int).SetString("340282366920938463463374607431768211455", 10)
	row := &aggregated.QueryRow{Function: "f", BuildID: "build", CommitID: "commit", BinaryPath: "/bin/app", Filename: "src/a.cc", SourceLine: 17, SourceColumn: 2, SelfCycles: *n, CumulativeCycles: *n}
	key := queryKey(row)
	require.Equal(t, "commit", key.GetCommitID())
	require.Equal(t, "/bin/app", key.GetBinaryPath())
	require.Equal(t, uint32(17), key.GetSourceLine())
	require.Equal(t, uint32(2), key.GetSourceColumn())
	require.Nil(t, key.Service)
	require.Greater(t, averageCPU(n, time.Hour), 1e20)
	require.Equal(t, "340282366920938463463374607431768211455", n.String())
	row.SourceLine = 0
	row.SourceColumn = 0
	key = queryKey(row)
	require.Nil(t, key.SourceLine)
	require.Nil(t, key.SourceColumn)
}

func TestQueryDoesNotReturnPartialRowsOnFailure(t *testing.T) {
	storage := mocks.NewMockStorage(gomock.NewController(t))
	storage.EXPECT().GetGeneration(gomock.Any(), uint32(42)).Return(queryGeneration(), nil)
	storage.EXPECT().QueryClusterTop(gomock.Any(), gomock.Any()).Return(&aggregated.QueryResult{Rows: []*aggregated.QueryRow{{Function: "partial"}}}, context.DeadlineExceeded)
	resp, err := NewService(xlog.ForTest(t), storage, Config{}).QueryClusterTop(t.Context(), queryRequest())
	require.Nil(t, resp)
	require.Equal(t, codes.DeadlineExceeded, status.Code(err))
}
