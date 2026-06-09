package jvmsupportservice

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/jvm"
	"github.com/yandex/perforator/perforator/internal/linguist/jvm/jvmscanner"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/jvmsupp"
)

type Options struct {
	SocketPath string
}

type Service struct {
	l       xlog.Logger
	opts    Options
	scanner *jvmscanner.Scanner

	firstKeepaliveCall   atomic.Bool
	keepaliveCallStarted chan struct{}
	keepaliveCallFailed  chan error

	procsmu sync.RWMutex
	procs   map[linux.CurrentNamespacePID]*jvmscanner.ProcessState

	cheatsheetsmu sync.RWMutex
	cheatsheets   map[uint64]*jvm.Cheatsheet
}

func New(logger xlog.Logger, scanner *jvmscanner.Scanner, opts Options) *Service {
	return &Service{
		l:                    logger,
		opts:                 opts,
		scanner:              scanner,
		keepaliveCallStarted: make(chan struct{}),
		keepaliveCallFailed:  make(chan error, 1),
		procs:                make(map[linux.CurrentNamespacePID]*jvmscanner.ProcessState),
		cheatsheets:          make(map[uint64]*jvm.Cheatsheet),
	}
}

func (s *Service) Scan(ctx context.Context, req *jvmsupp.ScanRequest) (*jvmsupp.ScanResponse, error) {
	pid := linux.CurrentNamespacePID(req.Pid)
	if int64(pid) != req.Pid {
		return nil, status.Errorf(codes.InvalidArgument, "pid does not fit into uint32_t: %d", req.Pid)
	}
	s.procsmu.RLock()
	proc := s.procs[pid]
	s.procsmu.RUnlock()
	if proc == nil {
		return nil, status.Errorf(codes.NotFound, "process %d not found", req.Pid)
	}

	scanned, err := s.scanner.ScanProcess(ctx, proc)
	if err != nil {
		var pnfe *jvmscanner.ProcessNotFoundError
		if errors.As(err, &pnfe) {
			return nil, status.Errorf(codes.FailedPrecondition, "scan error (probably process does not exist anymore): %v", err)
		}

		return nil, status.Errorf(codes.Internal, "scan error: %v", err)
	}
	res := &jvmsupp.ScanResponse{
		Methods: scanned,
	}
	return res, nil
}

func (s *Service) KeepAlive(ctx context.Context, req *jvmsupp.KeepAliveRequest) (*jvmsupp.KeepAliveResponse, error) {
	if s.firstKeepaliveCall.Swap(true) {
		s.l.Error(ctx, "Unexpected additional KeepAlive call arrived")
		return nil, status.Error(codes.FailedPrecondition, "only one KeepAlive call cay be started")
	}
	close(s.keepaliveCallStarted)
	<-ctx.Done()
	s.keepaliveCallFailed <- context.Cause(ctx)
	return nil, ctx.Err()
}

func (s *Service) Symbolize(ctx context.Context, req *jvmsupp.SymbolizeRequest) (*jvmsupp.SymbolizeResponse, error) {
	if len(req.GetMethods()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "no methods to symbolize")
	}
	s.procsmu.RLock()
	proc := s.procs[linux.CurrentNamespacePID(req.GetPid())]
	s.procsmu.RUnlock()
	if proc == nil {
		return nil, status.Errorf(codes.NotFound, "process %d not found", req.Pid)
	}
	results := make([]*jvmsupp.AddressSymbolization, len(req.GetMethods()))
	for i, addr := range req.Methods {
		name, err := s.scanner.Symbolize(ctx, proc, addr)
		if err != nil {
			results[i] = &jvmsupp.AddressSymbolization{
				Error: &jvmsupp.SymbolizationError{
					Message: err.Error(),
				},
			}
			continue
		}
		results[i] = &jvmsupp.AddressSymbolization{
			Name: name,
		}
	}

	return &jvmsupp.SymbolizeResponse{
		Symbolized: results,
	}, nil
}

