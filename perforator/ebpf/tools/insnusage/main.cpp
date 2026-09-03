// SPDX-License-Identifier: GPL-2.0-or-later
//
// insnusage: per-instruction BPF verifier complexity profiler.
//
// Loads each BPF program from an ELF with the verifier log enabled
// (LOG_LEVEL=2 → per-instruction trace), parses per-offset visit counts,
// and uses LLVM's DWARFContext to attribute every visit to its full inline
// call stack (DW_TAG_inlined_subroutine with DW_AT_call_file/line). For
// ELFs without DWARF, falls back to BTFContext (same DIContext API, lower
// fidelity).

#include <perforator/lib/llvmex/llvm_exception.h>
#include <perforator/lib/profile/builder.h>
#include <perforator/lib/profile/pprof.h>
#include <perforator/proto/profile/profile.pb.h>

#include <library/cpp/getopt/last_getopt.h>

#include <util/digest/numeric.h>
#include <util/folder/path.h>
#include <util/generic/hash.h>
#include <util/generic/string.h>
#include <util/generic/vector.h>
#include <util/stream/file.h>
#include <util/stream/output.h>
#include <util/stream/zlib.h>
#include <util/string/printf.h>

#include <llvm/DebugInfo/BTF/BTFContext.h>
#include <llvm/DebugInfo/DIContext.h>
#include <llvm/DebugInfo/DWARF/DWARFContext.h>
#include <llvm/DebugInfo/DWARF/DWARFCompileUnit.h>
#include <llvm/DebugInfo/DWARF/DWARFDebugLine.h>
#include <llvm/DebugInfo/DWARF/DWARFDie.h>
#include <llvm/DebugInfo/DWARF/DWARFUnit.h>
#include <llvm/Object/Binary.h>
#include <llvm/Object/ELFObjectFile.h>
#include <llvm/Object/ObjectFile.h>
#include <llvm/Support/MemoryBuffer.h>

#include <linux/bpf.h>

#include <bpf.h>
#include <btf.h>
#include <libbpf.h>

#include <cerrno>
#include <cstring>
#include <map>
#include <string>

namespace {

// Kernel caps `bpf_attr.log_size` at `UINT_MAX >> 2` (~1 GiB) all the way
// back to 5.4 — verified against the source.
constexpr size_t kVerifierLogBufSize = UINT_MAX >> 2;

// Hand-rolled parsers for the two verifier-log line shapes we care about.
// std::regex took 30+ s on a 40 MB log here (the inner state-machine and
// per-call allocations dominated); these are linear scans with no heap.
//
// Instruction visit:  "  123: (b7) r0 = 0  ; ..."
// Function marker:    "func#3 @186"

// Scan an unsigned decimal at line[i..]; advance i past the digits and store
// the value in *v. Returns false if no digit is at line[i].
bool ScanU32(TStringBuf line, size_t& i, ui32& v) {
    if (i >= line.size() || line[i] < '0' || line[i] > '9') return false;
    v = 0;
    while (i < line.size() && line[i] >= '0' && line[i] <= '9') {
        v = v * 10 + (line[i] - '0');
        i++;
    }
    return true;
}

bool ParseFuncMarker(TStringBuf line, ui32* idx, ui32* off) {
    if (!line.StartsWith("func#")) return false;
    line.Skip(5);
    size_t i = 0;
    if (!ScanU32(line, i, *idx)) return false;
    if (i >= line.size() || line[i] != ' ') return false;
    while (i < line.size() && line[i] == ' ') i++;
    if (i >= line.size() || line[i] != '@') return false;
    i++;
    return ScanU32(line, i, *off);
}

// Verifier log uses *pre-xlated* offsets (libbpf's instruction stream as-is),
// but `bpf_prog_info.func_info.insn_off` from the kernel reports *post*-xlated
// positions (after spectre fixups, helper inlining, LD_IMM64 expansion).
// The log preamble's `func#N @M` headers expose pre-xlated function starts
// in BTF order — the only path to ELF byte addresses without reimplementing
// libbpf's merge logic against raw BTF.ext.
struct TLogFuncMarker {
    ui32 Index = 0;
    ui32 Offset = 0;
};

// A unique verifier-visit position: the visited insn offset plus the chain
// of call-site offsets that landed us here. CallerOffsets[0] is the
// outermost caller's call insn; CallerOffsets.back() is the immediate
// parent's call insn. Empty for frame-0 visits (entry/global subprograms).
//
// Static BPF subprograms (the ones the verifier walks INTO at every call
// site, accruing complexity per site) need this distinction — without it,
// their body's visits collapse onto a single root and we can't tell which
// caller drove them. Global BPF subprograms verify at frame 0 with empty
// CallerOffsets, so they naturally end up as their own roots.
struct TVerifierVisitKey {
    ui32 Offset = 0;
    TStackVec<ui32, 4> CallerOffsets;

