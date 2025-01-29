#include <perforator/agent/preprocessing/lib/analyze.h>
#include <perforator/agent/preprocessing/proto/unwind/table.pb.h>

#include <library/cpp/streams/zstd/zstd.h>

#include <util/folder/iterator.h>
#include <util/folder/path.h>
#include <util/generic/function_ref.h>
#include <util/generic/hash.h>
#include <util/generic/hash_set.h>
#include <util/generic/map.h>
#include <util/generic/vector.h>
#include <util/stream/file.h>
#include <util/stream/format.h>
#include <util/stream/length.h>
#include <util/system/filemap.h>
#include <util/system/info.h>


class TUnwindStats {
public:
    void AddRule(NPerforator::NBinaryProcessing::NUnwind::UnwindRule rule) {
        switch (rule.GetKindCase()) {
        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::KIND_NOT_SET:
            return;

        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kRegisterOffset:
            rule.mutable_register_offset()->clear_offset();
            break;

        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kRegisterDerefOffset:
            rule.mutable_register_deref_offset()->clear_offset();
            break;

        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kCfaPlusOffset:
            // rule.mutable_cfa_plus_offset()->clear_offset();
            break;

        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kUnsupported:
        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kCfaMinus8:
        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kPltSection:
        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kConstant:
            break;

        default:
            Cerr << "Unsupported unwind rule " << static_cast<int>(rule.Kind_case()) << Endl;
            break;
        }

        Counts_[rule.ShortUtf8DebugString()]++;
        Sum_++;
    }

    void Dump(IOutputStream& out) const {
        TVector<std::pair<TString, ui64>> ordered(Counts_.begin(), Counts_.end());
        SortBy(ordered, [](auto&& kv) {
            return kv.second;
        });

        for (auto&& [k, v] : ordered) {
            out << v << '\t' << Prec(100.0 * v / Sum_, PREC_POINT_DIGITS, 3) << "%\t" << k << Endl;
        }
    }

private:
    THashMap<TString, ui64> Counts_;
    ui64 Sum_ = 0;
};

class TRowRef {
public:
    explicit TRowRef(const NPerforator::NBinaryProcessing::NUnwind::UnwindTable* table, int index)
        : Table_{table}
        , Index_{index}
    {
    }

    ui64 start_pc() const {
        return Table_->start_pc(Index_);
    }

    ui64 pc_range() const {
        return Table_->pc_range(Index_);
    }

    const NPerforator::NBinaryProcessing::NUnwind::UnwindRule& rbp() const {
        return Table_->dict(Table_->rbp(Index_));
    }

    const NPerforator::NBinaryProcessing::NUnwind::UnwindRule& cfa() const {
        return Table_->dict(Table_->cfa(Index_));
    }

    const NPerforator::NBinaryProcessing::NUnwind::UnwindRule& ra() const {
        return Table_->dict(Table_->ra(Index_));
    }

private:
    const NPerforator::NBinaryProcessing::NUnwind::UnwindTable* Table_ = nullptr;
    int Index_ = 0;
};

void ForEachRow(const NPerforator::NBinaryProcessing::NUnwind::UnwindTable& table, TFunctionRef<void(const TRowRef& row)> cb) {
    for (int i = 0; i < table.start_pc_size(); ++i) {
        cb(TRowRef{&table, i});
    }
}

void ForEachThreadLocal(
    const NPerforator::NBinaryProcessing::NTls::TLSConfig& config,
    TFunctionRef<void(const NPerforator::NBinaryProcessing::NTls::TLSVariable&)> cb
) {
    for (auto&& var : config.variables()) {
        cb(var);
    }
}

class TUnwindStatsCollector {
public:
    void AddTable(const NPerforator::NBinaryProcessing::NUnwind::UnwindTable& table) {
        ForEachRow(table, [this](TRowRef row) {
            CFA_.AddRule(row.cfa());
            RBP_.AddRule(row.rbp());
            RA_.AddRule(row.ra());
        });
    }

