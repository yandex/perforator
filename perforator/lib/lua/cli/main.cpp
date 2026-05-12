#include <llvm/Object/ObjectFile.h>
#include <llvm/Support/TargetSelect.h>
#include <perforator/lib/llvmex/llvm_exception.h>
#include <perforator/lib/lua/lua.h>
#include <util/stream/format.h>

int main(int argc, const char* argv[]) {
  llvm::InitializeNativeTarget();
  llvm::InitializeNativeTargetDisassembler();

  Y_THROW_UNLESS(argc == 2);
  auto objectFile =
      Y_LLVM_RAISE(llvm::object::ObjectFile::createObjectFile(argv[1]));

  NPerforator::NLinguist::NLua::TLuaAnalyzer analyzer{*objectFile.getBinary()};
  TMaybe<NPerforator::NLinguist::NLua::TParsedLuaVersion> version =
      analyzer.ParseVersion();

  if (!version) {
    Cout << "Does not seem like Lua binary" << Endl;
    return 0;
  }

  Cout << "Detected Lua binary" << Endl;

  if (version) {
    Cout << "Parsed Lua binary version " << version->ToString() << Endl;
  } else {
    Cout << "Could not parse Lua version" << Endl;
  }

  auto offset = analyzer.ParseOffsetGtoL();
  if (!offset) {
    Cout << "Found no `OFFSET_G_TO_L` offset" << Endl;
  } else {
    Cout << "Found `OFFSET_G_TO_L` offset: " << *offset << Endl;
  }

  offset = analyzer.ParseOffsetGtoDispatch();
  if (!offset) {
    Cout << "Found no `OFFSET_G_TO_DISPATCH` offset" << Endl;
  } else {
    Cout << "Found `OFFSET_G_TO_DISPATCH` offset: " << *offset << Endl;
  }
}
