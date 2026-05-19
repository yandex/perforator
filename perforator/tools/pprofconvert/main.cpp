#include <perforator/lib/profile/merge.h>
#include <perforator/lib/profile/parallel_merge.h>
#include <perforator/lib/profile/pprof.h>
#include <perforator/lib/profile/profile.h>
#include <perforator/lib/profile/flat_diffable.h>
#include <perforator/lib/profile/validate.h>

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>
#include <library/cpp/digest/murmur/murmur.h>
#include <library/cpp/iterator/enumerate.h>
#include <library/cpp/threading/future/async.h>
#include <library/cpp/threading/future/wait/wait_group.h>
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
#include <util/string/builder.h>
#include <util/system/yassert.h>
#include <util/thread/pool.h>

#include <google/protobuf/arena.h>

#include <concepts>
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

template <typename F>
static decltype(auto) LogTime(TStringBuf before, TStringBuf after, F&& func) {
    TInstant start = Now();
    Cerr << before << Endl;

    auto result = std::invoke(std::forward<F>(func));

    TInstant end = Now();
    Cerr << after << " in " << HumanReadable(end - start) << Endl;

    return result;
}

template <std::derived_from<google::protobuf::Message> M>
M ParseProtoTimed(const TString& path) {
    TString name = M::descriptor()->full_name();
    return LogTime("Parsing " + name  + " from " + path, "Parsed " + name, [path] {
        TFileInput in{path};
        M proto;
        Y_ENSURE(proto.ParseFromArcadiaStream(&in));
        return proto;
    });
}

TString DebugDump(NPerforator::NProfile::TStackFrame frame) {
    TStringBuilder builder;

    auto chain = frame.GetInlineChain();
    for (auto line : chain.GetLines()) {
        builder << line.GetFunction().GetFileName() << ":" << line.GetFunction().GetName() << ":" << line.GetFunction().GetStartLine() << ":(" << line.GetLine() << ":" << line.GetColumn() << ')';
    }

    builder << "@<" << frame.GetBinary().GetBuildId().View();
    ui64 offset = frame.GetAddress();
    if (offset >= 0) {
        builder << '+';
    }
    builder << Hex(offset, 0) << ">";

    return builder;
}

