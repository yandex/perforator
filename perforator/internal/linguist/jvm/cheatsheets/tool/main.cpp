#include <perforator/internal/linguist/jvm/analysis/static/static_analysis.h>

#include <library/cpp/getopt/last_getopt.h>

#include <library/cpp/json/json_value.h>
#include <library/cpp/json/json_writer.h>

#include <contrib/libs/protobuf/src/google/protobuf/text_format.h>

#include <util/stream/file.h>

#include <dlfcn.h>

#include <string>

namespace {
struct TDeleter {
    void operator()(void* handle) {
        int err = dlclose(handle);
        if (err != 0) {
            std::string msg = dlerror();
            Cerr << "failed to close libjvm.so: " << msg << Endl;
        }
    }
};

NPerforator::NLinguist::NJvm::TJvmAnalysis DumpDynamic(std::string libjvmPath, ui32 version) {
    using namespace NPerforator::NLinguist::NJvm;
    void* rawHandle = dlopen(libjvmPath.c_str(), RTLD_LAZY | RTLD_LOCAL);
    if (rawHandle == nullptr) {
        char* msg = dlerror();
        throw yexception() << "failed to load libjvm.so: " << msg;
    }
    std::unique_ptr<void, TDeleter> handle(rawHandle);

    auto GetSym = [&](const std::string& sym) {
        const void* addr = dlsym(handle.get(), sym.c_str());
        if (addr == nullptr) {
            char* msg = dlerror();
            throw yexception() << "failed to load symbol " << sym << ": " << msg;
        }
        return addr;
    };

    TVMStructsAddresses addresses;
    addresses.StructsAddress = GetSym(std::string{TVMStructsAddresses::StructsAddressSym});
    addresses.StructsStructNameOffset = GetSym(std::string{TVMStructsAddresses::StructsStructNameOffsetSym});
    addresses.StructsFieldNameOffset = GetSym(std::string{TVMStructsAddresses::StructsFieldNameOffsetSym});
    addresses.StructsTypeNameOffset = GetSym(std::string{TVMStructsAddresses::StructsTypeNameOffsetSym});
    addresses.StructsIsStaticOffset = GetSym(std::string{TVMStructsAddresses::StructsIsStaticOffsetSym});
    addresses.StructsOffsetOffset = GetSym(std::string{TVMStructsAddresses::StructsOffsetOffsetSym});
    addresses.StructsAddressOffset = GetSym(std::string{TVMStructsAddresses::StructsAddressOffsetSym});
    addresses.StructsStride = GetSym(std::string{TVMStructsAddresses::StructsStrideSym});

    addresses.TypesAddress = GetSym(std::string{TVMStructsAddresses::TypesAddressSym});
    addresses.TypesStructNameOffset = GetSym(std::string{TVMStructsAddresses::TypesStructNameOffsetSym});
    addresses.TypesSuperNameOffset = GetSym(std::string{TVMStructsAddresses::TypesSuperNameOffsetSym});
    addresses.TypesIsOopOffset = GetSym(std::string{TVMStructsAddresses::TypesIsOopOffsetSym});
    addresses.TypesIsIntegerOffset = GetSym(std::string{TVMStructsAddresses::TypesIsIntegerOffsetSym});
    addresses.TypesIsUnsignedOffset = GetSym(std::string{TVMStructsAddresses::TypesIsUnsignedOffsetSym});
    addresses.TypesSizeOffset = GetSym(std::string{TVMStructsAddresses::TypesSizeOffsetSym});
    addresses.TypesStride = GetSym(std::string{TVMStructsAddresses::TypesStrideSym});

    addresses.IntsAddress = GetSym(std::string{TVMStructsAddresses::IntsAddressSym});
    addresses.IntsNameOffset = GetSym(std::string{TVMStructsAddresses::IntsNameOffsetSym});
    addresses.IntsValueOffset = GetSym(std::string{TVMStructsAddresses::IntsValueOffsetSym});
    addresses.IntsStride = GetSym(std::string{TVMStructsAddresses::IntsStrideSym});

    return NPerforator::NLinguist::NJvm::ProcessDynamicLinkedJVM(addresses, version);
}

void Write(NPerforator::NBinaryProcessing::NJvm::Cheatsheet cheatsheet, const TString& path) {
    TProtoStringType repr;
    google::protobuf::TextFormat::PrintToString(cheatsheet, &repr);
    TUnbufferedFileOutput out{path};
    out << repr << Endl;
    out.Finish();
}

}

int main(int argc, char** argv) {
    using namespace std::literals;
    using namespace NPerforator::NLinguist::NJvm;

    NLastGetopt::TOpts opts;
    opts.AddLongOption("jvm-path").Required().Help("Path to libjvm.so");
    opts.AddLongOption("out-dir").DefaultValue("..").Help("output directory");


    NLastGetopt::TOptsParseResult parsed{&opts, argc, argv};

    TString out = parsed.Get("out-dir");
    TString libjvmPath = parsed.Get("jvm-path");

    TJvmAnalysis spec = NPerforator::NLinguist::NJvm::ProcessJVMHeaders();

    int version = spec.Version;

    TJvmAnalysis dynamic = DumpDynamic(libjvmPath, version);

    Cout << "Writing cheatsheets for JDK " << version << Endl;

    spec.Cheatsheet.MergeFrom(dynamic.Cheatsheet);
    Write(spec.Cheatsheet, out + std::format("/jdk{}.txtpb", version));
}
