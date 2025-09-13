#include "analyze.h"
#include "ehframe.h"

#include <perforator/lib/tls/parser/tls.h>
#include <perforator/lib/llvmex/llvm_exception.h>
#include <perforator/lib/pthread/pthread.h>
#include <perforator/lib/python/python.h>
#include <perforator/lib/php/php.h>

#include <library/cpp/streams/zstd/zstd.h>

#include <llvm/DebugInfo/DWARF/DWARFContext.h>
#include <llvm/MC/MCRegisterInfo.h>
#include <llvm/MC/TargetRegistry.h>
#include <llvm/Object/ObjectFile.h>
#include <llvm/Support/TargetSelect.h>

#include <util/generic/maybe.h>

namespace NPerforator::NBinaryProcessing::NTls {

NPerforator::NBinaryProcessing::NTls::TLSConfig BuildTlsConfig(llvm::object::ObjectFile* objectFile) {
    auto parser = NPerforator::NThreadLocal::TTlsParser(objectFile);
    NTls::TLSConfig conf;
    parser.VisitVariables([&](const NThreadLocal::TTlsParser::TVariableRef& symbol) {
        auto variable = conf.MutableVariables()->Add();
        variable->SetOffset(symbol.ThreadImageOffset);
        variable->SetName(symbol.Name.data(), symbol.Name.size());
    });

    return conf;
}

} // namespace NPerforator::NBinaryProcessing::NTls

namespace NPerforator::NBinaryProcessing::NPython {

NPerforator::NBinaryProcessing::NPython::PythonConfig BuildPythonConfig(llvm::object::ObjectFile* objectFile) {
    auto analyzer = NPerforator::NLinguist::NPython::TPythonAnalyzer{*objectFile};
    NPerforator::NBinaryProcessing::NPython::PythonConfig conf;
    auto version = analyzer.ParseVersion();
    if (!version) {
        return conf;
    }
    conf.MutableVersion()->SetMajor(version->Version.MajorVersion);
    conf.MutableVersion()->SetMinor(version->Version.MinorVersion);
    conf.MutableVersion()->SetMicro(version->Version.MicroVersion);

    auto threadStateTLSOffset = analyzer.ParseTLSPyThreadState();
    if (threadStateTLSOffset) {
        conf.SetPyThreadStateTLSOffset(*threadStateTLSOffset);
    }

    auto pyRuntimeAddress = analyzer.ParsePyRuntimeAddress();
    if (pyRuntimeAddress) {
        conf.SetRelativePyRuntimeAddress(*pyRuntimeAddress);
    }

    auto interpHeadAddress = analyzer.ParseInterpHeadAddress();
    if (interpHeadAddress) {
        conf.SetRelativePyInterpHeadAddress(*interpHeadAddress);
    }

    auto autoTSSkeyAddress = analyzer.ParseAutoTSSKeyAddress();
    if (autoTSSkeyAddress) {
        conf.SetRelativeAutoTSSkeyAddress(*autoTSSkeyAddress);
    }

    auto unicodeType = analyzer.ParseUnicodeType();
    if (unicodeType == NPerforator::NLinguist::NPython::EUnicodeType::UCS2) {
        conf.SetUnicodeTypeSizeLog2(1);
    } else if (unicodeType == NPerforator::NLinguist::NPython::EUnicodeType::UCS4) {
        conf.SetUnicodeTypeSizeLog2(2);
    }

    return conf;
}

} // namespace NPerforator::NBinaryProcessing::NPython

namespace NPerforator::NBinaryProcessing::NPthread {

TMaybe<NPerforator::NBinaryProcessing::NPthread::PthreadConfig> BuildPthreadConfig(llvm::object::ObjectFile* objectFile) {
    auto analyzer = NPerforator::NPthread::TLibPthreadAnalyzer{*objectFile};
    auto accessTSSInfo = analyzer.ParseAccessTSSInfo();
    if (!accessTSSInfo) {
        return Nothing();
    }

    NPerforator::NBinaryProcessing::NPthread::PthreadConfig conf;
    conf.MutableKeyData()->SetSize(accessTSSInfo->PthreadKeyData.Size);
    conf.MutableKeyData()->SetValueOffset(accessTSSInfo->PthreadKeyData.ValueOffset);
    conf.MutableKeyData()->SetSeqOffset(accessTSSInfo->PthreadKeyData.SeqOffset);
    conf.SetFirstSpecificBlockOffset(accessTSSInfo->FirstSpecificBlockOffset);
    conf.SetSpecificArrayOffset(accessTSSInfo->SpecificArrayOffset);
    conf.SetStructPthreadPointerOffset(accessTSSInfo->StructPthreadPointerOffset);
    conf.SetKeySecondLevelSize(accessTSSInfo->KeySecondLevelSize);
    conf.SetKeyFirstLevelSize(accessTSSInfo->KeyFirstLevelSize);
    conf.SetKeysMax(accessTSSInfo->KeysMax);

    return MakeMaybe(conf);
}

} // namespace NPerforator::NBinaryProcessing::NPthread

