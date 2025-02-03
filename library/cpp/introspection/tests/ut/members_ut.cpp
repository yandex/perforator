#include <library/cpp/introspection/introspection.h>
#include <library/cpp/introspection/tests/data.h>
#include <library/cpp/testing/gtest/gtest.h>


TEST(TReflectionMembers, TestEmpty) {
    NIntrospectionTest::TEmpty data;
    auto members = NIntrospection::Members(data);
    static_assert(std::is_same_v<std::tuple<>, decltype(members)>);
}

TEST(TReflectionMembers, TestOne) {
    NIntrospectionTest::TChar data{};
    auto members = NIntrospection::Members(data);
    static_assert(std::is_same_v<std::tuple<char&>, decltype(members)>);
}

TEST(TReflectionMembers, TestComposite) {
    NIntrospectionTest::TComposite7 data{};
    auto members = NIntrospection::Members(data);
    using TExpected = std::tuple<char&, ui32&, double&, TString&, NIntrospectionTest::TArray1&, i64&, NIntrospectionTest::TChar&>;
    static_assert(std::is_same_v<TExpected, decltype(members)>);
}

TEST(TReflectionMembers, TestMembersNonConst) {
    NIntrospectionTest::TChar data{'a'};
    auto members = NIntrospection::Members(data);
    static_assert(std::is_same_v<std::tuple<char&>, decltype(members)>);
}

TEST(TReflectionMembers, TestMembersConst) {
    NIntrospectionTest::TChar data{'a'};
    auto members = NIntrospection::Members(std::as_const(data));
    static_assert(std::is_same_v<std::tuple<const char&>, decltype(members)>);
}
