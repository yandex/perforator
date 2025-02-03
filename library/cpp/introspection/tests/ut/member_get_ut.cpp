#include <library/cpp/introspection/introspection.h>
#include <library/cpp/introspection/tests/data.h>
#include <library/cpp/testing/gtest/gtest.h>


TEST(TReflectionMembersGet, TestOne) {
    NIntrospectionTest::TChar data{'a'};
    EXPECT_EQ(1u, std::tuple_size_v<std::decay_t<decltype(NIntrospection::Members(data))>>);
    EXPECT_EQ(data.M1, NIntrospection::Member<0>(data));
    data.M1 = 'e';
    EXPECT_EQ(data.M1, NIntrospection::Member<0>(data));
}

TEST(TReflectionMembersGet, TestComposite) {
    NIntrospectionTest::TComposite7 data{'1', 2, 3.0, {}, {'1', '2', '3'}, -42, {'7'}};
    EXPECT_EQ(7u, std::tuple_size_v<std::decay_t<decltype(NIntrospection::Members(data))>>);
    EXPECT_EQ(data.M1, NIntrospection::Member<0>(data));
    EXPECT_EQ(data.M2, NIntrospection::Member<1>(data));
    EXPECT_EQ(data.M3, NIntrospection::Member<2>(data));
    EXPECT_EQ(data.M4, NIntrospection::Member<3>(data));
    EXPECT_EQ(&data.M4, &NIntrospection::Member<3>(data));
    EXPECT_EQ(data.M5.M1[0], NIntrospection::Member<4>(data).M1[0]);
    EXPECT_EQ(data.M6, NIntrospection::Member<5>(data));
    EXPECT_EQ(data.M7.M1, NIntrospection::Member<6>(data).M1);

    data.M5.M1[0] = 'a';
    EXPECT_EQ(data.M5.M1[0], NIntrospection::Member<4>(data).M1[0]);
}

TEST(TReflectionMembersGet, TestHuge) {
    NIntrospectionTest::THuge data = {};
    EXPECT_EQ(100u, std::tuple_size_v<std::decay_t<decltype(NIntrospection::Members(data))>>);
    EXPECT_EQ(data.M100, NIntrospection::Member<99>(data));
    data.M100 = 1ull;
    EXPECT_EQ(data.M100, NIntrospection::Member<99>(data));
}

TEST(TReflectionMembersGet, TestGetReference) {
    NIntrospectionTest::TChar data{'a'};
    NIntrospection::Member<0>(data) = 'b';
    EXPECT_EQ('b', data.M1);

    EXPECT_EQ('a', NIntrospection::Member<0>(NIntrospectionTest::TChar{'a'}));
}
