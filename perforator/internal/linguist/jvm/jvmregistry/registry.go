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

var _ process.IdentityListener = (*Registry)(nil)

type trackedProcess struct {
	key linux.ProcessKey
	// Serializes scans and symbol publication with retirement.
	mu          sync.Mutex
	initialized bool
	retired     bool
	// The helper may still own the bootstrap key while registration is retried.
	helperKey  *linux.ProcessKey
	curMethods map[string]*jvmsupp.MethodInfo
	alloc      *unwindtable.Allocation
}

type interpretedSymbolCacheKey struct {
	process linux.ProcessKey
	addr    uint64
}

type interpretedSymbolCacheValue struct {
	name string
}

type binaryData struct {
	cheatsheet *jvm.Cheatsheet
	version    uint32
}

type symFunc = func(addr uint64) []string

type compiledSymbol struct {
	key     linux.ProcessKey
	resolve symFunc
}

type Registry struct {
	l      xlog.Logger
	bpf    *programstate.State
	unwind *unwindtable.BPFManager

	disambiguateFrameSource bool

	compiledSyms         *symboltable.Table[linux.CurrentNamespacePID, compiledSymbol]
	interpretedSymsCache *expirable.LRU[interpretedSymbolCacheKey, interpretedSymbolCacheValue]

	helperBinaryPath string
	helperSocketPath string

	enableLineInfoParsing bool
	enableBookmarks       bool

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
	methodScans    metrics.Counter
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

	EnableLineInfoParsing     bool
	EnableIncrementalScanning bool

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
		compiledSyms:            symboltable.New[linux.CurrentNamespacePID, compiledSymbol](),
		interpretedSymsCache:    interpretedSymsCache,
		disambiguateFrameSource: opts.DisambiguateFrameSource,

		helperBinaryPath: opts.ScannerBinaryPath,
		helperSocketPath: opts.SocketPath,

		enableLineInfoParsing: opts.EnableLineInfoParsing,
		enableBookmarks:       opts.EnableIncrementalScanning,

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
		r.methodScans = reg.Counter("scan.methods.count")
		r.methodNameReads = reg.Counter("scan.method.name_reads.count")
	} else {
		r.scanIterations = &nop.Counter{}
		r.processScans = &nop.Counter{}
		r.methodScans = &nop.Counter{}
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
			r.ensureRegistered(ctx, info.Key(), m.ID(), m.BaseAddress())
		}
	}
}

func (r *Registry) OnProcessRescan(ctx context.Context, info process.ProcessInfo) {
	r.OnProcessDiscovery(ctx, info)
}

func (r *Registry) SymbolizeInterpreted(ctx context.Context, pid linux.CurrentNamespacePID, addr uint64) (string, error) {
	return r.symbolizeInterpreted(ctx, linux.ProcessKey{Pid: pid}, addr)
}

func (r *Registry) SymbolizeInterpretedForProcess(ctx context.Context, process linux.ProcessKey, addr uint64) (string, bool, error) {
	r.trackedMu.Lock()
	tp := r.tracked[process.Pid]
	matched := process.ProcessStartTime != 0 && tp != nil && tp.key == process
	r.trackedMu.Unlock()
	if !matched {
		return "", false, nil
	}
	name, err := r.symbolizeInterpreted(ctx, process, addr)
	r.trackedMu.Lock()
	matched = r.tracked[process.Pid] == tp
	r.trackedMu.Unlock()
	if !matched {
		return "", false, nil
	}
	if status.Code(err) == codes.FailedPrecondition || status.Code(err) == codes.NotFound {
		return "", false, nil
	}
	return name, true, err
}

