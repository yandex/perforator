package jvmregistry

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/ctxlog"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/library/go/core/resource"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/unwindtable"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine/programstate"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/process"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profilerext"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/jvm"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/parse"
	_ "github.com/yandex/perforator/perforator/internal/linguist/jvm/cheatsheets"
	"github.com/yandex/perforator/perforator/internal/logfield"
	"github.com/yandex/perforator/perforator/internal/symboltable"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/jvmsupp"
)

const (
	methodKindSuffixInterpreted = " (method-kind:interpreted)"
	methodKindSuffixJIT         = " (method-kind:jit)"
	methodKindSuffixStubs       = " (method-kind:stubs)"
	methodKindSuffixInlined     = " (method-kind:inlined)"
)

type trackedProcess struct {
	pid         linux.CurrentNamespacePID
	mu          sync.Mutex
	initialized bool
	alloc       *unwindtable.Allocation
}

type interpretedSymbolCacheKey struct {
	pid  linux.CurrentNamespacePID
	addr uint64
}

type interpretedSymbolCacheValue struct {
	name string
	// TODO: add some unique process id as a measure against pid reuse
}

type binaryData struct {
	cheatsheet *jvm.Cheatsheet
	version    uint32
}

type symFunc = func(addr uint64) []string

type Registry struct {
	l      xlog.Logger
	bpf    *programstate.State
	unwind *unwindtable.BPFManager

	disambiguateFrameSource bool

	compiledSyms         *symboltable.Table[linux.CurrentNamespacePID, symFunc]
	interpretedSymsCache *expirable.LRU[interpretedSymbolCacheKey, interpretedSymbolCacheValue]

	helperBinaryPath string
	helperSocketPath string

	enableLineInfoParsing bool

	mapPrefix  string
	grpcClient *grpc.ClientConn
	helper     jvmsupp.JvmSupportServiceClient

	cheatsheetsMu sync.Mutex
	// TODO: remove version field from InitProcess rpc and change this field back to
	// map[uint64]*jvm.Cheatsheet
	cheatsheets map[uint64]*binaryData

	trackedMu sync.Mutex
	tracked   map[linux.CurrentNamespacePID]*trackedProcess

	trackedProcessCount metrics.IntGauge

	scanIterations metrics.Counter
	processScans   metrics.Counter
	// we track it separately because it is expensive leaf operation
	methodNameReads metrics.Counter

	compiledMethodCount    metrics.IntGauge
	interpretedMethodCount metrics.FuncIntGauge

	compiledMethodResolveSuccess metrics.Counter
	compiledMethodResolveFailure metrics.Counter

	interpretedMethodResolveCacheHit      metrics.Counter
	interpretedMethodResolveDone          metrics.Counter
	interpretedMethodResolveFailure       metrics.Counter
	interpretedMethodResolveInternalError metrics.Counter

	invalidMethodSymbolizationTable    metrics.Counter
	methodSymbolizationTableLookupFail metrics.Counter
}

type Options struct {
	SocketPath        string
	ScannerBinaryPath string

	DisambiguateFrameSource   bool
	MapPrefix                 string
	InterpetedSymbolCacheSize int
	InterpretedSymbolCacheTTL time.Duration

	EnableLineInfoParsing bool

	EnableScanMetrics  bool
	EnableCacheMetrics bool
}

