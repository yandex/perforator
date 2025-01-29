#include <library/cpp/getopt/last_getopt.h>
#include <library/cpp/getopt/modchooser.h>
#include <library/cpp/getopt/small/last_getopt_opts.h>
#include <library/cpp/getopt/small/last_getopt_parse_result.h>

#include <iostream>

#include <util/folder/path.h>
#include <util/stream/file.h>
#include <util/stream/zlib.h>

#include <perforator/proto/pprofprofile/profile.pb.h>
#include <perforator/symbolizer/lib/autofdo/autofdo_input_builder.h>

namespace {

void ProcessProfile(const NPerforator::NProto::NPProf::Profile& profile, const std::string& buildId) {
    NPerforator::NAutofdo::TInputBuilder builder{buildId};
    builder.AddProfile(profile);

    const auto autofdoInputData = std::move(builder).Finalize();
    const auto autofdoInput = NPerforator::NAutofdo::SerializeAutofdoInput(autofdoInputData);

    std::cout << autofdoInput << std::endl;
}

}

int main(int argc, const char *argv[]) {
    TFsPath profilePath;
    std::string buildId{};

    NLastGetopt::TOpts opts;
    opts.AddLongOption('p', "profile-path", "Input profile path")
        .Required()
        .StoreResult(&profilePath);
    opts.AddLongOption('b', "build-id", "Binary Build ID")
        .Required()
        .StoreResult(&buildId);

    NLastGetopt::TOptsParseResult res(&opts, argc, argv);

    TFileInput profileProto(profilePath);
    NPerforator::NProto::NPProf::Profile profile;
    if (TString{profilePath}.EndsWith(".tar.gz")) {
        TZLibDecompress decompresedInput(&profileProto);
        profile.ParseFromArcadiaStream(&decompresedInput);
    } else {
        profile.ParseFromArcadiaStream(&profileProto);
    }

    ProcessProfile(profile, buildId);

    return 0;
}
