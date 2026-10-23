#include <perforator/symbolizer/lib/cluster_top/cluster_top_c.h>
#include <perforator/symbolizer/lib/cluster_top/perf_top_aggregator.h>
#include <perforator/symbolizer/lib/cluster_top/top_aggregator_session.h>

#include <perforator/lib/profile/pprof.h>
#include <perforator/lib/profile/builder.h>
#include <perforator/proto/pprofprofile/profile.pb.h>
#include <perforator/proto/profile/profile.pb.h>

#include <library/cpp/testing/gtest/gtest.h>

#include <llvm/DebugInfo/GSYM/GsymCreator.h>
#include <llvm/DebugInfo/GSYM/OutputAggregator.h>
#include <util/system/tempfile.h>

#include <limits>
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
    line->set_line(42);
    line->set_column(7);

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
    ASSERT_EQ(top.Functions.size(), 1u);
    EXPECT_EQ(top.Functions[0].BuildId, "userBuild");
    EXPECT_TRUE(top.Functions[0].SourceFile.empty());
    EXPECT_EQ(top.Functions[0].SourceLine, 42u);
    EXPECT_EQ(top.Functions[0].SourceColumn, 7u);
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
    ASSERT_EQ(top.Functions.size(), 1u);
    EXPECT_EQ(top.Functions[0].BuildId, "buildIdNoGsym");
    EXPECT_TRUE(top.Functions[0].SourceFile.empty());
    EXPECT_EQ(top.Functions[0].SourceLine, 0u);
    EXPECT_EQ(top.Functions[0].SourceColumn, 0u);
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
    for (const char* s : {"", "cpu", "cycles", "inlinedLeaf", "outerCaller", "buildId", "inline.h", "caller.cpp"}) {
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
    fInlined->set_filename(6);
    fInlined->set_start_line(1);

    auto* fOuter = profile.add_function();
    fOuter->set_id(2);
    fOuter->set_name(4); // "outerCaller"
    fOuter->set_filename(7);

    // Location with inline chain: leaf-to-root order.
    // [0] = inlinedLeaf (most-inlined), [1] = outerCaller (root).
    auto* loc = profile.add_location();
    loc->set_id(1);
    loc->set_mapping_id(1);
    loc->set_address(0x110000);
    auto* inlineLine = loc->add_line();
    inlineLine->set_function_id(1);
    inlineLine->set_line(42);
    inlineLine->set_column(7);
    auto* callerLine = loc->add_line();
    callerLine->set_function_id(2);
    callerLine->set_line(81);
    callerLine->set_column(9);

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
    for (const auto& function : top.Functions) {
        EXPECT_EQ(function.BuildId, "buildId");
        const bool leaf = function.Name == "inlinedLeaf";
        EXPECT_EQ(function.SourceFile, leaf ? "inline.h" : "caller.cpp");
        EXPECT_EQ(function.SourceLine, leaf ? 42u : 81u);
        EXPECT_EQ(function.SourceColumn, leaf ? 7u : 9u);
    }
}

TEST(TopAggregatorTest, SourceLocationsRemainDistinctAcrossProfilesAndMerges) {
    NProto::NPProf::Profile profile;
    for (const char* value : {"", "cpu", "cycles", "same", "a.cpp", "b.cpp", "build-a", "build-b"}) {
        profile.add_string_table(value);
    }
    auto* type = profile.add_sample_type();
    type->set_type(1);
    type->set_unit(2);
    for (ui64 id = 1; id <= 2; ++id) {
        auto* mapping = profile.add_mapping();
        mapping->set_id(id);
        mapping->set_build_id(5 + id);
        auto* function = profile.add_function();
        function->set_id(id);
        function->set_name(3);
        function->set_filename(3 + id);
    }
    for (ui64 id = 1; id <= 5; ++id) {
        auto* location = profile.add_location();
        location->set_id(id);
        location->set_mapping_id(id == 4 ? 2 : 1);
        auto* line = location->add_line();
        line->set_function_id(id == 3 ? 2 : 1);
        line->set_line(id == 2 ? 11 : 10);
        line->set_column(id == 5 ? 3 : 2);
    }
    auto* recursive = profile.add_sample();
    for (ui64 id : {1, 2, 1}) {
        recursive->add_location_id(id);
    }
    recursive->add_value(100);
    for (ui64 id = 3; id <= 5; ++id) {
        auto* sample = profile.add_sample();
        sample->add_location_id(id);
        sample->add_value(7);
    }

    TTopAggregatorSession destination;
    {
        TTopAggregatorSession source;
        const auto bytes = BuildYaprofBytes(profile);
        source.AddYaprofBytes({bytes.data(), bytes.size()});
        destination.MergeAggregator(source);
    }
    // Reuse the session's protobuf storage, then destroy all input strings.
    const auto bytes = BuildYaprofBytes(profile);
    destination.AddYaprofBytes({bytes.data(), bytes.size()});
    profile.Clear();
    const auto top = destination.ExtractEntries();
    EXPECT_EQ(top.TotalCycles, ui128{242});
    ASSERT_EQ(top.Functions.size(), 5u);
    for (const auto& function : top.Functions) {
        EXPECT_EQ(function.Name, "same");
        if (function.SourceLine == 11) {
            EXPECT_EQ(function.SelfCycles, ui128{0});
            EXPECT_EQ(function.CumulativeCycles, ui128{200});
        } else if (function.BuildId == "build-a" && function.SourceFile == "a.cpp" && function.SourceColumn == 2) {
            EXPECT_EQ(function.SelfCycles, ui128{200});
            EXPECT_EQ(function.CumulativeCycles, ui128{200});
        } else {
            EXPECT_EQ(function.SelfCycles, ui128{14});
            EXPECT_EQ(function.CumulativeCycles, ui128{14});
        }
    }
}

