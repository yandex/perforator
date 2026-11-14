#include "runtime.hpp"

#include <util/generic/scope.h>
#include <util/generic/yexception.h>

#include <cstdlib>
#include <utility>

extern "C" int PerforatorGoResolveRuntimeFrames(
    uint64_t,
    TPerforatorRuntimeRequestPtr,
    TPerforatorRuntimeResult*,
    char**
) __attribute__((weak));

namespace {

void ValidateArenaRange(
    const TPerforatorRuntimeResult& data,
    size_t offset,
    size_t count,
    size_t size,
    size_t alignment
)
{
    Y_ENSURE(offset <= data.ArenaBytes && count <= (data.ArenaBytes - offset) / size,
             "runtime result range points outside its arena");
    Y_ENSURE(count == 0 || offset % alignment == 0, "misaligned runtime result range");
}

void ValidateArenaString(
    const TPerforatorRuntimeResult& data,
    TPerforatorRuntimeString value
)
{
    ValidateArenaRange(data, value.Offset, value.Size, 1, 1);
}

} // anonymous namespace

namespace NPerforator::NAgent::NNativePipeline {

TRuntimeFrameBatch::TRuntimeFrameBatch(TPerforatorRuntimeResult data) noexcept
    : Data{data}
{
}

TRuntimeFrameBatch::~TRuntimeFrameBatch() {
    std::free(Data.Arena);
}

TRuntimeFrameBatch::TRuntimeFrameBatch(TRuntimeFrameBatch&& other) noexcept
    : Data{std::exchange(other.Data, {})}
{
}

TRuntimeFrameBatch& TRuntimeFrameBatch::operator=(TRuntimeFrameBatch&& other) noexcept {
    if (this != &other) {
        std::free(Data.Arena);
        Data = std::exchange(other.Data, {});
    }
    return *this;
}

TGoRuntimeResolver::TGoRuntimeResolver(ui64 resolverId)
    : ResolverId_{resolverId}
{
}

TRuntimeFrameBatch TGoRuntimeResolver::ResolveRuntimeFrames(
    const TPerforatorRuntimeRequest& request
)
{
    if (request.NativeStackSize == 0 && request.LanguagesSize == 0) {
        return {};
    }
    Y_ENSURE(PerforatorGoResolveRuntimeFrames, "Go runtime resolver callback is not linked");

    TRuntimeFrameBatch batch;
    char* errorMessage = nullptr;
    Y_DEFER {
        std::free(errorMessage);
    };
    const int status = PerforatorGoResolveRuntimeFrames(ResolverId_, &request, &batch.Data, &errorMessage);
    Y_ENSURE(status == PerforatorResolveOk,
             "Go runtime resolver failed: " << (errorMessage ? errorMessage : "unspecified failure"));
    const auto& data = batch.Data;
    Y_ENSURE(data.Arena && data.NativePrefixSize == request.NativeStackSize &&
                 data.FrameCount >= data.NativePrefixSize && data.LanguageCount == request.LanguagesSize,
             "runtime resolver returned invalid result sizes");
    ValidateArenaRange(data, data.FramesOffset, data.FrameCount, sizeof(TPerforatorResolvedFrame), alignof(TPerforatorResolvedFrame));
    ValidateArenaRange(data, data.LanguagesOffset, data.LanguageCount, sizeof(TPerforatorResolvedLanguageSection), alignof(TPerforatorResolvedLanguageSection));
    for (size_t i = 0; i < data.LanguageCount; ++i) {
        const auto& section = batch.Language(i);
        Y_ENSURE(section.Language == request.Languages[i].Language && section.FrameBegin <= data.FrameCount &&
                     section.FrameCount <= data.FrameCount - section.FrameBegin,
                 "runtime resolver returned an invalid language range");
    }
    const auto* frames = reinterpret_cast<const TPerforatorResolvedFrame*>(data.Arena + data.FramesOffset);
    for (size_t i = 0; i < data.FrameCount; ++i) {
        const auto& frame = frames[i];
        ValidateArenaString(data, frame.Mapping);
        ValidateArenaRange(data, frame.SymbolsOffset, frame.SymbolCount, sizeof(TPerforatorResolvedSymbol), alignof(TPerforatorResolvedSymbol));
        const auto* symbols = reinterpret_cast<const TPerforatorResolvedSymbol*>(data.Arena + frame.SymbolsOffset);
        for (size_t j = 0; j < frame.SymbolCount; ++j) {
            ValidateArenaString(data, symbols[j].Name);
            ValidateArenaString(data, symbols[j].SystemName);
            ValidateArenaString(data, symbols[j].File);
        }
    }
    return batch;
}

} // namespace NPerforator::NAgent::NNativePipeline
