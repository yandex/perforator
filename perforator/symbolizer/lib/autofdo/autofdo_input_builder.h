#pragma once

#include <optional>
#include <string>
#include <vector>

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>

#include <util/digest/multi.h>
#include <util/generic/array_ref.h>

namespace NPerforator::NProto::NPProf {
class Profile;
}

namespace NPerforator::NAutofdo {

ui64 GetBinaryInstructionsBytesSize(TStringBuf binaryPath);

// A bunch of maps create_llvm_prof expects as a possible input.
struct TAutofdoInputData final {
    // Technically not a "branch", but rather control-flow jump (i.e. callq/ret as well).
    struct TTakenBranch final {
        ui64 From;
        ui64 To;

        ui64 MappingOffset;

        bool operator==(const TTakenBranch& other) const noexcept {
            return From == other.From && To == other.To;
        }
    };

    struct TTakenBranchHash final {
        std::size_t operator()(const TTakenBranch& value) const noexcept {
            return MultiHash(value.From, value.To);
        }
    };

    // A range of instructions without taken "branches" in it.
    struct TRange final {
        ui64 From;
        ui64 To;

        ui64 MappingOffset;

        bool operator==(const TRange& other) const noexcept {
            return From == other.From && To == other.To;
        }
    };

    struct TRangeHash final {
        std::size_t operator()(const TRange& value) const noexcept {
            return MultiHash(value.From, value.To);
        }
    };

    struct TMetadata final {
        ui64 TotalProfiles{0};

        ui64 TotalBranches{0};
        ui64 TotalSamples{0};
        ui64 BogusLbrEntries{0};

        TMetadata& operator+=(const TMetadata& other);
    };

    absl::flat_hash_map<TTakenBranch, ui64, TTakenBranchHash> BranchCountMap;
    absl::flat_hash_map<TRange, ui64, TRangeHash> RangeCountMap;
    absl::flat_hash_map<ui64, ui64> AddressCountMap;

    TMetadata Meta{};
};

// Given the structured input data, serialize it into "text" format
// consumed by create_llvm_prof (--profile="text" option).
std::string SerializeAutofdoInput(const TAutofdoInputData& data);

// Given the structured input data, serialize it into pre-aggregated format
// consumed by llvm-bolt (-pa option).
std::string SerializeAutofdoInputInBoltPreaggregatedFormat(const TAutofdoInputData& data);

// Class that aggregates lbr-profiles into structured create_llvm_prof's input.
// One should call `AddProfile`/`AddData` to feed profiles into the builder,
// and then `Finalize` it to extract the aggregated input.
class TInputBuilder final {
public:
    // Constructs the builder for this buildId (readelf -n <binary>),
    // samples not belonging to the buildId will be filtered out.
    explicit TInputBuilder(const std::string& buildId);

    // Parses the raw profile bytes and adds the profile into builder.
    void AddProfile(TArrayRef<const char> profileBytes);
    // Adds the profile into builder.
    void AddProfile(const NPerforator::NProto::NPProf::Profile& profile);

    // Add the pre-aggregated data into builder.
    void AddData(TAutofdoInputData&& otherData);

    // Extracts the aggregared data from the builder.
    // The builder shouldn't be used after this call.
    TAutofdoInputData&& Finalize() &&;

private:
    std::string BuildId_;
    TAutofdoInputData Data_;
};

// A wrapper over `TInputBuilder`, which allows one to consume profiles in parallel.
// Given N threads, one should create the builder with N `buildersCount`, and for thread I
// operate on a builder acquired by `GetBuilder(I)` call.
class TBatchInputBuilder final {
public:
    // Constructs the builder with `buildersCount` inner builders, each filtering profiles
    // by `buildId`.
    TBatchInputBuilder(ui64 buildersCount, std::string buildId);

    // Acquire a reference to the builder at index `builderIndex`.
    TInputBuilder& GetBuilder(ui64 builderIndex);

    // Extracts and merges aggregared data from the inner builders.
    // The builder shouldn't be used after this call.
    TAutofdoInputData Finalize() &&;

private:
    std::vector<TInputBuilder> Builders_;
};

// A simple frequency-counter for BuildIDs in `Feed`-ed profiles.
// One should `FeedProfile`-s into the class, and then access the resulting frequency map
// via GetFrequencyMap().
class TBuildIdGuesser final {
public:
    // Parses the bytes provided into Profile and aggregates BuildID frequencies
    // from it.
    void FeedProfile(TArrayRef<const char> profileBytes);

    // Aggregates BuildID frequencies from the profile.
    void FeedProfile(const NPerforator::NProto::NPProf::Profile& profile);

    // Returns a reference to the frequency map of Feed-ed buildIDs.
    const absl::flat_hash_map<std::string, ui64>& GetFrequencyMap() const;

private:
    absl::flat_hash_map<std::string, ui64> BuildIdCount_;
};

// A wrapper over `TBuildIdGuesser`, which allows one to consume profiles in parallel.
// Given N threads, one should create the guesser with N `guessersCount`, and for thread I
// operate on a guesser acquired by `GetGuesser(I)` call.
class TBatchBuildIdGuesser final {
public:
    // Constructs the guesser with `guessersCount` inner guessers.
    explicit TBatchBuildIdGuesser(ui64 guessersCount);

    // Acquire a reference to the guesser at index `guesserIndex`.
    TBuildIdGuesser& GetGuesser(ui64 guesserIndex);

    // Returns the most frequently encountered BuildID, or std::nullopt if
    // there are no BuildIDs encountered.
    std::optional<std::string> GuessBuildID() const;

private:
    std::vector<TBuildIdGuesser> Guessers_;
};

}