    bool operator==(const TVerifierVisitKey& o) const noexcept {
        return Offset == o.Offset && CallerOffsets == o.CallerOffsets;
    }
};

struct TVerifierVisitKeyHash {
    size_t operator()(const TVerifierVisitKey& v) const noexcept {
        size_t h = v.Offset;
        for (ui32 c : v.CallerOffsets) {
            h = CombineHashes<size_t>(h, c);
        }
        return h;
    }
};

using TVisitMap = THashMap<TVerifierVisitKey, ui64, TVerifierVisitKeyHash>;

// Parse the verifier log line-by-line. Beyond the per-instruction visit
// counts we already extracted, also reconstruct the verifier's frame stack
// at every visited instruction. The kernel emits four explicit markers we
// rely on (kernel/bpf/verifier.c):
//
//   "caller:\n"                      — about to print caller state at call
//   "callee:\n"                      — about to print callee state at call
//                                       (frame just pushed)
//   "returning from callee:\n"       — about to print returning state
//   "to caller at <K>:\n"            — frame just popped, next insn is K
//
// We push the most recent insn offset (the `call` instruction) onto the
// stack on the caller→callee transition, and pop on the callee→caller
// transition. "Validating <name>() func#<N>..." headers reset the stack
// to empty (each globally-verified subprog starts at frame 0).
//
// State backtracking ("from X to Y:") prints the new state inline, which
// includes a " frame<N>:" prefix when N > 0. We use this to truncate the
// stack if the verifier rewound past one or more call frames.
void ParseVerifierLog(
    const TStringBuf log,
    TVisitMap& visits,
    ui64& totalVisits,
    TVector<TLogFuncMarker>& markers
) {
    TStackVec<ui32, 4> callStack;
    ui32 lastInsnOffset = Max<ui32>();

    // The kernel's verifier `push_stack(env, target_idx, branch_idx, ...)`
    // saves the full call frame whenever it encounters a conditional branch
    // and decides to explore one path while saving the other for later.
    // Saves are keyed by `(target_idx, depth)` and form a LIFO that gets
    // unwound by `pop_stack` as the verifier exhausts each branch. We
    // mirror the kernel's stack: each conditional branch we observe pushes
    // a snapshot of the current callStack onto BOTH possible target keys
    // (we don't know which path the verifier explored first), and each
    // state-line resync pops the top of the matching key. LIFO push order
    // = LIFO pop order, regardless of which target the kernel actually
    // saved at each branch (every push records the same chain).
    struct TSnapKey {
        ui32 Offset = 0;
        ui32 Depth = 0;
        bool operator==(const TSnapKey& o) const noexcept {
            return Offset == o.Offset && Depth == o.Depth;
        }
    };
    struct TSnapKeyHash {
        size_t operator()(const TSnapKey& k) const noexcept {
            return CombineHashes<size_t>(k.Offset, k.Depth);
        }
    };
    THashMap<TSnapKey, TVector<TStackVec<ui32, 4>>, TSnapKeyHash> snapshots;

    // Parse the trailing ` frame<N>:` from a state line, defaulting to 0
    // when absent (kernel only emits the prefix when frameno > 0). Returns
    // false on malformed input.
    auto parseFrameDepth = [](TStringBuf s, ui32& f) -> bool {
        constexpr TStringBuf kFramePrefix = " frame";
        size_t pos = s.find(kFramePrefix);
        if (pos == TStringBuf::npos) {
            f = 0;
            return true;
        }
        size_t i = pos + kFramePrefix.size();
        if (!ScanU32(s, i, f)) return false;
        return i < s.size() && s[i] == ':';
    };

    auto popSnapshot = [&](ui32 x, ui32 depth) {
        auto it = snapshots.find(TSnapKey{x, depth});
        if (it != snapshots.end() && !it->second.empty()) {
            callStack = it->second.back();
            it->second.pop_back();
            return;
        }
        while (callStack.size() > depth) callStack.pop_back();
    };

    auto resyncFromState = [&](TStringBuf s) {
        // "from X to Y: <state>" — pop the snapshot keyed by the target
        // Y at the embedded depth N. Kernel saves are keyed by target.
        ui32 depth = 0;
        if (!parseFrameDepth(s, depth)) return;
        size_t toPos = s.find(" to ");
        if (toPos == TStringBuf::npos) return;
        size_t i = toPos + 4;
        ui32 y = 0;
        if (!ScanU32(s, i, y)) return;
        popSnapshot(y, depth);
    };

    TStringBuf rest = log;
    while (!rest.empty()) {
        TStringBuf line;
        if (!rest.TrySplit('\n', line, rest)) {
            line = rest;
            rest = TStringBuf{};
        }

        TStringBuf trimmed = line;
        while (!trimmed.empty() && (trimmed[0] == ' ' || trimmed[0] == '\t')) {
            trimmed.Skip(1);
        }
        if (trimmed.empty()) continue;

        // The dominant line shape is `<digit>: (<hex>) ...` (instruction
        // visits). Dispatch by first character so the 99% case skips the
        // marker checks entirely.
        switch (trimmed[0]) {
        case 'c':
            // `caller:` is just a label preceding `callee:`; the push
            // happens on `callee:` (the most recent insn was the call).
            if (trimmed.StartsWith("callee:")) {
                callStack.push_back(lastInsnOffset);
            }
            continue;
        case 'r':
            // "returning from callee:" — noop, the matching "to caller
            // at K:" pops the stack.
            continue;
        case 't':
            if (trimmed.StartsWith("to caller at ") && !callStack.empty()) {
                callStack.pop_back();
            }
            continue;
        case 'f':
            if (trimmed.StartsWith("from ")) {
                resyncFromState(trimmed);
            } else {
                ui32 idx = 0, off = 0;
                if (ParseFuncMarker(trimmed, &idx, &off)) {
                    markers.push_back({idx, off});
                }
            }
            continue;
        case 'V':
            if (trimmed.StartsWith("Validating ")) {
                callStack.clear();
            }
            continue;
        default:
            break;
        }

        // Both instruction lines and state-print lines start with
        // `<idx>:`. After the colon and any whitespace, an instruction
        // continues with `(`; a state line continues with `R<n>=...` or
        // `frame<N>: R<n>=...` (sometimes ` fp<n>=...` for stack slots).
        // Pre-5.6 kernels at LOG_LEVEL=2 emit a state line *before every
        // instruction visit* — which is the only signal we get for
        // pop_stack restores on those kernels (they never emit `from X
        // to Y:` at LEVEL=2). Treat any digit-led non-instruction line as
        // a state line and resync from the embedded frame depth.
        size_t i = 0;
        ui32 off = 0;
        if (!ScanU32(trimmed, i, off)) continue;
        if (i >= trimmed.size() || trimmed[i] != ':') continue;
        i++;
        while (i < trimmed.size() && (trimmed[i] == ' ' || trimmed[i] == '\t')) i++;
        if (i >= trimmed.size()) continue;
        char c = trimmed[i];

        if (c == 'R' || c == 'f') {
            // State line: depth mismatch ⇒ kernel did a state-cache
            // pop_stack we didn't see explicit markers for; pop our
            // matching snapshot.
            ui32 depth = 0;
            if (!parseFrameDepth(trimmed, depth)) continue;
            if (depth == callStack.size()) continue;
            popSnapshot(off, depth);
            continue;
        }

        if (c != '(') continue; // unknown digit-led line shape (e.g. "X: safe")

        TVerifierVisitKey k;
        k.Offset = off;
        k.CallerOffsets.assign(callStack.begin(), callStack.end());
        visits[k]++;
        totalVisits++;
        lastInsnOffset = off;

        // Conditional-branch detection: the kernel's `push_stack` for the
        // alternative target happens here. Instructions like `if r1 == 0
        // goto pc+5` save the call frame keyed by ONE of the two
        // potential targets (taken/fall-through) — we don't know which,
        // so push the same snapshot under both keys. Since both keys get
        // the same chain (the chain at this branch insn), kernel pops at
        // either target draw the right chain off our stack.
        if (!callStack.empty()) {
            constexpr TStringBuf kGotoPc = " goto pc";
            size_t ifPos = trimmed.find(" if ");
            size_t gotoPos = trimmed.find(kGotoPc);
            if (ifPos != TStringBuf::npos && gotoPos != TStringBuf::npos && gotoPos > ifPos) {
                size_t op = gotoPos + kGotoPc.size();
                bool neg = false;
                if (op < trimmed.size() && (trimmed[op] == '+' || trimmed[op] == '-')) {
                    neg = (trimmed[op] == '-');
                    op++;
                }
                ui32 jmpOff = 0;
                if (ScanU32(trimmed, op, jmpOff)) {
                    ui32 takenTarget = neg ? off + 1 - jmpOff : off + 1 + jmpOff;
                    ui32 altTarget = off + 1;
                    ui32 d = ui32(callStack.size());
                    snapshots[TSnapKey{takenTarget, d}].push_back(callStack);
                    if (altTarget != takenTarget) {
                        snapshots[TSnapKey{altTarget, d}].push_back(callStack);
                    }
                }
            }
        }
    }
}

// DWARF queries need (byte_addr, section_index), but kernel func_info exposes
// merged-stream offsets and type ids. We index sections by name and function
// symbols by (section, offset, size) to bridge the two coordinate spaces.
class TElfInspector {
public:
    struct TFunctionSymbol {
        TString SectionName;
        ui64 SectionIndex = 0;
        ui64 ByteOffsetInSection = 0;
        ui64 Size = 0;
    };

