#include "visitor.h"

namespace NPerforator::NProfile {

void VisitProfile(
    const NProto::NProfile::Profile& profile,
    IProfileVisitor& visitor
) {
    visitor.VisitWholeProfile(profile);
    visitor.VisitStringTable(profile.strtab());
    visitor.VisitMetadata(profile.metadata());
    visitor.VisitFeatures(profile.features());
    visitor.VisitComments(profile.comments());
    visitor.VisitLabels(profile.labels());
    visitor.VisitThreads(profile.threads());
    visitor.VisitBinaries(profile.binaries());
    visitor.VisitFunctions(profile.functions());
    visitor.VisitInlineChains(profile.inline_chains());
    visitor.VisitStackFrames(profile.stack_frames());
    visitor.VisitStacks(profile.stacks());
    visitor.VisitSampleKeys(profile.sample_keys());
    visitor.VisitSamples(profile.samples());
}

} // namespace NPerforator::NProfile
