#include <perforator/symbolizer/lib/cluster_top/cluster_top_c.h>
#include <perforator/proto/pprofprofile/profile.pb.h>
#include <perforator/lib/profile/pprof.h>
#include <perforator/proto/profile/profile.pb.h>

#include <library/cpp/testing/gtest/gtest.h>

namespace NPerforator::NClusterTop {
namespace {
TString MakeKernelProfile() {
    NProto::NPProf::Profile profile;
    for (const char* value : {"", "cpu", "cycles", "[kernel]", "kernelFunc", "build-id", "kernel.cpp"}) {
        profile.add_string_table(value);
    }
    auto* type = profile.add_sample_type();
    type->set_type(1);
    type->set_unit(2);
    auto* mapping = profile.add_mapping();
    mapping->set_id(1);
    mapping->set_filename(3);
    mapping->set_build_id(5);
    auto* function = profile.add_function();
    function->set_id(1);
    function->set_name(4);
    function->set_filename(6);
    auto* location = profile.add_location();
    location->set_id(1);
    location->set_mapping_id(1);
    auto* line = location->add_line();
    line->set_function_id(1);
    line->set_line(42);
    line->set_column(7);
    auto* sample = profile.add_sample();
    sample->add_location_id(1);
    sample->add_value(7);
    NProto::NProfile::Profile native;
    NProfile::ConvertFromPProf(profile.SerializeAsStringOrThrow(), &native);
    return native.SerializeAsStringOrThrow();
}

void CheckAndFreeResult(void* aggregator, bool empty) {
    ui64 count = 0;
    TClusterTopFunction* functions = nullptr;
    FinalizePerfTopAggregator(aggregator, &count, &functions);
    EXPECT_EQ(count, empty ? 0u : 1u);
    if (count == 1) {
        EXPECT_STREQ(functions[0].Name, "[kernel] kernelFunc");
        EXPECT_STREQ(functions[0].BuildId, "build-id");
        EXPECT_STREQ(functions[0].SourceFile, "kernel.cpp");
        EXPECT_EQ(functions[0].SourceLine, 42u);
        EXPECT_EQ(functions[0].SourceColumn, 7u);
        for (size_t i = 0; i < 15; ++i) {
            EXPECT_EQ(functions[0].SelfCycles[i], 0);
            EXPECT_EQ(functions[0].CumulativeCycles[i], 0);
        }
        EXPECT_EQ(functions[0].SelfCycles[15], 7);
        EXPECT_EQ(functions[0].CumulativeCycles[15], 7);
    }
    FreePerfTopAggregatorResult(count, functions);
}
} // namespace

TEST(PerfTopAggregatorResult, MergePreservesKernelTagAndCycles) {
    void* destination = MakePerfTopAggregator();
    void* source = MakePerfTopAggregator();
    const auto profile = MakeKernelProfile();
    ASSERT_EQ(AddYaprofIntoPerfTopAggregator(source, profile.data(), profile.size()), nullptr);
    MergePerfTopAggregators(destination, source);
    DestroyPerfTopAggregator(source);
    CheckAndFreeResult(destination, false);
    CheckAndFreeResult(destination, false);
    DestroyPerfTopAggregator(destination);
}

TEST(PerfTopAggregatorResult, FreeEmptyResult) {
    void* aggregator = MakePerfTopAggregator();
    CheckAndFreeResult(aggregator, true);
    DestroyPerfTopAggregator(aggregator);
    FreePerfTopAggregatorResult(0, nullptr);
}
} // namespace NPerforator::NClusterTop
