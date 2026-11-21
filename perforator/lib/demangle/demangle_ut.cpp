#include <perforator/lib/demangle/rustc.h>
#include <perforator/lib/demangle/demangle.h>

#include <library/cpp/testing/gtest/gtest.h>

#include <string_view>
#include <utility>

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
        "core::ptr::drop_in_place<dynamo_llm::http::service::openai::chat_completions::{{closure}}>");
    check(
        "_ZL31basic_lookup_transparent_type_1P7objfileiPKc.isra.14",
        "basic_lookup_transparent_type_1(objfile*, int, char const*)");
    check(
        "_ZL18read_signed_leb128P3bfdPKhPj.isra.14",
        "read_signed_leb128(bfd*, unsigned char const*, unsigned int*)");
    check(
        "_ZNK2lm11TSmallArrayImE3endEv",
        "lm::TSmallArray<unsigned long>::end() const");
    check(
        "tcp_collapse",
        "tcp_collapse");
    check(
        "_ZN4core3ptr1494drop_in_place$LT$$LT$dynamo_runtime..pipeline..nodes..PipelineOperatorBackwardEdge$LT$dynamo_runtime..pipeline..context..Context$LT$dynamo_llm..protocols..common..preprocessor..PreprocessedRequest$GT$$C$core..pin..Pin$LT$alloc..boxed..Box$LT$dyn$u20$dynamo_runtime..engine..AsyncEngineStream$LT$dynamo_runtime..protocols..annotated..Annotated$LT$dynamo_llm..protocols..common..llm_backend..BackendOutput$GT$$GT$$u2b$Item$u20$$u3d$$u20$dynamo_runtime..protocols..annotated..Annotated$LT$dynamo_llm..protocols..common..llm_backend..BackendOutput$GT$$GT$$GT$$C$dynamo_runtime..pipeline..context..Context$LT$dynamo_llm..protocols..common..preprocessor..PreprocessedRequest$GT$$C$core..pin..Pin$LT$alloc..boxed..Box$LT$dyn$u20$dynamo_runtime..engine..AsyncEngineStream$LT$dynamo_runtime..protocols..annotated..Annotated$LT$dynamo_llm..protocols..common..llm_backend..LLMEngineOutput$GT$$GT$$u2b$Item$u20$$u3d$$u20$dynamo_runtime..protocols..annotated..Annotated$LT$dynamo_llm..protocols..common..llm_backend..LLMEngineOutput$GT$$GT$$GT$$GT$$u20$as$u20$dynamo_runtime..pipeline..nodes..Sink$LT$core..pin..Pin$LT$alloc..boxed..Box$LT$dyn$u20$dynamo_runtime..engine..AsyncEngineStream$LT$dynamo_runtime..protocols..annotated..Annotated$LT$dynamo_llm..protocols..common..llm_backend..LLMEngineOutput$GT$$GT$$u2b$Item$u20$$u3d$$u20$dynamo_runtime..protocols..annotated..Annotated$LT$dynamo_llm..protocols..common..llm_backend..LLMEngineOutput$GT$$GT$$GT$$GT$$GT$..on_data..$u7b$$u7b$closure$u7d$$u7d$$GT$17hb60ab5c70e57c846E",
        "core::ptr::drop_in_place<<dynamo_runtime::pipeline::nodes::PipelineOperatorBackwardEdge<dynamo_runtime::pipeline::context::Context<dynamo_llm::protocols::common::preprocessor::PreprocessedRequest>,core::pin::Pin<alloc::boxed::Box<dyn dynamo_runtime::engine::AsyncEngineStream<dynamo_runtime::protocols::annotated::Annotated<dynamo_llm::protocols::common::llm_backend::BackendOutput>>+Item = dynamo_runtime::protocols::annotated::Annotated<dynamo_llm::protocols::common::llm_backend::BackendOutput>>>,dynamo_runtime::pipeline::context::Context<dynamo_llm::protocols::common::preprocessor::PreprocessedRequest>,core::pin::Pin<alloc::boxed::Box<dyn dynamo_runtime::engine::AsyncEngineStream<dynamo_runtime::protocols::annotated::Annotated<dynamo_llm::protocols::common::llm_backend::LLMEngineOutput>>+Item = dynamo_runtime::protocols::annotated::Annotated<dynamo_llm::protocols::common::llm_backend::LLMEngineOutput>>>> as dynamo_runtime::pipeline::nodes::Sink<core::pin::Pin<alloc::boxed::Box<dyn dynamo_runtime::engine::AsyncEngineStream<dynamo_runtime::protocols::annotated::Annotated<dynamo_llm::protocols::common::llm_backend::LLMEngineOutput>>+Item = dynamo_runtime::protocols::annotated::Annotated<dynamo_llm::protocols::common::llm_backend::LLMEngineOutput>>>>>::on_data::{{closure}}>");
}