func (r *Registry) symbolizeInterpreted(ctx context.Context, process linux.ProcessKey, addr uint64) (string, error) {
	key := interpretedSymbolCacheKey{
		process: process,
		addr:    addr,
	}
	cached, ok := r.interpretedSymsCache.Get(key)
	if ok {
		r.interpretedMethodResolveCacheHit.Inc()
		return cached.name, nil
	}
	req := &jvmsupp.SymbolizeRequest{
		Pid:              int64(process.Pid),
		ProcessStartTime: process.ProcessStartTime,
		Methods:          []uint64{addr},
	}
	res, err := r.helper.Symbolize(ctx, req)
	if err != nil {
		r.interpretedMethodResolveInternalError.Inc()
		return "", fmt.Errorf("remote symbolization failed: %w", err)
	}
	if len(res.GetSymbolized()) != 1 || res.GetSymbolized()[0] == nil {
		return "", fmt.Errorf("invalid JVM symbolization response")
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
	symbol, ok := r.compiledSyms.Find(pid, ip)
	if ok {
		r.compiledMethodResolveSuccess.Inc()
		names := symbol.resolve(ip)
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

func (r *Registry) ResolveForProcess(key linux.ProcessKey, ip uint64) (profilerext.JITSymbolizerOutput, bool) {
	symbol, ok := r.compiledSyms.Find(key.Pid, ip)
	if !ok || key.ProcessStartTime == 0 || symbol.key != key {
		r.compiledMethodResolveFailure.Inc()
		return profilerext.JITSymbolizerOutput{}, false
	}
	r.compiledMethodResolveSuccess.Inc()
	names := symbol.resolve(ip)
	symbols := make([]profilerext.JITSymbol, len(names))
	for i, name := range names {
		symbols[i].Name = name
	}
	return profilerext.JITSymbolizerOutput{MappingName: profile.JVMSpecialMapping, Symbols: symbols}, true
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

func (r *Registry) updateSymbols(ctx context.Context, key linux.ProcessKey, methods []*jvmsupp.MethodInfo) {
	var s []symboltable.Entry[compiledSymbol]
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

		s = append(s, symboltable.Entry[compiledSymbol]{
			Data:  compiledSymbol{key: key, resolve: symfunc},
			Begin: m.CodeBegin,
			Size:  m.CodeEnd - m.CodeBegin,
		})
	}
	slices.SortFunc(s, func(lhs, rhs symboltable.Entry[compiledSymbol]) int {
		return cmp.Compare(lhs.Begin, rhs.Begin)
	})
	r.l.Info(ctx, "Updating compiled symbols", logfield.CurrentNamespacePID(key.Pid), log.Int("count", len(s)))
	r.compiledSyms.Put(key.Pid, s)
}

func (r *Registry) scanSingleProcess(ctx context.Context, tp *trackedProcess) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	if !tp.initialized || tp.retired {
		return nil
	}
	var bookmarks []string
	if r.enableBookmarks {
		for bookmark := range tp.curMethods {
			bookmarks = append(bookmarks, bookmark)
		}
	}
	res, err := r.helper.Scan(ctx, &jvmsupp.ScanRequest{
		Pid:              int64(tp.key.Pid),
		ProcessStartTime: tp.key.ProcessStartTime,
		Bookmarks:        bookmarks,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.FailedPrecondition, codes.NotFound:
			r.l.Debug(ctx, "Skipping JVM scan for unavailable process state", logfield.CurrentNamespacePID(tp.key.Pid), log.Error(err))
			return nil
		default:
			return fmt.Errorf("scanner call failed: %w", err)
		}
	}
	r.methodNameReads.Add(res.Metrics.MethodNameReads)
	r.methodScans.Add(int64(len(res.Methods)))
	fullMethods := slices.Clone(res.Methods)
	for _, b := range res.UnchangedMethods {
		m, ok := tp.curMethods[b]
		if !ok {
			return fmt.Errorf("scanner response mentions unknown string %q as an unchanged method bookmark", b)
		}
		fullMethods = append(fullMethods, m)
	}
	if r.enableBookmarks {
		clear(tp.curMethods)
		for _, m := range fullMethods {
			tp.curMethods[m.Bookmark] = m
		}
	}

	r.updateSymbols(ctx, tp.key, fullMethods)
	dwarf, err := synthesizeDWARF(fullMethods)
	if err != nil {
		return fmt.Errorf("failed to synthesize dwarf for detected method: %w", err)
	}
	oldAlloc := tp.alloc
	if oldAlloc != nil {
		// TODO(PERFORATOR-1170): Concurrent BPF readers may miss the root between Release
		// and Add, or read rewritten pages if Add reuses the previous allocation.
		r.l.Debug(ctx, "Releasing previous unwind table allocation", logfield.CurrentNamespacePID(tp.key.Pid))
		r.unwind.Release(tp.alloc)
	}
	alloc, err := r.unwind.Add(unwindtable.AllocationID{PID: tp.key.Pid}, dwarf)
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

func (r *Registry) scanAll(ctx context.Context) {
	r.scanIterations.Add(1)
	targets := r.listTargets()

	r.l.Debug(ctx, "Scanning targets", log.Int("count", len(targets)))

	for _, tp := range targets {
		if ctx.Err() != nil {
			return
		}
		r.processScans.Add(1)
		err := r.scanSingleProcess(ctx, tp)
		if err != nil {
			r.l.Warn(ctx, "Error during background JVM process refresh", logfield.CurrentNamespacePID(tp.key.Pid), log.Error(err))
		}
	}
}

func (r *Registry) runProcessScanner(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		r.scanAll(ctx)
		select {
		case <-ctx.Done():
			return
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
			isCanceledByContext := ctx.Err() != nil && (errors.Is(err, context.Canceled) || (ok && grpcStatus.Code() == codes.Canceled))
			if isCanceledByContext {
				return nil
			}
			return fmt.Errorf("keepalive call failed: %w", err)
		}
		return fmt.Errorf("keepalive call succeeded unexpectedly")
	})
	eg.Go(func() error {
		r.runProcessScanner(ctx)
		return nil
	})

	return eg.Wait()
}

func (r *Registry) ensureRegistered(ctx context.Context, key linux.ProcessKey, binaryID uint64, baseAddress uint64) {
	pid := key.Pid
	var tp *trackedProcess
	var ok bool
	r.trackedMu.Lock()
	tp, ok = r.tracked[pid]
	if ok && tp.key != key {
		tp.mu.Lock()
		tp.retired = true
		if tp.alloc != nil {
			r.unwind.Release(tp.alloc)
			tp.alloc = nil
		}
		tp.mu.Unlock()
		r.compiledSyms.Remove(pid)
		r.trackedProcessCount.Add(-1)
		ok = false
	}
	if !ok {
		tp = &trackedProcess{
			key:        key,
			curMethods: make(map[string]*jvmsupp.MethodInfo),
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
		Pid:              uint32(pid),
		ProcessStartTime: key.ProcessStartTime,
		LibjvmBinaryId:   uint64(binaryID),
		Version:          data.version,
		Cheatsheet:       data.cheatsheet,
		BaseAddress:      baseAddress,
	})
	if err != nil {
		r.l.Info(ctx, "Failed to initialize process", logfield.CurrentNamespacePID(pid), log.Error(err))
		return
	}
	tp.helperKey = &key
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

func (r *Registry) OnProcessDeath(ctx context.Context, key linux.ProcessKey) {
	r.retireProcess(ctx, key)
}

func (r *Registry) OnProcessIdentityChanged(ctx context.Context, previous, current linux.ProcessKey) {
	if previous.Pid != current.Pid || previous.ProcessStartTime != 0 || current.ProcessStartTime == 0 {
		return
	}
	r.trackedMu.Lock()
	defer r.trackedMu.Unlock()
	old := r.tracked[previous.Pid]
	if old == nil || old.key != previous {
		return
	}
	old.mu.Lock()
	defer old.mu.Unlock()
	old.retired = true
	// Keep Go symbols and unwind pages until the new registration is ready.
	helperKey := old.helperKey
	if helperKey == nil {
		helperKey = &previous
	}
	r.tracked[current.Pid] = &trackedProcess{
		key:        current,
		helperKey:  helperKey,
		alloc:      old.alloc,
		curMethods: make(map[string]*jvmsupp.MethodInfo),
	}
	old.helperKey = nil
	old.alloc = nil
}

func (r *Registry) retireProcess(ctx context.Context, key linux.ProcessKey) {
	pid := key.Pid
	logCtx := ctxlog.WithFields(ctx, logfield.CurrentNamespacePID(pid))
	r.trackedMu.Lock()
	tp := r.tracked[pid]
	if tp == nil || tp.key != key {
		r.trackedMu.Unlock()
		return
	}
	tp.mu.Lock()
	tp.retired = true
	r.compiledSyms.Remove(pid)
	r.trackedProcessCount.Add(-1)
	delete(r.tracked, pid)
	if tp.alloc != nil {
		r.l.Debug(logCtx, "Releasing unwind table allocation of retired process")
		r.unwind.Release(tp.alloc)
		tp.alloc = nil
	}
	helperKey := tp.helperKey
	tp.helperKey = nil
	tp.mu.Unlock()
	r.trackedMu.Unlock()
	releaseKeys := []linux.ProcessKey{key}
	if helperKey != nil && *helperKey != key {
		// A failed Init RPC may have replaced the helper state before returning.
		releaseKeys = append(releaseKeys, *helperKey)
	}
	for _, releaseKey := range releaseKeys {
		if _, err := r.helper.ReleaseProcess(ctx, &jvmsupp.ReleaseProcessRequest{Pid: uint32(pid), ProcessStartTime: releaseKey.ProcessStartTime}); err != nil {
			r.l.Warn(ctx, "Failed to release JVM process state", log.Error(err))
		}
	}
}
