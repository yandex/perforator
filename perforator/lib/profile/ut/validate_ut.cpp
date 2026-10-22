#include <perforator/lib/profile/builder.h>
#include <perforator/lib/profile/validate.h>

#include <library/cpp/testing/gtest/gtest.h>

using namespace NPerforator::NProfile;

namespace {

NPerforator::NProto::NProfile::Profile BuildEmptyProfile() {
    NPerforator::NProto::NProfile::Profile profile;
    TProfileBuilder builder{&profile};
    std::move(builder).Finish();
    return profile;
}

class ProfileValidationOffsets : public testing::TestWithParam<bool> {};

TEST_P(ProfileValidationOffsets, AcceptsEmptyProfileWithSentinels) {
    const auto profile = BuildEmptyProfile();
    ASSERT_EQ(profile.inline_chains().lines_offset_size(), 1);
    ASSERT_EQ(profile.stack_segments().frame_ids_offset_size(), 1);
    EXPECT_NO_THROW(ValidateProfile(profile, {.CheckIndices = GetParam()}));
}

TEST_P(ProfileValidationOffsets, RejectsMissingInlineChainOffsets) {
    auto profile = BuildEmptyProfile();
    profile.mutable_inline_chains()->clear_lines_offset();
    EXPECT_THROW(ValidateProfile(profile, {.CheckIndices = GetParam()}), yexception);
}

TEST_P(ProfileValidationOffsets, RejectsMissingStackSegmentOffsets) {
    auto profile = BuildEmptyProfile();
    profile.mutable_stack_segments()->clear_frame_ids_offset();
    EXPECT_THROW(ValidateProfile(profile, {.CheckIndices = GetParam()}), yexception);
}

INSTANTIATE_TEST_SUITE_P(CheckIndices, ProfileValidationOffsets, testing::Bool());

NPerforator::NProto::NProfile::Profile BuildProfileWithStack() {
    NPerforator::NProto::NProfile::Profile profile;
    TProfileBuilder builder{&profile};
    const auto cpu = builder.AddValueType("cpu", "cycles");
    const auto stack = builder.AddStack(TStackInfo{});
    const auto key = builder.AddSampleKey(TSampleKeyInfo{.Stacks = {stack}});
    builder.AddSample(TSampleInfo{.Key = key, .Values = {{cpu, 7}}});
    std::move(builder).Finish();
    return profile;
}

class ProfileValidationStacks : public testing::TestWithParam<bool> {};

TEST_P(ProfileValidationStacks, AcceptsMatchingStackArrays) {
    const auto profile = BuildProfileWithStack();
    ASSERT_EQ(profile.stacks().top_frame_id_size(), 1);
    ASSERT_EQ(profile.stacks().stack_segment_id_size(), 1);
    EXPECT_NO_THROW(ValidateProfile(profile, {.CheckIndices = GetParam()}));
}

TEST_P(ProfileValidationStacks, AcceptsPopulatedStacksWithInlineChains) {
    NPerforator::NProto::NProfile::Profile profile;
    TProfileBuilder builder{&profile};
    const auto cpu = builder.AddValueType("cpu", "cycles");
    const auto leaf = builder.AddFunction(TFunctionInfo{.Name = builder.AddString("leaf")});
    const auto caller = builder.AddFunction(TFunctionInfo{.Name = builder.AddString("caller")});
    const auto root = builder.AddFunction(TFunctionInfo{.Name = builder.AddString("root")});
    const auto leafChain = builder.AddInlineChain(TInlineChainInfo{.Lines = {
        {.Function = leaf, .Line = 10},
        {.Function = caller, .Line = 20},
    }});
    const auto rootChain = builder.AddInlineChain(TInlineChainInfo{.Lines = {
        {.Function = root, .Line = 30},
    }});
    const auto firstFrame = builder.AddStackFrame(TStackFrameInfo{.Address = 0x10, .InlineChain = leafChain});
    const auto secondFrame = builder.AddStackFrame(TStackFrameInfo{.Address = 0x20, .InlineChain = leafChain});
    const auto rootFrame = builder.AddStackFrame(TStackFrameInfo{.Address = 0x30, .InlineChain = rootChain});
    const auto firstSegment = builder.AddStackSegment(TStackSegmentInfo{.Stack = {rootFrame}});
    const auto secondSegment = builder.AddStackSegment(TStackSegmentInfo{.Stack = {firstFrame, rootFrame}});
    const auto firstStack = builder.AddStack(TStackInfo{.TopFrame = firstFrame, .StackSegment = firstSegment});
    const auto secondStack = builder.AddStack(TStackInfo{.TopFrame = secondFrame, .StackSegment = secondSegment});
    const auto key = builder.AddSampleKey(TSampleKeyInfo{.Stacks = {firstStack, secondStack}});
    builder.AddSample(TSampleInfo{.Key = key, .Values = {{cpu, 7}}});
    std::move(builder).Finish();

    // Include the empty entries that the builder reserves at index zero.
    ASSERT_EQ(profile.stacks().top_frame_id_size(), 3);
    ASSERT_EQ(profile.stacks().stack_segment_id_size(), 3);
    ASSERT_EQ(profile.stack_segments().frame_ids_offset_size(), 3);
    ASSERT_EQ(profile.stack_segments().frame_ids_size(), 3);
    ASSERT_EQ(profile.inline_chains().lines_offset_size(), 3);
    ASSERT_EQ(profile.inline_chains().lines().function_id_size(), 3);
    EXPECT_NO_THROW(ValidateProfile(profile, {.CheckIndices = GetParam()}));
}

TEST_P(ProfileValidationStacks, RejectsMissingTopFrames) {
    auto profile = BuildProfileWithStack();
    profile.mutable_stacks()->clear_top_frame_id();
    EXPECT_THROW(ValidateProfile(profile, {.CheckIndices = GetParam()}), yexception);
}

TEST_P(ProfileValidationStacks, RejectsMissingStackSegments) {
    auto profile = BuildProfileWithStack();
    profile.mutable_stacks()->clear_stack_segment_id();
    EXPECT_THROW(ValidateProfile(profile, {.CheckIndices = GetParam()}), yexception);
}

TEST_P(ProfileValidationStacks, RejectsExtraTopFrames) {
    auto profile = BuildProfileWithStack();
    profile.mutable_stacks()->add_top_frame_id(0);
    EXPECT_THROW(ValidateProfile(profile, {.CheckIndices = GetParam()}), yexception);
}

TEST_P(ProfileValidationStacks, RejectsExtraStackSegments) {
    auto profile = BuildProfileWithStack();
    profile.mutable_stacks()->add_stack_segment_id(0);
    EXPECT_THROW(ValidateProfile(profile, {.CheckIndices = GetParam()}), yexception);
}

INSTANTIATE_TEST_SUITE_P(CheckIndices, ProfileValidationStacks, testing::Bool());

} // namespace