TEST(Demangle, Cython) {
    auto check = [](std::string mangled, std::string expected) {
        EXPECT_EQ(expected, NPerforator::NDemangle::Demangle(std::move(mangled)));
    };
    check("__pyx_f_6mtrand_cont0_array", "cont0_array");
    check("__pyx_fuse_0__pyx_f_8implicit_4_als_axpy", "axpy");
    check("__pyx_fuse_1__pyx_f_8implicit_3bpr_has_non_zero", "has_non_zero");
    check("___pyx_pw_4test_1foo", "foo");
    check("__pyx_pf_4test_foo", "foo");
    check("___pyx_pw_4demo_1foo", "foo");
    check("___pyx_pf_4demo_foo", "foo");
    check("__pyx_f_4test_4Test_foo", "foo");
    for (const auto* name : {"__Pyx_PyObject_Call", "__pyx_f", "__pyx_f_", "__pyx_f_0foo",
                             "__pyx_f_6uvloop_1", "__pyx_f_3abcdef", "__pyx_f_999999999999999999999999999999foo"}) {
        check(name, name);
    }
}

TEST(Demangle, CythonLowDiscriminators) {
    auto check = [](std::string mangled, std::string expected) {
        EXPECT_EQ(expected, NPerforator::NDemangle::Demangle(std::move(mangled)));
    };
    check("__pyx_pw_4test_1foo", "foo");
    check("__pyx_pf_4test_1long_function", "long_function");
    check("__pyx_pw_4test_1_long_function", "_long_function");
    check("__pyx_pw_4test_1a_01b", "a_01b");
    check("__pyx_pw_4test_1a_18446744073709551616b", "a_18446744073709551616b");
    check("__pyx_pw_4test_1a_", "a_");

    for (const auto* name : {"__pyx_pw_4test_1", "__pyx_pf_4test_0foo",
                             "__pyx_pw_4test_999999999999999999999999999999foo"}) {
        check(name, name);
    }
}

TEST(Demangle, CythonVendorSuffix) {
    for (const auto* suffix : {".llvm.abc", ".llvm.123456789", ".isra.0", ".isra.", ".llvm.abc.isra.0"}) {
        const std::string name = "__pyx_pw_4test_1foo" + std::string{suffix};
        EXPECT_EQ("foo", NPerforator::NDemangle::Demangle(name));
        EXPECT_EQ("foo" + std::string{suffix},
                  NPerforator::NDemangle::Demangle(name, {.DropVendorSpecificSuffix = false}));
        const std::string ambiguous = "__pyx_pw_4test_4Test_5foo" + std::string{suffix};
        EXPECT_EQ("__pyx_pw_4test_4Test_5foo", NPerforator::NDemangle::Demangle(ambiguous));
        EXPECT_EQ(ambiguous, NPerforator::NDemangle::Demangle(ambiguous, {.DropVendorSpecificSuffix = false}));
    }
}

