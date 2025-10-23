#include <perforator/lib/demangle/rustc.h>
#include <perforator/lib/demangle/demangle.h>

#include <library/cpp/testing/gtest/gtest.h>

TEST(Demangle, RustcCleanup) {
#define CHECK(mangled, demangled) \
    EXPECT_EQ(NPerforator::NDemangle::MaybePostprocessLegacyRustSymbol(mangled), demangled)

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

TEST(Demangle, Simple) {
    auto check = [](std::string mangled, std::string expected) {
        EXPECT_EQ(expected, NPerforator::NDemangle::Demangle(std::move(mangled)));
    };

    check("_Z3foov", "foo()");
    check("_ZN4llvm3foo3bar3bazEv", "llvm::foo::bar::baz()");
    check("_ZN4llvm4llvmEid", "llvm::llvm(int, double)");
    check("_ZN4llvm4llvmEid.llvm.123456", "llvm::llvm(int, double)");
    check("ZSTD_decodeLiteralsBlock.llvm.8240084484405978173", "ZSTD_decodeLiteralsBlock");

    check(
        "_ZN4core3ptr101drop_in_place$LT$dynamo_llm..http..service..openai..chat_completions..$u7b$$u7b$closure$u7d$$u7d$$GT$17h3357c529a15f68fcE.llvm.13108998173474012399",
        "core::ptr::drop_in_place<dynamo_llm::http::service::openai::chat_completions::{{closure}}>"
    );
    check(
        "_ZL31basic_lookup_transparent_type_1P7objfileiPKc.isra.14",
        "basic_lookup_transparent_type_1(objfile*, int, char const*)"
    );
    check(
        "_ZL18read_signed_leb128P3bfdPKhPj.isra.14",
        "read_signed_leb128(bfd*, unsigned char const*, unsigned int*)"
    );
    check(
        "_ZNK2lm11TSmallArrayImE3endEv",
        "lm::TSmallArray<unsigned long>::end() const"
    );
    check(
        "tcp_collapse",
        "tcp_collapse"
    );
    check(
        "_ZN4core3ptr1494drop_in_place$LT$$LT$dynamo_runtime..pipeline..nodes..PipelineOperatorBackwardEdge$LT$dynamo_runtime..pipeline..context..Context$LT$dynamo_llm..protocols..common..preprocessor..PreprocessedRequest$GT$$C$core..pin..Pin$LT$alloc..boxed..Box$LT$dyn$u20$dynamo_runtime..engine..AsyncEngineStream$LT$dynamo_runtime..protocols..annotated..Annotated$LT$dynamo_llm..protocols..common..llm_backend..BackendOutput$GT$$GT$$u2b$Item$u20$$u3d$$u20$dynamo_runtime..protocols..annotated..Annotated$LT$dynamo_llm..protocols..common..llm_backend..BackendOutput$GT$$GT$$GT$$C$dynamo_runtime..pipeline..context..Context$LT$dynamo_llm..protocols..common..preprocessor..PreprocessedRequest$GT$$C$core..pin..Pin$LT$alloc..boxed..Box$LT$dyn$u20$dynamo_runtime..engine..AsyncEngineStream$LT$dynamo_runtime..protocols..annotated..Annotated$LT$dynamo_llm..protocols..common..llm_backend..LLMEngineOutput$GT$$GT$$u2b$Item$u20$$u3d$$u20$dynamo_runtime..protocols..annotated..Annotated$LT$dynamo_llm..protocols..common..llm_backend..LLMEngineOutput$GT$$GT$$GT$$GT$$u20$as$u20$dynamo_runtime..pipeline..nodes..Sink$LT$core..pin..Pin$LT$alloc..boxed..Box$LT$dyn$u20$dynamo_runtime..engine..AsyncEngineStream$LT$dynamo_runtime..protocols..annotated..Annotated$LT$dynamo_llm..protocols..common..llm_backend..LLMEngineOutput$GT$$GT$$u2b$Item$u20$$u3d$$u20$dynamo_runtime..protocols..annotated..Annotated$LT$dynamo_llm..protocols..common..llm_backend..LLMEngineOutput$GT$$GT$$GT$$GT$$GT$..on_data..$u7b$$u7b$closure$u7d$$u7d$$GT$17hb60ab5c70e57c846E",
        "core::ptr::drop_in_place<<dynamo_runtime::pipeline::nodes::PipelineOperatorBackwardEdge<dynamo_runtime::pipeline::context::Context<dynamo_llm::protocols::common::preprocessor::PreprocessedRequest>,core::pin::Pin<alloc::boxed::Box<dyn dynamo_runtime::engine::AsyncEngineStream<dynamo_runtime::protocols::annotated::Annotated<dynamo_llm::protocols::common::llm_backend::BackendOutput>>+Item = dynamo_runtime::protocols::annotated::Annotated<dynamo_llm::protocols::common::llm_backend::BackendOutput>>>,dynamo_runtime::pipeline::context::Context<dynamo_llm::protocols::common::preprocessor::PreprocessedRequest>,core::pin::Pin<alloc::boxed::Box<dyn dynamo_runtime::engine::AsyncEngineStream<dynamo_runtime::protocols::annotated::Annotated<dynamo_llm::protocols::common::llm_backend::LLMEngineOutput>>+Item = dynamo_runtime::protocols::annotated::Annotated<dynamo_llm::protocols::common::llm_backend::LLMEngineOutput>>>> as dynamo_runtime::pipeline::nodes::Sink<core::pin::Pin<alloc::boxed::Box<dyn dynamo_runtime::engine::AsyncEngineStream<dynamo_runtime::protocols::annotated::Annotated<dynamo_llm::protocols::common::llm_backend::LLMEngineOutput>>+Item = dynamo_runtime::protocols::annotated::Annotated<dynamo_llm::protocols::common::llm_backend::LLMEngineOutput>>>>>::on_data::{{closure}}>"
    );
}