    explicit TElfInspector(llvm::object::ObjectFile* obj) {
        for (auto sec : obj->sections()) {
            auto name = Y_LLVM_RAISE(sec.getName());
            SectionsByName_[TString(name.data(), name.size())] = sec.getIndex();
        }
        for (auto sym : obj->symbols()) {
            auto type = Y_LLVM_RAISE(sym.getType());
            if (type != llvm::object::SymbolRef::ST_Function) {
                continue;
            }
            auto secIt = Y_LLVM_RAISE(sym.getSection());
            if (secIt == obj->section_end()) {
                continue;
            }
            auto secName = Y_LLVM_RAISE(secIt->getName());
            auto name = Y_LLVM_RAISE(sym.getName());
            TFunctionSymbol fs;
            fs.SectionName = TString(secName.data(), secName.size());
            fs.SectionIndex = secIt->getIndex();
            fs.ByteOffsetInSection = Y_LLVM_RAISE(sym.getValue());
            // BPF .o symbols carry their function size via ELFSymbolRef::getSize.
            fs.Size = llvm::object::ELFSymbolRef(sym).getSize();
            Functions_[TString(name.data(), name.size())] = fs;
        }
    }

    const TFunctionSymbol* FunctionByName(const TString& name) const {
        auto it = Functions_.find(name);
        return it == Functions_.end() ? nullptr : &it->second;
    }

    ui64 SectionIndexByName(const TString& name) const {
        auto it = SectionsByName_.find(name);
        return it == SectionsByName_.end()
            ? llvm::object::SectionedAddress::UndefSection
            : it->second;
    }

private:
    THashMap<TString, ui64> SectionsByName_;
    THashMap<TString, TFunctionSymbol> Functions_;
};

// TFuncInfo carries a single bpf_func_info entry as returned by the kernel
// after load. `InsnOffMerged` is the merged-stream instruction offset (the
// units the verifier log uses); `Name` comes from BTF resolved via TypeId.
struct TFuncInfo {
    ui32 InsnOffMerged = 0;
    ui32 TypeId = 0;
    TString Name;
};

struct TLoadedProgram {
    TString Name;
    TString SectionName;
    TVector<char> Log;
    bool Loaded = false;
    TString LoadError;
    TVector<TFuncInfo> FuncInfo; // sorted by InsnOffMerged
    ui32 TotalInsns = 0;

