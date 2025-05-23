#include <llvm/Support/TargetSelect.h>
#include <perforator/lib/php/php.h>
#include <perforator/lib/llvmex/llvm_exception.h>

#include <util/stream/format.h>

#include <llvm/Object/ObjectFile.h>

int main(int argc, const char* argv[]) {
    llvm::InitializeNativeTarget();
    llvm::InitializeNativeTargetDisassembler();

    Y_THROW_UNLESS(argc == 2);
    auto objectFile = Y_LLVM_RAISE(llvm::object::ObjectFile::createObjectFile(argv[1]));

    NPerforator::NLinguist::NPhp::TZendPhpAnalyzer analyzer{*objectFile.getBinary()};
    TMaybe<NPerforator::NLinguist::NPhp::TParsedPhpVersion> version = analyzer.ParseVersion();
    if (version) {
        Cout << "Parsed php binary version: "
             << version->ToString() << Endl;
    } else {
        Cout << "Could not parse php version" << Endl;
    }
    TMaybe<NPerforator::NLinguist::NPhp::EZendVmKind> vmKind = analyzer.ParseZendVmKind();
    if (vmKind) {
        Cout << "Parsed zend vm kind: "
             << NPerforator::NLinguist::NPhp::ToString(*vmKind) << Endl;
    } else {
        Cout << "Could not parse zend vm kind" << Endl;
    }
}
