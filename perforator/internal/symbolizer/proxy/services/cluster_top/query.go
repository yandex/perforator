package cluster_top

import (
	"context"
	"errors"
	"math/big"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/perforator/pkg/storage/cluster_top/aggregated"
	pb "github.com/yandex/perforator/perforator/proto/perforator"
)

func (s *APIService) QueryClusterTop(ctx context.Context, req *pb.ClusterTopQueryRequest) (*pb.ClusterTopQueryResponse, error) {
	err := aggregated.ValidateQuery(req)
	if errors.Is(err, aggregated.ErrUnsupportedShape) {
		result, detailErr := status.New(codes.Unimplemented, err.Error()).WithDetails(&errdetails.ErrorInfo{
			Reason: "UNSUPPORTED_QUERY_SHAPE", Domain: "perforator",
		})
		if detailErr != nil {
			return nil, status.Error(codes.Internal, "cannot encode query error")
		}
		return nil, result.Err()
	}
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	ctx, cancel := context.WithTimeout(ctx, s.conf.QueryTimeout)
	defer cancel()
	generation, err := s.clusterTopGenerationStorage.GetGeneration(ctx, req.Query.Generation)
	if err != nil {
		return nil, queryStorageError(err)
	}
	if generation == nil {
		return nil, status.Error(codes.NotFound, "Cluster Top generation is unavailable")
	}
	interval := generation.To.AsTime().Sub(generation.From.AsTime())
	data, err := s.clusterTopGenerationStorage.QueryClusterTop(ctx, req)
	if err != nil {
		return nil, queryStorageError(err)
	}
	result := &pb.ClusterTopQueryResponse{Generation: generation, HasMore: data.HasMore}
	for _, row := range data.Rows {
		result.Rows = append(result.Rows, &pb.ClusterTopQueryRow{
			Key: queryKey(row),
			Metrics: &pb.ClusterTopQueryMetrics{
				SelfCPU:       averageCPU(&row.SelfCycles, interval),
				CumulativeCPU: averageCPU(&row.CumulativeCycles, interval),
			},
		})
	}
	return result, nil
}

func queryStorageError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return status.FromContextError(err).Err()
	}
	return err
}

// TODO: Apply the sampling multiplier in the worker and remove the hardcoded
// multiplier here once stored counters are normalized.
func averageCPU(cycles *big.Int, interval time.Duration) float64 {
	numerator := new(big.Int).Mul(cycles, big.NewInt(perforatorSamplingModulo*int64(time.Second)))
	denominator := new(big.Int).Mul(big.NewInt(estimatedCpuFreq), big.NewInt(int64(interval)))
	cpu, _ := new(big.Rat).SetFrac(numerator, denominator).Float64()
	return cpu
}

func queryKey(row *aggregated.QueryRow) *pb.ClusterTopQueryKey {
	key := &pb.ClusterTopQueryKey{
		Function:   knownString(row.Function),
		Service:    knownString(row.Service),
		BuildID:    knownString(row.BuildID),
		CommitID:   knownString(row.CommitID),
		BinaryPath: knownString(row.BinaryPath),
		Filename:   knownString(row.Filename),
	}
	if row.SourceLine != 0 {
		key.SourceLine = proto.Uint32(row.SourceLine)
	}
	if row.SourceColumn != 0 {
		key.SourceColumn = proto.Uint32(row.SourceColumn)
	}
	return key
}

func knownString(value string) *string {
	if value == "" {
		return nil
	}
	return proto.String(value)
}
