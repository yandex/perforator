#include <library/cpp/introspection/introspection.h>
#include <library/cpp/introspection/tests/data.h>
#include <library/cpp/testing/gtest/gtest.h>

TEST(TReflectionMembersCount,TestBasic) {
    EXPECT_EQ(0u, NIntrospection::MembersCount<NIntrospectionTest::TEmpty>());
    EXPECT_EQ(1u, NIntrospection::MembersCount<NIntrospectionTest::TChar>());
    EXPECT_EQ(7u, NIntrospection::MembersCount<NIntrospectionTest::TComposite7>());
    EXPECT_EQ(100u, NIntrospection::MembersCount<NIntrospectionTest::THuge>());
}