    explicit TLoadedProgram(size_t logSize)
        : Log(logSize, '\0')
    {}
};

// Pulls bpf_prog_info from the kernel for the loaded program; specifically,
// the merged func_info table (one entry per distinct subprogram in the
// merged stream) and BTF type id used to resolve subprogram names. We do
// this in two passes: query sizes, then allocate and re-query.
TVector<TFuncInfo> FetchFuncInfo(int progFd, struct btf* btfObj) {
    TVector<TFuncInfo> out;
    bpf_prog_info info = {};
    ui32 infoLen = sizeof(info);
    if (bpf_obj_get_info_by_fd(progFd, &info, &infoLen) != 0) {
        return out;
    }
    if (info.nr_func_info == 0 || info.func_info_rec_size < sizeof(ui32) * 2) {
        return out;
    }
    TVector<char> buf(info.nr_func_info * info.func_info_rec_size);
    bpf_prog_info q = {};
    q.func_info_rec_size = info.func_info_rec_size;
    q.func_info = reinterpret_cast<ui64>(buf.data());
    q.nr_func_info = info.nr_func_info;
    infoLen = sizeof(q);
    if (bpf_obj_get_info_by_fd(progFd, &q, &infoLen) != 0) {
        return out;
    }
    out.reserve(q.nr_func_info);
    for (ui32 i = 0; i < q.nr_func_info; i++) {
        // Each record's first 8 bytes are insn_off (u32) + type_id (u32).
        // libbpf may extend the record format; we only read the kernel-min
        // prefix.
        const char* rec = buf.data() + (ui64)i * info.func_info_rec_size;
        TFuncInfo fi;
        std::memcpy(&fi.InsnOffMerged, rec, 4);
        std::memcpy(&fi.TypeId, rec + 4, 4);
        if (btfObj && fi.TypeId) {
            const auto* t = btf__type_by_id(btfObj, fi.TypeId);
            if (t && btf_kind(t) == BTF_KIND_FUNC) {
                if (const char* nm = btf__name_by_offset(btfObj, t->name_off)) {
                    fi.Name = nm;
                }
            }
        }
        out.push_back(std::move(fi));
    }
    std::sort(out.begin(), out.end(), [](const TFuncInfo& a, const TFuncInfo& b) {
        return a.InsnOffMerged < b.InsnOffMerged;
    });
    return out;
}

// Given a merged offset and the per-program func_info table (sorted), find
// which function entry contains it. Returns nullptr if before the first or
// outside coverage.
const TFuncInfo* LookupFuncForOffset(const TVector<TFuncInfo>& funcs, ui32 off) {
    if (funcs.empty() || off < funcs.front().InsnOffMerged) {
        return nullptr;
    }
    auto it = std::upper_bound(funcs.begin(), funcs.end(), off, [](ui32 v, const TFuncInfo& f) {
        return v < f.InsnOffMerged;
    });
    if (it == funcs.begin()) {
        return nullptr;
    }
    --it;
    return &*it;
}

// Resolves a merged offset to a (section_index, byte_address) pair suitable
// for DWARFContext::getInliningInfoForAddress. Returns false if any step
// failed (unknown function, missing symbol).
bool MergedOffsetToAddress(
    ui32 mergedOff,
    const TVector<TFuncInfo>& funcs,
    const TElfInspector& elf,
    ui64& outSectionIndex,
    ui64& outByteAddr,
    TString& outFnName
) {
    const auto* fn = LookupFuncForOffset(funcs, mergedOff);
    if (!fn || fn->Name.empty()) {
        return false;
    }
    const auto* sym = elf.FunctionByName(fn->Name);
    if (!sym) {
        return false;
    }
    outSectionIndex = sym->SectionIndex;
    outByteAddr = sym->ByteOffsetInSection + (ui64)(mergedOff - fn->InsnOffMerged) * sizeof(bpf_insn);
    outFnName = fn->Name;
    return true;
}

// Loads each program in the ELF with verifier log capture enabled. When
// `programFilter` is non-empty, programs whose name doesn't match are set
// to autoload=false so the kernel never verifies them — saves substantial
// kernel time when only one program is wanted.
TVector<TLoadedProgram> LoadAllPrograms(const TString& elfPath, const TString& programFilter) {
    TVector<TLoadedProgram> out;

    LIBBPF_OPTS(bpf_object_open_opts, openOpts);
    bpf_object* obj = bpf_object__open_file(elfPath.c_str(), &openOpts);
    if (!obj) {
        ythrow yexception() << "bpf_object__open_file(" << elfPath << ") failed: "
            << ::strerror(errno);
    }

    bpf_program* prog = nullptr;
    bpf_object__for_each_program(prog, obj) {
        TString name = bpf_program__name(prog);
        if (!programFilter.empty() && name != programFilter) {
            bpf_program__set_autoload(prog, false);
            continue;
        }
        out.emplace_back(kVerifierLogBufSize);
        auto& slot = out.back();
        slot.Name = std::move(name);
        slot.SectionName = bpf_program__section_name(prog);
        // BPF_LOG_LEVEL2 (= 2) enables the per-instruction trace; the kernel
        // also prefixes each new source line with `; <text>` when BTF.ext
        // is attached. The macro isn't in the vendored headers, hence the
        // literal.
        bpf_program__set_log_level(prog, 2);
        bpf_program__set_log_buf(prog, slot.Log.data(), slot.Log.size());
    }

    int err = bpf_object__load(obj);
    if (err) {
        for (auto& p : out) {
            p.Loaded = false;
            p.LoadError = TString::Join(
                "bpf_object__load failed: ", ::strerror(-err));
        }
        bpf_object__close(obj);
        return out;
    }

    struct btf* btfObj = bpf_object__btf(obj); // borrowed
    bpf_program* p = nullptr;
    bpf_object__for_each_program(p, obj) {
        if (!bpf_program__autoload(p)) {
            continue;
        }
        TString name = bpf_program__name(p);
        for (auto& slot : out) {
            if (slot.Name != name) continue;
            slot.Loaded = true;
            int fd = bpf_program__fd(p);
            if (fd >= 0) {
                slot.FuncInfo = FetchFuncInfo(fd, btfObj);
                slot.TotalInsns = bpf_program__insn_cnt(p);
            }
            break;
        }
    }

    bpf_object__close(obj);
    return out;
}

// DWARF / BTF debug info wrapper. `Ctx` is the DIContext we query for
// inline-stack lookups by byte address. `Object` keeps the ObjectFile alive
// since DIContext borrows from it.
struct TDebugInfo {
    llvm::object::OwningBinary<llvm::object::Binary> Owning;
    llvm::object::ObjectFile* Object = nullptr;
    std::unique_ptr<llvm::DIContext> Ctx;
    bool UsesDwarf = false;
};

bool DwarfHasUsableContent(llvm::DIContext* ctx) {
    auto* dwarf = llvm::dyn_cast<llvm::DWARFContext>(ctx);
    if (!dwarf) {
        return false;
    }
    return dwarf->getNumCompileUnits() > 0 || dwarf->getNumDWOCompileUnits() > 0;
}

TDebugInfo OpenDebugInfo(const TString& elfPath) {
    TDebugInfo di;
    auto bin = Y_LLVM_RAISE(llvm::object::createBinary(elfPath.c_str()));
    di.Owning = std::move(bin);
    auto* obj = llvm::dyn_cast<llvm::object::ObjectFile>(di.Owning.getBinary());
    if (!obj) {
        ythrow yexception() << elfPath << ": not an ObjectFile";
    }
    di.Object = obj;

    di.Ctx = llvm::DWARFContext::create(*obj);
    if (DwarfHasUsableContent(di.Ctx.get())) {
        di.UsesDwarf = true;
    } else {
        di.Ctx = llvm::BTFContext::create(*obj);
        di.UsesDwarf = false;
    }
    return di;
}

// Per-function aggregate keyed by leaf-frame name, so visits to instructions
// inlined into many BPF subprograms still credit the source-level function.
struct TLogicalFn {
    TString Name;
    TString DeclFile;
    ui32 DeclLine = 0;
    ui64 Visits = 0;
    size_t InsnCount = 0;
};

// Section-aware DWARF index. We need this because LLVM's
// DWARFContext::getInliningInfoForAddress -> DWARFUnit::getSubroutineForAddress
// builds a per-CU `AddrDieMap` keyed by `LowPC` only — it discards the
// `SectionIndex` carried by each `DWARFAddressRange`. For relocatable BPF
// `.o` files (where many SEC()-tagged entry functions all have low_pc=0
// in their respective sections), that map collides, returning a DIE from
// some random section instead of the one we asked about. Confirmed by
// reading lib/DebugInfo/DWARF/DWARFUnit.cpp updateAddressDieMap (line 717
// in LLVM 18; same in 21).
//
// The class mirrors LLVM's algorithm exactly (DWARFUnit's `AddrDieMap` +
// `getSubroutineForAddress` + `getInlinedChainForAddress`), with one
// change: the address map is keyed by `(section, LowPC)` instead of
// `LowPC`. Children DIEs are inserted after parents with overlap-
// splitting, so on byte-identical ranges (e.g., `return foo();` where
// the inner inline covers the whole outer inline) the deeper DIE
// naturally wins. The inline chain is built lazily at query time by
// walking DIE parents — same as LLVM.
struct TFrame {
    TString FunctionName;
    TString FileName;
    ui32 Line = 0;
    ui32 Column = 0;
};

struct TAnalysisResult {
    ui64 TotalVisits = 0;
    ui64 Resolved = 0;
    ui64 Unresolved = 0;
    THashMap<TString, TLogicalFn> Functions;
    // Pool of distinct DWARF inline chains, addressed by index from
    // TSample::Frames. Owns the storage so sample frame pointers stay
    // alive past AnalyzeProgram.
    TVector<TVector<TFrame>> ChainPool;
    // Per-visit stack samples. Frames is a list of verifier-frame inline
    // chains, leaf-first (Frames[0] = innermost call, Frames.back() =
    // outermost BPF subprogram, i.e. frame 0 in verifier terms). Each
    // entry indexes ChainPool. TProfileBuilder dedupes by hash on emit,
    // so we don't pre-aggregate.
    struct TSample {
        TStackVec<ui32, 4> Frames;
        ui64 Visits = 0;
    };
    TVector<TSample> Samples;
    // Per-file per-line visit counts for lcov output
    THashMap<TString, THashMap<ui32, ui64>> FileLineVisits;
};

struct TFunctionDecl {
    TString File;
    ui32 Line = 0;
};

class TSectionAwareDwarf {
public:
    explicit TSectionAwareDwarf(llvm::DWARFContext* ctx)
        : Ctx_{ctx}
    {
        if (!ctx) return;
        for (const auto& cu : ctx->compile_units()) {
            (void)cu->getNumDIEs(); // force extractDIEsIfNeeded(false)
            llvm::DWARFDie root = cu->getUnitDIE();
            if (root) UpdateAddressDieMap(root);
        }
    }