func New(log xlog.Logger, reg metrics.Registry, bpf *machine.BPF, unwind *unwindtable.BPFManager, opts Options) (*Registry, error) {
	client, err := grpc.NewClient(
		fmt.Sprintf("unix:%s", opts.SocketPath),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create scanner client: %w", err)
	}
	interpretedSymsCache := expirable.NewLRU(
		opts.InterpetedSymbolCacheSize,
		func(key interpretedSymbolCacheKey, value interpretedSymbolCacheValue) {

		},
		opts.InterpretedSymbolCacheTTL,
	)
	registry := &Registry{
		l:                       log.WithName("jvmregistry"),
		tracked:                 make(map[linux.CurrentNamespacePID]*trackedProcess),
		bpf:                     bpf.State(),
		unwind:                  unwind,
		compiledSyms:            symboltable.New[linux.CurrentNamespacePID, symFunc](),
		interpretedSymsCache:    interpretedSymsCache,
		disambiguateFrameSource: opts.DisambiguateFrameSource,

		helperBinaryPath: opts.ScannerBinaryPath,
		helperSocketPath: opts.SocketPath,

		enableLineInfoParsing: opts.EnableLineInfoParsing,

		mapPrefix:  opts.MapPrefix,
		grpcClient: client,
		helper:     jvmsupp.NewJvmSupportServiceClient(client),

		cheatsheets: make(map[uint64]*binaryData),

		trackedProcessCount: reg.IntGauge("tracked_processes.count"),

		compiledMethodCount: reg.IntGauge("scan.compiled_methods.cache.size"),
		interpretedMethodCount: reg.FuncIntGauge("interpreted_methods.cache.size", func() int64 {
			return int64(interpretedSymsCache.Len())
		}),

		compiledMethodResolveSuccess: reg.WithTags(map[string]string{"outcome": "success"}).Counter("compiled_methods.resolve.count"),
		compiledMethodResolveFailure: reg.WithTags(map[string]string{"outcome": "failure"}).Counter("compiled_methods.resolve.count"),

		interpretedMethodResolveCacheHit:      reg.WithTags(map[string]string{"outcome": "cache_hit"}).Counter("interpreted_methods.resolve.count"),
		interpretedMethodResolveDone:          reg.WithTags(map[string]string{"outcome": "done"}).Counter("interpreted_methods.resolve.count"),
		interpretedMethodResolveFailure:       reg.WithTags(map[string]string{"outcome": "failure"}).Counter("interpreted_methods.resolve.count"),
		interpretedMethodResolveInternalError: reg.WithTags(map[string]string{"outcome": "internal_error"}).Counter("interpreted_methods.resolve.count"),

		invalidMethodSymbolizationTable:    reg.Counter("symbolization.invalid_table.count"),
		methodSymbolizationTableLookupFail: reg.Counter("symbolization.table_lookup_fail.count"),
	}

	registry.initMetrics(reg, opts)

	return registry, nil
}

func (r *Registry) initMetrics(reg metrics.Registry, opts Options) {
	if opts.EnableScanMetrics {
		r.scanIterations = reg.Counter("scan.iterations.count")
		r.processScans = reg.Counter("scan.processes.count")
		r.methodNameReads = reg.Counter("scan.method.name_reads.count")
	} else {
		r.scanIterations = &nop.Counter{}
		r.processScans = &nop.Counter{}
		r.methodNameReads = &nop.Counter{}
	}
}

func (r *Registry) installBinaryConfig(binaryID uint64, cheatSheet *jvm.Cheatsheet) error {
	var bpfConf unwinder.JvmBinaryConfig

	if cheatSheet.FrameInterpreterFrameMethodOffset == nil {
		return fmt.Errorf("incomplete cheatsheet: frame_interpreter_frame_method_offset missing")
	}
	bpfConf.InterpreterStackFrameMethodOffset = int32(*cheatSheet.FrameInterpreterFrameMethodOffset)

	if cheatSheet.FrameReturnAddrOffset == nil {
		return fmt.Errorf("incomplete cheatsheet: frame_return_addr_offset missing")
	}
	bpfConf.StackFrameReturnAddrOffset = int32(*cheatSheet.FrameReturnAddrOffset)

	err := r.bpf.PutJVMBinaryConfig(unwinder.BinaryId(binaryID), bpfConf)
	if err != nil {
		return fmt.Errorf("failed to put config: %v", err)
	}

	return nil
}

