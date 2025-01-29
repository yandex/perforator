#include <perforator/symbolizer/lib/gsym/gsym.h>

#include <string_view>

#include <llvm/DebugInfo/DWARF/DWARFContext.h>
#include <llvm/DebugInfo/GSYM/GsymCreator.h>
#include <llvm/DebugInfo/GSYM/DwarfTransformer.h>
#include <llvm/DebugInfo/GSYM/ObjectFileTransformer.h>

#include <llvm/Object/Binary.h>
#include <llvm/Object/ObjectFile.h>
#include <llvm/Object/ELF.h>

#include <perforator/lib/llvmex/llvm_elf.h>

namespace {

constexpr const char* kNullInputError = "input is nullptr";
constexpr const char* kNullOutputError = "output is nullptr";
constexpr const char* kNotAnELFError = "provided binary is not an ELF object file";

template <class ELTF>
TMaybe<ui64> GetImageBaseAddress(const llvm::object::ELFObjectFile<ELTF>& elfFile) {
    auto phdrRangeOrErr = elfFile.getELFFile().program_headers();
    if (!phdrRangeOrErr) {
        return Nothing();
    }

    for (const auto& phdr : *phdrRangeOrErr) {
        if (phdr.p_type == llvm::ELF::PT_LOAD) {
            return MakeMaybe(static_cast<ui64>(phdr.p_vaddr));
        }
    }

    return Nothing();
}

// At the time of writing, ObjectFileTransformer ignores ST_Unknown symbols.
// This is problematic, since some handwritten assembly symbols lack ".type function"
// directive, and are thus absent in the resulting GSYM.
// What's even worse, handwritten assembly might lack ".size" directive, and GSYM lookups
// might match these zero-size symbols _instead_ of absent ST_Unknown symbols in some cases.
// (https://github.com/llvm/llvm-project/blob/release/18.x/llvm/lib/DebugInfo/GSYM/GsymReader.cpp#L283).
//
// We fixup such symbols by basically copying what ObjectFileTransformer::convert
// (https://github.com/llvm/llvm-project/blob/release/18.x/llvm/lib/DebugInfo/GSYM/ObjectFileTransformer.cpp#L70)
// does, but only looking for ST_Unknown symbols.
//
// TODO : remove when/if https://github.com/llvm/llvm-project/pull/119307 is resolved
llvm::Error FixupObjectFileTransformation(
    const llvm::object::ObjectFile& obj,
    llvm::gsym::GsymCreator& gsymCreator
) {
    if (!llvm::isa<llvm::object::ELFObjectFileBase>(obj)) {
        return llvm::createStringError(std::errc::invalid_argument,
            "Binary is not an ELF file");
    }

    for (const llvm::object::SymbolRef& sym : obj.symbols()) {
        auto symTypeOrErr = sym.getType();
        if (!symTypeOrErr) {
            llvm::consumeError(symTypeOrErr.takeError());
            continue;
        }
        const auto symType = *symTypeOrErr;

        auto addrOrErr = sym.getValue();
        if (!addrOrErr) {
            return addrOrErr.takeError();
        }
        const auto addr = *addrOrErr;

        if (symType != llvm::object::SymbolRef::Type::ST_Unknown || !gsymCreator.IsValidTextAddress(addr)) {
            continue;
        }

        auto nameOrErr = sym.getName();
        if (!nameOrErr) {
            llvm::consumeError(nameOrErr.takeError());
            continue;
        }
        const auto name = *nameOrErr;
        if (name.empty()) {
            continue;
        }

        const auto size = llvm::object::ELFSymbolRef{sym}.getSize();

        gsymCreator.addFunctionInfo(
            llvm::gsym::FunctionInfo{addr, size, gsymCreator.insertString(name,  /* Copy = */false)}
        );
    }

    return llvm::Error::success();
}

// This is a close adaptation of how llvm-gsymutil-18 does the convertion
// https://github.com/llvm/llvm-project/blob/release/18.x/llvm/tools/llvm-gsymutil/llvm-gsymutil.cpp#L303
llvm::Error ConvertDWARFToGSYM(llvm::object::ObjectFile& obj, std::string_view output, ui32 convertNumThreads) {
    // We might want to caprute the logs in the future, so this could be a pipe instead
    auto &os = llvm::outs();

    llvm::gsym::GsymCreator gsymCreator(true /* quiet */);

    if (auto imageBaseAddr = NPerforator::NLLVM::VisitELF(&obj,
    [](const auto& elfFile) { return GetImageBaseAddress(elfFile); })) {
        if (*imageBaseAddr) {
            gsymCreator.setBaseAddress(**imageBaseAddr);
        }
    }

    llvm::AddressRanges textRanges;
    for (const auto& section : obj.sections()) {
        if (!section.isText()) {
            continue;
        }

        const auto sectionSize = section.getSize();
        if (sectionSize == 0) {
            continue;
        }

        const auto sectionStartAddress = section.getAddress();
        textRanges.insert(llvm::AddressRange{sectionStartAddress, sectionStartAddress + sectionSize});
    }

    auto dwarfCtx = llvm::DWARFContext::create(
        obj,
        /* RelocAction = */ llvm::DWARFContext::ProcessDebugRelocations::Process,
        /* LoadedObjectInfo = */ nullptr,
        /* DWPName = */ "",
        /* RecoverableErrorHandler = */ llvm::WithColor::defaultErrorHandler,
        /* WarningHandler = */ llvm::WithColor::defaultWarningHandler,
        /* ThreadSafe = */ true
    );
    if (!dwarfCtx) {
        return llvm::createStringError(std::errc::invalid_argument, "unable to create DWARF context");
    }

    llvm::gsym::DwarfTransformer dwarfTransformer{*dwarfCtx, gsymCreator};
    if (!textRanges.empty()) {
        gsymCreator.SetValidTextRanges(textRanges);
    }

    if (auto err = dwarfTransformer.convert(convertNumThreads, nullptr)) {
        return err;
    }

    if (auto err = llvm::gsym::ObjectFileTransformer::convert(obj, nullptr, gsymCreator)) {
        return err;
    }
    if (auto err = FixupObjectFileTransformation(obj, gsymCreator)) {
        return err;
    }

    if (auto err = gsymCreator.finalize(os)) {
        return err;
    }

    const auto endian = obj.makeTriple().isLittleEndian() ? llvm::endianness::little : llvm::endianness::big;
    if (auto err = gsymCreator.save(output, endian)) {
        return err;
    }

    return llvm::Error::success();
}

const char* CopyErrorString(llvm::Error err) {
    const auto errStr = llvm::toString(std::move(err));

    return strndup(errStr.data(), errStr.size());
}

}

extern "C" {

const char* ConvertDWARFToGSYM(const char *input, const char *output, ui32 convertNumThreads) {
    if (input == nullptr) {
        return kNullInputError;
    }
    if (output == nullptr) {
        return kNullOutputError;
    }

    auto binary = llvm::object::createBinary(input);
    if (!binary) {
        return CopyErrorString(binary.takeError());
    }

    auto* obj = llvm::dyn_cast<llvm::object::ObjectFile>(binary->getBinary());
    if (!obj) {
        return kNotAnELFError;
    }

    if (auto err = ConvertDWARFToGSYM(*obj, output, convertNumThreads)) {
        return CopyErrorString(std::move(err));
    }

    return nullptr;
}

}
