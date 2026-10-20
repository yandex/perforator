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
	// An empty path runs the service for in-process calls without a gRPC listener.
	SocketPath string
}

type processEntry struct {
	mu    sync.RWMutex
	state *jvmscanner.ProcessState
}

func (p *processEntry) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	_ = p.state.Close()
}

type Service struct {
	l       xlog.Logger
	opts    Options
	scanner *jvmscanner.Scanner

	firstKeepaliveCall   atomic.Bool
	keepaliveCallStarted chan struct{}
	keepaliveCallFailed  chan error

	procsmu        sync.RWMutex
	procs          map[linux.CurrentNamespacePID]*processEntry
	closed         bool
	processUpdates sync.WaitGroup

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
		procs:                make(map[linux.CurrentNamespacePID]*processEntry),
		cheatsheets:          make(map[uint64]*jvm.Cheatsheet),
	}
}

// acquireProcess pins the selected state; callers must release its read lock.
func (s *Service) acquireProcess(key linux.ProcessKey) (*processEntry, error) {
	s.procsmu.RLock()
	defer s.procsmu.RUnlock()
	proc := s.procs[key.Pid]
	if proc == nil {
		return nil, status.Errorf(codes.NotFound, "process %d not found", key.Pid)
	}
	if key.ProcessStartTime != 0 && proc.state.Key() != key {
		return nil, status.Errorf(codes.NotFound, "JVM state for process %d with start time %d is not registered (registered start time: %d)", key.Pid, key.ProcessStartTime, proc.state.Key().ProcessStartTime)
	}
	proc.mu.RLock()
	return proc, nil
}

