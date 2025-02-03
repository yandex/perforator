#include <library/cpp/introspection/hash_ops.h>
#include <library/cpp/introspection/introspection.h>
#include <library/cpp/introspection/tests/data.h>
#include <library/cpp/testing/gtest/gtest.h>


Y_GENERATE_T_HASH_AND_EQUALS(NIntrospectionTest, TChar)
Y_GENERATE_T_HASH_AND_EQUALS(NIntrospectionTest, TCompositePrimitive)


TEST(TReflectionHashOps, TestHashAndEquals) {
    NIntrospectionTest::TCompositePrimitive a{};

    {
        auto tuple = std::make_tuple(a.M1, a.M2, a.M3, a.M4, a.M5, a.M6, a.M7);
        auto manualHash = THash<decltype(tuple)>()(tuple);
        EXPECT_EQ(manualHash, THash<NIntrospectionTest::TCompositePrimitive>()(a));

        auto b = a;
        EXPECT_EQ(true, a == b);
        b.M1 = 'k';
        EXPECT_EQ(false, a == b);
    }

    a.M1 = 'f';
    a.M3 = 9.9;

    {
        auto tuple = std::make_tuple(a.M1, a.M2, a.M3, a.M4, a.M5, a.M6, a.M7);
        auto manualHash = THash<decltype(tuple)>()(tuple);
        EXPECT_EQ(manualHash, THash<NIntrospectionTest::TCompositePrimitive>()(a));

        auto b = a;
        EXPECT_EQ(true, a == b);
        b.M7.M1 = 'p';
        EXPECT_EQ(false, a == b);
    }

    a.M2 = 55;
    a.M4 = "even tstring";

    {
        auto tuple = std::make_tuple(a.M1, a.M2, a.M3, a.M4, a.M5, a.M6, a.M7);
        auto manualHash = THash<decltype(tuple)>()(tuple);
        EXPECT_EQ(manualHash, THash<NIntrospectionTest::TCompositePrimitive>()(a));

        auto b = a;
        EXPECT_EQ(true, a == b);
        b.M4 += "hello there";
        EXPECT_EQ(false, a == b);
    }

    a.M5 = -7;
    a.M6 = 1.3333;
    a.M7.M1 = 'z';

    {
        auto tuple = std::make_tuple(a.M1, a.M2, a.M3, a.M4, a.M5, a.M6, a.M7);
        auto manualHash = THash<decltype(tuple)>()(tuple);
        EXPECT_EQ(manualHash, THash<NIntrospectionTest::TCompositePrimitive>()(a));

        auto b = a;
        EXPECT_EQ(true, a == b);
        b.M2 = 900;
        EXPECT_EQ(false, a == b);
    }
}
