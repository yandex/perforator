#pragma once

#include <library/cpp/containers/absl/flat_hash_map.h>
#include <library/cpp/int128/int128.h>

#include <perforator/symbolizer/lib/gsym/gsym_symbolizer.h>

#include <perforator/proto/profile/profile.pb.h>

#include <util/generic/array_ref.h>
#include <util/digest/multi.h>

#include <vector>

namespace NPerforator::NClusterTop {

struct TFunctionLocation final {
    TString Name;
    TString BuildId;
    TString SourceFile;
    ui32 SourceLine = 0;
    ui32 SourceColumn = 0;
    bool IsKernel = false;

    bool operator==(const TFunctionLocation&) const = default;
};

struct TFunctionLocationHash final {
    size_t operator()(const TFunctionLocation& location) const {
        return MultiHash(location.Name, location.BuildId, location.SourceFile,
                         location.SourceLine, location.SourceColumn, location.IsKernel);
    }
};

class TCachingGSYMSymbolizer final {
public:
    TCachingGSYMSymbolizer(std::string_view gsymPath);

    const std::vector<TFunctionLocation>& Symbolize(ui64 addr);

    void PruneCaches();

    std::size_t GetCacheSize() const {
        return SymbolizationCache_.size();
    }

private:
    NPerforator::NGsym::TSymbolizer Symbolizer_;

    absl::flat_hash_map<ui64, std::vector<TFunctionLocation>> SymbolizationCache_;
};

class TPerfTopAggregator final {
public:
    TPerfTopAggregator();

    void InitializeSymbolizer(TArrayRef<const char> buildId, TArrayRef<const char> gsymPath);

    void AddProfile(const NPerforator::NProto::NProfile::Profile& profile);

    void MergeAggregator(const TPerfTopAggregator& other);

    struct Function final {
        TString Name;
        TString BuildId;
        TString SourceFile;
        ui32 SourceLine;
        ui32 SourceColumn;
        ui128 SelfCycles;
        ui128 CumulativeCycles;
    };
    struct PerfTop final {
        std::vector<Function> Functions;
        ui128 TotalCycles;
    };
    PerfTop ExtractEntries();

private:
    void MaybePruneCaches();

    absl::flat_hash_map<TString, TCachingGSYMSymbolizer> Symbolizers_;

    struct TFunctionStats final {
        ui128 SelfCycles = 0;
        ui128 CumulativeCycles = 0;
    };
    absl::flat_hash_map<TFunctionLocation, TFunctionStats, TFunctionLocationHash> CyclesByFunction_;
    ui128 TotalCycles_{0};

    ui64 TotalProfiles_{0};
};

}