    const TFunctionDecl* DeclByName(const TString& name) const {
        auto it = DeclByName_.find(name);
        return it == DeclByName_.end() ? nullptr : &it->second;
    }

    // Look up the inline chain at (section, byte_addr). Frame 0 is the
    // innermost; the last frame is the real subprogram. Empty if no DIE
    // covers the address in this section.
    TVector<TFrame> Lookup(ui64 secIdx, ui64 addr) const {
        TVector<TFrame> out;
        llvm::DWARFDie leaf = GetSubroutineForAddress(secIdx, addr);
        if (!leaf) return out;

        // Walk parent DIE pointers, mirroring LLVM's
        // getInlinedChainForAddress: collect inlined_subroutine DIEs leaf
        // first, stop after the enclosing DW_TAG_subprogram.
        TStackVec<llvm::DWARFDie, 8> chain;
        for (llvm::DWARFDie die = leaf; die; die = die.getParent()) {
            if (die.isSubprogramDIE()) {
                chain.push_back(die);
                break;
            }
            if (die.getTag() == llvm::dwarf::DW_TAG_inlined_subroutine) {
                chain.push_back(die);
            }
        }
        if (chain.empty()) return out;

        out.reserve(chain.size());
        // Each frame's source location is the call site of the deeper
        // frame (or, for the leaf, the actual instruction's line table
        // entry). Same convention as LLVM's DIInliningInfo.
        ui32 pendingFile = 0, pendingLine = 0, pendingColumn = 0;
        for (size_t i = 0; i < chain.size(); i++) {
            llvm::DWARFDie die = chain[i];
            TFrame f;
            if (const char* nm = die.getSubroutineName(llvm::DINameKind::ShortName)) {
                f.FunctionName = nm;
            }
            if (i == 0) {
                FillLeafSourceLine(die.getDwarfUnit(), secIdx, addr, f);
            } else if (pendingFile || pendingLine) {
                f.FileName = ResolveCallFile(die, pendingFile);
                f.Line = pendingLine;
                f.Column = pendingColumn;
            }
            // Capture this frame's call_* attributes for the next outer
            // frame to consume (DW_AT_call_file/line are stored on the
            // inlined_subroutine itself and describe where it was
            // inlined in its parent).
            ui32 cd = 0;
            die.getCallerFrame(pendingFile, pendingLine, pendingColumn, cd);
            out.push_back(std::move(f));
        }
        return out;
    }

private:
    struct TKey {
        ui64 SecIdx;
        ui64 LowPc;
        bool operator<(const TKey& o) const {
            if (SecIdx != o.SecIdx) return SecIdx < o.SecIdx;
            return LowPc < o.LowPc;
        }
    };

    // Mirrors DWARFUnit::updateAddressDieMap. Inserts each subroutine
    // DIE's ranges, splitting any existing range that overlaps so the
    // child entry overrides the parent on byte-identical ranges. Walks
    // children depth-first after self so children appear after parents
    // in the map and overwrite them as needed.
    void UpdateAddressDieMap(llvm::DWARFDie die) {
        if (die.isSubroutineDIE()) {
            if (die.isSubprogramDIE()) {
                if (const char* nm = die.getSubroutineName(llvm::DINameKind::ShortName)) {
                    TFunctionDecl decl;
                    std::string df = die.getDeclFile(
                        llvm::DILineInfoSpecifier::FileLineInfoKind::AbsoluteFilePath);
                    if (!df.empty()) decl.File = df.c_str();
                    decl.Line = die.getDeclLine();
                    if (decl.Line || !decl.File.empty()) {
                        DeclByName_[nm] = std::move(decl);
                    }
                }
            }
            auto rangesOrErr = die.getAddressRanges();
            if (rangesOrErr) {
                for (const auto& r : *rangesOrErr) {
                    if (r.LowPC == r.HighPC) continue;
                    InsertRange(r.SectionIndex, r.LowPC, r.HighPC, die);
                }
            } else {
                llvm::consumeError(rangesOrErr.takeError());
            }
        }
        for (llvm::DWARFDie child = die.getFirstChild(); child;
             child = child.getSibling()) {
            UpdateAddressDieMap(child);
        }
    }