// OnBinaryDiscovery implements binary.Listener
func (r *Registry) OnBinaryDiscovery(ctx context.Context, binaryID uint64, buildID string, analysis *parse.BinaryAnalysis) {
	if analysis.Jvm == nil {
		return
	}
	switch analysis.Jvm.Status {
	case jvm.JvmAnalysis_STATUS_OK:
		jdkVersion := analysis.Jvm.Version
		var cheatSheetResourceName string
		cheatSheetResourceName = fmt.Sprintf("jvm-cheatsheets/jdk%d.txtpb", jdkVersion)

		rawStaticData := resource.Get(cheatSheetResourceName)
		if rawStaticData == nil {
			r.l.Error(ctx, "Failed to find JVM cheatsheet", log.String("cheatsheet", cheatSheetResourceName))
			return
		}
		cheatSheet := new(jvm.Cheatsheet)
		err := prototext.Unmarshal(rawStaticData, cheatSheet)
		if err != nil {
			r.l.Error(ctx, "Failed to parse builtin JVM cheatsheet", log.Error(err), log.String("cheatsheet", cheatSheetResourceName))
			return
		}
		proto.Merge(cheatSheet, analysis.Jvm.Cheatsheet)

		r.cheatsheetsMu.Lock()
		r.cheatsheets[binaryID] = &binaryData{
			version:    jdkVersion,
			cheatsheet: cheatSheet,
		}
		r.cheatsheetsMu.Unlock()

		r.l.Info(ctx, "Assigning jvm config", log.String("binary_id", buildID), log.Any("config", cheatSheet), log.String("cheatsheet", cheatSheetResourceName))
		err = r.installBinaryConfig(binaryID, cheatSheet)
		if err != nil {
			// TODO: improve error handling
			r.l.Error(ctx, "Failed to install jvm config", log.String("binary_id", buildID), log.Any("analysis", analysis.Jvm), log.Error(err))
		}
	case jvm.JvmAnalysis_STATUS_ERROR:
		r.l.Error(ctx, "JVM analysis failed", log.String("binary_id", buildID), log.Any("config", analysis.Jvm), log.String("error", analysis.Jvm.ErrorMessage))
	case jvm.JvmAnalysis_STATUS_NOT_JVM:
	default:
		r.l.Error(ctx, "Unknown JVM analysis status", log.String("status", analysis.Jvm.Status.String()))
	}
}

func (r *Registry) OnProcessDiscovery(ctx context.Context, info process.ProcessInfo) {
	for _, m := range info.Mappings() {
		if m.BinaryClass() == dso.JvmBinaryClass {
			r.l.Info(ctx, "Ensuring JVM process is tracked")
			r.ensureRegistered(ctx, info.ProcessID(), m.ID(), m.BaseAddress())
		}
	}
}

func (r *Registry) OnProcessRescan(ctx context.Context, info process.ProcessInfo) {
	r.OnProcessDiscovery(ctx, info)
}

func (r *Registry) SymbolizeInterpreted(ctx context.Context, pid linux.CurrentNamespacePID, addr uint64) (string, error) {
	key := interpretedSymbolCacheKey{
		pid:  pid,
		addr: addr,
	}
	cached, ok := r.interpretedSymsCache.Get(key)
	if ok {
		r.interpretedMethodResolveCacheHit.Inc()
		return cached.name, nil
	}
	req := &jvmsupp.SymbolizeRequest{
		Pid:     int64(pid),
		Methods: []uint64{addr},
	}
	res, err := r.helper.Symbolize(ctx, req)
	if err != nil {
		r.interpretedMethodResolveInternalError.Inc()
		return "", fmt.Errorf("remote symbolization failed: %w", err)
	}
	sym := res.GetSymbolized()[0]
	if sym.Error != nil {
		// TODO: better error classification
		r.interpretedMethodResolveInternalError.Inc()
		return "", fmt.Errorf("remote symbolization failed: %s", sym.Error.GetMessage())
	}

	name := sym.Name
	r.interpretedMethodResolveDone.Inc()
	if r.disambiguateFrameSource {
		name = name + methodKindSuffixInterpreted
	}
	r.interpretedSymsCache.Add(key, interpretedSymbolCacheValue{
		name: name,
	})
	return name, nil
}