TEST(Demangle, CythonAmbiguousQualification) {
    // For example, a_21b and a.b can both produce __pyx_pw_4test_1a_21b.
    for (const auto* name : {"__pyx_pw_4test_1a_1foo", "__pyx_pw_4test_1a_1bar",
                             "__pyx_pw_4test_2ab_1foo", "__pyx_pf_4test_8My_Class_1long_function",
                             "__pyx_pw_8my_tests_9_My_Class_1_long_function",
                             "__pyx_pw_4test_1a_20b", "__pyx_pw_4test_1a_21b",
                             "__pyx_pw_4test_4Test_5foo",
                             "__pyx_pw_6uvloop_4loop_4Loop_45run_until_complete",
                             "__pyx_pf_6uvloop_4loop_4Loop_44run_until_complete",
                             "__pyx_pf_8implicit_4_als_30_least_squares_cg",
                             "__pyx_fuse_1_0__pyx_pw_8implicit_4_als_31_least_squares_cg",
                             "___pyx_pw_6uvloop_4loop_4Loop_45run_until_complete"}) {
        EXPECT_EQ(name, NPerforator::NDemangle::Demangle(name));
    }
}

TEST(Demangle, CythonMalformedNames) {
    for (const auto* name : {"__pyx_pw_04test_1foo", "__pyx_pw_4test_01foo",
                             "__pyx_pw_4test_1foo!", "__pyx_pw_4test_1foo.llvm.",
                             "__pyx_pw_4test_1foo.llvm.abcXYZ", "__pyx_pw_4test_1foo.unknown.0",
                             "__pyx_pw_4test_1foo.llvm.abc.llvm.def", "__pyx_fuse_02__pyx_f_4test_foo"}) {
        EXPECT_EQ(name, NPerforator::NDemangle::Demangle(name, {.DropVendorSpecificSuffix = false}));
    }
    EXPECT_EQ("__pyx_pw_4test_1fooXYZ", NPerforator::NDemangle::Demangle("__pyx_pw_4test_1foo.llvm.abcXYZ"));
    EXPECT_EQ("plainXYZ", NPerforator::NDemangle::Demangle("plain.llvm.abcXYZ"));
}

TEST(Demangle, CythonFusedAndCpp) {
    using NPerforator::NDemangle::Demangle;
    EXPECT_EQ("foo", Demangle("__pyx_fuse_10_0_13__pyx_f_4test_foo"));
    EXPECT_EQ("__pyx_fuse_01__pyx_f_4test_foo", Demangle("__pyx_fuse_01__pyx_f_4test_foo"));
    EXPECT_EQ("__pyx_fuse_a__pyx_f_4test_foo", Demangle("__pyx_fuse_a__pyx_f_4test_foo"));
    EXPECT_EQ("foo(int)", Demangle("_ZL19__pyx_pw_4test_1fooi"));
    EXPECT_EQ("__pyx_pw_4test_1a_21b(int)", Demangle("_ZL21__pyx_pw_4test_1a_21bi"));
}

TEST(Demangle, CythonSlotsAndProperties) {
    using NPerforator::NDemangle::Demangle;
    EXPECT_EQ("test.C [tp_dealloc]", Demangle("__pyx_tp_dealloc_4test_C"));
    EXPECT_EQ("test.a_dot_b [tp_new]", Demangle("__pyx_tp_new_4test_a_dot_b"));
    EXPECT_EQ("test.a_dot_b.p_dot_q [get]", Demangle("__pyx_getprop_4test_7a_dot_b_p_dot_q"));
    EXPECT_EQ("test.C [mp_ass_subscript]", Demangle("__pyx_mp_ass_subscript_4test_C"));
    EXPECT_EQ("features.C [tp_new__initialisation]", Demangle("__pyx_tp_new__initialisation_8features_C"));
    EXPECT_EQ("test.C [nb_inplace_add]", Demangle("__pyx_nb_inplace_add_4test_C"));
    EXPECT_EQ("test.C.prop [get]", Demangle("__pyx_getprop_4test_1C_prop"));
    EXPECT_EQ("test.C.prop [set]", Demangle("__pyx_setprop_4test_1C_prop"));
    EXPECT_EQ("__pyx_tp_dealloc_4test_1a_21b", Demangle("__pyx_tp_dealloc_4test_1a_21b"));
    EXPECT_EQ("test.C [tp_new].llvm.abc", Demangle("__pyx_tp_new_4test_C.llvm.abc", {.DropVendorSpecificSuffix = false}));
    EXPECT_EQ("__pyx_tp_new_04test_C", Demangle("__pyx_tp_new_04test_C"));
}

