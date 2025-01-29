#include <perforator/lib/python/python.h>
#include <perforator/lib/llvmex/llvm_exception.h>

#include <util/stream/format.h>

#include <llvm/Object/ObjectFile.h>

constexpr TStringBuf kThreadLocalPyThreadStateVarName = "_Py_tss_tstate";

int main(int argc, const char* argv[]) {
    llvm::InitializeNativeTarget();
    llvm::InitializeNativeTargetDisassembler();

    Y_THROW_UNLESS(argc == 2);
    auto objectFile = Y_LLVM_RAISE(llvm::object::ObjectFile::createObjectFile(argv[1]));

    if (!NPerforator::NLinguist::NPython::IsPythonBinary(objectFile.getBinary())) {
        Cout << "Does not seem like python binary" << Endl;
        return 0;
    }

    Cout << "Detected Python binary" << Endl;

    NPerforator::NLinguist::NPython::TPythonAnalyzer analyzer{objectFile.getBinary()};
    TMaybe<NPerforator::NLinguist::NPython::TParsedPythonVersion> version = analyzer.ParseVersion();
    if (version) {
        Cout << "Parsed python binary version "
            << version->ToString() << Endl;
    } else {
        Cout << "Could not parse python version" << Endl;
    }

    auto offset = analyzer.ParseTLSPyThreadState();
    if (!offset) {
        Cout
            << "Found no `" << kThreadLocalPyThreadStateVarName
            << "` thread image offset" << Endl;
    } else {
        Cout
            << "Found offset for `" << kThreadLocalPyThreadStateVarName
            << "`: " << *offset << Endl;
    }

    auto pyRuntimeAddress = analyzer.ParsePyRuntimeAddress();
    if (!pyRuntimeAddress) {
        Cout << "Found no `_PyRuntime` address" << Endl;
    } else {
        Cout << "Found `_PyRuntime` address: " << *pyRuntimeAddress << Endl;
    }
}
