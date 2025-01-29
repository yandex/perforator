#include "analyze.h"
#include "ehframe.h"

#include <perforator/lib/tls/parser/tls.h>
#include <perforator/lib/llvmex/llvm_exception.h>
#include <perforator/lib/python/python.h>

#include <library/cpp/streams/zstd/zstd.h>

#include <llvm/DebugInfo/DWARF/DWARFContext.h>
#include <llvm/MC/MCRegisterInfo.h>
#include <llvm/MC/TargetRegistry.h>
#include <llvm/Object/ObjectFile.h>
#include <llvm/Support/TargetSelect.h>

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
    auto analyzer = NPerforator::NLinguist::NPython::TPythonAnalyzer{objectFile};
    NPerforator::NBinaryProcessing::NPython::PythonConfig conf;
    auto version = analyzer.ParseVersion();
    if (!version) {
        return conf;
    }
    conf.MutableVersion()->SetMajor(version->Version.MajorVersion);
    conf.MutableVersion()->SetMinor(version->Version.MinorVersion);
    conf.MutableVersion()->SetMicro(version->Version.MicroVersion);

    auto threadStateTLSOffset = analyzer.ParseTLSPyThreadState();
    if (!threadStateTLSOffset) {
        return conf;
    }
    conf.SetPyThreadStateTLSOffset(*threadStateTLSOffset);

    auto pyRuntimeAddress = analyzer.ParsePyRuntimeAddress();
    if (!pyRuntimeAddress) {
        return conf;
    }
    conf.SetRelativePyRuntimeAddress(*pyRuntimeAddress);

    return conf;
}

} // namespace NPerforator::NBinaryProcessing::NPython

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

    NPerforator::NBinaryProcessing::BinaryAnalysis result;
    *result.MutableUnwindTable() = std::move(unwtable);
    *result.MutableTLSConfig() = std::move(tlsConfig);
    *result.MutablePythonConfig() = std::move(pythonConfig);

    return result;
}

} // namespace NPerforator::NBinaryProcessing
