#include <perforator/lib/profile/merge.h>
#include <perforator/lib/profile/pprof.h>
#include <perforator/lib/profile/profile.h>
#include <perforator/lib/profile/validate.h>

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>
#include <library/cpp/digest/murmur/murmur.h>
#include <library/cpp/iterator/enumerate.h>
#include <library/cpp/yt/compact_containers/compact_vector.h>

#include <util/datetime/base.h>
#include <util/digest/city.h>
#include <util/digest/multi.h>
#include <util/generic/bitops.h>
#include <util/generic/function_ref.h>
#include <util/generic/hash_set.h>
#include <util/generic/size_literals.h>
#include <util/stream/file.h>
#include <util/stream/format.h>
#include <util/stream/input.h>
#include <util/stream/zlib.h>
#include <util/system/yassert.h>
#include <util/thread/pool.h>

#include <google/protobuf/arena.h>

#include <type_traits>


template <typename Range>
size_t CountBits(Range&& range) {
    ui64 mask = 0;

    for (std::integral auto x : range) {
        mask |= static_cast<ui64>(x);
    }

    size_t count = 0;
    while (mask) {
        count += mask & 1;
        mask >>= 1;
    }

    return count;
}

bool FilterWellKnownHighCardinalityLabels(const NPerforator::NProfile::TLabel& label) {
    TStringBuf key = label.GetKey().View();
    return !key.StartsWith("tls:") && !key.StartsWith("cgroup");
}

