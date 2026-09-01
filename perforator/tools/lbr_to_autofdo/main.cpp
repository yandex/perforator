#include <library/cpp/getopt/last_getopt.h>
#include <library/cpp/getopt/modchooser.h>
#include <library/cpp/getopt/small/last_getopt_opts.h>
#include <library/cpp/getopt/small/last_getopt_parse_result.h>

#include <iostream>

#include <util/folder/path.h>
#include <util/stream/file.h>
#include <util/stream/zlib.h>

#include <perforator/lib/profile/pprof.h>
#include <perforator/proto/profile/profile.pb.h>
#include <perforator/symbolizer/lib/autofdo/autofdo_input_builder.h>

namespace {

constexpr std::string_view kPlaceholderServiceName{"serviceName"};

void ProcessProfile(
    const NPerforator::NProto::NProfile::Profile& profile,
    const std::string& buildId,
    const std::string& binaryPath
) {
    NPerforator::NAutofdo::TBatchInputBuilder builder{1, buildId, binaryPath};
    builder.GetBuilder(0).AddProfile(kPlaceholderServiceName, profile);

    auto autofdoInputData = std::move(builder).Finalize();
    const auto [autofdoInput, _] = NPerforator::NAutofdo::SerializePGOInputsForBinary(autofdoInputData, binaryPath);

    std::cout << autofdoInput << std::endl;
}

}

int main(int argc, const char *argv[]) {
    TFsPath profilePath;
    std::string buildId{};
    std::string binaryPath{};

    NLastGetopt::TOpts opts;
    opts.AddLongOption('p', "profile-path", "Input profile path")
        .Required()
        .StoreResult(&profilePath);
    opts.AddLongOption('b', "build-id", "Binary Build ID")
        .Required()
        .StoreResult(&buildId);
    opts.AddLongOption("binary-path", "Profiled ELF binary path")
        .Required()
        .StoreResult(&binaryPath);

    NLastGetopt::TOptsParseResult res(&opts, argc, argv);

    TFileInput profileProto(profilePath);
    NPerforator::NProto::NPProf::Profile pprof;
    if (TString{profilePath}.EndsWith(".tar.gz")) {
        TZLibDecompress decompresedInput(&profileProto);
        pprof.ParseFromArcadiaStream(&decompresedInput);
    } else {
        pprof.ParseFromArcadiaStream(&profileProto);
    }

    NPerforator::NProto::NProfile::Profile profile;
    NPerforator::NProfile::ConvertFromPProf(pprof, &profile);

    ProcessProfile(profile, buildId, binaryPath);

    return 0;
}