func (r *Registry) Resolve(pid linux.CurrentNamespacePID, ip uint64) (profilerext.JITSymbolizerOutput, bool) {
	symfn, ok := r.compiledSyms.Find(pid, ip)
	if ok {
		r.compiledMethodResolveSuccess.Inc()
		names := symfn(ip)
		var syms []profilerext.JITSymbol
		for _, n := range names {
			syms = append(syms, profilerext.JITSymbol{
				Name: n,
			})
		}
		return profilerext.JITSymbolizerOutput{
			Symbols:     syms,
			MappingName: profile.JVMSpecialMapping,
		}, true
	}
	r.compiledMethodResolveFailure.Inc()
	return profilerext.JITSymbolizerOutput{}, false
}

func (r *Registry) symbolizeLocation(ctx context.Context, method *jvmsupp.MethodInfo, offset uint64) []string {
	mst := method.SymbolizationTable
	pos, ok := slices.BinarySearchFunc(mst.Instructions, offset, func(insn *jvmsupp.InstructionRange, off uint64) int {
		if off < insn.Begin {
			return 1
		}
		if off >= insn.End {
			return -1
		}
		return 0
	})
	if !ok {
		r.l.Debug(ctx, "Offset not covered by any instruction range", log.UInt64("offset", offset))
		r.methodSymbolizationTableLookupFail.Inc()
		return []string{method.Name}
	}
	node := mst.Instructions[pos].TreeNode
	var res []string
	for {
		if int(node) >= len(mst.Parent) {
			r.l.Error(ctx, "Invalid symbolization table: no parent information for node", log.UInt32("node", node))
			r.invalidMethodSymbolizationTable.Inc()
			break
		}
		methodNamePos := mst.Method[node]
		if int(methodNamePos) >= len(mst.StringTable) {
			r.l.Error(ctx, "Invalid symbolization table: method name index points out-of-bounds", log.UInt32("method", methodNamePos))
			r.invalidMethodSymbolizationTable.Inc()
			break
		}
		mname := mst.StringTable[methodNamePos]
		parent := mst.Parent[node]
		if int(node) >= len(mst.Method) {
			r.l.Error(ctx, "Invalid symbolization table: no method information for node", log.UInt32("node", node))
			r.invalidMethodSymbolizationTable.Inc()
			break
		}
		if r.disambiguateFrameSource {
			if parent == node {
				mname = mname + methodKindSuffixJIT
			} else {
				mname = mname + methodKindSuffixInlined
			}
		}
		res = append(res, mname)
		if len(res) > len(mst.Parent) {
			r.l.Error(ctx, "Invalid symbolization table: cycle detected")
			r.invalidMethodSymbolizationTable.Inc()
			break
		}
		if parent == node {
			break
		}
		node = parent
	}

	return res
}

func (r *Registry) updateSymbols(ctx context.Context, pid linux.CurrentNamespacePID, methods []*jvmsupp.MethodInfo) {
	var s []symboltable.Entry[symFunc]
	for _, m := range methods {
		name := m.Name
		if r.disambiguateFrameSource {
			if m.IsJit {
				name = name + methodKindSuffixJIT
			} else {
				name = name + methodKindSuffixStubs
			}
		}
		var symfunc func(uint64) []string
		if m.SymbolizationTable != nil {
			symfunc = func(addr uint64) []string {
				offset := addr - m.CodeBegin
				return r.symbolizeLocation(ctx, m, offset)
			}
		} else {
			symfunc = func(addr uint64) []string {
				return []string{name}
			}
		}

		s = append(s, symboltable.Entry[symFunc]{
			Data:  symfunc,
			Begin: m.CodeBegin,
			Size:  m.CodeEnd - m.CodeBegin,
		})
	}
	slices.SortFunc(s, func(lhs, rhs symboltable.Entry[symFunc]) int {
		return cmp.Compare(lhs.Begin, rhs.Begin)
	})
	r.l.Info(ctx, "Updating compiled symbols", logfield.CurrentNamespacePID(pid), log.Int("count", len(s)))
	r.compiledSyms.Put(pid, s)
}