func (s *Service) Scan(ctx context.Context, req *jvmsupp.ScanRequest) (*jvmsupp.ScanResponse, error) {
	pid := linux.CurrentNamespacePID(req.Pid)
	if int64(pid) != req.Pid {
		return nil, status.Errorf(codes.InvalidArgument, "pid does not fit into uint32_t: %d", req.Pid)
	}
	proc, err := s.acquireProcess(linux.ProcessKey{Pid: pid, ProcessStartTime: req.ProcessStartTime})
	if err != nil {
		return nil, err
	}
	defer proc.mu.RUnlock()

	res, err := s.scanner.ScanProcess(ctx, proc.state, req.Bookmarks)
	if err != nil {
		if _, ok := errors.AsType[*jvmscanner.ProcessNotFoundError](err); ok {
			return nil, status.Errorf(codes.FailedPrecondition, "scan error (probably process does not exist anymore): %v", err)
		}

		return nil, status.Errorf(codes.Internal, "scan error: %v", err)
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
	proc, err := s.acquireProcess(linux.ProcessKey{Pid: linux.CurrentNamespacePID(req.GetPid()), ProcessStartTime: req.ProcessStartTime})
	if err != nil {
		return nil, err
	}
	defer proc.mu.RUnlock()
	results := make([]*jvmsupp.AddressSymbolization, len(req.GetMethods()))
	for i, addr := range req.Methods {
		name, err := s.scanner.Symbolize(ctx, proc.state, addr)
		if err != nil {
			if _, ok := errors.AsType[*jvmscanner.ProcessNotFoundError](err); ok {
				return nil, status.Errorf(codes.FailedPrecondition, "process lifetime ended: %v", err)
			}
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

// Admission and shutdown share procsmu so no update can start after cleanup begins.
func (s *Service) beginProcessUpdate() error {
	s.procsmu.Lock()
	defer s.procsmu.Unlock()
	if s.closed {
		return status.Error(codes.Unavailable, "JVM service is closed")
	}
	s.processUpdates.Add(1)
	return nil
}

func (s *Service) InitProcess(ctx context.Context, req *jvmsupp.InitProcessRequest) (*jvmsupp.InitProcessResponse, error) {
	if err := s.beginProcessUpdate(); err != nil {
		return nil, err
	}
	defer s.processUpdates.Done()

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
	state, bpfInfo, err := s.scanner.InitProcess(linux.ProcessKey{Pid: linux.CurrentNamespacePID(req.Pid), ProcessStartTime: req.ProcessStartTime}, cheatsheet, req.Version, req.BaseAddress)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to initialize process: %v", err)
	}

	s.procsmu.Lock()
	if s.closed {
		s.procsmu.Unlock()
		_ = state.Close()
		return nil, status.Error(codes.Unavailable, "JVM service is closed")
	}
	old := s.procs[linux.CurrentNamespacePID(req.Pid)]
	s.procs[linux.CurrentNamespacePID(req.Pid)] = &processEntry{state: state}
	s.procsmu.Unlock()
	if old != nil {
		old.close()
	}

	return &jvmsupp.InitProcessResponse{
		InterpreterBegin: bpfInfo.InterpreterBegin,
		InterpreterEnd:   bpfInfo.InterpreterEnd,
	}, nil
}

func (s *Service) RetainProcesses(ctx context.Context, req *jvmsupp.RetainProcessesRequest) (*jvmsupp.RetainProcessesResponse, error) {
	if err := s.beginProcessUpdate(); err != nil {
		return nil, err
	}
	defer s.processUpdates.Done()

	pids := make(map[linux.CurrentNamespacePID]struct{})
	for _, pid := range req.Pids {
		pids[linux.CurrentNamespacePID(pid)] = struct{}{}
	}
	var removed []*processEntry
	s.procsmu.Lock()
	maps.DeleteFunc(s.procs, func(pid linux.CurrentNamespacePID, proc *processEntry) bool {
		_, ok := pids[pid]
		if !ok {
			removed = append(removed, proc)
		}
		return !ok
	})
	s.procsmu.Unlock()
	for _, proc := range removed {
		proc.close()
	}
	return &jvmsupp.RetainProcessesResponse{}, nil
}

func (s *Service) ReleaseProcess(ctx context.Context, req *jvmsupp.ReleaseProcessRequest) (*jvmsupp.ReleaseProcessResponse, error) {
	if err := s.beginProcessUpdate(); err != nil {
		return nil, err
	}
	defer s.processUpdates.Done()

	key := linux.ProcessKey{Pid: linux.CurrentNamespacePID(req.Pid), ProcessStartTime: req.ProcessStartTime}
	s.procsmu.Lock()
	proc := s.procs[key.Pid]
	if proc != nil && proc.state.Key() == key {
		delete(s.procs, key.Pid)
	} else {
		proc = nil
	}
	s.procsmu.Unlock()
	if proc != nil {
		proc.close()
	}
	return &jvmsupp.ReleaseProcessResponse{}, nil
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

func (s *Service) close() {
	s.procsmu.Lock()
	s.closed = true
	procs := s.procs
	s.procs = nil
	s.procsmu.Unlock()
	s.processUpdates.Wait()
	for _, proc := range procs {
		proc.close()
	}
}

// Run must be called once. It releases process state before returning.
func (s *Service) Run(runCtx context.Context) error {
	defer s.close()

	eg, gctx := errgroup.WithContext(runCtx)
	if s.opts.SocketPath != "" {
		srv := grpc.NewServer()
		jvmsupp.RegisterJvmSupportServiceServer(srv, s)
		listener, err := net.Listen("unix", s.opts.SocketPath)
		if err != nil {
			return fmt.Errorf("failed to bind UDS listener: %w", err)
		}
		eg.Go(func() error {
			s.l.Info(gctx, "Starting grpc server")
			if err := srv.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
				return fmt.Errorf("grpc server failed: %w", err)
			}
			return nil
		})
		eg.Go(func() error {
			<-gctx.Done()
			srv.Stop()
			return nil
		})
	}
	eg.Go(func() error {
		keepaliveStartCtx, cancel := context.WithTimeoutCause(
			gctx,
			15*time.Second,
			errors.New("keepalive call establishment timeout exceeded"),
		)
		defer cancel()
		select {
		case <-keepaliveStartCtx.Done():
			if gctx.Err() != nil {
				return nil
			}
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
			if gctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("keepalive call failed: %w", err)
		}
	})
	return eg.Wait()
}
