#include "runtime.hpp"

#include <absl/container/inlined_vector.h>

#include <library/cpp/cache/cache.h>

#include <util/digest/multi.h>
#include <util/generic/hash.h>
#include <util/generic/scope.h>
#include <util/generic/yexception.h>

#include <algorithm>
#include <chrono>
#include <cstdlib>
#include <memory>

extern "C" int PerforatorGoResolveInterpreterFrames(
    uint64_t,
    TPerforatorInterpreterFrameRequest*,
    size_t,
    TPerforatorResolvedInterpreterFrame*,
    char*,
    size_t,
    size_t*,
    uint64_t*,
    char**
) __attribute__((weak));
extern "C" int PerforatorGoResolveNativeFrames(
    uint64_t,
    TPerforatorNativeFrameRequest*,
    size_t,
    TPerforatorResolvedNativeFrame*,
    TPerforatorResolvedNativeSymbol*,
    size_t,
    size_t*,
    char*,
    size_t,
    size_t*,
    uint64_t*,
    char**
) __attribute__((weak));

extern "C" void PerforatorGoReleaseRuntimeResolution(uint64_t) __attribute__((weak));

namespace NPerforator::NAgent::NNativePipeline {

struct TInterpreterCacheKey {
    ui64 Hash = 0;
    ui64 ProcessStartTime = 0;
    ui64 ProcessImageId = 0;
    ui32 Pid = 0;
    ui8 Language = 0;
    absl::InlinedVector<ui8, 32> Frame;

    bool operator==(const TInterpreterCacheKey&) const noexcept = default;
};

} // namespace NPerforator::NAgent::NNativePipeline

template <>
struct THash<NPerforator::NAgent::NNativePipeline::TInterpreterCacheKey> {
    size_t operator()(
        const NPerforator::NAgent::NNativePipeline::TInterpreterCacheKey& key
    ) const noexcept {
        return key.Hash;
    }
};

namespace NPerforator::NAgent::NNativePipeline {
namespace {

constexpr size_t MaxRuntimeArenaSize = 16 * 1024 * 1024;

TString CopyArenaString(
    TStringBuf arena,
    TPerforatorStringView value,
    TStringBuf field
)
{
    if (value.Size == 0) {
        return {};
    }
    Y_ENSURE(value.Data, field << " has a null data pointer");
    const uintptr_t arenaBegin = reinterpret_cast<uintptr_t>(arena.data());
    const uintptr_t data = reinterpret_cast<uintptr_t>(value.Data);
    Y_ENSURE(data >= arenaBegin, field << " points outside the runtime result arena");
    const size_t offset = data - arenaBegin;
    Y_ENSURE(offset <= arena.size() && value.Size <= arena.size() - offset,
             field << " points outside the runtime result arena");
    return TString{value.Data, value.Size};
}

TInterpreterCacheKey MakeCacheKey(const TPerforatorInterpreterFrameRequest& request) {
    Y_ENSURE(request.FrameData && request.FrameSize != 0, "invalid runtime frame request");

    TInterpreterCacheKey key{
        .ProcessStartTime = request.ProcessStartTime,
        .ProcessImageId = request.ProcessImageId,
        .Pid = request.Pid,
        .Language = request.Language,
    };
    const auto* frame = static_cast<const ui8*>(request.FrameData);
    key.Frame.assign(frame, frame + request.FrameSize);
    key.Hash = MultiHash(
        request.Language,
        request.Pid,
        request.ProcessStartTime,
        request.ProcessImageId,
        THash<TStringBuf>{}(TStringBuf{
            static_cast<const char*>(request.FrameData),
            request.FrameSize,
        }));
    return key;
}

TInterpreterFrameId MakeProfileId(const TResolvedInterpreterFrame& frame) {
    const ui8 flags =
        (frame.HasStartLine ? 1u : 0u) |
        (frame.HasLine ? 2u : 0u);
    const i32 startLine = frame.HasStartLine ? frame.StartLine : 0;
    const i32 line = frame.HasLine ? frame.Line : 0;
    return {
        .First = MultiHash(
            0x2c4e73ad91b568d1ull,
            frame.Address,
            frame.Mapping,
            frame.Name,
            frame.FileName,
            startLine,
            line,
            flags),
        .Second = MultiHash(
            0x91e10da5c79e7b1dull,
            flags,
            line,
            startLine,
            frame.FileName,
            frame.Name,
            frame.Mapping,
            frame.Address),
    };
}

} // anonymous namespace

class TGoRuntimeResolver::TImpl {
private:
    struct TCacheValue {
        std::shared_ptr<const TResolvedInterpreterFrame> Value;
        std::chrono::steady_clock::time_point ExpiresAt;
    };