int main(int argc, const char* argv[]) {
    Y_ENSURE(argc > 1);

    if (argv[1] == "convert"sv) {
        auto start = Now();

        Y_ENSURE(argc == 4);
        TFileInput in{argv[2]};
        TFileOutput out{argv[3]};

        NPerforator::NProto::NPProf::Profile oldp;
        Y_ENSURE(oldp.ParseFromArcadiaStream(&in));
        Cerr << "Parsed profile with strtab of size " << oldp.string_table_size() << " in " << HumanReadable(Now() - start) << Endl;

        NPerforator::NProto::NProfile::Profile newp;
        NPerforator::NProfile::ConvertFromPProf(oldp, &newp);

        Y_ENSURE(newp.SerializeToArcadiaStream(&out));

        Cerr << "Converted profile in " << HumanReadable(Now() - start) << Endl;

        return 0;
    }

    if (argv[1] == "bench-convert"sv) {
        Y_ENSURE(argc == 4);
        while (true) {
            TFileInput in{argv[2]};
            TFileOutput out{argv[3]};

            auto start = Now();

            NPerforator::NProto::NPProf::Profile oldp;
            Y_ENSURE(oldp.ParseFromArcadiaStream(&in));

            NPerforator::NProto::NProfile::Profile newp;
            NPerforator::NProfile::ConvertFromPProf(oldp, &newp);

            Y_ENSURE(newp.SerializeToArcadiaStream(&out));

            auto end = Now();

            Cout << "Converted profile in " << HumanReadable(end - start) << Endl;
        }

        return 0;
    }

    if (argv[1] == "validate"sv) {
        Y_ENSURE(argc == 3);

        TFileInput in{argv[2]};

        NPerforator::NProto::NProfile::Profile profile;
        Y_ENSURE(profile.ParseFromArcadiaStream(&in));

        NPerforator::NProfile::ValidateProfile(profile, {
            .CheckIndices = false,
        });

        NPerforator::NProfile::ValidateProfile(profile, {
            .CheckIndices = true,
        });

        return 0;
    }

    if (argv[1] == "convert-old"sv) {
        Y_ENSURE(argc == 4);
        TFileInput in{argv[2]};
        TFileOutput out{argv[3]};

        NPerforator::NProto::NProfile::Profile newp;
        Y_ENSURE(newp.ParseFromArcadiaStream(&in));

        NPerforator::NProto::NPProf::Profile oldp;
        NPerforator::NProfile::ConvertToPProf(newp, &oldp);

        Y_ENSURE(oldp.SerializeToArcadiaStream(&out));
        return 0;
    }

    if (argv[1] == "bulk-convert"sv) {
        TThreadPool pool;
        pool.Start(20);

        std::atomic<int> processed = 0;
        for (int i = 2; i < argc; ++i) {
            pool.SafeAddFunc([i, argv, argc, &processed] {
                try {
                    TFileInput in{argv[i]};
                    TFileOutput out{TString{argv[i]} + ".new"};

                    google::protobuf::Arena arena;
                    auto* oldp = arena.CreateMessage<NPerforator::NProto::NPProf::Profile>(&arena);
                    Y_ENSURE(oldp->ParseFromArcadiaStream(&in));

                    auto* newp = arena.CreateMessage<NPerforator::NProto::NProfile::Profile>(&arena);
                    NPerforator::NProfile::ConvertFromPProf(*oldp, newp);
                    Y_ENSURE(newp->SerializeToArcadiaStream(&out));

                    Cerr << "Processed " << processed.fetch_add(1) + 1 << " / " << argc - 2 << " profiles" << Endl;
                } catch (...) {
                    Cerr << "Failed to convert profile " << i << ": " << CurrentExceptionMessage() << Endl;
                }
            });
        }

        pool.Stop();

        return 0;
    }

    if (argv[1] == "parse-new"sv) {
        auto start = Now();

        Y_ENSURE(argc == 3);
        TFileInput in{argv[2]};
        NPerforator::NProto::NProfile::Profile profile;
        Y_ENSURE(profile.ParseFromArcadiaStream(&in));

        Cerr << "Parsed profile in " << HumanReadable(Now() - start) << Endl;

        auto arrstats = [](const char* name, auto&& arr) {
            Cerr << name << " size: " << arr.size() << "\n";
            auto bits = CountBits(arr);
            Cerr << "\t" << bits << Endl;
            /*
            for (auto [i, count] : Enumerate(bits)) {
                if (count) {
                    Cerr << "\t" << i << ": " << count << Endl;
                }
            }
            */
        };

        ui64 total = 0;
        ui64 zero = 0;
        for (ui32 id : profile.sample_keys().stacks().user_stack_id()) {
            if (id == 0) {
                ++zero;
            }
            ++total;
        }
        Cerr << "Found " << zero << " / " << total << " zero locations" << Endl;
        for (ui32 id : profile.sample_keys().stacks().kernel_stack_id()) {
            if (id == 0) {
                ++zero;
            }
            ++total;
        }
        Cerr << "Found " << zero << " / " << total << " zero locations" << Endl;

        Cerr << "Parsed profile" << Endl;
        arrstats("samples.labels.first_label_id", profile.sample_keys().labels().first_label_id());
        arrstats("samples.labels.packed_label_id", profile.sample_keys().labels().packed_label_id());
        arrstats("stacks.frame_id", profile.stacks().frame_id());

        volatile bool loop = true;
        while (loop) {
        }

        return 0;
    }

    if (argv[1] == "parse-old"sv) {
        auto start = Now();

        Y_ENSURE(argc == 3);
        TFileInput in{argv[2]};
        NPerforator::NProto::NPProf::Profile profile;
        Y_ENSURE(profile.ParseFromArcadiaStream(&in));

        Cerr << "Parsed profile in " << HumanReadable(Now() - start) << Endl;

        volatile bool loop = true;
        while (loop) {
        }

        return 0;
    }

    if (argv[1] == "merge-threaded"sv) {
        Y_ENSURE(argc > 3);

        const int threadCount = 10;

        TThreadPool tp;
        tp.Start(threadCount);

        TVector<NPerforator::NProto::NProfile::Profile> profiles(threadCount);

        for (int tid = 0; tid < threadCount; ++tid) {
            tp.SafeAddFunc([tid, argv, argc, &profiles] {
                NPerforator::NProfile::TProfileMerger merger{&profiles[tid], {
                    .KeepProcesses = false,
                    .KeepTimestamps = false,
                    .LabelFilter = FilterWellKnownHighCardinalityLabels,
                }};

                NPerforator::NProto::NProfile::Profile profile;
                for (int i = 3 + tid; i < argc; i += threadCount) {
                    TFileInput in{argv[i]};
                    Y_ENSURE(profile.ParseFromArcadiaStream(&in));
                    merger.Add(profile);
                }

                merger.Finish();
            });
        }

        Cerr << "Waiting for the profile mergers to finish" << Endl;

        tp.Stop();

        Cerr << "Merging final profile" << Endl;

        NPerforator::NProto::NProfile::Profile merged;
        NPerforator::NProfile::TProfileMerger merger{&merged, {
            .KeepProcesses = false,
            .KeepTimestamps = false,
            .LabelFilter = FilterWellKnownHighCardinalityLabels,
        }};
        for (auto& profile : profiles) {
            merger.Add(profile);
        }
        merger.Finish();

        TFileOutput out{argv[2]};
        merged.SerializeToArcadiaStream(&out);

        return 0;
    }

    if (argv[1] == "merge"sv) {
        Y_ENSURE(argc > 3);

        auto start = Now();

        NPerforator::NProto::NProfile::Profile merged;
        NPerforator::NProfile::TProfileMerger merger{&merged, {
            .KeepProcesses = false,
            .KeepTimestamps = false,
            .LabelFilter = FilterWellKnownHighCardinalityLabels,
        }};

        int cnt = 0;

        NPerforator::NProto::NProfile::Profile profile;
        for (int i = 3; i < argc; ++i) {
            TFileInput in{argv[i]};
            Y_ENSURE(profile.ParseFromArcadiaStream(&in));

            merger.Add(profile);
            Cerr << "Merged profile #" << cnt++ << Endl;
        }

        merger.Finish();

        TFileOutput out{argv[2]};
        merged.SerializeToArcadiaStream(&out);

        Cerr << "Merged " << cnt << " profiles in " << HumanReadable(Now() - start) << Endl;

        return 0;
    }

    if (argv[1] == "check"sv) {
        Y_ENSURE(argc == 3);
        TFileInput in{argv[2]};
        NPerforator::NProto::NProfile::Profile profile;
        Y_ENSURE(profile.ParseFromArcadiaStream(&in));

        NPerforator::NProfile::TProfile prof{&profile};

        for (auto sample : prof.Samples()) {
            if (sample.GetValue(0) == 0) {
                continue;
            }

            auto stack = sample.GetKey().GetKernelStack();
            if (stack.GetStackFrameCount() < 1) {
                continue;
            }

            for (i32 i = 0; i < sample.GetKey().GetLabelCount(); ++i) {
                auto label = sample.GetKey().GetLabel(i);

                Cout << "{" << label.GetKey() << ":";
                if (label.IsNumber()) {
                    Cout << label.GetNumber();
                } else {
                    Cout << '"' << label.GetString() << '"';
                }
                Cout << "}";
            }

            for (i32 i = 0; i < stack.GetStackFrameCount(); ++i) {
                auto inlining = stack.GetStackFrame(i).GetInlineChain();
                Cout << "[";
                for (i32 i = 0; i < inlining.GetLineCount(); ++i) {
                    Cout << *inlining.GetLine(i).GetFunction().GetIndex() << ":" << inlining.GetLine(i).GetFunction().GetName() << ',';
                }
                Cout << "]";
            }
            Cout << "\n";

            continue;

            auto address = stack.GetStackFrame(0);
            if (address.GetInlineChain().GetLineCount() < 1) {
                continue;
            }

            auto name = address.GetInlineChain().GetLine(0).GetFunction().GetName();
            Cout << name << Endl;
        }
    }

    if (argv[1] == "dump"sv) {
        Y_ENSURE(argc == 3);
        TFileInput in{argv[2]};
        NPerforator::NProto::NProfile::Profile profile;
        Y_ENSURE(profile.ParseFromArcadiaStream(&in));

        NPerforator::NProfile::TProfile prof{&profile};

        NJson::TJsonWriter writer{&Cout, false};

        writer.OpenMap();
        writer.WriteKey("samples");

        writer.OpenArray();
        for (auto sample : prof.Samples()) {
            sample.DumpJson(writer);
        }
        writer.CloseArray();

        writer.CloseMap();
    }
}