TEST(Demangle, CythonGeneratorBody) {
    using NPerforator::NDemangle::Demangle;
    EXPECT_EQ("features.6generator [generator body]", Demangle("__pyx_gb_8features_6generator"));
    EXPECT_EQ("features.12generator2 [generator body]", Demangle("__pyx_gb_8features_12generator2"));
    EXPECT_EQ("__pyx_gb_8features_generator", Demangle("__pyx_gb_8features_generator"));
}

TEST(Demangle, CythonSpecialCallables) {
    using NPerforator::NDemangle::Demangle;
    EXPECT_EQ("foo", Demangle("__pyx_specialmethod___pyx_pw_4test_1foo"));
    EXPECT_EQ("__pyx_specialmethod___pyx_pw_4test_1a_21b", Demangle("__pyx_specialmethod___pyx_pw_4test_1a_21b"));
    EXPECT_EQ("cygrpc [module exec]", Demangle("__pyx_pymod_exec_cygrpc"));
    EXPECT_EQ("features.Pair [from_py]", Demangle("__pyx_convert__from_py_struct____pyx_t_8features_Pair"));
    EXPECT_EQ("features.Pair [to_py]", Demangle("__pyx_convert__to_py_struct____pyx_t_8features_Pair"));
    EXPECT_EQ("lambda [lambda]", Demangle("__pyx_lambda_funcdef_lambda"));
    EXPECT_EQ("__pyx_ff_match_signatures", Demangle("__pyx_ff_match_signatures"));
    EXPECT_EQ("café", Demangle("__pyx_Upw_8features_18caf_yfc"));
    EXPECT_EQ("café", Demangle("___pyx_Upw_8features_18caf_yfc"));
    EXPECT_EQ("features.Ω [tp_new]", Demangle("__pyx_Utp_new_8features__o4p"));
    EXPECT_EQ("__pyx_Ubroken", Demangle("__pyx_Ubroken"));
    EXPECT_EQ("__pyx_Upw_4test_1_ox59o", Demangle("__pyx_Upw_4test_1_ox59o"));
    EXPECT_EQ("__pyx_Upw_4test_1ab_c29j", Demangle("__pyx_Upw_4test_1ab_c29j"));
    EXPECT_EQ("__pyx_Upw_4test_1a_21_u2b", Demangle("__pyx_Upw_4test_1a_21_u2b"));
}

TEST(Demangle, CythonUnicodeLengthLimit) {
    const auto name = "__pyx_Upw_4test_1" + std::string(1000, 'a') + "_b91g";
    EXPECT_EQ(name, NPerforator::NDemangle::Demangle(name));

    const auto atLimit = "__pyx_U" + std::string(1994, 'a');
    const auto overLimit = atLimit + "a";
    ASSERT_EQ(2001, atLimit.size());
    ASSERT_EQ(2002, overLimit.size());
    EXPECT_EQ(atLimit, NPerforator::NDemangle::Demangle(atLimit));
    EXPECT_EQ(overLimit, NPerforator::NDemangle::Demangle(overLimit));
    EXPECT_EQ("_" + overLimit, NPerforator::NDemangle::Demangle("_" + overLimit));
}

TEST(Demangle, CythonLambdaScopes) {
    using NPerforator::NDemangle::Demangle;
    // Module and function-local lambdas have different Cython scope encodings.
    EXPECT_EQ("test.lambda [lambda]", Demangle("__pyx_lambda_funcdef_4test_lambda"));
    EXPECT_EQ("lambda [lambda]", Demangle("__pyx_lambda_funcdef_lambda"));
    EXPECT_EQ("__pyx_lambda_funcdef_04test_lambda", Demangle("__pyx_lambda_funcdef_04test_lambda"));
}

