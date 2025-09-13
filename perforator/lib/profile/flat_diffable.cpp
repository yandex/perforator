#include "flat_diffable.h"
#include "compact_map.h"

#include <perforator/lib/profile/profile.h>

#include <library/cpp/iterator/enumerate.h>
#include <library/cpp/iterator/zip.h>
#include <library/cpp/json/json_value.h>
#include <library/cpp/json/json_writer.h>


namespace NPerforator::NProfile {

struct TFlatSampleKeyBuilder {
public:
    TFlatSampleKeyBuilder(TFlatDiffableProfileOptions& options)
        : Options_{options}
    {}

    TFlatSampleKeyBuilder& SetTimestamp(TInstant ts) {
        if (Options_.PrintTimestamps) {
            Value_.InsertValue("timestamp", ts.MicroSeconds());
        }
        return *this;
    }

    template <typename T>
    TFlatSampleKeyBuilder& AddLabel(TStringBuf key, const T& value) {
        if (Options_.LabelBlacklist.contains(key)) {
            return *this;
        }

        auto& labels = Value_["labels"][key];
        if (labels.IsDefined()) {
            NJson::TJsonValue prev = std::move(labels);
            labels.AppendValue(std::move(prev));
            labels.AppendValue(value);
        } else {
            labels = value;
        }

        return *this;
    }

    TFlatSampleKeyBuilder& AddLabel(TStringBuf key, const TStringRef& value) {
        return AddLabel(key, value.View());
    }

    TFlatSampleKeyBuilder& AddFrame(
        TStringBuf buildid,
        TStringBuf path,
        ui64 offset,
        TStringBuf sourceFile = "",
        TStringBuf sourceFunction = "",
        ui32 line = 0
    ) {
        NJson::TJsonValue frame;
        if (buildid && Options_.PrintBuildIds) {
            frame["binary"]["buildid"] = buildid;
        }
        if (path) {
            frame["binary"]["path"] = path;
        }
        if (buildid && Options_.PrintAddresses) {
            frame["address"] = offset;
        }

        if (sourceFile) {
            frame["file"] = sourceFile;
        }
        frame["line"] = line;
        frame["function"] = sourceFunction;

        Value_["stack"].AppendValue(std::move(frame));
        return *this;
    }

