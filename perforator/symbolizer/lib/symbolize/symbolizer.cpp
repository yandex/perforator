#include "symbolizer.h"

#include <library/cpp/logger/global/global.h>

#include <perforator/lib/llvmex/llvm_elf.h>
#include <perforator/lib/llvmex/llvm_exception.h>

#include <util/generic/algorithm.h>
#include <util/generic/hash.h>
#include <util/generic/vector.h>
#include <util/generic/deque.h>
#include <util/stream/format.h>

#include <llvm/Object/ELF.h>
#include <llvm/Object/ELFObjectFile.h>

#include <contrib/libs/re2/re2/re2.h>

namespace NPerforator::NSymbolize {

namespace {

constexpr std::size_t kMaxCachedOffsets = 1'000'000;
constexpr std::size_t kMaxLLVMSymbolizerCacheSize = [] {
    if constexpr (sizeof(std::size_t) == 4) {
        return 512 * 1024 * 1024; // 512Mb
    } else {
        return 32ULL * 1024 * 1024 * 1024; // 32Gb
    }
}();

template <typename ELFT>
TMaybe<ui64> getFirstPhdrVirtualAddressImpl(llvm::object::ObjectFile* file) {
    using Elf_Phdr_Range = typename ELFT::PhdrRange;

    llvm::object::ELFObjectFile<ELFT>* elf = llvm::dyn_cast<llvm::object::ELFObjectFile<ELFT>>(file);
    if (!elf) {
        return Nothing();
    }

    llvm::Expected<Elf_Phdr_Range> range = elf->getELFFile().program_headers();
    if (!range) {
        return Nothing();
    }

    for (auto&& phdr : *range) {
        if (phdr.p_type != llvm::ELF::PT_LOAD) {
            continue;
        }

        return phdr.p_vaddr;
    }

    return Nothing();
}

TMaybe<ui64> getFirstPhdrVirtualAddress(llvm::object::ObjectFile* file) {
#define TRY_ELF_TYPE(ELFT) \
    if (auto res = getFirstPhdrVirtualAddressImpl<ELFT>(file)) { \
        return res; \
    }
    Y_LLVM_FOR_EACH_ELF_TYPE(TRY_ELF_TYPE)
#undef TRY_ELF_TYPE

    return Nothing();
}

ui64 CalcOffsetForModule(TStringBuf moduleName) {
    // We can't feed these objects directly into LLVMSymbolizer::symbolizeInlinedCode:
    // this is the object that owns mmap-ed region, and lots of LLVMSymbolizer internals
    // index into this region, which ties the lifetime of this object to the lifetime of
    // the symbolizer.
    // There's no interface to prune symbolizer caches by the object file, so we have to
    // keep it forever, and that OOMs sooner or later.
    auto binary = llvm::object::ObjectFile::createObjectFile(moduleName);
    if (!binary) {
        return 0;
    }

    return getFirstPhdrVirtualAddress(binary->getBinary()).GetOrElse(0);
}

}

std::string DemangleFunctionName(const std::string& name) {
    return llvm::symbolize::LLVMSymbolizer::DemangleName(name, nullptr);
}

std::string CleanupFunctionName(std::string&& name) {
    // https://itanium-cxx-abi.github.io/cxx-abi/abi.html#mangling-structure
    // If the name looks like a mangled name, remove vendor-specific suffix from it
    // (for example, llvm's LTO adds .llvm.<sone hash>, gcc adds .isra/.part etc)
    if (name.starts_with("_Z")) {
        const auto vendorSpecificSuffixStart = name.find('.');
        if (vendorSpecificSuffixStart != std::string::npos) {
            name = name.substr(0, vendorSpecificSuffixStart);
        }
    } else {
        // Otherwise, try to remove llvm's LTO suffix,
        // which could be added to C (hence not mangled) names just as well
        static const auto toErase = [] () {
            std::deque<re2::RE2> patterns;
            Y_ENSURE(patterns.emplace_back(R"(\.llvm\.[0-9a-f]+)").ok());
            return patterns;
        }();
        for (const auto& pattern : toErase) {
             Y_ENSURE(pattern.ok(), "Failed to compile regex");
             re2::RE2::Replace(&name, pattern, "");
        }
    }

    return std::move(name);
}

TCodeSymbolizer::TCodeSymbolizer()
    : Symbolizer_{llvm::symbolize::LLVMSymbolizer::Options{
        .Demangle = false,
        .MaxCacheSize = kMaxLLVMSymbolizerCacheSize,
    }}
{
}

TSmallVector<llvm::DILineInfo> TCodeSymbolizer::Symbolize(TStringBuf moduleName, ui64 addr) {
    const auto offset = GetOffsetByModule(moduleName);

    // For some reason LLVMSymbolizer doesn't accept std::string_view as moduleName,
    // so let's cache the string as a minor optimization.
    if (LastSymbolizedModuleName_ != moduleName) {
        LastSymbolizedModuleName_ = moduleName;
    }
    auto inliningInfo = Y_LLVM_RAISE(Symbolizer_.symbolizeInlinedCode(
        LastSymbolizedModuleName_,
        llvm::object::SectionedAddress{.Address = addr + offset}
    ));

    TSmallVector<llvm::DILineInfo> result;
    result.reserve(inliningInfo.getNumberOfFrames());

    for (size_t i = static_cast<size_t>(inliningInfo.getNumberOfFrames()) - 1; ~i; --i) {
        result.push_back(std::move(*inliningInfo.getMutableFrame(i)));
    }

    return result;
}

TSmallVector<llvm::DILineInfo> TCodeSymbolizer::SymbolizeGsym(TStringBuf moduleName, ui64 addr) {
    auto& symbolizer = [this, moduleName] () -> NPerforator::NGsym::TSymbolizer& {
        const auto it = GSYMSymbolizers_.find(moduleName);
        if (it != GSYMSymbolizers_.end()) {
            return it->second;
        }

        return GSYMSymbolizers_.emplace(moduleName, moduleName).first->second;
    }();

    return symbolizer.Symbolize(addr);
}

void TCodeSymbolizer::PruneCaches() {
    Symbolizer_.pruneCache();

    if (OffsetByModule_.size() > kMaxCachedOffsets) {
        OffsetByModule_.clear();
    }

    GSYMSymbolizers_.clear();
}

ui64 TCodeSymbolizer::GetOffsetByModule(TStringBuf moduleName) {
    const auto it = OffsetByModule_.find(moduleName);
    if (it != OffsetByModule_.end()) {
        return it->second;
    }

    const auto offset = CalcOffsetForModule(moduleName);
    OffsetByModule_[moduleName] = offset;
    return offset;
}

class TLocationSymbolizer {
public:
    explicit TLocationSymbolizer(NPerforator::NProto::NPProf::Profile& profile, TCodeSymbolizer& symbolizer);

