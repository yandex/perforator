#include <perforator/lib/tls/parser/tls.h>
#include <perforator/lib/llvmex/llvm_exception.h>

#include <util/stream/format.h>

#include <llvm/Object/ObjectFile.h>


int main(int argc, const char* argv[]) {
    Y_THROW_UNLESS(argc == 2);
    auto objectFile = Y_LLVM_RAISE(llvm::object::ObjectFile::createObjectFile(argv[1]));

    NPerforator::NThreadLocal::TTlsParser parser{objectFile.getBinary()};
    parser.VisitVariables([](auto&& symbol) {
        Cerr
            << "Found TLS symbol " << symbol.Name
            << " with offset " << Hex(symbol.ThreadImageOffset, HF_ADDX)
            << Endl;
    });
}
