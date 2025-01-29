#pragma once

#include <util/generic/hash.h>
#include <util/generic/string.h>


#include <llvm/DebugInfo/GSYM/GsymReader.h>
#include <llvm/DebugInfo/Symbolize/Symbolize.h>
#include <llvm/DebugInfo/DIContext.h>
#include <llvm/Object/ObjectFile.h>

#include <library/cpp/yt/compact_containers/compact_vector.h>

#include <perforator/proto/pprofprofile/profile.pb.h>
#include <perforator/symbolizer/lib/gsym/gsym_symbolizer.h>

namespace NPerforator::NSymbolize {

////////////////////////////////////////////////////////////////////////////////

std::string DemangleFunctionName(const std::string& name);
std::string CleanupFunctionName(std::string&& name);

////////////////////////////////////////////////////////////////////////////////

template<typename T>
using TSmallVector = NYT::TCompactVector<T, 4>;

class TCodeSymbolizer : TNonCopyable {
public:
    TCodeSymbolizer();

    TSmallVector<llvm::DILineInfo> Symbolize(TStringBuf moduleName, ui64 addr);

    TSmallVector<llvm::DILineInfo> SymbolizeGsym(TStringBuf moduleName, ui64 addr);

    void PruneCaches();

private:
    llvm::object::ObjectFile* GetObjectFile(TStringBuf moduleName);
    ui64 GetOffsetByModule(TStringBuf moduleName);

private:
    llvm::symbolize::LLVMSymbolizer Symbolizer_;
    THashMap<TString, ui64> OffsetByModule_;

    std::string LastSymbolizedModuleName_;

    THashMap<TString, NPerforator::NGsym::TSymbolizer> GSYMSymbolizers_;
};

////////////////////////////////////////////////////////////////////////////////

class TProfileSymbolizer {
public:
    //  Inplace symbolization for profile.proto.
    //  Symbolizes every address in callstack into function names if possible
    //  For each location's address provides lines information (multiple lines in case of inlined functions)
    void Symbolize(NPerforator::NProto::NPProf::Profile& profile);

private:
    TCodeSymbolizer CodeSymbolizer_;
};

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NSymbolize