    void InsertRange(ui64 sec, ui64 lo, ui64 hi, llvm::DWARFDie die) {
        TKey key{sec, lo};
        auto B = AddrDieMap_.upper_bound(key);
        if (B != AddrDieMap_.begin()) {
            auto P = std::prev(B);
            // Only consider an enclosing range from the same section.
            if (P->first.SecIdx == sec && lo < P->second.HighPc) {
                if (hi < P->second.HighPc) {
                    AddrDieMap_[TKey{sec, hi}] = P->second;
                }
                if (lo > P->first.LowPc) {
                    P->second.HighPc = lo;
                }
            }
        }
        AddrDieMap_[key] = TEntry{hi, die};
    }

    llvm::DWARFDie GetSubroutineForAddress(ui64 sec, ui64 addr) const {
        auto R = AddrDieMap_.upper_bound(TKey{sec, addr});
        if (R == AddrDieMap_.begin()) return llvm::DWARFDie();
        --R;
        if (R->first.SecIdx != sec) return llvm::DWARFDie();
        if (addr >= R->second.HighPc) return llvm::DWARFDie();
        return R->second.Die;
    }

    void FillLeafSourceLine(llvm::DWARFUnit* unit, ui64 secIdx, ui64 addr, TFrame& out) const {
        if (!Ctx_ || !unit) return;
        const auto* lt = Ctx_->getLineTableForUnit(unit);
        if (!lt) return;
        llvm::DILineInfo info;
        if (lt->getFileLineInfoForAddress(
                {addr, secIdx}, unit->getCompilationDir(),
                llvm::DILineInfoSpecifier::FileLineInfoKind::AbsoluteFilePath,
                info)) {
            out.FileName = info.FileName.c_str();
            out.Line = info.Line;
            out.Column = info.Column;
        }
    }

    TString ResolveCallFile(llvm::DWARFDie die, ui32 fileIdx) const {
        if (!Ctx_ || !fileIdx) return {};
        const auto* lt = Ctx_->getLineTableForUnit(die.getDwarfUnit());
        if (!lt) return {};
        std::string s;
        if (lt->getFileNameByIndex(
                fileIdx, die.getDwarfUnit()->getCompilationDir(),
                llvm::DILineInfoSpecifier::FileLineInfoKind::AbsoluteFilePath, s)) {
            return TString(s.c_str());
        }
        return {};
    }

    struct TEntry {
        ui64 HighPc = 0;
        llvm::DWARFDie Die;
    };

