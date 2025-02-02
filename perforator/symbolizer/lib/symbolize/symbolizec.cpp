#include <perforator/symbolizer/lib/symbolize/symbolizec.h>
#include <perforator/symbolizer/lib/symbolize/symbolizer.h>
#include <perforator/lib/llvmex/llvm_exception.h>

#include <library/cpp/yt/compact_containers/compact_vector.h>

#include <util/stream/format.h>
#include <util/stream/str.h>

namespace {

void FillLineInfo(TLineInfo& lineInfo, llvm::DILineInfo&& info) {
    lineInfo.FunctionName = strdup(info.FunctionName.c_str());

    const auto demangledName = NPerforator::NSymbolize::DemangleFunctionName(
        NPerforator::NSymbolize::CleanupFunctionName(std::move(info.FunctionName))
    );
    lineInfo.DemangledFunctionName = strdup(demangledName.c_str());

    lineInfo.FileName = strdup(info.FileName.c_str());

    lineInfo.Line = info.Line;
    lineInfo.StartLine = info.StartLine;
    lineInfo.Column = info.Column;
    lineInfo.Discriminator = info.Discriminator;
}

} // anonymous namespace

extern "C" {

void* MakeSymbolizer(char** error) {
    void* buf = ::operator new(sizeof(NPerforator::NSymbolize::TCodeSymbolizer));
    NPerforator::NSymbolize::TCodeSymbolizer* symbolizer = nullptr;
    try {
        symbolizer = new (buf) NPerforator::NSymbolize::TCodeSymbolizer();
    } catch (...) {
        ::operator delete(buf);
        TString excMessage = CurrentExceptionMessage();
        *error = strdup(excMessage.c_str());
        return nullptr;
    }

    return reinterpret_cast<void*>(symbolizer);
}

TLineInfo* Symbolize(
    void* symb,
    char* modulePath,
    ui64 modulePathLen,
    ui64 addr,
    ui64* linesCount,
    char** error,
    ui32 useGsym
) {
    auto symbolizer = reinterpret_cast<NPerforator::NSymbolize::TCodeSymbolizer*>(symb);

    const TStringBuf moduleName{modulePath, modulePathLen};
    NPerforator::NSymbolize::TSmallVector<llvm::DILineInfo> lines;
    try {
        lines = useGsym
            ? symbolizer->SymbolizeGsym(moduleName, addr)
            : symbolizer->Symbolize(moduleName, addr);
    } catch (const TLLVMException& exc) {
        TStringStream ss;
        ss << "Failed to symbolize address " << Hex(addr) << " in mapping " << modulePath << ": " << exc.AsStrBuf() << Endl;
        *error = strdup(ss.Str().c_str());
        return nullptr;
    }

    *linesCount = lines.size();
    TLineInfo* result = new TLineInfo[lines.size()];
    for (size_t i = 0; i < lines.size(); ++i) {
        FillLineInfo(result[i], std::move(lines[i]));
    }

    return result;
}

void PruneCaches(void* symb) {
    reinterpret_cast<NPerforator::NSymbolize::TCodeSymbolizer*>(symb)->PruneCaches();
}

void DestroySymbolizeResult(TLineInfo* result, ui64 linesCount) {
    for (size_t i = 0; i < linesCount; ++i) {
        free(result[i].FunctionName);
        free(result[i].DemangledFunctionName);
        free(result[i].FileName);
    }
    delete[] result;
}

void DestroySymbolizer(void* symb) {
    auto symbolizer = reinterpret_cast<NPerforator::NSymbolize::TCodeSymbolizer*>(symb);
    symbolizer->~TCodeSymbolizer();
    ::operator delete(symb);
}

} // extern "C"
