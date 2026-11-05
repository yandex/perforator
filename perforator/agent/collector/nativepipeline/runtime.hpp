#pragma once

#include "runtime.h"

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
    ui64 Address() const noexcept {
        return Frame_->Address;
    }

    TStringBuf MappingName() const noexcept {
        return RuntimeStringView(Arena_, Frame_->Mapping);
    }

    size_t SymbolCount() const noexcept {
        return Frame_->SymbolCount;
    }

    TResolvedSymbolView Symbol(size_t index) const noexcept {
        return TResolvedSymbolView{Arena_, reinterpret_cast<const TPerforatorResolvedSymbol*>(Arena_ + Frame_->SymbolsOffset) + index};
    }

private:
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

    size_t NativePrefixSize() const noexcept {
        return Data.NativePrefixSize;
    }

    size_t LanguageCount() const noexcept {
        return Data.LanguageCount;
    }

    const TPerforatorResolvedLanguageSection& Language(size_t index) const noexcept {
        return reinterpret_cast<const TPerforatorResolvedLanguageSection*>(Data.Arena + Data.LanguagesOffset)[index];
    }

    TResolvedFrameView Frame(size_t index) const noexcept {
        return TResolvedFrameView{Data.Arena, reinterpret_cast<const TPerforatorResolvedFrame*>(Data.Arena + Data.FramesOffset) + index};
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
    explicit TGoRuntimeResolver(ui64 resolverId);

    TRuntimeFrameBatch ResolveRuntimeFrames(
        const TPerforatorRuntimeRequest& request
    ) override;

private:
    ui64 ResolverId_;
};

} // namespace NPerforator::NAgent::NNativePipeline