func (s *Service) InitProcess(ctx context.Context, req *jvmsupp.InitProcessRequest) (*jvmsupp.InitProcessResponse, error) {
	var cheatsheet *jvm.Cheatsheet
	if req.Cheatsheet != nil {
		cheatsheet = req.Cheatsheet
		s.cheatsheetsmu.Lock()
		s.cheatsheets[req.LibjvmBinaryId] = cheatsheet
		s.cheatsheetsmu.Unlock()
	} else {
		s.cheatsheetsmu.RLock()
		cheatsheet = s.cheatsheets[req.LibjvmBinaryId]
		s.cheatsheetsmu.RUnlock()
		if cheatsheet == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "cheatsheet is unavailable for binary id %d", req.LibjvmBinaryId)
		}
	}
	if req.Version == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "version is required")
	}
	state, bpfInfo, err := s.scanner.InitProcess(linux.CurrentNamespacePID(req.Pid), cheatsheet, req.Version, req.BaseAddress)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to initialize process: %v", err)
	}

	s.procsmu.Lock()
	defer s.procsmu.Unlock()
	s.procs[linux.CurrentNamespacePID(req.Pid)] = state

	return &jvmsupp.InitProcessResponse{
		InterpreterBegin: bpfInfo.InterpreterBegin,
		InterpreterEnd:   bpfInfo.InterpreterEnd,
	}, nil
}

func (s *Service) RetainProcesses(ctx context.Context, req *jvmsupp.RetainProcessesRequest) (*jvmsupp.RetainProcessesResponse, error) {
	pids := make(map[linux.CurrentNamespacePID]struct{})
	for _, pid := range req.Pids {
		pids[linux.CurrentNamespacePID(pid)] = struct{}{}
	}
	s.procsmu.Lock()
	defer s.procsmu.Unlock()

	maps.DeleteFunc(s.procs, func(pid linux.CurrentNamespacePID, _ *jvmscanner.ProcessState) bool {
		_, ok := pids[pid]
		return !ok
	})
	return &jvmsupp.RetainProcessesResponse{}, nil
}

func (s *Service) RetainBinaries(ctx context.Context, req *jvmsupp.RetainBinariesRequest) (*jvmsupp.RetainBinariesResponse, error) {
	ids := make(map[uint64]struct{})
	for _, id := range req.BinaryIds {
		ids[id] = struct{}{}
	}
	s.cheatsheetsmu.Lock()
	defer s.cheatsheetsmu.Unlock()

	maps.DeleteFunc(s.cheatsheets, func(id uint64, _ *jvm.Cheatsheet) bool {
		_, ok := ids[id]
		return !ok
	})
	return &jvmsupp.RetainBinariesResponse{}, nil
}

func (s *Service) Run(runCtx context.Context) error {
	srv := grpc.NewServer()
	jvmsupp.RegisterJvmSupportServiceServer(srv, s)
	listener, err := net.Listen("unix", s.opts.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to bind UDS listener: %w", err)
	}
	eg, gctx := errgroup.WithContext(runCtx)
	eg.Go(func() error {
		s.l.Info(gctx, "Starting grpc server")
		err := srv.Serve(listener)
		if err != nil {
			return fmt.Errorf("grpc server failed: %w", err)
		}
		return nil
	})
	eg.Go(func() error {
		<-gctx.Done()
		srv.Stop()
		return nil
	})
	eg.Go(func() error {
		keepaliveStartCtx, cancel := context.WithTimeoutCause(
			gctx,
			15*time.Second,
			errors.New("keepalive call establishment timeout exceeded"),
		)
		defer cancel()
		select {
		case <-keepaliveStartCtx.Done():
			return context.Cause(keepaliveStartCtx)
		case <-s.keepaliveCallStarted:
			s.l.Info(gctx, "Keepalive call established")
			return nil
		}
	})
	eg.Go(func() error {
		select {
		case <-gctx.Done():
			return nil
		case err := <-s.keepaliveCallFailed:
			return fmt.Errorf("keepalive call failed: %w", err)
		}
	})
	return eg.Wait()
}
