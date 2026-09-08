#include <perforator/internal/linguist/jvm/analysis/static/static_analysis.h>

#include <library/cpp/getopt/last_getopt.h>

#include <library/cpp/json/json_value.h>
#include <library/cpp/json/json_writer.h>

#include <contrib/libs/protobuf/src/google/protobuf/text_format.h>
#include <contrib/libs/protobuf/src/google/protobuf/util/message_differencer.h>

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

NPerforator::NLinguist::NJvm::TJvmAnalysis DumpDynamic(std::string libjvmPath) {
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

    return NPerforator::NLinguist::NJvm::ProcessDynamicLinkedJVM(addresses);
}

void Write(NPerforator::NBinaryProcessing::NJvm::Cheatsheet cheatsheet, const std::string& path) {
    TProtoStringType repr;
    google::protobuf::TextFormat::PrintToString(cheatsheet, &repr);
    TUnbufferedFileOutput out{path};
    out << repr << Endl;
    out.Finish();
}

bool Validate(NPerforator::NBinaryProcessing::NJvm::Cheatsheet cheatsheet, const std::string& path) {
    TUnbufferedFileInput in{path};
    std::string data = in.ReadAll();
    NPerforator::NBinaryProcessing::NJvm::Cheatsheet actual;
    google::protobuf::TextFormat::ParseFromString(data, &actual);
    return google::protobuf::util::MessageDifferencer::Equals(cheatsheet, actual);
}

}

int main(int argc, char** argv) {
    using namespace std::literals;
    using namespace NPerforator::NLinguist::NJvm;

    bool validateOnly = false;
    NLastGetopt::TOpts opts;
    opts.AddLongOption("jvm-path").Required().Help("Path to libjvm.so");
    opts.AddLongOption("dbg-path").Required().Help("Path to libjvm debuginfo");
    opts.AddLongOption("out-dir").DefaultValue("..").Help("output directory");
    opts.AddLongOption("validate").SetFlag(&validateOnly).Help("Check that output file is up-to-date instead of updating it");


    NLastGetopt::TOptsParseResult parsed{&opts, argc, argv};

    TString out = parsed.Get("out-dir");
    TString libjvmPath = parsed.Get("jvm-path");
    TString dbgPath = parsed.Get("dbg-path");


    TJvmAnalysis dynamic = DumpDynamic(libjvmPath);

    TJvmAnalysis spec = NPerforator::NLinguist::NJvm::ProcessJVMDwarf(dbgPath, dynamic.Version);

    spec.Version = dynamic.Version;

    Cout << "Writing cheatsheets for JDK " << spec.Version << Endl;

    spec.Cheatsheet.MergeFrom(dynamic.Cheatsheet);
    std::string path = out + std::format("/jdk{}.txtpb", spec.Version);
    if (validateOnly) {
        bool ok = Validate(spec.Cheatsheet, path);
        if (!ok) {
            Cout << "Cheatsheet " << path << " is outdated" << Endl;
            return 1;
        }
    } else {
        Write(spec.Cheatsheet, path);
    }

    Cout << "OK" << Endl;
}
