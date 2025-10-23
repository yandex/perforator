package custom_profiling_operation

import (
	"context"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	cpo_internal "github.com/yandex/perforator/perforator/internal/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/internal/symbolizer/proxy/services"
	"github.com/yandex/perforator/perforator/pkg/storage/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	perforator_proto "github.com/yandex/perforator/perforator/proto/perforator"
)

var (
	_ services.GRPCService = (*APIService)(nil)
)

type APIService struct {
	l                xlog.Logger
	operationStorage custom_profiling_operation.Storage
	metrics          *serviceMetrics
}

type serviceMetrics struct {
	successfulSchedules metrics.Counter
	failedSchedules     metrics.Counter
}

func NewService(
	l xlog.Logger,
	r metrics.Registry,
	operationStorage custom_profiling_operation.Storage,
) *APIService {
	r = r.WithPrefix("cpo.api")

	return &APIService{
		l:                l,
		operationStorage: operationStorage,
		metrics: &serviceMetrics{
			successfulSchedules: r.WithTags(map[string]string{"status": "success"}).Counter("schedule.count"),
			failedSchedules:     r.WithTags(map[string]string{"status": "fail"}).Counter("schedule.count"),
		},
	}
}

func (s *APIService) Register(server *grpc.Server) error {
	perforator_proto.RegisterCustomProfilingOperationAPIServer(server, s)
	return nil
}

func genOperationID() (custom_profiling_operation.OperationID, error) {
	uid, err := uuid.NewV7()
	if err != nil {
		return custom_profiling_operation.OperationID(""), err
	}

	return custom_profiling_operation.OperationID(uid.String()), nil
}

func (s *APIService) Schedule(ctx context.Context, req *perforator_proto.ScheduleProfilingOperationRequest) (*perforator_proto.ScheduleProfilingOperationResponse, error) {
	if req == nil || req.OperationSpec == nil {
		return nil, status.Errorf(codes.InvalidArgument, "req is nil or req.OperationSpec is nil")
	}

	if err := cpo_internal.ValidateOperationSpec(req.OperationSpec); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid operation spec: %v", err)
	}

	operationID, err := genOperationID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate operation ID: %v", err)
	}

	operation, err := s.operationStorage.InsertOperation(ctx, operationID, req.OperationSpec)
	if err != nil {
		s.metrics.failedSchedules.Inc()
		s.l.Error(ctx, "Failed to insert operation", log.Any("operation_spec", req.OperationSpec), log.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to insert operation: %v", err)
	}

	s.metrics.successfulSchedules.Inc()
	s.l.Info(ctx, "Scheduled operation", log.Any("operation", operation))

	return &perforator_proto.ScheduleProfilingOperationResponse{
		OperationID: operation.ID,
	}, nil
}

func (s *APIService) Stop(ctx context.Context, req *perforator_proto.StopProfilingOperationRequest) (*perforator_proto.StopProfilingOperationResponse, error) {
	if req == nil || req.OperationID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "req is nil or req.OperationID is empty")
	}

	err := s.operationStorage.StopOperation(ctx, custom_profiling_operation.OperationID(req.OperationID))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to stop operation: %v", err)
	}

	return &perforator_proto.StopProfilingOperationResponse{}, nil
}

func (s *APIService) Get(ctx context.Context, req *perforator_proto.GetProfilingOperationRequest) (*perforator_proto.GetProfilingOperationResponse, error) {
	if req == nil || req.OperationID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "req is nil or req.OperationID is empty")
	}

	operation, err := s.operationStorage.GetOperation(ctx, custom_profiling_operation.OperationID(req.OperationID))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get operation: %v", err)
	}

	return &perforator_proto.GetProfilingOperationResponse{
		Operation: operation,
	}, nil
}
