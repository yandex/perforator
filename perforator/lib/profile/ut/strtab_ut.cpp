#include <perforator/lib/profile/builder.h>
#include <perforator/lib/profile/profile.h>
#include <perforator/lib/profile/validate.h>

#include <library/cpp/testing/gtest/gtest.h>

#include <limits>


using namespace NPerforator::NProfile;

namespace {

NPerforator::NProto::NProfile::Profile MakeProfile() {
    NPerforator::NProto::NProfile::Profile proto;
    TProfileBuilder builder{&proto};
    std::move(builder).Finish();

    auto& strtab = *proto.mutable_strtab();
    strtab.Clear();
    strtab.set_strings("abcd");
    strtab.add_offset(0);
    strtab.add_length(0);
    strtab.add_offset(0);
    strtab.add_length(4);
    return proto;
}

} // namespace

TEST(ProfileStringTable, RejectsInvalidRanges) {
    constexpr ui32 max = std::numeric_limits<ui32>::max();
    const struct {
        ui32 Offset;
        ui32 Length;
    } cases[] = {
        {max, 2}, // Wraps to 1.
        {1, max}, // Wraps to 0 with an in-bounds offset.
        {max, max},
        {5, 0}, // Offset beyond the buffer.
        {3, 2}, // End beyond the buffer without overflow.
    };

    for (const auto& test : cases) {
        SCOPED_TRACE(testing::Message() << test.Offset << ", " << test.Length);
        auto proto = MakeProfile();
        const auto id = TStringId::FromInternalIndex(1);
        ASSERT_NO_THROW(ValidateProfile(proto, {.CheckIndices = true}));

        proto.mutable_strtab()->set_offset(*id, test.Offset);
        proto.mutable_strtab()->set_length(*id, test.Length);
        EXPECT_THROW(ValidateProfile(proto, {.CheckIndices = true}), yexception);
        EXPECT_THROW(TProfile{&proto}.String(id).View(), yexception);
    }
}

TEST(ProfileStringTable, AcceptsValidBoundaryRanges) {
    const struct {
        ui32 Offset;
        ui32 Length;
        TStringBuf Expected;
    } cases[] = {
        {0, 4, "abcd"},
        {1, 3, "bcd"},
        {0, 0, ""},
        {4, 0, ""},
    };

    for (const auto& test : cases) {
        SCOPED_TRACE(testing::Message() << test.Offset << ", " << test.Length);
        auto proto = MakeProfile();
        const auto id = TStringId::FromInternalIndex(1);

        proto.mutable_strtab()->set_offset(*id, test.Offset);
        proto.mutable_strtab()->set_length(*id, test.Length);
        EXPECT_NO_THROW(ValidateProfile(proto, {.CheckIndices = true}));
        EXPECT_EQ(TProfile{&proto}.String(id).View(), test.Expected);
    }
}