func (r *Registry) scanSingleProcess(ctx context.Context, tp *trackedProcess) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	if !tp.initialized {
		return nil
	}
	res, err := r.helper.Scan(ctx, &jvmsupp.ScanRequest{
		Pid: int64(tp.pid),
	})
	if err != nil {
		status, ok := status.FromError(err)
		if ok && status.Code() == codes.FailedPrecondition {
			r.l.Info(ctx, "Ignoring FAILED_PRECONDITION scan error", logfield.CurrentNamespacePID(tp.pid), log.String("message", status.Message()))
		} else {
			r.l.Error(ctx, "Unexpected scan error", logfield.CurrentNamespacePID(tp.pid), log.Stringer("code", status.Code()), log.String("message", status.Message()))
			// TODO:
			// return fmt.Errorf("scanner call failed: %w", err)
		}
		return nil
	}
	r.methodNameReads.Add(res.Metrics.MethodNameReads)
	r.updateSymbols(ctx, tp.pid, res.Methods)
	dwarf, err := synthesizeDWARF(res.Methods)
	if err != nil {
		return fmt.Errorf("failed to synthesize dwarf for detected method: %w", err)
	}
	oldAlloc := tp.alloc
	if oldAlloc != nil {
		// TODO(PERFORATOR-1170): we are releasing old allocation before adding new one.
		// This is bad as ongoing BPF program executions may behave incorrectly in two ways:
		// 1. program looks root up between Release and Add and finds nothing.
		// 2. program looks root up before Release and starts traversing the tree,
		// but pages are rewritten by a concurrent Add that reuses them (as they are already in the freelist after Release),
		// so program reads effectively garbage.
		// In both cases recorded sample will be truncated or wrong.
		r.l.Debug(ctx, "Releasing previous unwind table allocation", logfield.CurrentNamespacePID(tp.pid))
		r.unwind.Release(tp.alloc)
	}
	alloc, err := r.unwind.Add(unwindtable.AllocationID{PID: tp.pid}, dwarf)
	if err != nil {
		return fmt.Errorf("failed to allocate bpf unwind table: %w", err)
	}

	tp.alloc = alloc

	return nil
}

func (r *Registry) listTargets() []*trackedProcess {
	r.trackedMu.Lock()
	defer r.trackedMu.Unlock()
	var targets []*trackedProcess
	for _, tp := range r.tracked {
		targets = append(targets, tp)
	}
	return targets
}

