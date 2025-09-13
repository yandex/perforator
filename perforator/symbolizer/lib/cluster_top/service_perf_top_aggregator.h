#pragma once

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>
#include <library/cpp/containers/absl_flat_hash/flat_hash_set.h>
#include <library/cpp/int128/int128.h>

#include <perforator/symbolizer/lib/gsym/gsym_symbolizer.h>

#include <perforator/proto/pprofprofile/lightweightprofile.pb.h>

#include <util/generic/array_ref.h>

#include <vector>
#include <string>

namespace NPerforator::NClusterTop {

class TCachingGSYMSymbolizer final {
public:
    TCachingGSYMSymbolizer(std::string_view gsymPath);

    const std::vector<std::string>& Symbolize(ui64 addr);

    void PruneCaches();

    std::size_t GetCacheSize() const {
        return SymbolizationCache_.size();
    }

private:
    NPerforator::NGsym::TSymbolizer Symbolizer_;

    absl::flat_hash_map<ui64, std::vector<std::string>> SymbolizationCache_;
};

class TServicePerfTopAggregator final {
public:
    TServicePerfTopAggregator();

    void InitializeSymbolizer(TArrayRef<const char> buildId, TArrayRef<const char> gsymPath);

    void AddProfile(TArrayRef<const char> service, TArrayRef<const char> profileBytes);

    void AddProfile(TArrayRef<const char> service, const NPerforator::NProto::NPProf::ProfileLight& profile);

    void MergeAggregator(const TServicePerfTopAggregator& other);

    struct Function final {
        TString Name;
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

    absl::flat_hash_set<TString, THash<TString>, TEqualTo<TString>> KernelFunctions_;

    struct TFunctionStats final {
        ui128 SelfCycles = 0;
        ui128 CumulativeCycles = 0;
    };
    absl::flat_hash_map<TString, TFunctionStats, THash<TString>, TEqualTo<TString>> CyclesByFunction_;
    ui128 TotalCycles_{0};

    ui64 TotalProfiles_{0};

    NPerforator::NProto::NPProf::ProfileLight ProfileForMemoryReuse_;
};

}
