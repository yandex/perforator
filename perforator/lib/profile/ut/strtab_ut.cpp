#include <perforator/lib/profile/builder.h>
#include <perforator/lib/profile/profile.h>
#include <perforator/lib/profile/string_table.h>
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
        EXPECT_THROW(TStringInterner{proto.mutable_strtab()}, yexception);
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

TEST(ProfileStringInterner, KeepsViewsValidAfterGrowth) {
    NPerforator::NProto::NProfile::StringTable table;
    TStringInterner interner{&table};
    auto first = interner.Intern("first");
    EXPECT_EQ(first.GetIndex(), TStringId::Zero());
    EXPECT_EQ(interner.Intern("first").GetIndex(), first.GetIndex());
    for (ui32 i = 0; i < 1000; ++i) {
        interner.Intern(TString(i + 1, 'x'));
    }
    EXPECT_EQ(first.AsStringBuf(), "first");
    EXPECT_EQ(interner.Get(first.GetIndex()), first);
    EXPECT_EQ(interner.Intern("first").GetIndex(), first.GetIndex());
    EXPECT_EQ(table.offset_size(), 1001);
}

TEST(ProfileStringInterner, ReusesExistingTable) {
    NPerforator::NProto::NProfile::StringTable table;
    table.set_strings("firstsecondfirst");
    for (const auto& [offset, length] : {std::pair{0, 5}, std::pair{5, 6}, std::pair{11, 5}}) {
        table.add_offset(offset);
        table.add_length(length);
    }
    TStringInterner interner{&table};
    EXPECT_EQ(*interner.Intern("first").GetIndex(), 0);
    EXPECT_EQ(*interner.Intern("second").GetIndex(), 1);
    EXPECT_EQ(*interner.Intern("third").GetIndex(), 3);
    EXPECT_EQ(interner.Get(TStringId::FromInternalIndex(2)).AsStringBuf(), "first");
}

TEST(ProfileStringInterner, InternsSubstringFromItsOwnTable) {
    NPerforator::NProto::NProfile::StringTable table;
    TStringInterner interner{&table};
    auto original = interner.Intern(TString(8192, 'x'));
    auto substring = interner.Intern(original.AsStringBuf().SubStr(1));
    EXPECT_EQ(substring.AsStringBuf(), TString(8191, 'x'));
    EXPECT_EQ(original.AsStringBuf(), TString(8192, 'x'));
    EXPECT_NE(substring.GetIndex(), original.GetIndex());
    EXPECT_EQ(interner.Intern(TString(8191, 'x')).GetIndex(), substring.GetIndex());
}

TEST(ProfileStringInterner, ComparesViewsFromDifferentTables) {
    NPerforator::NProto::NProfile::StringTable firstTable;
    NPerforator::NProto::NProfile::StringTable secondTable;
    TStringInterner first{&firstTable};
    TStringInterner second{&secondTable};
    auto a = first.Intern("a");
    auto b = second.Intern("b");
    EXPECT_NE(a, b);
    EXPECT_EQ(a, second.Intern("a"));
}