TString DebugDump(NPerforator::NProfile::TStack stack) {
    TStringBuilder builder;
    builder << "[";
    for (auto [i, frame] : Enumerate(stack.GetFrames())) {
        if (i != 0) {
            builder << DebugDump(frame) << ",";
        }
    }
    if (builder.back() == ',') {
        builder.pop_back();
    }
    builder << ']';
    return builder;
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
        for (ui32 id : profile.sample_keys().stacks().stack_ids()) {
            if (id == 0) {
                ++zero;
            }
            ++total;
        }
        Cerr << "Found " << zero << " / " << total << " zero locations" << Endl;

        Cerr << "Parsed profile" << Endl;
        arrstats("samples.labels.labels.packed_label_ids_offset", profile.sample_keys().labels().packed_label_ids_offset());
        arrstats("samples.labels.labels.packed_label_ids", profile.sample_keys().labels().packed_label_ids());
        arrstats("samples.labels.label_group_id", profile.sample_keys().label_group_id());

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

        if (argv[1] == "merge-threaded-old"sv) {
        Y_ENSURE(argc > 3);

        const int threadCount = 10;

        TThreadPool tp;
        tp.Start(threadCount);

        TVector<NPerforator::NProto::NProfile::Profile> profiles(threadCount);

        for (int tid = 0; tid < threadCount; ++tid) {
            tp.SafeAddFunc([tid, argv, argc, &profiles] {
                NPerforator::NProfile::TProfileMerger merger{&profiles[tid]};

                NPerforator::NProto::NProfile::Profile profile;
                for (int i = 3 + tid; i < argc; i += threadCount) {
                    TFileInput in{argv[i]};
                    Y_ENSURE(profile.ParseFromArcadiaStream(&in));
                    merger.Add(profile, {});
                }

                std::move(merger).Finish();
            });
        }

        Cerr << "Waiting for the profile mergers to finish" << Endl;

        tp.Stop();

        Cerr << "Merging final profile" << Endl;

        NPerforator::NProto::NProfile::Profile merged;
        NPerforator::NProfile::TProfileMerger merger{&merged};
        for (auto& profile : profiles) {
            merger.Add(profile, {});
        }
        std::move(merger).Finish();

        TFileOutput out{argv[2]};
        merged.SerializeToArcadiaStream(&out);

        return 0;
    }

    if (argv[1] == "merge-threaded"sv) {
        Y_ENSURE(argc > 3);

        const TInstant start = Now();
        const int threadCount = 20;

        TThreadPool tp;
        tp.Start(threadCount);

        NPerforator::NProto::NProfile::Profile merged;

        NPerforator::NProfile::TParallelProfileMergerOptions mergerOptions;
        mergerOptions.MergeOptions = {};
        mergerOptions.ConcurrencyLevel = 16;
        mergerOptions.BufferSize = 20;

        NPerforator::NProfile::TParallelProfileMerger merger{&merged, mergerOptions, &tp};

        // Parallel parsing and feeding to merger
        NThreading::TWaitGroup<NThreading::TWaitPolicy::TAll> wg;
        std::atomic<int> processed = 0;
        const int total = argc - 3;

        for (int i = 3; i < argc; ++i) {
            wg.Add(NThreading::Async([&merger, &processed, i, total, argv] {
                TFileInput in{argv[i]};
                NPerforator::NProto::NProfile::Profile profile;
                Y_ENSURE(profile.ParseFromArcadiaStream(&in));
                merger.Add(std::move(profile));

                int current = processed.fetch_add(1) + 1;
                if (current % 10 == 0 || current == total) {
                    Cerr << "Parsed and queued " << current << " / " << total << " profiles" << Endl;
                }
            }, tp));
        }

        Cerr << "Waiting for parsing to complete" << Endl;

        // Wait for all parsing to complete
        std::move(wg).Finish().GetValueSync();

        Cerr << "All profiles parsed, finishing merge" << Endl;

        std::move(merger).Finish();

        TFileOutput out{argv[2]};
        merged.SerializeToArcadiaStream(&out);

        Cerr << "Finished merge in " << HumanReadable(Now() - start) << Endl;

        return 0;
    }

    if (argv[1] == "merge"sv) {
        Y_ENSURE(argc > 3);

        auto start = Now();

        NPerforator::NProto::NProfile::Profile merged;
        NPerforator::NProfile::TProfileMerger merger{&merged};

        int cnt = 0;

        int prevSize = 0;
        NPerforator::NProto::NProfile::Profile profile;
        for (int i = 3; i < argc; ++i) {
            TFileInput in{argv[i]};
            Y_ENSURE(profile.ParseFromArcadiaStream(&in));

            merger.Add(profile, {});
            Cerr << "Merged profile #" << cnt++ << Endl;

            int size = merged.stack_segments().frame_ids_offset_size();
            int delta = size - prevSize;

            Cout
                << cnt
                << '\t' << size
                << '\t' << delta
                << '\t' << Prec(size * 1.0 / cnt, PREC_POINT_DIGITS, 2)
                << '\t' << Prec(delta * 100.0 / profile.stack_segments().frame_ids_offset_size(), PREC_POINT_DIGITS, 2) << "% new stack segments"
                << Endl;
            prevSize = size;
        }

        std::move(merger).Finish();

        TFileOutput out{argv[2]};
        merged.SerializeToArcadiaStream(&out);

        Cerr << "Merged " << cnt << " profiles in " << HumanReadable(Now() - start) << Endl;

        return 0;
    }

    if (argv[1] == "dump"sv) {
        Y_ENSURE(argc == 3);

        auto proto = LogTime("Parsing profile", "Parsed profile", [argv] {
            TFileInput in{argv[2]};
            NPerforator::NProto::NProfile::Profile proto;
            Y_ENSURE(proto.ParseFromArcadiaStream(&in));
            return proto;
        });

        NPerforator::NProfile::TProfile profile{&proto};

        // Brief per-sample text dump. A richer human-readable dumper for the
        // profile format would belong in its own library, not in lib/profile
        // (which is consumed from wasm and must stay free of json deps).
        for (auto sample : profile.Samples()) {
            Cout << "sample id=" << *sample.GetIndex()
                 << " key=" << *sample.GetKey().GetIndex();
            for (auto sv : sample.GetValues()) {
                Cout << ' '
                     << sv.GetValueType().GetType().View()
                     << '.'
                     << sv.GetValueType().GetUnit().View()
                     << '=' << sv.GetValue();
            }
            Cout << Endl;
        }
    }

    if (argv[1] == "stats"sv) {
        Y_ENSURE(argc == 3);
        TFileInput in{argv[2]};
        NPerforator::NProto::NProfile::Profile proto;
        Y_ENSURE(proto.ParseFromArcadiaStream(&in));

        NPerforator::NProfile::TProfile profile{&proto};

        Cerr << "Profile has " << profile.Stacks().size() << " stacks" << Endl;
    }

    if (argv[1] == "dump-stacks"sv) {
        Y_ENSURE(argc == 3);
        TFileInput in{argv[2]};
        NPerforator::NProto::NProfile::Profile proto;
        Y_ENSURE(proto.ParseFromArcadiaStream(&in));

        NPerforator::NProfile::TProfile profile{&proto};

        Cerr << "Profile has " << profile.Stacks().size() << " stacks" << Endl;

        for (auto stack : profile.Stacks()) {
            TString key = DebugDump(stack);
            Cout << Hex(MultiHash(key)) << ": " << key << Endl;
        }
    }

    if (argv[1] == "diff-stacks"sv) {
        Y_ENSURE(argc == 4);

        NPerforator::NProto::NProfile::Profile protos[] {
            ParseProtoTimed<NPerforator::NProto::NProfile::Profile>(argv[2]),
            ParseProtoTimed<NPerforator::NProto::NProfile::Profile>(argv[3]),
        };

        NPerforator::NProfile::TProfile profiles[]{
            NPerforator::NProfile::TProfile{&protos[0]},
            NPerforator::NProfile::TProfile{&protos[1]},
        };

        auto iterateStackValues = [](const NPerforator::NProfile::TProfile& profile, auto&& consumer) {
            for (auto sample : profile.Samples()) {
                auto key = sample.GetKey();
                for (auto stack : key.GetStacks()) {
                    for (auto sampleValue : sample.GetValues()) {
                        consumer(stack, sampleValue.GetValue());
                    }
                }
            }
        };

        THashMap<TString, double> weights;
        iterateStackValues(profiles[0], [&weights](NPerforator::NProfile::TStack stack, ui64 value) {
            weights[DebugDump(stack)] += value;
        });

        ui32 commonStacks = 0;
        ui32 totalStacks = 0;
        for (auto stack : profiles[1].Stacks()) {
            TString key = DebugDump(stack);
            commonStacks += weights.contains(key);
            totalStacks += 1;
        }

        Cerr << commonStacks << " / " << totalStacks << " common stacks" << Endl;
    }

    if (argv[1] == "dump-diffable"sv) {
        Y_ENSURE(argc == 3);

        auto proto = ParseProtoTimed<NPerforator::NProto::NProfile::Profile>(argv[2]);
        NPerforator::NProfile::TProfile profile{&proto};
        NPerforator::NProfile::TFlatDiffableProfile{profile}.WriteTo(Cout);
    }

    if (argv[1] == "dump-diffable-pprof"sv) {
        Y_ENSURE(argc == 3);

        auto profile = ParseProtoTimed<NPerforator::NProto::NPProf::Profile>(argv[2]);
        NPerforator::NProfile::TFlatDiffableProfile{profile, {
            .LabelBlacklist = {"comm"},
        }}.WriteTo(Cout);
    }
}
