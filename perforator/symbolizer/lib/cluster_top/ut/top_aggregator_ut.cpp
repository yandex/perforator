#include <perforator/symbolizer/lib/cluster_top/cluster_top_c.h>
#include <perforator/symbolizer/lib/cluster_top/perf_top_aggregator.h>
#include <perforator/symbolizer/lib/cluster_top/top_aggregator_session.h>

#include <perforator/lib/profile/pprof.h>
#include <perforator/lib/profile/builder.h>
#include <perforator/proto/pprofprofile/profile.pb.h>
#include <perforator/proto/profile/profile.pb.h>

#include <library/cpp/testing/gtest/gtest.h>

#include <unordered_map>

namespace NPerforator::NClusterTop {

namespace {

// Builds a minimal pprof Profile proto with the following stacks:
//
//   Sample 0 (100 cpu/cycles): leaf=funcA, caller=funcB
//   Sample 1 (200 cpu/cycles): leaf=funcA, funcA (recursive), caller=funcB
//   Sample 2 ( 50 cpu/cycles): leaf=kernelFunc  (from "[kernel]" mapping)
//
// Expected aggregation results:
//   TotalCycles = 350
//   SelfCycles:       funcA=300, kernelFunc=50
//   CumulativeCycles: funcA=300, funcB=300, kernelFunc=50
//   KernelFunctions:  kernelFunc -> shown as "[kernel] kernelFunc"
NPerforator::NProto::NPProf::Profile BuildTestProfile() {
    NPerforator::NProto::NPProf::Profile profile;

    // String table (index 0 must be "")
    // 0: ""
    // 1: "cpu"
    // 2: "cycles"
    // 3: "funcA"
    // 4: "funcB"
    // 5: "[kernel]"
    // 6: "kernelFunc"
    // 7: "buildId_user"
    // 8: "buildId_kernel"
    for (const char* s : {"", "cpu", "cycles", "funcA", "funcB", "[kernel]", "kernelFunc", "buildId_user", "buildId_kernel"}) {
        profile.add_string_table(s);
    }

    // Sample type: cpu/cycles
    {
        auto* st = profile.add_sample_type();
        st->set_type(1); // "cpu"
        st->set_unit(2); // "cycles"
    }

    // Mappings
    // id=1: user binary
    {
        auto* m = profile.add_mapping();
        m->set_id(1);
        m->set_memory_start(0x100000);
        m->set_memory_limit(0x200000);
        m->set_file_offset(0);
        m->set_build_id(7); // "buildId_user"
    }
    // id=2: kernel
    {
        auto* m = profile.add_mapping();
        m->set_id(2);
        m->set_memory_start(0xFFFF000000000000ULL);
        m->set_memory_limit(0xFFFFFFFFFFFFFFFFULL);
        m->set_file_offset(0);
        m->set_filename(5); // "[kernel]"
        m->set_build_id(8); // "buildId_kernel"
    }

    // Functions
    // id=1: funcA
    {
        auto* f = profile.add_function();
        f->set_id(1);
        f->set_name(3); // "funcA"
    }
    // id=2: funcB
    {
        auto* f = profile.add_function();
        f->set_id(2);
        f->set_name(4); // "funcB"
    }
    // id=3: kernelFunc
    {
        auto* f = profile.add_function();
        f->set_id(3);
        f->set_name(6); // "kernelFunc"
    }

    // Locations
    // id=1: funcA (user)
    {
        auto* loc = profile.add_location();
        loc->set_id(1);
        loc->set_mapping_id(1);
        loc->set_address(0x110000);
        auto* line = loc->add_line();
        line->set_function_id(1);
    }
    // id=2: funcB (user)
    {
        auto* loc = profile.add_location();
        loc->set_id(2);
        loc->set_mapping_id(1);
        loc->set_address(0x120000);
        auto* line = loc->add_line();
        line->set_function_id(2);
    }
    // id=3: kernelFunc (kernel)
    {
        auto* loc = profile.add_location();
        loc->set_id(3);
        loc->set_mapping_id(2);
        loc->set_address(0xFFFF000000001000ULL);
        auto* line = loc->add_line();
        line->set_function_id(3);
    }

    // Samples
    // Sample 0: stack [funcA leaf, funcB caller], 100 cycles
    {
        auto* s = profile.add_sample();
        s->add_location_id(1); // leaf
        s->add_location_id(2); // caller
        s->add_value(100);
    }
    // Sample 1: stack [funcA leaf, funcA (recursive), funcB caller], 200 cycles
    {
        auto* s = profile.add_sample();
        s->add_location_id(1); // leaf (funcA)
        s->add_location_id(1); // same function again (recursive)
        s->add_location_id(2); // caller (funcB)
        s->add_value(200);
    }
    // Sample 2: kernel stack [kernelFunc leaf], 50 cycles
    {
        auto* s = profile.add_sample();
        s->add_location_id(3);
        s->add_value(50);
    }

    return profile;
}

NPerforator::NProto::NProfile::Profile ConvertPProf(
    const NPerforator::NProto::NPProf::Profile& pprofProfile) {
    NPerforator::NProto::NProfile::Profile profile;
    NProfile::ConvertFromPProf(pprofProfile.SerializeAsStringOrThrow(), &profile);
    return profile;
}

TString BuildYaprofBytes(const NPerforator::NProto::NPProf::Profile& pprofProfile) {
    return ConvertPProf(pprofProfile).SerializeAsStringOrThrow();
}

// Helper: build result map from extracted functions.
std::unordered_map<std::string, std::pair<ui128, ui128>> ToMap(
    const std::vector<TPerfTopAggregator::Function>& functions) {
    std::unordered_map<std::string, std::pair<ui128, ui128>> result;
    for (const auto& f : functions) {
        result[f.Name.c_str()] = {f.SelfCycles, f.CumulativeCycles};
    }
    return result;
}

} // namespace

TEST(TopAggregatorTest, TotalCycles) {
    TPerfTopAggregator agg;

    agg.AddProfile(ConvertPProf(BuildTestProfile()));

    auto top = agg.ExtractEntries();
    EXPECT_EQ(top.TotalCycles, ui128{350});
}

TEST(TopAggregatorTest, SelfCycles) {
    TPerfTopAggregator agg;

    agg.AddProfile(ConvertPProf(BuildTestProfile()));

    auto top = agg.ExtractEntries();
    auto m = ToMap(top.Functions);

    ASSERT_TRUE(m.count("funcA")) << "funcA must appear in top";
    EXPECT_EQ(m["funcA"].first, ui128{300}) << "funcA self cycles";

    // funcB has no self cycles (it's always a caller, never a leaf)
    if (m.count("funcB")) {
        EXPECT_EQ(m["funcB"].first, ui128{0}) << "funcB self cycles must be 0";
    }
}

TEST(TopAggregatorTest, CumulativeCyclesNoDoubleCounting) {
    TPerfTopAggregator agg;

    agg.AddProfile(ConvertPProf(BuildTestProfile()));

    auto top = agg.ExtractEntries();
    auto m = ToMap(top.Functions);

    // funcA appears in all 3 samples (and twice in sample 1, but should not be double-counted)
    ASSERT_TRUE(m.count("funcA")) << "funcA must appear in top";
    EXPECT_EQ(m["funcA"].second, ui128{300}) << "funcA cumulative cycles";

    // funcB appears in samples 0 and 1
    ASSERT_TRUE(m.count("funcB")) << "funcB must appear in top";
    EXPECT_EQ(m["funcB"].second, ui128{300}) << "funcB cumulative cycles";
}

TEST(TopAggregatorTest, KernelPrefix) {
    TPerfTopAggregator agg;

    agg.AddProfile(ConvertPProf(BuildTestProfile()));

    auto top = agg.ExtractEntries();
    auto m = ToMap(top.Functions);

    const std::string kernelEntry = "[kernel] kernelFunc";
    ASSERT_TRUE(m.count(kernelEntry)) << "kernel function must appear with [kernel] prefix";
    EXPECT_EQ(m[kernelEntry].first, ui128{50}) << "kernelFunc self cycles";
    EXPECT_EQ(m[kernelEntry].second, ui128{50}) << "kernelFunc cumulative cycles";

    // The bare "kernelFunc" must not appear without the prefix
    EXPECT_FALSE(m.count("kernelFunc")) << "bare kernelFunc must not appear";
}

TEST(TopAggregatorTest, MergeAggregators) {
    const auto& parsedProfile = ConvertPProf(BuildTestProfile());

    TPerfTopAggregator agg1;
    agg1.AddProfile(parsedProfile);

    TPerfTopAggregator agg2;
    agg2.AddProfile(parsedProfile);

    agg1.MergeAggregator(agg2);

    auto top = agg1.ExtractEntries();
    EXPECT_EQ(top.TotalCycles, ui128{700});

    auto m = ToMap(top.Functions);
    EXPECT_EQ(m["funcA"].first, ui128{600}) << "funcA self cycles after merge";
    EXPECT_EQ(m["funcA"].second, ui128{600}) << "funcA cumulative cycles after merge";
}

TEST(TopAggregatorTest, EmptyYaprofProfileSkipped) {
    TTopAggregatorSession session;

    session.AddYaprofBytes({});

    auto top = session.ExtractEntries();
    EXPECT_EQ(top.TotalCycles, ui128{0});
    EXPECT_TRUE(top.Functions.empty());
}

TEST(TopAggregatorTest, YaprofBytesPath) {
    const auto yaprofBytes = BuildYaprofBytes(BuildTestProfile());

    TTopAggregatorSession session;
    session.AddYaprofBytes({yaprofBytes.data(), yaprofBytes.size()});

    auto top = session.ExtractEntries();
    EXPECT_EQ(top.TotalCycles, ui128{350});
}

// Verify that a Line with function_id == 0 in pprof gets aggregated as
// "<UNKNOWN (No function)>" — same semantics as the old ProfileLight code.
TEST(TopAggregatorTest, LineWithZeroFunctionId) {
    NPerforator::NProto::NPProf::Profile profile;

    for (const char* s : {"", "cpu", "cycles", "userBuild"}) {
        profile.add_string_table(s);
    }

    auto* st = profile.add_sample_type();
    st->set_type(1);
    st->set_unit(2);

    auto* m = profile.add_mapping();
    m->set_id(1);
    m->set_memory_start(0x100000);
    m->set_memory_limit(0x200000);
    m->set_build_id(3);

    // Location with one line whose function_id is 0 (i.e. "no function").
    auto* loc = profile.add_location();
    loc->set_id(1);
    loc->set_mapping_id(1);
    loc->set_address(0x110000);
    auto* line = loc->add_line();
    line->set_function_id(0);

    auto* s = profile.add_sample();
    s->add_location_id(1);
    s->add_value(100);

    TPerfTopAggregator agg;
    agg.AddProfile(ConvertPProf(profile));

    auto top = agg.ExtractEntries();
    auto m_top = ToMap(top.Functions);

    EXPECT_EQ(top.TotalCycles, ui128{100});
    ASSERT_TRUE(m_top.count("<UNKNOWN (No function)>"))
        << "function_id == 0 must yield <UNKNOWN (No function)>";
    EXPECT_EQ(m_top["<UNKNOWN (No function)>"].first, ui128{100});
}

// Verify that a Location without a mapping yields "<UNKNOWN (No mapping)>".
TEST(TopAggregatorTest, LocationWithoutMappingNorLines) {
    NPerforator::NProto::NPProf::Profile profile;

    for (const char* s : {"", "cpu", "cycles"}) {
        profile.add_string_table(s);
    }

    auto* st = profile.add_sample_type();
    st->set_type(1);
    st->set_unit(2);

    // Location with no mapping_id and no lines.
    auto* loc = profile.add_location();
    loc->set_id(1);
    loc->set_address(0x110000);

    auto* s = profile.add_sample();
    s->add_location_id(1);
    s->add_value(42);

    TPerfTopAggregator agg;
    agg.AddProfile(ConvertPProf(profile));

    auto top = agg.ExtractEntries();
    auto m_top = ToMap(top.Functions);

    EXPECT_EQ(top.TotalCycles, ui128{42});
    ASSERT_TRUE(m_top.count("<UNKNOWN (No mapping)>"));
}

// Verify that a Location with a mapping but no GSYM (and no lines) yields
// "<UNKNOWN (No GSYM)>" — buildId is present but no symbolizer was registered.
TEST(TopAggregatorTest, MappingWithoutGSYM) {
    NPerforator::NProto::NPProf::Profile profile;

    for (const char* s : {"", "cpu", "cycles", "buildIdNoGsym"}) {
        profile.add_string_table(s);
    }

    auto* st = profile.add_sample_type();
    st->set_type(1);
    st->set_unit(2);

    auto* m = profile.add_mapping();
    m->set_id(1);
    m->set_memory_start(0x100000);
    m->set_memory_limit(0x200000);
    m->set_build_id(3);

    auto* loc = profile.add_location();
    loc->set_id(1);
    loc->set_mapping_id(1);
    loc->set_address(0x110000);
    // No lines: requires GSYM symbolization.

    auto* s = profile.add_sample();
    s->add_location_id(1);
    s->add_value(7);

    TPerfTopAggregator agg;
    agg.AddProfile(ConvertPProf(profile));

    auto top = agg.ExtractEntries();
    auto m_top = ToMap(top.Functions);

    EXPECT_EQ(top.TotalCycles, ui128{7});
    ASSERT_TRUE(m_top.count("<UNKNOWN (No GSYM)>"));
}

// Verify that a profile without cpu/cycles sample type is silently skipped
// and produces no output.
TEST(TopAggregatorTest, NoCpuCyclesSampleType) {
    NPerforator::NProto::NPProf::Profile profile;

    for (const char* s : {"", "wall", "nanoseconds"}) {
        profile.add_string_table(s);
    }

    auto* st = profile.add_sample_type();
    st->set_type(1); // "wall"
    st->set_unit(2); // "nanoseconds"

    auto* loc = profile.add_location();
    loc->set_id(1);
    loc->set_address(0x100);

    auto* s = profile.add_sample();
    s->add_location_id(1);
    s->add_value(99999);

    TPerfTopAggregator agg;
    agg.AddProfile(ConvertPProf(profile));

    auto top = agg.ExtractEntries();
    EXPECT_EQ(top.TotalCycles, ui128{0});
    EXPECT_TRUE(top.Functions.empty());
}

// Self cycles belong to the innermost inline function at the leaf address.
TEST(TopAggregatorTest, InlineChainSelfAttribution) {
    NPerforator::NProto::NPProf::Profile profile;

    // 0:"", 1:"cpu", 2:"cycles", 3:"inlinedLeaf", 4:"outerCaller", 5:"buildId"
    for (const char* s : {"", "cpu", "cycles", "inlinedLeaf", "outerCaller", "buildId"}) {
        profile.add_string_table(s);
    }

    auto* st = profile.add_sample_type();
    st->set_type(1);
    st->set_unit(2);

    auto* m = profile.add_mapping();
    m->set_id(1);
    m->set_memory_start(0x100000);
    m->set_memory_limit(0x200000);
    m->set_build_id(5);

    auto* fInlined = profile.add_function();
    fInlined->set_id(1);
    fInlined->set_name(3); // "inlinedLeaf"

    auto* fOuter = profile.add_function();
    fOuter->set_id(2);
    fOuter->set_name(4); // "outerCaller"

    // Location with inline chain: leaf-to-root order.
    // [0] = inlinedLeaf (most-inlined), [1] = outerCaller (root).
    auto* loc = profile.add_location();
    loc->set_id(1);
    loc->set_mapping_id(1);
    loc->set_address(0x110000);
    loc->add_line()->set_function_id(1); // inlinedLeaf
    loc->add_line()->set_function_id(2); // outerCaller (back -> outermost)

    auto* s = profile.add_sample();
    s->add_location_id(1);
    s->add_value(123);

    TPerfTopAggregator agg;
    agg.AddProfile(ConvertPProf(profile));

    auto top = agg.ExtractEntries();
    auto m_top = ToMap(top.Functions);

    EXPECT_EQ(top.TotalCycles, ui128{123});

    // Both functions are on the stack, but only the inline leaf is executing.
    ASSERT_TRUE(m_top.count("outerCaller"));
    EXPECT_EQ(m_top["outerCaller"].first, ui128{0});

    // Both inlined functions get cumulative cycles (each appears once in the chain).
    EXPECT_EQ(m_top["outerCaller"].second, ui128{123});
    ASSERT_TRUE(m_top.count("inlinedLeaf"));
    EXPECT_EQ(m_top["inlinedLeaf"].first, ui128{123});
    EXPECT_EQ(m_top["inlinedLeaf"].second, ui128{123});
}

TEST(TopAggregatorTest, ReusesProfileStorageWithoutLosingAggregatedNames) {
    const auto yaprof = BuildYaprofBytes(BuildTestProfile());
    auto otherProfile = BuildTestProfile();
    otherProfile.set_string_table(3, "replacementFunc");
    const auto other = BuildYaprofBytes(otherProfile);
    TTopAggregatorSession session;
    session.AddYaprofBytes({yaprof.data(), yaprof.size()});
    session.AddYaprofBytes({other.data(), other.size()});
    session.AddYaprofBytes({yaprof.data(), yaprof.size()});
    const auto top = session.ExtractEntries();
    EXPECT_EQ(top.TotalCycles, ui128{1050});
    const auto functions = ToMap(top.Functions);
    EXPECT_EQ(functions.at("funcA").first, ui128{600});
    EXPECT_EQ(functions.at("replacementFunc").first, ui128{300});
    EXPECT_EQ(functions.at("funcB").second, ui128{900});
    EXPECT_EQ(functions.at("[kernel] kernelFunc").first, ui128{150});
}

TEST(TopAggregatorTest, EmptyStackHasNoFunctions) {
    auto profile = BuildTestProfile();
    profile.clear_sample();
    profile.add_sample()->add_value(17);
    TTopAggregatorSession session;
    const auto yaprof = BuildYaprofBytes(profile);
    session.AddYaprofBytes({yaprof.data(), yaprof.size()});
    const auto top = session.ExtractEntries();
    EXPECT_EQ(top.TotalCycles, ui128{17});
    EXPECT_TRUE(top.Functions.empty());
}

TEST(TopAggregatorTest, MalformedProfilesReturnErrorsAcrossCAbi) {
    void* session = MakePerfTopAggregator();
    const TString malformed("\xff");
    auto error = AddYaprofIntoPerfTopAggregator(session, malformed.data(), malformed.size());
    ASSERT_NE(error, nullptr);
    EXPECT_NE(PerforatorErrorString(error), nullptr);
    PerforatorErrorDispose(error);
    const auto good = BuildYaprofBytes(BuildTestProfile());
    EXPECT_EQ(AddYaprofIntoPerfTopAggregator(session, good.data(), good.size()), nullptr);
    DestroyPerfTopAggregator(session);
}

TEST(TopAggregatorTest, InvalidYaprofReferencesRejectedBeforeAggregation) {
    auto invalid = ConvertPProf(BuildTestProfile());
    invalid.mutable_functions()->set_name(1, 1000000);
    const auto bytes = invalid.SerializeAsStringOrThrow();
    TTopAggregatorSession session;
    EXPECT_THROW(session.AddYaprofBytes({bytes.data(), bytes.size()}), yexception);
    EXPECT_TRUE(session.ExtractEntries().Functions.empty());
    EXPECT_EQ(session.ExtractEntries().TotalCycles, ui128{0});
}

TEST(TopAggregatorTest, NativeMultipleStacksMatchPprofSemantics) {
    NProto::NProfile::Profile native;
    NProfile::TProfileBuilder builder{&native};
    const auto cpu = builder.AddValueType("cpu", "cycles");
    const auto other = builder.AddValueType("wall", "nanoseconds");
    const auto inner = builder.AddFunction().SetName("inner").Finish();
    const auto middle = builder.AddFunction().SetName("middle").Finish();
    const auto outer = builder.AddFunction().SetName("outer").Finish();
    const auto caller = builder.AddFunction().SetName("caller").Finish();
    const auto root = builder.AddFunction().SetName("root").Finish();
    const auto chain = builder.AddInlineChain(NProfile::TInlineChainInfo{
        .Lines = {{.Function = inner}, {.Function = middle}, {.Function = outer}},
    });
    const auto callerChain = builder.AddInlineChain(NProfile::TInlineChainInfo{
        .Lines = {{.Function = caller}},
    });
    const auto rootChain = builder.AddInlineChain(NProfile::TInlineChainInfo{
        .Lines = {{.Function = root}},
    });
    const auto leaf = builder.AddStackFrame(NProfile::TStackFrameInfo{.InlineChain = chain});
    const auto callerFrame = builder.AddStackFrame(NProfile::TStackFrameInfo{.InlineChain = callerChain});
    const auto rootFrame = builder.AddStackFrame(NProfile::TStackFrameInfo{.InlineChain = rootChain});
    // Distinct physical endpoints catch choosing the wrong stack or frame.
    // Repeating leaf in the second stack also checks cumulative deduplication.
    const auto first = builder.AddStack(NProfile::TStackInfo{
        .TopFrame = leaf,
        .StackSegment = builder.AddStackSegment(NProfile::TStackSegmentInfo{.Stack = {callerFrame}}),
    });
    const auto second = builder.AddStack(NProfile::TStackInfo{
        .TopFrame = callerFrame,
        .StackSegment = builder.AddStackSegment(NProfile::TStackSegmentInfo{.Stack = {leaf, rootFrame}}),
    });
    const auto key = builder.AddSampleKey(NProfile::TSampleKeyInfo{.Stacks = {first, second}});
    builder.AddSample(NProfile::TSampleInfo{.Key = key, .Values = {{cpu, 123}, {other, 999}}});
    std::move(builder).Finish();

    const auto bytes = native.SerializeAsStringOrThrow();
    TTopAggregatorSession direct;
    direct.AddYaprofBytes({bytes.data(), bytes.size()});
    const auto top = direct.ExtractEntries();
    const auto functions = ToMap(top.Functions);
    EXPECT_EQ(top.TotalCycles, ui128{123});
    ASSERT_EQ(functions.size(), 5);
    EXPECT_EQ(functions.at("outer").first, ui128{0});
    EXPECT_EQ(functions.at("outer").second, ui128{123});
    EXPECT_EQ(functions.at("middle").first, ui128{0});
    EXPECT_EQ(functions.at("middle").second, ui128{123});
    EXPECT_EQ(functions.at("inner").first, ui128{123});
    EXPECT_EQ(functions.at("inner").second, ui128{123});
    EXPECT_EQ(functions.at("caller").first, ui128{0});
    EXPECT_EQ(functions.at("caller").second, ui128{123});
    EXPECT_EQ(functions.at("root").first, ui128{0});
    EXPECT_EQ(functions.at("root").second, ui128{123});

    TString pprof;
    NProfile::ConvertToPProf(native, &pprof);
    TTopAggregatorSession legacy;
    NProto::NProfile::Profile converted;
    NProfile::ConvertFromPProf(pprof, &converted);
    const auto legacyBytes = converted.SerializeAsStringOrThrow();
    legacy.AddYaprofBytes({legacyBytes.data(), legacyBytes.size()});
    EXPECT_EQ(ToMap(legacy.ExtractEntries().Functions), functions);
}

} // namespace NPerforator::NClusterTop