TEST(Demangle, CythonCallableFamilySuffixes) {
    using NPerforator::NDemangle::Demangle;
    static constexpr std::pair<std::string_view, std::string_view> cases[] = {
        {"__pyx_pw_4test_1foo", "foo"},
        {"__pyx_pf_4test_foo", "foo"},
        {"__pyx_f_4test_foo", "foo"},
        {"__pyx_fuse_10__pyx_f_5fused_wide", "wide"},
        {"__pyx_tp_dealloc_4lxml_5etree__Document", "lxml.etree._Document [tp_dealloc]"},
        {"__pyx_getprop_4lxml_5etree_8_Element_text", "lxml.etree._Element.text [get]"},
        {"__pyx_setprop_8features_9Extension_prop", "features.Extension.prop [set]"},
        {"__pyx_convert__from_py_struct____pyx_t_8features_Pair", "features.Pair [from_py]"},
        {"__pyx_convert__to_py_struct____pyx_t_8features_Pair", "features.Pair [to_py]"},
        {"__pyx_gb_8features_6generator", "features.6generator [generator body]"},
        {"__pyx_lambda_funcdef_4test_lambda", "test.lambda [lambda]"},
        {"__pyx_lambda_funcdef_lambda", "lambda [lambda]"},
        {"__pyx_pymod_exec_cygrpc", "cygrpc [module exec]"},
        {"__pyx_specialmethod___pyx_pw_4test_1foo", "foo"},
        {"__pyx_Upw_8features_18caf_yfc", "café"},
        {"__pyx_Utp_new_8features__o4p", "features.Ω [tp_new]"},
    };
    for (const auto& [symbol, expected] : cases) {
        for (const auto prefix : {"", "_"}) {
            const auto name = prefix + std::string{symbol};
            SCOPED_TRACE(name);
            EXPECT_EQ(expected, Demangle(name));
            for (const auto suffix : {".llvm.abc", ".isra.0", ".llvm.abc.isra.0"}) {
                const auto decorated = name + suffix;
                EXPECT_EQ(expected, Demangle(decorated));
                EXPECT_EQ(std::string{expected} + suffix, Demangle(decorated, {.DropVendorSpecificSuffix = false}));
            }
        }
    }
}

TEST(Demangle, CythonUnicodeInlineBoundary) {
    EXPECT_EQ(std::string(36, 'a') + "café", NPerforator::NDemangle::Demangle("__pyx_Upw_8features_1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaacaf_ygf"));
    EXPECT_EQ(std::string(37, 'a') + "café", NPerforator::NDemangle::Demangle("__pyx_Upw_8features_1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaacaf_zjf"));
    EXPECT_EQ(std::string(38, 'a') + "café", NPerforator::NDemangle::Demangle("__pyx_Upw_8features_1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaacaf_0mf"));
}

TEST(Demangle, CythonUnicodeCanonicalFastPath) {
    EXPECT_EQ("Café", NPerforator::NDemangle::Demangle("__pyx_Upw_8features_1Caf_xcc"));
    EXPECT_EQ(std::string(39, 'a') + "café", NPerforator::NDemangle::Demangle("__pyx_Upw_8features_1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaacaf_1pf"));
    EXPECT_EQ(std::string(40, 'a') + "café", NPerforator::NDemangle::Demangle("__pyx_Upw_8features_1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaacaf_2sf"));
    EXPECT_EQ(std::string(41, 'a') + "café", NPerforator::NDemangle::Demangle("__pyx_Upw_8features_1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaacaf_3vf"));
    EXPECT_EQ(std::string(80, 'a') + "café", NPerforator::NDemangle::Demangle("__pyx_Upw_8features_1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaacaf_78i"));
    EXPECT_EQ("__pyx_Upw_8features_18caf_YFC", NPerforator::NDemangle::Demangle("__pyx_Upw_8features_18caf_YFC"));
}

TEST(Demangle, CythonMachOPreservesUnsupportedNames) {
    for (const auto* name : {"____pyx_pw_4test_1foo", "___pyx_pw_4test_1a_21b",
                             "___pyx_ff_match_signatures", "___pyx_pw_04test_1foo",
                             "___pyx_fuse_0___pyx_f_4test_foo",
                             "___pyx_specialmethod____pyx_pw_4test_1foo"}) {
        EXPECT_EQ(name, NPerforator::NDemangle::Demangle(name));
    }
}
