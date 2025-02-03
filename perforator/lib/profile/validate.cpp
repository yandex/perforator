#include "validate.h"
#include "visitor.h"

#include <perforator/proto/profile/profile.pb.h>

#include <library/cpp/iterator/zip.h>

#include <util/generic/maybe.h>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

namespace {

template <typename First, typename ...Rest>
void RequireCongruentContainers(const First& first, const Rest& ...rest) {
    auto size = first.size();
    [[maybe_unused]] int _ = (0 + ... + [&size](const auto& container){
        Y_ENSURE(
            container.size() == size,
            "Expected size of " << size << " elements, got " << container.size()
        );
        return 0;
     }(rest));
}

class TProfileValidator final : public INopProfileVisitor {
public:
    TProfileValidator(
        const NProto::NProfile::Profile& profile,
        TProfileValidationOptions options
    )
        : Profile_{profile}
        , Options_{options}
    {}

    void Validate() {
        VisitProfile(Profile_, *this);
    }

private:
    void VisitStringTable(const NProto::NProfile::StringTable& strtab) override {
        RequireCongruentContainers(strtab.offset(), strtab.length());

        // Check that first string is empty.
        Y_ENSURE(strtab.length_size() > 0);
        Y_ENSURE(strtab.length(0) == 0);

        if (Options_.CheckIndices) {
            for (auto [offset, length] : Zip(strtab.offset(), strtab.length())) {
                Y_ENSURE(offset + length <= strtab.strings().size());
            }
        }
    }

    void VisitComments(const NProto::NProfile::Comments& comments) override {
        RequireStringArray(comments.comment());
    }

    void VisitLabels(const NProto::NProfile::Labels& labels) override {
        RequireStringLabels(labels.strings());
        RequireNumericLabels(labels.numbers());
    }

    void RequireStringLabels(const NProto::NProfile::StringLabels& labels) const {
        RequireCongruentContainers(labels.key(), labels.value());
        RequireStringArray(labels.key());
        RequireStringArray(labels.value());
    }

    void RequireNumericLabels(const NProto::NProfile::NumberLabels& labels) const {
        RequireCongruentContainers(labels.key(), labels.value());
        RequireStringArray(labels.key());
    }

    void VisitThreads(const NProto::NProfile::Threads& threads) override {
        RequireCongruentContainers(
            threads.thread_id(),
            threads.process_id(),
            threads.thread_name(),
            threads.process_name(),
            threads.container_offset()
        );

        // Check that the first thread is empty.
        Y_ENSURE(threads.thread_id_size() > 0);
        Y_ENSURE(threads.thread_id(0) == 0);
        Y_ENSURE(threads.process_id(0) == 0);
        Y_ENSURE(threads.thread_name(0) == 0);
        Y_ENSURE(threads.process_name(0) == 0);
        Y_ENSURE(threads.container_offset(0) == 0);
        Y_ENSURE(threads.container_offset().size() == 1 || threads.container_offset(1) == 0);

        RequireStringArray(threads.thread_name());
        RequireStringArray(threads.process_name());
        RequireFlattenedArray(threads.container_offset(), threads.container_names());
        RequireStringArray(threads.container_names());
    }

    void VisitBinaries(const NProto::NProfile::Binaries& binaries) override {
        RequireCongruentContainers(binaries.build_id(), binaries.path());
        RequireStringArray(binaries.build_id());
        RequireStringArray(binaries.path());

        // Check that the first binary is empty.
        Y_ENSURE(binaries.build_id_size() > 0);
        Y_ENSURE(binaries.build_id(0) == 0);
        Y_ENSURE(binaries.path(0) == 0);
    }

    void VisitFunctions(const NProto::NProfile::Functions& functions) override {
        RequireCongruentContainers(
            functions.name(),
            functions.system_name(),
            functions.filename(),
            functions.start_line()
        );
        RequireStringArray(functions.name());
        RequireStringArray(functions.system_name());
        RequireStringArray(functions.filename());

        // Check that the first function is empty.
        Y_ENSURE(functions.name_size() > 0);
        Y_ENSURE(functions.name(0) == 0);
        Y_ENSURE(functions.system_name(0) == 0);
        Y_ENSURE(functions.filename(0) == 0);
        Y_ENSURE(functions.start_line(0) == 0);
    }

    void VisitInlineChains(const NProto::NProfile::InlineChains& inlineChains) override {
        RequireFlattenedArray(inlineChains.offset(), inlineChains.function_id());

        RequireCongruentContainers(
            inlineChains.function_id(),
            inlineChains.line(),
            inlineChains.column()
        );

        ExpectEntityArray(inlineChains.function_id(), Profile_.functions().name(), "function");

        // Check that the first inline chain is empty.
        Y_ENSURE(inlineChains.offset(0) == 0);
        Y_ENSURE(inlineChains.offset_size() == 1 || inlineChains.offset(1) == 0);
    }

