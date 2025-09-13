#include "golden.h"

#include <perforator/lib/profile/flat_diffable.h>

#include <contrib/libs/re2/re2/re2.h>

#include <util/folder/iterator.h>
#include <util/folder/path.h>
#include <util/generic/vector.h>
#include <util/stream/file.h>
#include <util/stream/zlib.h>


namespace NPerforator::NProfile::NTest {

NPerforator::NProto::NPProf::Profile ParsePprof(const TFsPath& path) {
    TFileInput serialized{path};
    TZLibDecompress uncompressed{&serialized};
    NPerforator::NProto::NPProf::Profile proto;
    Y_ENSURE(proto.ParseFromArcadiaStream(&uncompressed));
    return proto;
}

TVector<TFsPath> ListGoldenProfiles(TStringBuf pattern, TMaybe<size_t> expectedProfileCount) {
    re2::RE2 regex{pattern};

    TFsPath dir = SRC_("testprofiles");
    TVector<TFsPath> children;
    dir.List(children);

    TVector<TFsPath> profiles;

    TDirIterator iterator{dir, TDirIterator::TOptions{FTS_LOGICAL | FTS_XDEV | FTS_NOSTAT}};
    for (auto entry : iterator) {
        if (entry.fts_info != FTS_F) {
            continue;
        }

        TStringBuf absPath{entry.fts_path, entry.fts_pathlen};
        TFsPath localPath = TFsPath{absPath}.RelativeTo(dir);
        if (!re2::RE2::PartialMatch(localPath.GetPath(), regex)) {
            continue;
        }

        profiles.emplace_back(absPath);
    }

    if (expectedProfileCount) {
        Y_ENSURE(
            profiles.size() == *expectedProfileCount,
            "Expected " << *expectedProfileCount << " profiles matching " << pattern << ", got " << profiles.size()
        );
    }

    return profiles;
}

TString SerializeFlatProfile(const NPerforator::NProfile::TFlatDiffableProfile& profile) {
    TStringStream out;
    profile.WriteTo(out);
    return out.Str();
}

TMap<TString, ui64> CountFlatProfileEvents(const NPerforator::NProfile::TFlatDiffableProfile& profile) {
    TMap<TString, ui64> sum;

    profile.IterateSamples([&](TStringBuf, const TMap<TString, ui64>& values) {
        for (auto& [type, value] : values) {
            sum[type] += value;
        }
    });

    return sum;
}

} // namespace NPerforator::NProfile::NTest