    const NPerforator::NProto::NPProf::Mapping* GetMappingForLocation(const NPerforator::NProto::NPProf::Location* loc) const;
    NPerforator::NProto::NPProf::Location* GetLocation(size_t id);

    // Returns true if location was symbolized successfully
    bool SymbolizeLocation(NPerforator::NProto::NPProf::Location* loc, TLog& logger);

private:
    void AddLine(
        NPerforator::NProto::NPProf::Location* loc,
        llvm::DILineInfo&& lineInfo
    );

    ui64 AddFunction(
        TString&& functionName,
        TString&& fileName,
        ui32 line
    );

private:
    NPerforator::NProto::NPProf::Profile& Profile_;
    TCodeSymbolizer& Symbolizer_;

    THashMap<ui64, NPerforator::NProto::NPProf::Location*> Locations_;
    THashMap<ui64, const NPerforator::NProto::NPProf::Mapping*> Mappings_;
    TVector<const NPerforator::NProto::NPProf::Mapping*> SortedMappings_;
};

TLocationSymbolizer::TLocationSymbolizer(NPerforator::NProto::NPProf::Profile& profile, TCodeSymbolizer& symbolizer)
    : Profile_(profile)
    , Symbolizer_(symbolizer)
    , SortedMappings_(profile.mappingSize() - 1)
{
    for (size_t i = 0; i < profile.locationSize(); ++i) {
        Locations_[profile.location(i).id()] = profile.mutable_location(i);
    }

    for (size_t i = 0; i < profile.mappingSize(); ++i) {
        Mappings_[profile.mapping(i).id()] = &profile.mapping(i);
        if (i > 0) {
            SortedMappings_[i - 1] = &profile.mapping(i);
        }
    }

    SortBy(SortedMappings_, [](const NPerforator::NProto::NPProf::Mapping* mapping) {
        return mapping->memory_start();
    });
}

const NPerforator::NProto::NPProf::Mapping* TLocationSymbolizer::GetMappingForLocation(const NPerforator::NProto::NPProf::Location* loc) const {
    if (loc->mapping_id() != 0) {
        auto mappingPtrPtr = Mappings_.FindPtr(loc->mapping_id());
        return mappingPtrPtr != nullptr ? *mappingPtrPtr : nullptr;
    }

    // binary search the appropriate mapping for address
    auto it = UpperBoundBy(
        SortedMappings_.begin(),
        SortedMappings_.end(),
        loc->address(),
        [](const NPerforator::NProto::NPProf::Mapping* mapping) {
            return mapping->memory_start();
    });
    if (it == SortedMappings_.begin()) {
        return nullptr;
    }
    --it;

    return (*it)->memory_limit() >= loc->address() ? *it : nullptr;
}

NPerforator::NProto::NPProf::Location* TLocationSymbolizer::GetLocation(size_t id) {
    return Locations_[id];
}

bool TLocationSymbolizer::SymbolizeLocation(NPerforator::NProto::NPProf::Location* loc, TLog& logger) {
    const NPerforator::NProto::NPProf::Mapping* mapping = GetMappingForLocation(loc);

    if (mapping == nullptr) {
        logger << TLOG_ERR << "No mapping was found for address " << Hex(loc->address()) << Endl;
        return false;
    }

    TSmallVector<llvm::DILineInfo> lines;
    try {
        lines = Symbolizer_.Symbolize(
            Profile_.string_table(mapping->filename()),
            loc->address() - mapping->memory_start() + mapping->file_offset()
        );
    } catch (const TLLVMException& exc) {
        logger << TLOG_ERR << "Failed to symbolize address " << Hex(loc->address()) << " in mapping " << Profile_.string_table(mapping->filename()) << ": " << exc.AsStrBuf() << Endl;
        return false;
    }

    for (auto&& lineInfo : lines) {
        AddLine(loc, std::move(lineInfo));
    }

    return true;
}

ui64 TLocationSymbolizer::AddFunction(
    TString&& functionName,
    TString&& fileName,
    ui32 startLine
) {
    std::string demangledName = DemangleFunctionName(functionName);
    std::string cleanFunctionName = CleanupFunctionName(std::move(demangledName));

    Profile_.add_string_table(std::move(functionName));
    Profile_.add_string_table(std::move(cleanFunctionName));
    Profile_.add_string_table(std::move(fileName));

    NPerforator::NProto::NPProf::Function* func = Profile_.add_function();
    func->set_id(Profile_.function_size()); // do not allow zero id for function
    func->set_system_name(Profile_.string_table_size() - 3);
    func->set_name(Profile_.string_table_size() - 2);
    func->set_filename(Profile_.string_table_size() - 1);
    func->set_start_line(startLine);

    return func->id();
}

void TLocationSymbolizer::AddLine(NPerforator::NProto::NPProf::Location* loc, llvm::DILineInfo&& lineInfo) {
    ui64 funcId = AddFunction(
        std::move(lineInfo.FunctionName),
        std::move(lineInfo.FileName),
        lineInfo.StartLine
    );
    NPerforator::NProto::NPProf::Line* line = loc->add_line();
    line->set_function_id(funcId);
    line->set_line(lineInfo.Line);
}

void TProfileSymbolizer::Symbolize(NPerforator::NProto::NPProf::Profile& profile) {
    TLocationSymbolizer locationSymbolizer(profile, CodeSymbolizer_);

    for (size_t i = 0; i < profile.sampleSize(); ++i) {
        NPerforator::NProto::NPProf::Sample sample = profile.sample(i);

        for (size_t j = 0; j < sample.location_idSize(); ++j) {
            NPerforator::NProto::NPProf::Location* loc = locationSymbolizer.GetLocation(sample.location_id(j));

            Y_ENSURE_EX(
                loc != nullptr,
                yexception{} << "invalid profile.proto, location id " << sample.location_id(j) << " does not exist"
            );

            locationSymbolizer.SymbolizeLocation(loc, TLoggerOperator<TGlobalLog>::Log());
        }
    }
}

} // namespace NPerforator::NSymbolize