    void VisitStackFrames(const NProto::NProfile::StackFrames& stackFrames) override {
        RequireCongruentContainers(
            stackFrames.binary_id(),
            stackFrames.binary_offset(),
            stackFrames.inline_chain_id()
        );

        // Check that the first stack frame is empty.
        Y_ENSURE(stackFrames.binary_id_size() > 0);
        Y_ENSURE(stackFrames.binary_id(0) == 0);
        Y_ENSURE(stackFrames.binary_offset(0) == 0);
        Y_ENSURE(stackFrames.inline_chain_id(0) == 0);

        ExpectEntityArray(stackFrames.binary_id(), Profile_.binaries().build_id(), "binary");
        ExpectEntityArray(stackFrames.inline_chain_id(), Profile_.inline_chains().offset(), "inline_chain");
    }

    void VisitStacks(const NProto::NProfile::Stacks& stacks) override {
        RequireFlattenedArray(stacks.offset(), stacks.frame_id());
        ExpectEntityArray(stacks.frame_id(), Profile_.stack_frames().binary_id(), "stack_frame");

        Y_ENSURE(stacks.offset_size() > 0, "The first stack must be defined empty");
        Y_ENSURE(stacks.offset(0) == 0, "The first stack must be defined empty");
        Y_ENSURE(stacks.offset_size() == 1 || stacks.offset(1) == 0, "The first stack must be defined empty");
    }

    void VisitSampleKeys(const NProto::NProfile::SampleKeys& keys) override {
        RequireCongruentContainers(
            keys.stacks().user_stack_id(),
            keys.stacks().kernel_stack_id(),
            keys.labels().first_label_id(),
            keys.threads().thread_id()
        );

        RequireFlattenedArray(keys.labels().first_label_id(), keys.labels().packed_label_id());
        ExpectEntityArray(keys.stacks().user_stack_id(), Profile_.stacks().offset(), "stack");
        ExpectEntityArray(keys.stacks().kernel_stack_id(), Profile_.stacks().offset(), "stack");
        ExpectEntityArray(keys.threads().thread_id(), Profile_.threads().thread_id(), "thread");

        // Packed labels require custom checks.
        for (ui32 label : keys.labels().packed_label_id()) {
            bool isNumber = label & 1;
            ui32 labelIndex = label >> 1;
            if (isNumber) {
                Y_ENSURE(labelIndex < (ui32)Profile_.labels().numbers().key_size());
            } else {
                Y_ENSURE(labelIndex < (ui32)Profile_.labels().strings().key_size());
            }
        }
    }

    void VisitSamples(const NProto::NProfile::Samples& samples) override {
        for (auto&& values : samples.values()) {
            RequireCongruentContainers(
                samples.key(),
                values.value()
            );
        }
        if (samples.has_timestamps()) {
            RequireCongruentContainers(samples.key(), samples.timestamps().delta_nanoseconds());
        }

        ExpectEntityArray(
            samples.key(),
            Profile_.sample_keys().stacks().user_stack_id(),
            "sample_keys"
        );
    }

private:
    template <typename Offsets, typename Values>
    void RequireFlattenedArray(Offsets&& offsets, Values&& values) const {
        if (!Options_.CheckIndices) {
            return;
        }

        TMaybe<ui32> lastOffset;
        ui32 maxOffset = values.size();

        for (std::same_as<ui32> auto offset : offsets) {
            Y_ENSURE(lastOffset <= offset);
            Y_ENSURE(offset <= maxOffset);
            lastOffset = offset;
        }
    }

    template <typename Arr>
    void RequireStringArray(Arr&& array) const {
        ExpectEntityArray(array, Profile_.strtab().length(), "string");
    }

    template <typename Idx, typename Arr>
    void ExpectEntityArray(Idx&& indicies, Arr&& arr, TStringBuf name) const {
        if (!Options_.CheckIndices) {
            return;
        }

        ui32 size = arr.size();

        for (std::same_as<ui32> auto idx : indicies) {
            Y_ENSURE(idx < size,
                "Invalid " << name
                << " index, expected indices in range [0, " << size << ")"
                << ", got " << idx)
            ;
        }
    }

private:
    const NProto::NProfile::Profile& Profile_;
    const TProfileValidationOptions Options_;
};

} // anonymous namespace

////////////////////////////////////////////////////////////////////////////////

void ValidateProfile(
    const NProto::NProfile::Profile& profile,
    TProfileValidationOptions options
) {
    TProfileValidator{profile, options}.Validate();
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
