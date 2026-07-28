#include <llvm/Object/ObjectFile.h>
#include <llvm/Support/TargetSelect.h>
#include <perforator/lib/llvmex/llvm_exception.h>
#include <perforator/lib/lua/lua.h>
#include <util/stream/format.h>

int main(int argc, const char *argv[]) {
    llvm::InitializeNativeTarget();
    llvm::InitializeNativeTargetDisassembler();

    Y_THROW_UNLESS(argc == 2);
    auto objectFile =
        Y_LLVM_RAISE(llvm::object::ObjectFile::createObjectFile(argv[1]));

    NPerforator::NLinguist::NLua::TLuaAnalyzer analyzer{
        *objectFile.getBinary()};

    TMaybe<NPerforator::NLinguist::NLua::TParsedLuaVersion> version =
        analyzer.ParseVersion();
    if (!version) {
        Cout << "Does not seem like Lua binary" << Endl;
        return 0;
    }

    Cout << "Detected Lua binary" << Endl;
    Cout << "Binary size: " << analyzer.GetBinarySize() << Endl;

    if (version) {
        Cout << "Parsed Lua binary version " << version->ToString() << Endl;
    } else {
        Cout << "Could not parse Lua version" << Endl;
    }

    auto offset = analyzer.ParseOffsetGtoL();
    if (!offset) {
        Cout << "Found no `offsetof(global_State, cur_L)`" << Endl;
    } else {
        Cout << "Found `offsetof(global_State, cur_L)`: " << *offset << Endl;
    }

    offset = analyzer.ParseOffsetGtoDispatch();
    if (!offset) {
        Cout << "Found no `GG_G2DISP` offset" << Endl;
    } else {
        Cout << "Found `GG_G2DISP` offset: " << *offset << Endl;
    }
}