    void Dump(IOutputStream& out) const {
        out << "CFA:\n"; CFA_.Dump(out);
        out << "RBP:\n"; RBP_.Dump(out);
        out << "RA:\n"; RA_.Dump(out);
    }

private:
    TUnwindStats CFA_;
    TUnwindStats RBP_;
    TUnwindStats RA_;
};

void CountCFAStats(const NPerforator::NBinaryProcessing::NUnwind::UnwindTable& table, auto&& selector) {
    THashMap<TString, ui32> counts;
    ui32 sum = 0;

    auto visit = [&](NPerforator::NBinaryProcessing::NUnwind::UnwindRule rule) {
        switch (rule.GetKindCase()) {
        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::KIND_NOT_SET:
            return;

        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kRegisterOffset:
            rule.mutable_register_offset()->clear_offset();
            break;

        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kRegisterDerefOffset:
            rule.mutable_register_deref_offset()->clear_offset();
            break;

        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kCfaPlusOffset:
            rule.mutable_cfa_plus_offset()->clear_offset();
            break;

        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kUnsupported:
        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kCfaMinus8:
        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kPltSection:
        case NPerforator::NBinaryProcessing::NUnwind::UnwindRule::kConstant:
            break;

        default:
            Cerr << "Unsupported unwind rule " << static_cast<int>(rule.Kind_case()) << Endl;
            break;
        }

        counts[rule.ShortUtf8DebugString()]++;
        sum++;
    };

    ForEachRow(table, [&](const TRowRef& row) {
        visit(selector(row));
    });

    TVector<std::pair<TString, ui32>> ordered(counts.begin(), counts.end());
    SortBy(ordered, [](auto&& kv) {
        return kv.second;
    });

    for (auto&& [k, v] : ordered) {
        Cout << v << '\t' << Prec(100.0 * v / sum, PREC_NDIGITS, 2) << "%\t" << k << Endl;
    }
}

bool LooksLikeElf(const TFsPath& path) {
    constexpr TStringBuf magic = "\x7f\x45\x4c\x46";
    char buf[magic.size()] = {0};

    TFile file{path, RdOnly};
    size_t len = file.Read(buf, sizeof(buf));

    TStringBuf hdr{buf, len};
    return hdr == magic;
}

void WalkBinaries(const char* root) {
    TUnwindStatsCollector stats;
    THashSet<TString> visited;

    for (auto&& ent : TDirIterator{root, FTS_LOGICAL|FTS_XDEV}) {
        if (ent.fts_type != FTS_F) {
            continue;
        }

        if ((ent.fts_statp->st_mode & S_IXOTH) != S_IXOTH) {
            continue;
        }

        auto path = TFsPath{TStringBuf{ent.fts_path, ent.fts_pathlen}}.RealLocation();
        if (auto [_, ok] = visited.emplace(path.GetPath()); !ok) {
            continue;
        }

        if (!LooksLikeElf(path)) {
            Cerr << "Found strange executable file that does not look like elf: " << path.GetPath() << Endl;
            continue;
        }

        Cerr << "Found executable file " << path.GetPath() << Endl;
        auto analysis = NPerforator::NBinaryProcessing::AnalyzeBinary(path.GetPath().c_str());
        stats.AddTable(analysis.GetUnwindTable());
    }

    stats.Dump(Cout);
}

