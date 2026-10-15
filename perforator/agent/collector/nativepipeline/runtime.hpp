#pragma once

#include "runtime.h"

#include <util/generic/string.h>
#include <util/generic/vector.h>
#include <util/system/types.h>

#include <array>
#include <memory>
#include <utility>

namespace NPerforator::NAgent::NNativePipeline {

// Stable semantic identity of the yaprof location produced for a runtime
// frame. It is independent of the native LRU lifetime and of opaque fields
// (for example, different Python instruction pointers on the same line).
struct TInterpreterFrameId {
    ui64 First = 0;
    ui64 Second = 0;

    bool operator==(const TInterpreterFrameId&) const noexcept = default;

    template <typename H>
    friend H AbslHashValue(H hash, const TInterpreterFrameId& id) {
        return H::combine(std::move(hash), id.First, id.Second);
    }
};

struct TResolvedInterpreterFrame {
    TString Mapping;
    TString Name;
    TString FileName;
    ui64 Address = 0;
    TInterpreterFrameId ProfileId;
    i32 StartLine = 0;
    i32 Line = 0;
    bool Unsymbolized = false;
    bool HasStartLine = false;
    bool HasLine = false;
};

struct TResolvedNativeFrame {
    TString JitMapping;
    TVector<TString> JitSymbols;
    TString JvmName;
};

inline constexpr size_t LanguageCount = PERFORATOR_LANGUAGE_COUNT;

struct TInterpreterLanguageCacheOptions {
    ui64 MaxEntries = 8192;
    ui64 TtlNanos = 5ull * 60 * 1'000'000'000;
};

struct TInterpreterCacheOptions {
    std::array<TInterpreterLanguageCacheOptions, LanguageCount> Languages;
};

struct TInterpreterCacheStats {
    std::array<ui64, LanguageCount> Hits{};
    std::array<ui64, LanguageCount> Misses{};
    std::array<ui64, LanguageCount> Evictions{};
    std::array<ui64, LanguageCount> Size{};
    std::array<ui64, LanguageCount> Capacity{};
    // Non-empty interpreter miss batches, including failed resolutions.
    // Arena resize retries belong to the same batch.
    ui64 MissBatches = 0;
};

// Requests are borrowed for the call; returned frames own their strings and
// remain valid after subsequent calls and resolver destruction.
class IRuntimeResolver {
public:
    virtual ~IRuntimeResolver() = default;

    virtual TVector<std::shared_ptr<const TResolvedInterpreterFrame>> ResolveInterpreterFrames(
        const TVector<TPerforatorInterpreterFrameRequest>& requests
    ) = 0;

    virtual TVector<TResolvedNativeFrame> ResolveNativeFrames(
        const TVector<TPerforatorNativeFrameRequest>& requests
    ) = 0;
};

// Calls, including GetCacheStats, require external serialization. The Go
// callback handle must remain live for every resolution call.
class TGoRuntimeResolver final: public IRuntimeResolver {
public:
    TGoRuntimeResolver(ui64 resolverId, TInterpreterCacheOptions options);
    ~TGoRuntimeResolver();

    TVector<std::shared_ptr<const TResolvedInterpreterFrame>> ResolveInterpreterFrames(
        const TVector<TPerforatorInterpreterFrameRequest>& requests
    ) override;

    TVector<TResolvedNativeFrame> ResolveNativeFrames(
        const TVector<TPerforatorNativeFrameRequest>& requests
    ) override;

    TInterpreterCacheStats GetCacheStats() const noexcept;

private:
    class TImpl;

    ui64 ResolverId_;
    std::unique_ptr<TImpl> Impl_;
};

} // namespace NPerforator::NAgent::NNativePipeline