    using TCache = TLRUCache<TInterpreterCacheKey, TCacheValue>;

    struct TLanguageCache {
        explicit TLanguageCache(const TInterpreterLanguageCacheOptions& options)
            : Ttl{static_cast<std::chrono::nanoseconds::rep>(options.TtlNanos)}
            , Cache{static_cast<size_t>(options.MaxEntries)}
        {
        }

        std::chrono::nanoseconds Ttl{};
        TCache Cache;
        ui64 Hits = 0;
        ui64 Misses = 0;
        ui64 Evictions = 0;
    };

public:
    explicit TImpl(const TInterpreterCacheOptions& options) {
        for (size_t i = 0; i < options.Languages.size(); ++i) {
            Languages_[i] = std::make_unique<TLanguageCache>(options.Languages[i]);
        }
    }

    std::shared_ptr<const TResolvedInterpreterFrame> Find(
        const TInterpreterCacheKey& key
    )
    {
        auto& language = Language(key.Language);
        auto entry = language.Cache.Find(key);
        if (entry == language.Cache.End()) {
            ++language.Misses;
            return {};
        }
        if (entry->ExpiresAt <= std::chrono::steady_clock::now()) {
            language.Cache.Erase(entry);
            ++language.Evictions;
            ++language.Misses;
            return {};
        }
        ++language.Hits;
        return entry->Value;
    }

    void Insert(
        TInterpreterCacheKey key,
        std::shared_ptr<const TResolvedInterpreterFrame> value
    )
    {
        auto& language = Language(key.Language);
        const size_t capacity = language.Cache.GetMaxSize();
        if (capacity == 0) {
            return;
        }
        const size_t sizeBefore = language.Cache.Size();
        const bool inserted = language.Cache.Insert(std::move(key), TCacheValue{
            .Value = std::move(value),
            .ExpiresAt = std::chrono::steady_clock::now() + language.Ttl,
        });
        if (inserted && sizeBefore >= capacity) {
            ++language.Evictions;
        }
    }

    TInterpreterCacheStats GetStats() const noexcept {
        TInterpreterCacheStats result;
        for (size_t i = 0; i < LanguageCount; ++i) {
            const auto& language = *Languages_[i];
            result.Hits[i] = language.Hits;
            result.Misses[i] = language.Misses;
            result.Evictions[i] = language.Evictions;
            result.Size[i] = language.Cache.Size();
            result.Capacity[i] = language.Cache.GetMaxSize();
        }
        result.MissBatches = MissBatches;
        return result;
    }