int main(int argc, const char* argv[]) {
    if (argc < 2) {
        return 1;
    }

    if (argv[1] == "print"sv) {
        NPerforator::NBinaryProcessing::BinaryAnalysis analysis;
        TZstdDecompress in{&Cin};
        Y_ENSURE(analysis.ParseFromArcadiaStream(&in));
        Cout << analysis.Utf8DebugString() << Endl;
        return 0;
    }

    if (argv[1] == "pretty-print"sv) {
        NPerforator::NBinaryProcessing::BinaryAnalysis analysis = NPerforator::NBinaryProcessing::DeserializeBinaryAnalysis(&Cin);
        ForEachThreadLocal(analysis.GetTLSConfig(), [&](const NPerforator::NBinaryProcessing::NTls::TLSVariable& var) {
            Cout << var.GetName() << ", offset: " << var.offset() << Endl;
        });
        ForEachRow(analysis.GetUnwindTable(), [&](TRowRef row) {
            Cout << Hex(row.start_pc()) << '\t' << Hex(row.pc_range()) << '\t' << row.cfa().ShortUtf8DebugString() << '\t' << row.rbp().ShortUtf8DebugString() << Endl;
        });
        return 0;
    }

    if (argv[1] == "walk"sv) {
        WalkBinaries(argv[2]);
        return 0;
    }

    if (argv[1] == "count"sv) {
        NPerforator::NBinaryProcessing::BinaryAnalysis analysis;
        TZstdDecompress in{&Cin};
        Y_ENSURE(analysis.ParseFromArcadiaStream(&in));
        auto unwtable = analysis.GetUnwindTable();

        Cout << "CFA:" << Endl;
        CountCFAStats(unwtable, [](auto&& row) {
            return row.cfa();
        });

        Cout << "RBP:" << Endl;
        CountCFAStats(unwtable, [](auto&& row) {
            return row.rbp();
        });

        Cout << "RA:" << Endl;
        CountCFAStats(unwtable, [](auto&& row) {
            return row.ra();
        });

        return 0;
    }

    if (argv[1] == "width"sv) {
        NPerforator::NBinaryProcessing::BinaryAnalysis analysis = NPerforator::NBinaryProcessing::DeserializeBinaryAnalysis(&Cin);

        TMap<ui64, ui32> counts;
        ui64 sum = 0;
        ui64 count = 0;
        ForEachRow(analysis.GetUnwindTable(), [&](TRowRef row) {
            counts[row.pc_range()]++;
            count++;
            sum += row.pc_range();
        });

        for (auto [width, count] : counts) {
            Cout << width << '\t' << count << '\t' << Prec(100.0 * count / analysis.GetUnwindTable().start_pc_size(), PREC_POINT_DIGITS, 2) << Endl;
        }

        Cout << "Total " << count << " rows, " << sum << " text bytes, avg " << 1.0 * sum / count << " bytes per row" << Endl;

        return 0;
    }

    while (true) {
        auto analysis = NPerforator::NBinaryProcessing::AnalyzeBinary(argv[1]);
        TString filename{"ehframe.pb.zstd"};
        if (argc > 2) {
            filename = argv[2];
        }
        TFileOutput out{filename};
        TCountingOutput counting{&out};
        NPerforator::NBinaryProcessing::SerializeBinaryAnalysis(std::move(analysis), &counting);
        counting.Finish();
        out.Finish();

        Cerr
            << "Generated unwind table with " << analysis.GetUnwindTable().start_pc_size() << " rows "
            << "(" << HumanReadableSize(counting.Counter(), SF_BYTES)
            << ", " << HumanReadableSize(static_cast<double>(counting.Counter()) / analysis.GetUnwindTable().start_pc_size(), SF_BYTES)
            << " per row)\n";

#ifdef PRINT_INSNS
        const uint8_t DWARF_CFI_PRIMARY_OPCODE_MASK = 0xc0;
        const uint8_t DWARF_CFI_PRIMARY_OPERAND_MASK = 0x3f;

        ui64 total = 0;
        for (ui64 i = 0; i < 256; ++i) {
            if (counts[i]) {
                Cout << names[i] << ": ";
                if (ui64 primary = i & DWARF_CFI_PRIMARY_OPCODE_MASK) {
                    Cout << "Primary " << (primary >> 6) << ", operand: " << (i & DWARF_CFI_PRIMARY_OPERAND_MASK) << ": " << counts[i] << Endl;
                } else {
                    Cout << Hex(i, HF_ADDX) << ": " << counts[i] << Endl;
                }
                total += counts[i];
            }
        }
        Cout << "Total " << total << " insns" << Endl;
#endif

        return 0;
    }
}