    llvm::DWARFContext* Ctx_;
    std::map<TKey, TEntry> AddrDieMap_;
    THashMap<TString, TFunctionDecl> DeclByName_;
};

TAnalysisResult AnalyzeProgram(
    const TLoadedProgram& prog,
    const TDebugInfo& di,
    const TElfInspector& elf,
    const TSectionAwareDwarf& sectionDwarf
) {
    TAnalysisResult r;
    if (!prog.Loaded || prog.FuncInfo.empty()) {
        return r;
    }

    TVisitMap visits;
    TVector<TLogFuncMarker> markers;
    // libbpf writes a NUL-terminated C string into Log; trim to its actual
    // length (~much smaller than the 512 MiB buffer for most programs).
    TStringBuf logView{prog.Log.data(), ::strnlen(prog.Log.data(), prog.Log.size())};
    ParseVerifierLog(logView, visits, r.TotalVisits, markers);

    // Replace BTF func_info's post-xlated offsets with pre-xlated offsets
    // from the verifier log's `func#N @M` headers, keeping BTF names. The
    // re-sort guards against funcs no longer being ascending after the
    // overlay (kernel-version-dependent — both inputs *should* be sorted,
    // but binary search downstream requires it).
    TVector<TFuncInfo> funcs = prog.FuncInfo;
    if (markers.size() == funcs.size()) {
        for (size_t i = 0; i < funcs.size(); i++) {
            funcs[i].InsnOffMerged = markers[i].Offset;
        }
        std::sort(funcs.begin(), funcs.end(), [](const TFuncInfo& a, const TFuncInfo& b) {
            return a.InsnOffMerged < b.InsnOffMerged;
        });
    }

    // Pool DWARF chain results by offset; identical offsets recur many
    // times across the visit stream and as caller offsets across call
    // sites, and the DIE walk + line-table query is the slowest part of
    // analysis. `kUnresolved` in chainIdByOffset marks an unresolvable
    // offset so we don't retry it.
    constexpr ui32 kUnresolved = Max<ui32>();
    THashMap<ui32, ui32> chainIdByOffset;

    // Populate decl info for every function we see in any chain (not just
    // leaves), so the emitted profile has start_line on each TFunctionInfo
    // and a generic source-renderer can navigate to definitions without
    // consulting DWARF.
    auto ensureFnDecl = [&](const TString& name) {
        auto& fn = r.Functions[name];
        if (!fn.Name.empty()) return;
        fn.Name = name;
        if (const auto* decl = sectionDwarf.DeclByName(name)) {
            fn.DeclFile = decl->File;
            fn.DeclLine = decl->Line;
        }
    };

    auto chainIdAtOffset = [&](ui32 off) -> ui32 {
        auto it = chainIdByOffset.find(off);
        if (it != chainIdByOffset.end()) return it->second;
        ui64 secIdx = 0, byteAddr = 0;
        TString fnName;
        if (!MergedOffsetToAddress(off, funcs, elf, secIdx, byteAddr, fnName)) {
            chainIdByOffset.emplace(off, kUnresolved);
            return kUnresolved;
        }
        TVector<TFrame> frames = sectionDwarf.Lookup(secIdx, byteAddr);
        if (frames.empty()) {
            TFrame f;
            f.FunctionName = fnName;
            auto lineOnly = di.Ctx->getLineInfoForAddress(
                {byteAddr, secIdx},
                llvm::DILineInfoSpecifier{
                    llvm::DILineInfoSpecifier::FileLineInfoKind::AbsoluteFilePath,
                    llvm::DILineInfoSpecifier::FunctionNameKind::None});
            f.FileName = lineOnly.FileName.c_str();
            f.Line = lineOnly.Line;
            f.Column = lineOnly.Column;
            frames.push_back(std::move(f));
        }
        for (const auto& frame : frames) {
            ensureFnDecl(frame.FunctionName);
        }
        ui32 id = static_cast<ui32>(r.ChainPool.size());
        r.ChainPool.push_back(std::move(frames));
        chainIdByOffset.emplace(off, id);
        return id;
    };

    r.Samples.reserve(visits.size());
    for (const auto& [key, n] : visits) {
        ui32 leafId = chainIdAtOffset(key.Offset);
        if (leafId == kUnresolved) {
            r.Unresolved += n;
            continue;
        }
        r.Resolved += n;

        TAnalysisResult::TSample sample;
        sample.Visits = n;
        sample.Frames.push_back(leafId);
        for (auto it = key.CallerOffsets.rbegin(); it != key.CallerOffsets.rend(); ++it) {
            ui32 id = chainIdAtOffset(*it);
            if (id != kUnresolved) sample.Frames.push_back(id);
        }

        const TVector<TFrame>& leafFrames = r.ChainPool[leafId];
        auto& fn = r.Functions[leafFrames[0].FunctionName];
        // Fallback when DW_TAG_subprogram's DW_AT_decl_* couldn't be
        // resolved (some BPF .o builds lack the CU line table the
        // resolution depends on): use the leaf source line — better than
        // `:0`.
        if (fn.DeclLine == 0 && leafFrames[0].Line > 0) {
            fn.DeclFile = leafFrames[0].FileName;
            fn.DeclLine = leafFrames[0].Line;
        }
        fn.Visits += n;
        fn.InsnCount++;

        // Track per-file per-line visit counts for lcov
        if (!leafFrames[0].FileName.empty() && leafFrames[0].Line > 0) {
            r.FileLineVisits[leafFrames[0].FileName][leafFrames[0].Line] += n;
        }

        r.Samples.push_back(std::move(sample));
    }
    return r;
}

void PrintFunctionTable(IOutputStream& out, const TAnalysisResult& r) {
    TVector<const TLogicalFn*> fns;
    fns.reserve(r.Functions.size());
    for (const auto& [_, fn] : r.Functions) {
        fns.push_back(&fn);
    }
    std::sort(fns.begin(), fns.end(), [](const auto* a, const auto* b) {
        return a->Visits > b->Visits;
    });
    out << "--- functions (sorted by total verifier visits, inline-aware) ---" << Endl;
    out << Sprintf("%-12s %-12s %-32s %s\n", "visits", "insns", "decl@", "name");
    for (const auto* fn : fns) {
        TString decl = Sprintf("%s:%u",
            TFsPath(fn->DeclFile).GetName().c_str(), fn->DeclLine);
        out << Sprintf("%-12llu %-12zu %-32s %s\n",
            (unsigned long long)fn->Visits,
            fn->InsnCount, decl.c_str(), fn->Name.c_str());
    }
}

// Build a perforator-native profile (one ValueType: verifier-visits/count)
// from per-instruction frame stacks. Each verifier call frame becomes its
// own pprof stack frame, with its DWARF inline chain attached. Static BPF
// subprograms (verified per call site) thus nest under each caller in the
// flamegraph; global subprograms / SEC() entries verify at frame 0 and
// appear as roots. With BTF-only debug info each frame's inline chain is
// just one entry, but the cross-frame stack structure is preserved.
void EmitProfile(IOutputStream& out, const TVector<TAnalysisResult>& results) {
    NPerforator::NProto::NProfile::Profile proto;
    NPerforator::NProfile::TProfileBuilder builder(&proto);

    auto valueType = builder.AddValueType("verifier-visits", "count");

    using NPerforator::NProfile::TFunctionId;
    using NPerforator::NProfile::TFunctionInfo;
    using NPerforator::NProfile::TInlineChainInfo;
    using NPerforator::NProfile::TStackFrameInfo;

    THashMap<TString, TFunctionId> fnByName;

    for (const auto& r : results) {
        auto getFn = [&](const TFrame& frame) {
            auto it = fnByName.find(frame.FunctionName);
            if (it != fnByName.end()) return it->second;
            TFunctionInfo info;
            info.Name = builder.AddString(frame.FunctionName);
            info.SystemName = info.Name;
            // Prefer the function's decl file/line over the per-instruction
            // file (which can be a header included from elsewhere). Falls
            // back to the frame's own file when decl info is missing.
            TString file = frame.FileName;
            ui32 startLine = 0;
            if (auto fnIt = r.Functions.find(frame.FunctionName); fnIt != r.Functions.end()) {
                if (!fnIt->second.DeclFile.empty()) file = fnIt->second.DeclFile;
                startLine = fnIt->second.DeclLine;
            }
            info.FileName = builder.AddString(file);
            info.StartLine = startLine;
            TFunctionId id = builder.AddFunction(info);
            fnByName.emplace(frame.FunctionName, id);
            return id;
        };

        // Pre-resolve each unique chain to a TStackFrameId so duplicate
        // chains across samples don't repeat the AddFunction/AddInlineChain
        // /AddStackFrame work.
        TVector<NPerforator::NProfile::TStackFrameId> frameIdByChain;
        frameIdByChain.reserve(r.ChainPool.size());
        for (const auto& chain : r.ChainPool) {
            TInlineChainInfo chainInfo;
            chainInfo.Lines.reserve(chain.size());
            // `chain` is already innermost-first (see Lookup), which is the
            // order perforator/proto/profile/profile.proto (InlineChains)
            // and the pprof spec want — copy it as-is.
            for (const auto& frame : chain) {
                chainInfo.Lines.push_back({
                    .Function = getFn(frame),
                    .Line = frame.Line,
                    .Column = frame.Column,
                });
            }
            TStackFrameInfo frameInfo;
            frameInfo.InlineChain = builder.AddInlineChain(chainInfo);
            frameIdByChain.push_back(builder.AddStackFrame(frameInfo));
        }

        for (const auto& sample : r.Samples) {
            auto skb = builder.AddSimpleSampleKey();
            for (ui32 chainId : sample.Frames) {
                skb.AddFrame(frameIdByChain[chainId]);
            }
            builder.AddSample()
                .SetSampleKey(skb.Finish())
                .AddValue(valueType, sample.Visits)
                .Finish();
        }
    }

    std::move(builder).Finish();

    // Convert to pprof and gzip-compress per pprof convention. The
    // perforator-native format isn't consumed by any external tool yet,
    // so on-disk output is pprof — readable by `go tool pprof`,
    // speedscope, perf-folded, etc.
    TString pprofBytes;
    NPerforator::NProfile::ConvertToPProf(proto, &pprofBytes);
    TZLibCompress gz(&out, ZLib::GZip, 1);
    gz.Write(pprofBytes.data(), pprofBytes.size());
    gz.Finish();
}

void EmitLcov(IOutputStream &out, TAnalysisResult &result) {
    // Group functions by file name (SF)
    THashMap<TString, TVector<const TLogicalFn *>> functionsByFile;
    for (const auto &[_, fn] : result.Functions) {
        // We need an absolute path
        TString filePath = TFsPath(fn.DeclFile);
        functionsByFile[filePath].push_back(&fn);
    }

    // Sort functions within each file by line number
    for (auto &[_, fns] : functionsByFile) {
        std::sort(fns.begin(), fns.end(), [](const auto *a, const auto *b) {
            return a->DeclLine < b->DeclLine;
        });
    }

    // Write lcov records
    for (const auto &[filePath, fns] : functionsByFile) {
        // SF line
        out << "SF:" << filePath << '\n';

        // DA lines per-file per-function visit counts
        for (const auto *fn : fns) {
            out << "DA:" << fn->DeclLine << ',' << fn->Visits << '\n';
        }

        // DA lines per-file per-line visit counts
        const auto& lineVisits = result.FileLineVisits[filePath];
        for (const auto& [line, visits] : lineVisits) {
            out << "DA:" << line << ',' << visits << '\n';
        }

        // end_of_record
        out << "end_of_record\n";
    }
}

TAnalysisResult DumpProgram(
    const TLoadedProgram& prog,
    const TDebugInfo& di,
    const TElfInspector& elf,
    const TSectionAwareDwarf& sectionDwarf,
    const TString& verifierLogOut
) {
    Cout << "=== program " << prog.Name
         << " section=" << prog.SectionName
         << " loaded=" << (prog.Loaded ? "yes" : "no") << " ===" << Endl;
    if (!prog.Loaded) {
        Cout << "  load error: " << prog.LoadError << Endl;
        // Kernel writes the rejection reason into our log buffer before
        // returning; dumping the tail (where the failure message lands)
        // tells the user exactly which verifier check tripped.
        TStringBuf logView{prog.Log.data(), ::strnlen(prog.Log.data(), prog.Log.size())};
        if (!logView.empty()) {
            constexpr size_t kTailBytes = 4096;

            if (!verifierLogOut.empty() && !prog.Log.empty()) {
                TFileOutput out(verifierLogOut);

                out << logView;
                Cout << "wrote full verifier log to " << verifierLogOut << Endl;
            }

            if (logView.size() > kTailBytes) {
                logView = logView.SubStr(logView.size() - kTailBytes);
            }
            Cout << "  --- verifier log tail ---\n" << logView << Endl;
        }
        return {};
    }

    auto r = AnalyzeProgram(prog, di, elf, sectionDwarf);
    Cout << "  total verifier visits: " << r.TotalVisits
         << "  funcs: " << r.Functions.size()
         << "  insns: " << prog.TotalInsns
         << "  resolved: " << r.Resolved
         << "  unresolved: " << r.Unresolved << Endl;

    PrintFunctionTable(Cout, r);
    return r;
}

} // namespace