    TVector<TPerforatorResolvedInterpreterFrame> RawResults;
    TVector<char> Arena;
    ui64 MissBatches = 0;

private:
    TLanguageCache& Language(ui8 language) {
        Y_ENSURE(language < Languages_.size(), "unsupported runtime language id");
        return *Languages_[language];
    }

private:
    std::array<std::unique_ptr<TLanguageCache>, LanguageCount> Languages_;
};

TGoRuntimeResolver::TGoRuntimeResolver(
    ui64 resolverId,
    TInterpreterCacheOptions options
)
    : ResolverId_{resolverId}
    , Impl_{std::make_unique<TImpl>(options)}
{
}

TGoRuntimeResolver::~TGoRuntimeResolver() = default;

TInterpreterCacheStats TGoRuntimeResolver::GetCacheStats() const noexcept {
    return Impl_->GetStats();
}

TVector<std::shared_ptr<const TResolvedInterpreterFrame>> TGoRuntimeResolver::ResolveInterpreterFrames(
    const TVector<TPerforatorInterpreterFrameRequest>& requests
)
{
    if (requests.empty()) {
        return {};
    }
    Y_ENSURE(PerforatorGoResolveInterpreterFrames && PerforatorGoReleaseRuntimeResolution, "Go runtime resolver callback is not linked");

    TVector<std::shared_ptr<const TResolvedInterpreterFrame>> result(requests.size());
    TVector<TPerforatorInterpreterFrameRequest> misses;
    TVector<TInterpreterCacheKey> missKeys;
    TVector<size_t> firstDestinations;
    TVector<size_t> requestMissIndices(requests.size(), requests.size());
    THashMap<TInterpreterCacheKey, size_t> missIndices;
    misses.reserve(requests.size());
    missKeys.reserve(requests.size());
    firstDestinations.reserve(requests.size());
    for (size_t i = 0; i < requests.size(); ++i) {
        const auto& request = requests[i];
        auto key = MakeCacheKey(request);
        if (auto cached = Impl_->Find(key)) {
            result[i] = std::move(cached);
            continue;
        }

        const auto [it, inserted] = missIndices.emplace(key, misses.size());
        requestMissIndices[i] = it->second;
        if (inserted) {
            misses.push_back(request);
            missKeys.push_back(std::move(key));
            firstDestinations.push_back(i);
        }
    }
    if (misses.empty()) {
        return result;
    }
    ++Impl_->MissBatches;

    Impl_->RawResults.assign(misses.size(), {});
    const size_t initialArenaSize = std::min(MaxRuntimeArenaSize, misses.size() * 256);
    if (Impl_->Arena.size() < initialArenaSize) {
        Impl_->Arena.resize(initialArenaSize);
    }
    size_t requiredArenaBytes = 0;

    uint64_t resolution = 0;
    char* errorMessage = nullptr;
    Y_DEFER {
        std::free(errorMessage);
        if (resolution) {
            PerforatorGoReleaseRuntimeResolution(resolution);
        }
    };
    for (size_t attempt = 0; attempt != 2; ++attempt) {
        const int status = PerforatorGoResolveInterpreterFrames(
            ResolverId_,
            misses.data(),
            misses.size(),
            Impl_->RawResults.data(),
            Impl_->Arena.data(),
            Impl_->Arena.size(),
            &requiredArenaBytes, &resolution, &errorMessage);
        if (status == PerforatorResolveOk) {
            Y_ENSURE(requiredArenaBytes <= Impl_->Arena.size(), "runtime resolver returned an invalid arena size");
            const TStringBuf arena{Impl_->Arena.data(), requiredArenaBytes};
            for (size_t i = 0; i < Impl_->RawResults.size(); ++i) {
                const auto& raw = Impl_->RawResults[i];
                const bool unsymbolized = (raw.Flags & PerforatorResolvedFrameUnsymbolized) != 0;
                const bool cacheable = !unsymbolized &&
                                       (raw.Flags & PerforatorResolvedFrameDoNotCache) == 0;
                auto resolved = std::make_shared<TResolvedInterpreterFrame>(TResolvedInterpreterFrame{
                    .Mapping = CopyArenaString(arena, raw.Mapping, "runtime frame mapping"),
                    .Name = CopyArenaString(arena, raw.Name, "runtime frame name"),
                    .FileName = CopyArenaString(arena, raw.File, "runtime frame file"),
                    .Address = raw.Address,
                    .StartLine = raw.StartLine,
                    .Line = raw.Line,
                    .Unsymbolized = unsymbolized,
                    .HasStartLine = (raw.Flags & PerforatorResolvedFrameHasStartLine) != 0,
                    .HasLine = (raw.Flags & PerforatorResolvedFrameHasLine) != 0,
                });
                resolved->ProfileId = MakeProfileId(*resolved);
                if (cacheable) {
                    Impl_->Insert(std::move(missKeys[i]), resolved);
                }
                result[firstDestinations[i]] = std::move(resolved);
            }
            for (size_t i = 0; i < requestMissIndices.size(); ++i) {
                const size_t missIndex = requestMissIndices[i];
                if (missIndex != requests.size() && i != firstDestinations[missIndex]) {
                    result[i] = result[firstDestinations[missIndex]];
                }
            }
            return result;
        }
        if (status != PerforatorResolveInsufficientCapacity) {
            ythrow yexception() << "Go runtime resolver failed: " << (errorMessage ? errorMessage : "unspecified failure");
        }
        Y_ENSURE(requiredArenaBytes <= MaxRuntimeArenaSize, "runtime resolver requested an excessive arena");
        Y_ENSURE(requiredArenaBytes > Impl_->Arena.size(), "runtime resolver did not increase required arena size");
        Impl_->Arena.resize(requiredArenaBytes);
    }

    ythrow yexception() << "Go runtime resolver changed its arena requirement during retry";
}

TVector<TResolvedNativeFrame> TGoRuntimeResolver::ResolveNativeFrames(
    const TVector<TPerforatorNativeFrameRequest>& requests
)
{
    if (requests.empty()) {
        return {};
    }
    Y_ENSURE(PerforatorGoResolveNativeFrames && PerforatorGoReleaseRuntimeResolution, "Go native frame resolver callback is not linked");
    auto rawRequests = requests;
    TVector<TPerforatorResolvedNativeFrame> rawResults(requests.size());
    TVector<TPerforatorResolvedNativeSymbol> rawSymbols(requests.size() * 2);
    TVector<char> arena(std::min(MaxRuntimeArenaSize, requests.size() * 128));
    uint64_t resolution = 0;
    char* errorMessage = nullptr;
    Y_DEFER {
        std::free(errorMessage);
        if (resolution) {
            PerforatorGoReleaseRuntimeResolution(resolution);
        }
    };
    for (size_t attempt = 0; attempt != 2; ++attempt) {
        size_t symbolCount = 0;
        size_t requiredArenaBytes = 0;
        const int status = PerforatorGoResolveNativeFrames(
            ResolverId_, rawRequests.data(), rawRequests.size(), rawResults.data(),
            rawSymbols.data(), rawSymbols.size(), &symbolCount,
            arena.data(), arena.size(), &requiredArenaBytes, &resolution, &errorMessage);
        if (status == PerforatorResolveOk) {
            Y_ENSURE(symbolCount <= rawSymbols.size() && requiredArenaBytes <= arena.size(),
                     "native resolver returned invalid result sizes");
            const TStringBuf resultArena{arena.data(), requiredArenaBytes};
            TVector<TResolvedNativeFrame> result;
            result.reserve(rawResults.size());
            for (const auto& raw : rawResults) {
                Y_ENSURE(raw.SymbolBegin <= symbolCount &&
                             raw.SymbolCount <= symbolCount - raw.SymbolBegin,
                         "native resolver returned invalid symbol range");
                TResolvedNativeFrame frame{
                    .JitMapping = CopyArenaString(resultArena, raw.JitMapping, "JIT mapping"),
                    .JvmName = CopyArenaString(resultArena, raw.JvmName, "JVM name"),
                };
                frame.JitSymbols.reserve(raw.SymbolCount);
                for (size_t i = 0; i < raw.SymbolCount; ++i) {
                    const auto& symbol = rawSymbols[raw.SymbolBegin + i];
                    frame.JitSymbols.push_back(CopyArenaString(resultArena, symbol.Name, "JIT symbol"));
                }
                result.push_back(std::move(frame));
            }
            return result;
        }
        Y_ENSURE(status == PerforatorResolveInsufficientCapacity,
                 "Go native frame resolver failed: " << (errorMessage ? errorMessage : "unspecified failure"));
        Y_ENSURE(requiredArenaBytes <= MaxRuntimeArenaSize, "native frame resolver requested an excessive arena");
        Y_ENSURE(requiredArenaBytes > arena.size() || symbolCount > rawSymbols.size(),
                 "native frame resolver did not increase result capacity");
        if (requiredArenaBytes > arena.size()) {
            arena.resize(requiredArenaBytes);
        }
        if (symbolCount > rawSymbols.size()) {
            rawSymbols.resize(symbolCount);
        }
    }
    ythrow yexception() << "Go native frame resolver changed requirements during retry";
}

} // namespace NPerforator::NAgent::NNativePipeline