    TString Finish() {
        return NJson::WriteJson(Value_, false, true);
    }

private:
    const TFlatDiffableProfileOptions& Options_;
    NJson::TJsonValue Value_;
};

TFlatDiffableProfile::TFlatDiffableProfile(const NProto::NProfile::Profile& profile, TFlatDiffableProfileOptions options)
    : TFlatDiffableProfile(TProfile{&profile}, options)
{}

template <typename EntityArray>
TCompactIntegerMap<ui64, ui32> EnumeratePProfEntities(EntityArray&& array) {
    TCompactIntegerMap<ui64, ui32> map;
    for (auto&& [i, entity] : Enumerate(array)) {
        map.EmplaceUnique(entity.id(), i);
    }
    return map;
}

template <typename T, typename Mapping>
const T& GetPProfField(const google::protobuf::RepeatedPtrField<T>& ref, Mapping&& m, ui32 i) {
    if (i == 0) {
        return Default<T>();
    }

    return ref.at(m.At(i));
}

TFlatDiffableProfile::TFlatDiffableProfile(const NProto::NPProf::Profile& profile, TFlatDiffableProfileOptions options) {
    TCompactIntegerMap<ui64, ui32> functions = EnumeratePProfEntities(profile.function());
    TCompactIntegerMap<ui64, ui32> mappings = EnumeratePProfEntities(profile.mapping());
    TCompactIntegerMap<ui64, ui32> locations = EnumeratePProfEntities(profile.location());

    auto str = [&](i64 id) -> TStringBuf {
        if (id == 0) {
            return "";
        }
        return profile.string_table(id);
    };

    for (auto&& sample : profile.sample()) {
        TFlatSampleKeyBuilder builder{options};

        for (auto&& label : sample.label()) {
            if (i64 value = label.num()) {
                builder.AddLabel(str(label.key()), value);
            } else {
                builder.AddLabel(str(label.key()), str(label.str()));
            }
        }

        for (ui64 id : sample.location_id()) {
            auto& location = GetPProfField(profile.location(), locations, id);
            auto& mapping = GetPProfField(profile.mapping(), mappings, location.mapping_id());
            i64 address = location.address() + (i64)mapping.file_offset() - (i64)mapping.memory_start();

            if (location.line().empty()) {
                builder.AddFrame(
                    str(mapping.build_id()),
                    str(mapping.filename()),
                    address
                );
            } else {
                for (auto&& line : location.line()) {
                    auto& function = GetPProfField(profile.function(), functions, line.function_id());

                    builder.AddFrame(
                        str(mapping.build_id()),
                        str(mapping.filename()),
                        address,
                        str(function.filename()),
                        str(function.name()),
                        line.line()
                    );
                }
            }
        }

        auto& values = Samples_[builder.Finish()];
        for (auto [i, value] : Enumerate(sample.value())) {
            TString key = TString::Join(str(profile.sample_type(i).type()), '.', str(profile.sample_type(i).unit()));
            values[std::move(key)] += value;
        }
    }
}

TFlatDiffableProfile::TFlatDiffableProfile(TProfile profile, TFlatDiffableProfileOptions options) {
    for (auto&& [i, sample] : Enumerate(profile.Samples())) {
        TFlatSampleKeyBuilder builder{options};

        if (auto ts = sample.GetInstantTimestamp()) {
            builder.SetTimestamp(*ts);
        }

        auto key = sample.GetKey();

        for (TLabel label : key.GetLabels()) {
            std::visit([&](auto&& value) {
                builder.AddLabel(label.GetKey().View(), value);
            }, label.GetValue());
        }

        {
            TThread thread = key.GetThread();
            if (auto id = thread.GetThreadId(); id != 0) {
                builder.AddLabel("tid", id);
            }
            if (auto id = thread.GetThreadName(); id != 0) {
                builder.AddLabel("thread_comm", id);
            }
            if (auto id = thread.GetProcessId(); id != 0) {
                builder.AddLabel("pid", id);
            }
            if (auto id = thread.GetProcessName(); id != 0) {
                builder.AddLabel("process_comm", id);
            }
            for (auto id : thread.GetContainers()) {
                builder.AddLabel("workload", id);
            }
        }

        for (TStack stack : key.GetStacks()) {
            auto visitFrame = [&](TStackFrame frame) {
                TInlineChain chain = frame.GetInlineChain();
                if (chain.GetLineCount() == 0) {
                    builder.AddFrame(
                        frame.GetBinary().GetBuildId().View(),
                        frame.GetBinary().GetPath().View(),
                        frame.GetBinaryOffset()
                    );

                } else {
                    for (TSourceLine line : chain.GetLines()) {
                        builder.AddFrame(
                            frame.GetBinary().GetBuildId().View(),
                            frame.GetBinary().GetPath().View(),
                            frame.GetBinaryOffset(),
                            line.GetFunction().GetFileName().View(),
                            line.GetFunction().GetName().View(),
                            line.GetLine()
                        );
                    }
                }
            };

            for (auto frame : stack.GetFrames()) {
                visitFrame(frame);
            }
        }

        auto& values = Samples_[builder.Finish()];
        for (auto [value, type] : Zip(sample.GetValues(), sample.GetValueTypes())) {
            TString key = TString::Join(type.GetType().View(), '.', type.GetUnit().View());
            values[std::move(key)] += value;
        }
    }
}

void TFlatDiffableProfile::IterateSamples(TFunctionRef<void(TStringBuf key, const TMap<TString, ui64>& values)> consumer) const {
    for (auto&& [key, values] : Samples_) {
        consumer(key, values);
    }
}

void TFlatDiffableProfile::WriteTo(IOutputStream& out) const {
    IterateSamples([&out](TStringBuf sample, const TMap<TString, ui64>& values) {
        out << sample << '\t';

        const char* sep = "";
        for (auto&& [key, value] : values) {
            out << std::exchange(sep, ",") << key << '=' << value;
        }

        out << '\n';
    });
}

} // namespace NPerforator::NProfile