int main(int argc, const char* argv[]) {
    NLastGetopt::TOpts opts = NLastGetopt::TOpts::Default();
    TString path;
    TString programFilter;
    TString profileOut;
    TString verifierLogOut;
    TString lcovOut;

    opts
        .AddLongOption('p', "path", "Path to the eBPF ELF file")
        .Required()
        .StoreResult(&path);
    opts
        .AddLongOption("program", "Analyze only this program (default: all)")
        .Optional()
        .StoreResult(&programFilter);
    opts
        .AddLongOption("profile-out", "Write a gzipped pprof profile to this path")
        .Optional()
        .StoreResult(&profileOut);
    opts
        .AddLongOption("verifier-log-out", "Write full untruncated verifier log to this path on load failure")
        .Optional()
        .StoreResult(&verifierLogOut);
    opts
        .AddLongOption("lcov-out", "Write lcov data to this path")
        .Optional()
        .StoreResult(&lcovOut);
    NLastGetopt::TOptsParseResult res(&opts, argc, argv);

    try {
        TDebugInfo di = OpenDebugInfo(path);
        Cerr << "debug-info: " << (di.UsesDwarf ? "DWARF" : "BTF only") << Endl;

        TElfInspector elf(di.Object);
        // Built once: indexes every CU's DIE tree, doesn't depend on
        // which BPF program is being analyzed. Empty for BTF-only ELFs;
        // analysis falls back to di.Ctx's plain line-table queries.
        TSectionAwareDwarf sectionDwarf(llvm::dyn_cast<llvm::DWARFContext>(di.Ctx.get()));

        auto progs = LoadAllPrograms(path, programFilter);
        TVector<TAnalysisResult> results;
        for (const auto& prog : progs) {
            results.push_back(DumpProgram(prog, di, elf, sectionDwarf, verifierLogOut));
        }
        if (results.empty()) {
            Cerr << "no programs matched";
            if (!programFilter.empty()) {
                Cerr << " (filter: " << programFilter << ")";
            }
            Cerr << Endl;
            return 1;
        }

        if (!profileOut.empty()) {
            TFileOutput out(profileOut);
            EmitProfile(out, results);
            Cerr << "wrote profile to " << profileOut << Endl;
        }

        if (!lcovOut.empty()) {
            TFileOutput out(lcovOut);
            EmitLcov(out, results.front());

            Cerr << "wrote lcov data to " << lcovOut << Endl;
        }
    } catch (const std::exception& e) {
        Cerr << "error: " << e.what() << Endl;
        return 1;
    }
    return 0;
}
