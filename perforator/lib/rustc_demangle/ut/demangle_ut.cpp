#include <perforator/lib/rustc_demangle/demangle.h>

#include <library/cpp/testing/gtest/gtest.h>

TEST(RustcDemangle, Simple) {
#define CHECK(mangled, demangled) \
    EXPECT_EQ(NPerforator::NDemangle::MaybeDemangleRustcName(mangled), demangled)

    CHECK("foo", "foo");
    CHECK("", "");
    CHECK("llvm::foo:bar::baz", "llvm::foo:bar::baz");

    CHECK("example::$u4f60$$u597d$::h3a915f2466537b48", "example::你好");
    CHECK("example::$u41f$$u440$$u438$$u432$$u435$$u442$::h589aaa819a611201", "example::Привет");
    CHECK("example::$u4f60$$u597d$::hZZZZZZZZZZZZZZZZ", "example::$u4f60$$u597d$::hZZZZZZZZZZZZZZZZ");
    CHECK("rayon::iter::plumbing::bridge_producer_consumer::helper::h9683a869826bdc08", "rayon::iter::plumbing::bridge_producer_consumer::helper");
    CHECK("rayon_core::join::join_context::_$u7b$$u7b$closure$u7d$$u7d$::h8d4829d020869bab", "rayon_core::join::join_context::_{{closure}}");

#undef CHECK
}