TEST(TopAggregatorTest, GsymSourceLocationsSurviveCachingAndAggregation) {
    TTempFileHandle file;
    llvm::gsym::GsymCreator creator(true);
    creator.setBaseAddress(0);
    const auto leafFile = creator.insertFile("src/inline.h");
    const auto callerFile = creator.insertFile("src/caller.cpp");
    llvm::gsym::FunctionInfo function(0x1000, 0x100, creator.insertString("outer"));
    function.OptLineTable.emplace();
    function.OptLineTable->push(llvm::gsym::LineEntry(0x1000, leafFile, 42));
    function.Inline.emplace();
    function.Inline->Name = function.Name;
    function.Inline->Ranges.insert(llvm::AddressRange(0x1000, 0x1100));
    auto& inner = function.Inline->Children.emplace_back();
    inner.Name = creator.insertString("inner");
    inner.CallFile = callerFile;
    inner.CallLine = 81;
    inner.Ranges.insert(llvm::AddressRange(0x1000, 0x1100));
    creator.addFunctionInfo(std::move(function));
    llvm::gsym::OutputAggregator output(nullptr);
    ASSERT_TRUE(llvm::toString(creator.finalize(output)).empty());
    ASSERT_TRUE(llvm::toString(creator.save(file.Name().c_str(), llvm::endianness::little)).empty());

    TCachingGSYMSymbolizer cache(file.Name());
    for (int i = 0; i < 2; ++i) {
        const auto& frames = cache.Symbolize(0x1000);
        ASSERT_EQ(frames.size(), 2u);
        EXPECT_EQ(frames[0].Name, "inner");
        EXPECT_EQ(frames[0].SourceFile, "src/inline.h");
        EXPECT_EQ(frames[0].SourceLine, 42u);
        EXPECT_EQ(frames[0].SourceColumn, 0u);
        EXPECT_EQ(frames[1].Name, "outer");
        EXPECT_EQ(frames[1].SourceFile, "src/caller.cpp");
        EXPECT_EQ(frames[1].SourceLine, 81u);
        EXPECT_EQ(frames[1].SourceColumn, 0u);
    }
    cache.PruneCaches();
    EXPECT_EQ(cache.Symbolize(0x1000)[0].SourceLine, 42u);

    NProto::NProfile::Profile native;
    NProfile::TProfileBuilder builder{&native};
    const auto cpu = builder.AddValueType("cpu", "cycles");
    const auto binary = builder.AddBinary().SetBuildId("gsym-build").Finish();
    const auto frame = builder.AddStackFrame(NProfile::TStackFrameInfo{.Binary = binary, .Address = 0x1000});
    const auto stack = builder.AddStack(NProfile::TStackInfo{.TopFrame = frame});
    const auto key = builder.AddSampleKey(NProfile::TSampleKeyInfo{.Stacks = {stack}});
    builder.AddSample(NProfile::TSampleInfo{.Key = key, .Values = {{cpu, 123}}});
    std::move(builder).Finish();
    TPerfTopAggregator aggregator;
    const TString buildId = "gsym-build";
    aggregator.InitializeSymbolizer({buildId.data(), buildId.size()}, {file.Name().data(), file.Name().size()});
    aggregator.AddProfile(native);
    aggregator.AddProfile(native);
    const auto top = aggregator.ExtractEntries();
    ASSERT_EQ(top.Functions.size(), 2u);
    for (const auto& entry : top.Functions) {
        const bool leaf = entry.Name == "inner";
        EXPECT_EQ(entry.BuildId, buildId);
        EXPECT_EQ(entry.SourceFile, leaf ? "src/inline.h" : "src/caller.cpp");
        EXPECT_EQ(entry.SourceLine, leaf ? 42u : 81u);
        EXPECT_EQ(entry.SourceColumn, 0u);
        EXPECT_EQ(entry.SelfCycles, leaf ? ui128{246} : ui128{0});
        EXPECT_EQ(entry.CumulativeCycles, ui128{246});
    }
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

TEST(TopAggregatorTest, ZeroCycleSamplePreservesOnlyLeafRow) {
    auto profile = BuildTestProfile();
    profile.clear_sample();
    auto* sample = profile.add_sample();
    sample->add_location_id(1);
    sample->add_location_id(2);
    sample->add_value(0);

    TPerfTopAggregator aggregator;
    aggregator.AddProfile(ConvertPProf(profile));
    const auto top = aggregator.ExtractEntries();
    EXPECT_EQ(top.TotalCycles, ui128{0});
    ASSERT_EQ(top.Functions.size(), 1u);
    EXPECT_EQ(top.Functions[0].Name, "funcA");
    EXPECT_EQ(top.Functions[0].SelfCycles, ui128{0});
    EXPECT_EQ(top.Functions[0].CumulativeCycles, ui128{0});
}

TEST(TopAggregatorTest, PerProfileCountersDoNotTruncateToUint64) {
    auto profile = BuildTestProfile();
    profile.clear_sample();
    const auto labelId = profile.string_table_size();
    profile.add_string_table("sample_id");
    constexpr auto value = std::numeric_limits<i64>::max();
    for (int i = 0; i < 3; ++i) {
        auto* sample = profile.add_sample();
        sample->add_location_id(1);
        sample->add_location_id(2);
        sample->add_value(value);
        // Keep samples distinct: pprof conversion otherwise coalesces their UInt64 values.
        auto* label = sample->add_label();
        label->set_key(labelId);
        label->set_num(i);
    }

    const auto native = ConvertPProf(profile);
    TPerfTopAggregator aggregator;
    aggregator.AddProfile(native);
    aggregator.AddProfile(native);
    const auto top = aggregator.ExtractEntries();
    const ui128 expected = ui128{static_cast<ui64>(value)} * 6;
    EXPECT_EQ(top.TotalCycles, expected);
    const auto functions = ToMap(top.Functions);
    ASSERT_EQ(functions.size(), 2u);
    EXPECT_EQ(functions.at("funcA").first, expected);
    EXPECT_EQ(functions.at("funcA").second, expected);
    EXPECT_EQ(functions.at("funcB").first, ui128{0});
    EXPECT_EQ(functions.at("funcB").second, expected);
}

TEST(TopAggregatorTest, DemanglingPreservesLocationsAndKernelTags) {
    auto profile = BuildTestProfile();
    for (int index : {3, 4, 6}) {
        profile.set_string_table(index, "_Z3foov");
    }
    const auto fileId = profile.string_table_size();
    profile.add_string_table("same.cpp");
    for (auto& function : *profile.mutable_function()) {
        function.set_filename(fileId);
    }
    for (int i = 0; i < profile.location_size(); ++i) {
        profile.mutable_location(i)->mutable_line(0)->set_line((i + 1) * 10);
    }
    TPerfTopAggregator aggregator;
    aggregator.AddProfile(ConvertPProf(profile));
    // A distinct raw name that demangles to the same output must still merge.
    profile.set_string_table(3, "foo()");
    aggregator.AddProfile(ConvertPProf(profile));
    for (int i = 0; i < 2; ++i) {
        const auto top = aggregator.ExtractEntries();
        EXPECT_EQ(top.TotalCycles, ui128{700});
        ASSERT_EQ(top.Functions.size(), 3u);
        for (const auto& function : top.Functions) {
            EXPECT_EQ(function.SourceFile, "same.cpp");
            if (function.SourceLine == 10) {
                EXPECT_EQ(function.Name, "foo()");
                EXPECT_EQ(function.SelfCycles, ui128{600});
                EXPECT_EQ(function.CumulativeCycles, ui128{600});
            } else if (function.SourceLine == 20) {
                EXPECT_EQ(function.Name, "foo()");
                EXPECT_EQ(function.SelfCycles, ui128{0});
                EXPECT_EQ(function.CumulativeCycles, ui128{600});
            } else {
                EXPECT_EQ(function.SourceLine, 30u);
                EXPECT_EQ(function.Name, "[kernel] foo()");
                EXPECT_EQ(function.SelfCycles, ui128{100});
                EXPECT_EQ(function.CumulativeCycles, ui128{100});
            }
        }
    }
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