func (r *Registry) scanAll(ctx context.Context) error {
	r.scanIterations.Add(1)
	targets := r.listTargets()

	r.l.Debug(ctx, "Scanning targets", log.Int("count", len(targets)))

	r.trackedMu.Lock()
	defer r.trackedMu.Unlock()
	for _, tp := range targets {
		r.processScans.Add(1)
		err := r.scanSingleProcess(ctx, tp)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) runProcessScanner(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Second)
	for {
		err := r.scanAll(ctx)
		if err != nil {
			if errors.Is(err, context.Cause(ctx)) {
				return nil
			}
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func isSignaled(e error) bool {
	var errExit *exec.ExitError
	ok := errors.As(e, &errExit)
	if !ok {
		return false
	}
	if errExit.Exited() {
		return false
	}
	waitStatus := errExit.Sys().(syscall.WaitStatus)
	// we don't check signal here because different cancellation scenarios may lead
	// to different signals. E.g. context cancellation leads to SIGKILL,
	// but user pressing Ctrl-C may lead to SIGINT instead.
	return waitStatus.Signaled()
}

func (r *Registry) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	r.startHelper(ctx, eg)
	eg.Go(func() error {
		<-ctx.Done()
		err := r.grpcClient.Close()
		if err != nil {
			r.l.Warn(ctx, "Failed to close helper client", log.Error(err))
		}
		return nil
	})
	eg.Go(func() error {
		_, err := r.helper.KeepAlive(ctx, &jvmsupp.KeepAliveRequest{}, grpc.WaitForReady(true))
		if err != nil {
			grpcStatus, ok := status.FromError(err)
			isCanceledByContext := ok && grpcStatus.Code() == codes.Canceled && ctx.Err() != nil
			if isCanceledByContext {
				// we are shutting down, and call got canceled. It's likely
				// that it is client-side cancelation, and that is fine.
				return nil
			}
			return fmt.Errorf("keepalive call failed: %w", err)
		}
		return fmt.Errorf("keepalive call succeeded unexpectedly")
	})
	eg.Go(func() error {
		err := r.runProcessScanner(ctx)
		if err != nil {
			return fmt.Errorf("background process scanner failed: %w", err)
		}
		return nil
	})

	return eg.Wait()
}

func (r *Registry) ensureRegistered(ctx context.Context, pid linux.CurrentNamespacePID, binaryID uint64, baseAddress uint64) {
	var tp *trackedProcess
	var ok bool
	r.trackedMu.Lock()
	tp, ok = r.tracked[pid]
	if !ok {
		tp = &trackedProcess{
			pid: pid,
		}
		r.tracked[pid] = tp
		r.trackedProcessCount.Add(1)
	}
	// make sure that other processes can't observe potentially-newly-created tp before us
	tp.mu.Lock()
	defer tp.mu.Unlock()
	r.trackedMu.Unlock()
	if !ok {
		r.l.Debug(ctx, "Initializing new tracked process", logfield.CurrentNamespacePID(pid))
	} else if !tp.initialized {
		r.l.Debug(ctx, "Retrying process initialization", logfield.CurrentNamespacePID(pid))
	} else {
		r.l.Debug(ctx, "Process is already registered and initialized", logfield.CurrentNamespacePID(pid))
		return
	}
	var data *binaryData
	r.cheatsheetsMu.Lock()
	data = r.cheatsheets[binaryID]
	r.cheatsheetsMu.Unlock()

	if data == nil {
		r.l.Error(ctx, "Missing per-binary information", logfield.CurrentNamespacePID(pid), log.UInt64("binary_id", binaryID))
		return
	}

	res, err := r.helper.InitProcess(ctx, &jvmsupp.InitProcessRequest{
		Pid:            uint32(pid),
		LibjvmBinaryId: uint64(binaryID),
		Version:        data.version,
		Cheatsheet:     data.cheatsheet,
		BaseAddress:    baseAddress,
	})
	if err != nil {
		r.l.Info(ctx, "Failed to initialize process", logfield.CurrentNamespacePID(pid), log.Error(err))
		return
	}
	err = r.bpf.PutJVMProcessConfig(pid, unwinder.JvmProcessConfig{
		InterpreterBegin: res.InterpreterBegin,
		InterpreterEnd:   res.InterpreterEnd,
	})
	if err != nil {
		r.l.Info(ctx, "Failed to store process config", logfield.CurrentNamespacePID(pid), log.Error(err))
		return
	}
	r.l.Info(ctx, "Process successfully initialized", logfield.CurrentNamespacePID(pid))
	r.cheatsheetsMu.Lock()
	data.cheatsheet = nil
	r.cheatsheetsMu.Unlock()
	tp.initialized = true
}

func (r *Registry) OnProcessDeath(ctx context.Context, pid linux.CurrentNamespacePID) {
	logCtx := ctxlog.WithFields(ctx, logfield.CurrentNamespacePID(pid), log.String("event", "discovery"))
	r.trackedMu.Lock()
	defer r.trackedMu.Unlock()
	tp := r.tracked[pid]
	if tp == nil {
		return
	}
	r.trackedProcessCount.Add(-1)
	delete(r.tracked, pid)
	tp.mu.Lock()
	defer tp.mu.Unlock()
	if tp.alloc != nil {
		r.l.Debug(logCtx, "Releasing unwind table allocation of dead process")
		r.unwind.Release(tp.alloc)
	}
}
