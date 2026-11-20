#pragma once

#include "runtime.h"

#include <util/generic/array_ref.h>
#include <util/generic/strbuf.h>
#include <util/system/types.h>

namespace NPerforator::NAgent::NNativePipeline {

inline TStringBuf RuntimeStringView(const char* arena, TPerforatorRuntimeString value) noexcept {
    return value.Size ? TStringBuf{arena + value.Offset, value.Size} : TStringBuf{};
}

class TResolvedSymbolView {
public:
    TStringBuf Name() const noexcept {
        return RuntimeStringView(Arena_, Symbol_->Name);
    }

    TStringBuf FileName() const noexcept {
        return RuntimeStringView(Arena_, Symbol_->File);
    }

    ui32 StartLine() const noexcept {
        return Symbol_->StartLine;
    }

    ui32 Line() const noexcept {
        return Symbol_->Line;
    }

private:
    friend class TResolvedFrameView;

    TResolvedSymbolView(
        const char* arena,
        const TPerforatorResolvedSymbol* symbol
    ) noexcept
        : Arena_{arena}
        , Symbol_{symbol}
    {
    }

    const char* Arena_;
    const TPerforatorResolvedSymbol* Symbol_;
};

// Views borrow the immutable batch arena and require retaining its owner.
class TResolvedFrameView {
public:
    bool IsNativeAddress() const noexcept {
        return Frame_->Representation == PerforatorFrameNativeAddress;
    }

    EPerforatorLanguage Language() const noexcept {
        return static_cast<EPerforatorLanguage>(Frame_->Language);
    }

    ui64 Address() const noexcept {
        return Frame_->Address;
    }

    TStringBuf MappingName() const noexcept {
        return RuntimeStringView(Arena_, Frame_->Mapping);
    }

    size_t SymbolCount() const noexcept {
        return Frame_->SymbolCount;
    }

    bool HasData() const noexcept {
        return Address() != 0 || SymbolCount() != 0 || !MappingName().empty();
    }

    TResolvedSymbolView Symbol(size_t index) const noexcept {
        return TResolvedSymbolView{Arena_, reinterpret_cast<const TPerforatorResolvedSymbol*>(Arena_ + Frame_->SymbolsOffset) + index};
    }

private:
    friend class TResolvedFramesView;
    friend struct TRuntimeFrameBatch;

    TResolvedFrameView(
        const char* arena,
        const TPerforatorResolvedFrame* frame
    ) noexcept
        : Arena_{arena}
        , Frame_{frame}
    {
    }

    const char* Arena_;
    const TPerforatorResolvedFrame* Frame_;
};

class TResolvedFramesView {
public:
    size_t size() const noexcept {
        return Frames_.size();
    }

    TResolvedFrameView operator[](size_t index) const noexcept {
        return TResolvedFrameView{Arena_, &Frames_[index]};
    }

private:
    friend struct TRuntimeFrameBatch;

    TResolvedFramesView(
        const char* arena,
        TConstArrayRef<TPerforatorResolvedFrame> frames
    ) noexcept
        : Arena_{arena}
        , Frames_{frames}
    {
    }

    const char* Arena_;
    TConstArrayRef<TPerforatorResolvedFrame> Frames_;
};

// Owns the malloc-allocated arena returned by Go. Moving transfers ownership
// without invalidating views; copying is disabled.
struct TRuntimeFrameBatch {
    TRuntimeFrameBatch() = default;
    explicit TRuntimeFrameBatch(TPerforatorRuntimeResult data) noexcept;
    ~TRuntimeFrameBatch();
    TRuntimeFrameBatch(TRuntimeFrameBatch&& other) noexcept;
    TRuntimeFrameBatch& operator=(TRuntimeFrameBatch&& other) noexcept;

    TPerforatorRuntimeResult Data = {};

    size_t FrameCount() const noexcept {
        return Data.FrameCount;
    }

    TResolvedFramesView GetUserFrames() const noexcept {
        return GetFrames(0, Data.UserFrameCount);
    }

    TResolvedFramesView GetUnboundLanguageFrames() const noexcept {
        return GetFrames(Data.UserFrameCount, Data.FrameCount - Data.UserFrameCount);
    }

    TResolvedFrameView Frame(size_t index) const noexcept {
        return TResolvedFrameView{Data.Arena, reinterpret_cast<const TPerforatorResolvedFrame*>(Data.Arena + Data.FramesOffset) + index};
    }

private:
    TResolvedFramesView GetFrames(size_t begin, size_t count) const noexcept {
        const auto* frames = count ? reinterpret_cast<const TPerforatorResolvedFrame*>(Data.Arena + Data.FramesOffset) + begin : nullptr;
        return TResolvedFramesView{Data.Arena, {frames, count}};
    }
};

// Requests are borrowed for the call. Returned results survive subsequent calls
// and resolver destruction; views require retaining the batch arena.
class IRuntimeResolver {
public:
    virtual ~IRuntimeResolver() = default;

    virtual TRuntimeFrameBatch ResolveRuntimeFrames(
        const TPerforatorRuntimeRequest& request
    ) = 0;
};

// Calls require external serialization. The Go callback handle must remain live
// for every resolution call.
class TGoRuntimeResolver final: public IRuntimeResolver {
public:
    explicit TGoRuntimeResolver(ui64 handle);

    TRuntimeFrameBatch ResolveRuntimeFrames(
        const TPerforatorRuntimeRequest& request
    ) override;

private:
    ui64 Handle_;
};

} // namespace NPerforator::NAgent::NNativePipeline
