#pragma once

#include <perforator/proto/profile/profile.pb.h>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

// Visit profile visits profile in topological order.
// If a visitor visits entity X, all dependencies of the X have been already visited.
// For example, stack frames are guranteed to be visited before stacks.
void VisitProfile(
    const NProto::NProfile::Profile& profile,
    class IProfileVisitor& visitor
);

////////////////////////////////////////////////////////////////////////////////

class IProfileVisitor {
public:
    virtual ~IProfileVisitor() = default;

    virtual void VisitWholeProfile(const NProto::NProfile::Profile& profile) = 0;
    virtual void VisitStringTable(const NProto::NProfile::StringTable& strtab) = 0;
    virtual void VisitMetadata(const NProto::NProfile::Metadata& metadata) = 0;
    virtual void VisitFeatures(const NProto::NProfile::Features& functions) = 0;
    virtual void VisitComments(const NProto::NProfile::Comments& comments) = 0;
    virtual void VisitLabels(const NProto::NProfile::Labels& labels) = 0;
    virtual void VisitThreads(const NProto::NProfile::Threads& threads) = 0;
    virtual void VisitBinaries(const NProto::NProfile::Binaries& binaries) = 0;
    virtual void VisitFunctions(const NProto::NProfile::Functions& functions) = 0;
    virtual void VisitInlineChains(const NProto::NProfile::InlineChains& inlineChains) = 0;
    virtual void VisitStackFrames(const NProto::NProfile::StackFrames& stackFrames) = 0;
    virtual void VisitStacks(const NProto::NProfile::Stacks& stacks) = 0;
    virtual void VisitSampleKeys(const NProto::NProfile::SampleKeys& sampleKeys) = 0;
    virtual void VisitSamples(const NProto::NProfile::Samples& samples) = 0;
};

class INopProfileVisitor : public IProfileVisitor {
public:
    void VisitWholeProfile(const NProto::NProfile::Profile&) override {}
    void VisitStringTable(const NProto::NProfile::StringTable&) override {}
    void VisitMetadata(const NProto::NProfile::Metadata&) override {}
    void VisitFeatures(const NProto::NProfile::Features&) override {}
    void VisitComments(const NProto::NProfile::Comments&) override {}
    void VisitLabels(const NProto::NProfile::Labels&) override {}
    void VisitThreads(const NProto::NProfile::Threads&) override {}
    void VisitBinaries(const NProto::NProfile::Binaries&) override {}
    void VisitFunctions(const NProto::NProfile::Functions&) override {}
    void VisitInlineChains(const NProto::NProfile::InlineChains&) override {}
    void VisitStackFrames(const NProto::NProfile::StackFrames&) override {}
    void VisitStacks(const NProto::NProfile::Stacks&) override {}
    void VisitSampleKeys(const NProto::NProfile::SampleKeys&) override {}
    void VisitSamples(const NProto::NProfile::Samples&) override {}
};

struct TCheckThatAllNopProfileVisitorMethodsAreImplemented final
    : INopProfileVisitor
{};

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
