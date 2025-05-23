#include "php.h"

#include <perforator/lib/llvmex/llvm_elf.h>
#include <perforator/lib/llvmex/llvm_exception.h>
#include <perforator/lib/php/asm/x86-64/decode.h>

#include <contrib/libs/re2/re2/stringpiece.h>

namespace NPerforator::NLinguist::NPhp {

TString TParsedPhpVersion::ToString() const {
    TStringBuilder builder;
    builder << ui64(Version.MajorVersion) << "." << ui64(Version.MinorVersion)
            << "." << ui64(Version.ReleaseVersion);
    builder << "(source: ";
    switch (Source) {
        case EPhpVersionSource::PhpVersionDissasembly:
            builder << "php_version dissasembly";
            break;
        case EPhpVersionSource::ZmInfoPhpCoreDissasembly:
            builder << "zm_info_php_core dissasembly";
            break;
        case EPhpVersionSource::RodataScan:
            builder << "rodata scanning";
            break;
        default:
            builder << "unknown source";
            break;
    }
    builder << ")";
    return builder;
}

TString ToString(const EZendVmKind vmKind) {
    switch (vmKind) {
        case EZendVmKind::Call:
            return "Call";
        case EZendVmKind::Switch:
            return "Switch";
        case EZendVmKind::Goto:
            return "Goto";
        case EZendVmKind::Hybrid:
            return "Hybrid";
        default:
            return "Unknown vm kind";
    }
}

TZendPhpAnalyzer::TZendPhpAnalyzer(const llvm::object::ObjectFile& file)
    : File_(file)
{
}

void TZendPhpAnalyzer::ParseSymbolLocations() {
    if (Symbols_) {
        return;
    }

    Symbols_ = MakeHolder<TSymbols>();
    auto dynamicSymbols = NELF::RetrieveDynamicSymbols(File_,
                                                       kPhpVersionSymbol,
                                                       KZendVmKindSymbol);

    auto setSymbolIfFound =
        [&](const THashMap<TStringBuf, NPerforator::NELF::TLocation>& symbols,
            TStringBuf symbolName, TMaybe<NELF::TLocation>& target) {
            if (auto it = symbols.find(symbolName); it != symbols.end()) {
                target = it->second;
            }
        };

    if (dynamicSymbols) {
        setSymbolIfFound(*dynamicSymbols, kPhpVersionSymbol, Symbols_->PhpVersion);
        setSymbolIfFound(*dynamicSymbols, KZendVmKindSymbol, Symbols_->ZendVmKind);
    }

    auto symbols = NELF::RetrieveSymbols(File_, kZmInfoPhpCoreSymbol);
    if (symbols) {
        setSymbolIfFound(*symbols, kZmInfoPhpCoreSymbol, Symbols_->ZmInfoPhpCore);
    }
}

TMaybe<TPhpVersion> TryScanVersion(TConstArrayRef<char> data) {
    re2::StringPiece input(data.data(), data.size());
    std::string major, minor, release;

    while (re2::RE2::FindAndConsume(&input, kPhpVersionRegex, &major, &minor,
                                    &release)) {
        ui8 majorVersion = static_cast<ui8>(std::stoi(major));
        ui8 minorVersion = static_cast<ui8>(std::stoi(minor));
        ui8 releaseVersion = release.empty() ? 0 : static_cast<ui8>(std::stoi(release));

        if (release.empty()) {
            continue;
        }

        return MakeMaybe(TPhpVersion{
            .MajorVersion = majorVersion,
            .MinorVersion = minorVersion,
            .ReleaseVersion = releaseVersion,
        });
    }

    return Nothing();
}

TMaybe<TPhpVersion> TryExtractPhpVersion(const llvm::object::ObjectFile& file, ui64 versionAddress) {
    static constexpr size_t kPhpMaxVersionLength = 20;
    NPerforator::NELF::TLocation phpVersionLocation{.Address = versionAddress,
                                                    .Size = kPhpMaxVersionLength};

    auto phpVersionContent = NPerforator::NELF::RetrieveContentFromRodataSection(
        file, phpVersionLocation);
    if (!phpVersionContent) {
        return Nothing();
    }

    return TryScanVersion(TConstArrayRef<char>(
        reinterpret_cast<const char*>(phpVersionContent->data()),
        phpVersionContent->size()));
}

TMaybe<TPhpVersion>
TryParsePhpVersion(const llvm::object::ObjectFile& file,
                    const NPerforator::NELF::TLocation& phpVersion) {
    NPerforator::NELF::TLocation symbol = phpVersion;
    if (symbol.Size == 0) {
        symbol.Size = 32;
    }

    TMaybe<TConstArrayRef<ui8>> bytecode =
        NPerforator::NELF::RetrieveContentFromTextSection(file, symbol);
    if (!bytecode) {
        return Nothing();
    }
    auto versionAddress = NAsm::NX86::DecodePhpVersion(file.makeTriple(), symbol.Address, *bytecode);
    if (!versionAddress) {
        return Nothing();
    }

    return TryExtractPhpVersion(file, *versionAddress);
}

TMaybe<TPhpVersion>
TryParseZmInfoPhpCore(const llvm::object::ObjectFile& file,
                        const NPerforator::NELF::TLocation& zmInfoPhpCore) {
    NPerforator::NELF::TLocation symbol = zmInfoPhpCore;
    if (symbol.Size == 0) {
        symbol.Size = 128;
    }

    TMaybe<TConstArrayRef<ui8>> bytecode =
        NPerforator::NELF::RetrieveContentFromTextSection(file, symbol);
    if (!bytecode) {
        return Nothing();
    }
    auto versionAddress = NAsm::NX86::DecodeZmInfoPhpCore(file.makeTriple(), symbol.Address, *bytecode);
    if (!versionAddress) {
        return Nothing();
    }

    return TryExtractPhpVersion(file, *versionAddress);

}

TMaybe<TPhpVersion>
TryFindVersionInRodata(const llvm::object::ObjectFile& file) {
    TMaybe<llvm::object::SectionRef> rodataSection = NELF::GetSection(file, NPerforator::NELF::NSections::kRoDataSectionName);
    if (!rodataSection) {
        return Nothing();
    }
    Y_LLVM_UNWRAP(content, rodataSection->getContents(), { return Nothing(); });
    size_t keyPhraseInd = content.find(kPhpVersionKeyPhrase);
    if (keyPhraseInd == llvm::StringRef::npos) {
        return Nothing();
    }
    size_t versionStart = keyPhraseInd + kPhpVersionKeyPhrase.size();
    size_t versionEnd = content.find('\0', versionStart);
    llvm::StringRef versionString = content.substr(versionStart, versionEnd - versionStart + 1);
    return TryScanVersion(TConstArrayRef<char>(
        reinterpret_cast<const char*>(versionString.bytes_begin()),
        versionString.size()));
}

TMaybe<TParsedPhpVersion> ParseVersion(const llvm::object::ObjectFile& file,
                const TZendPhpAnalyzer::TSymbols& symbols) {
    if (symbols.PhpVersion) {
        if (auto version = TryParsePhpVersion(file, *symbols.PhpVersion)) {
            return MakeMaybe(TParsedPhpVersion{
                .Version = *version,
                .Source = EPhpVersionSource::PhpVersionDissasembly});
        }
    }

    if (symbols.ZmInfoPhpCore) {
        if (auto version = TryParseZmInfoPhpCore(file, *symbols.ZmInfoPhpCore)) {
            return MakeMaybe(TParsedPhpVersion{
                .Version = *version,
                .Source = EPhpVersionSource::ZmInfoPhpCoreDissasembly});
        }
    }

    if (auto version = TryFindVersionInRodata(file)) {
        return MakeMaybe(TParsedPhpVersion{
            .Version = *version,
            .Source = EPhpVersionSource::RodataScan});
    }

    return Nothing();
}

TMaybe<TParsedPhpVersion> TZendPhpAnalyzer::ParseVersion() {
    ParseSymbolLocations();

    if (!Symbols_ || !NPerforator::NELF::IsElfFile(File_)) {
        return Nothing();
    }

    if (auto res = NPerforator::NLinguist::NPhp::ParseVersion(File_, *Symbols_.Get())) {
        return res;
    }
    return Nothing();
}

TMaybe<EZendVmKind> TZendPhpAnalyzer::ParseZendVmKind() {
    ParseSymbolLocations();

    if (!Symbols_ || !Symbols_->ZendVmKind) {
        return Nothing();
    }

    NPerforator::NELF::TLocation& zendVmKindSymbol = *Symbols_->ZendVmKind;
    if (zendVmKindSymbol.Size == 0) {
        zendVmKindSymbol.Size = 32;
    }

    auto bytecode = NPerforator::NELF::RetrieveContentFromTextSection(File_, zendVmKindSymbol);
    if (!bytecode) {
        return Nothing();
    }

    TMaybe<ui64> vmKindValue = NAsm::NX86::DecodeZendVmKind(
        File_.makeTriple(),
        zendVmKindSymbol.Address,
        *bytecode);

    if (!vmKindValue) {
        return Nothing();
    }
    EZendVmKind vmKindEnum;
    switch (*vmKindValue) {
        case 1:
            vmKindEnum = EZendVmKind::Call;
            break;
        case 2:
            vmKindEnum = EZendVmKind::Switch;
            break;
        case 3:
            vmKindEnum = EZendVmKind::Goto;
            break;
        case 4:
            vmKindEnum = EZendVmKind::Hybrid;
            break;
        default:
            return Nothing();
    }

    return MakeMaybe(vmKindEnum);
}

} // namespace NPerforator::NLinguist::NPhp