namespace NPerforator::NBinaryProcessing::NPhp {

TMaybe<NPerforator::NBinaryProcessing::NPhp::PhpConfig> BuildPhpConfig(llvm::object::ObjectFile* objectFile) {
    NPerforator::NLinguist::NPhp::TZendPhpAnalyzer analyzer{*objectFile};
    NPerforator::NBinaryProcessing::NPhp::PhpConfig conf;
    auto version = analyzer.ParseVersion();
    if (!version) {
        return Nothing();
    }
    conf.MutableVersion()->SetMajor(version->Version.MajorVersion);
    conf.MutableVersion()->SetMinor(version->Version.MinorVersion);
    conf.MutableVersion()->SetRelease(version->Version.ReleaseVersion);

    auto ztsEnabled = analyzer.ParseZts();
    if (!ztsEnabled) {
        return MakeMaybe(conf);
    }
    conf.SetZtsEnabled(*ztsEnabled);

    auto vmKind = analyzer.ParseZendVmKind();
    if (!vmKind) {
        return MakeMaybe(conf);
    }
    conf.SetZendVmKind(static_cast<ui32>(*vmKind));

    auto executorGlobalsAddress = analyzer.ParseExecutorGlobals();
    if (!executorGlobalsAddress) {
        return MakeMaybe(conf);
    }
    conf.SetExecutorGlobalsELFVaddr(*executorGlobalsAddress);

    return MakeMaybe(conf);
}

} // namespace NPerforator::NBinaryProcessing::NPhp

namespace NPerforator::NBinaryProcessing {

void SerializeBinaryAnalysis(BinaryAnalysis&& analysis, IOutputStream* out) {
    NUnwind::DeltaEncode(*analysis.MutableUnwindTable());
    TZstdCompress compress{out};
    Y_ENSURE(analysis.SerializeToArcadiaStream(&compress));
    compress.Finish();
}

BinaryAnalysis DeserializeBinaryAnalysis(IInputStream* input) {
    BinaryAnalysis analysis;

    TZstdDecompress in{input};
    Y_ENSURE(analysis.ParseFromArcadiaStream(&in));

    NUnwind::IntegrateUnwindTable(*analysis.MutableUnwindTable());

    return analysis;
}

NPerforator::NBinaryProcessing::BinaryAnalysis AnalyzeBinary(const char* path) {
    static std::once_flag once;
    std::call_once(once, [] {
        llvm::InitializeNativeTarget();
        llvm::InitializeNativeTargetDisassembler();
    });

    auto objectFile = Y_LLVM_RAISE(llvm::object::ObjectFile::createObjectFile(path));
    auto unwtable = NUnwind::BuildUnwindTable(objectFile.getBinary());
    auto tlsConfig = NTls::BuildTlsConfig(objectFile.getBinary());
    auto pythonConfig = NPython::BuildPythonConfig(objectFile.getBinary());
    auto pthreadConfig = NPthread::BuildPthreadConfig(objectFile.getBinary());
    auto phpConfig = NPhp::BuildPhpConfig(objectFile.getBinary());

    NPerforator::NBinaryProcessing::BinaryAnalysis result;
    *result.MutableUnwindTable() = std::move(unwtable);
    *result.MutableTLSConfig() = std::move(tlsConfig);
    *result.MutablePythonConfig() = std::move(pythonConfig);

    if (phpConfig) {
        *result.MutablePhpConfig() = std::move(phpConfig.GetRef());
    }

    if (pthreadConfig) {
        *result.MutablePthreadConfig() = std::move(pthreadConfig.GetRef());
    }

    return result;
}

} // namespace NPerforator::NBinaryProcessing
