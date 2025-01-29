#include <library/cpp/getopt/last_getopt.h>
#include <library/cpp/getopt/modchooser.h>
#include <library/cpp/getopt/small/last_getopt_opts.h>
#include <library/cpp/getopt/small/last_getopt_parse_result.h>
#include <library/cpp/logger/global/global.h>

#include <perforator/symbolizer/lib/symbolize/symbolizer.h>

#include <util/folder/path.h>
#include <util/stream/file.h>
#include <util/stream/zlib.h>

int main(int argc, const char* argv[]) {
    InitGlobalLog2Console();

    TFsPath profilePath;
    TFsPath outputPath;

    NLastGetopt::TOpts opts;
    opts
        .AddLongOption('p', "profile-path", "Input profile path")
        .Required()
        .StoreResult(&profilePath);
    opts
        .AddLongOption('o', "output-profile", "Output profile path (default - input profile path)")
        .Optional()
        .StoreResult(&outputPath);

    NLastGetopt::TOptsParseResult res(&opts, argc, argv);

    if (TString(outputPath) == "") {
        outputPath = profilePath;
    }

    TFileInput profileProto(profilePath);
    NPerforator::NProto::NPProf::Profile profile;
    if (TString{profilePath}.EndsWith(".tar.gz")) {
        TZLibDecompress decompresedInput(&profileProto);
        profile.ParseFromArcadiaStream(&decompresedInput);
    } else {
        profile.ParseFromArcadiaStream(&profileProto);
    }

    NPerforator::NSymbolize::TProfileSymbolizer symbolizer;
    symbolizer.Symbolize(profile);

    TFileOutput fileOutput(outputPath);
    TZLibCompress output(&fileOutput, ZLib::GZip, 1);

    profile.SerializeToArcadiaStream(&output);
}
