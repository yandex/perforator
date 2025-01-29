#include <perforator/symbolizer/lib/symbolize/symbolizer.h>

#include <library/cpp/testing/common/env.h>
#include <library/cpp/testing/gtest/gtest.h>

TEST(Symbolization, ElfObjectFile) {
    NPerforator::NSymbolize::TCodeSymbolizer symbolizer;

    THashMap<uint64_t, TVector<TString>> addressToFunctionNames = {
        {0x1345, TVector<TString>{"main"}},
        {0x1294, TVector<TString>{"foo(int)"}},
        {0x127a, TVector<TString>{"bar(int)"}},
        {0x1366, TVector<TString>{"main", "boo(int)"}}
    };

    TString moduleName = ArcadiaSourceRoot() + "/perforator/symbolizer/tests/sample_program.elf";

    for (const auto& [addr, functions]: addressToFunctionNames) {
        auto inlinedFunctions = symbolizer.Symbolize(moduleName, addr);
        ASSERT_EQ(functions.size(), inlinedFunctions.size());

        for (size_t i = 0; i < inlinedFunctions.size(); ++i) {
            ASSERT_EQ(NPerforator::NSymbolize::DemangleFunctionName(inlinedFunctions[i].FunctionName), functions[i]);
        }
    }
}

TEST(Symbolization, ProfileProto) {
    NPerforator::NSymbolize::TProfileSymbolizer symbolizer;

    NPerforator::NProto::NPProf::Profile profile;

    auto addLocation = [&](uint64_t address) -> uint64_t {
        NPerforator::NProto::NPProf::Location* loc = profile.add_location();
        loc->set_address(address);
        loc->set_id(profile.location_size() - 1);
        return loc->id();
    };

    auto addSample = [&](const TVector<uint64_t>& callstackAddresses) {
        NPerforator::NProto::NPProf::Sample* sample = profile.add_sample();

        for (uint64_t addr: callstackAddresses) {
            uint64_t locId = addLocation(addr);
            sample->add_location_id(locId);
        }
    };

    auto addMapping = [&](const TString& path, uint64_t start, uint64_t end, uint64_t fileOffset) -> uint64_t {
        NPerforator::NProto::NPProf::Mapping* mapping = profile.add_mapping();
        mapping->set_id(profile.mapping_size() - 1);
        mapping->set_file_offset(fileOffset);
        mapping->set_memory_start(start);
        mapping->set_memory_limit(end);
        profile.add_string_table(path);
        mapping->set_filename(profile.string_table_size() - 1);

        return mapping->id();
    };

    addSample({0x401201, 0x401340});
    addSample({0x7ffff7fc51b0, 0x401350});
    addMapping(ArcadiaSourceRoot() + "/perforator/symbolizer/tests/sample_program.elf", 0x400000, 0x405000, 0x0);
    addMapping(ArcadiaSourceRoot() + "/perforator/symbolizer/tests/libsample.so.elf", 0x7ffff7fc4000, 0x7ffff7fc9000, 0x0);

    TVector<TVector<TVector<TString>>> answers = {
        {
            {
                "std::basic_ostream<char, std::char_traits<char> >& std::operator<<<std::char_traits<char> >(std::basic_ostream<char, std::char_traits<char> >&, char const*)",
                "bar(int)"
            },
            {
                "boo(int)",
                "main"
            }
        },
        {
            {
                "bbbb(unsigned long)"
            },
            {
                "main"
            }
        },
    };

    symbolizer.Symbolize(profile);

    THashMap<uint64_t, const NPerforator::NProto::NPProf::Location*> locations;

    for (size_t i = 0; i < profile.locationSize(); ++i) {
        locations[profile.location(i).id()] = &profile.location(i);
    }

    THashMap<uint64_t, const NPerforator::NProto::NPProf::Function*> functions;

    for (size_t i = 0; i < profile.functionSize(); ++i) {
        functions[profile.function(i).id()] = &profile.function(i);
    }

    for (size_t i = 0; i < profile.sampleSize(); ++i) {
        auto sample = profile.sample(i);
        for (size_t j = 0; j < sample.location_idSize(); ++j) {
            auto loc = locations[sample.location_id(j)];

            for (size_t k = 0; k < loc->lineSize(); ++k) {
                auto func = functions[loc->line(k).function_id()];
                ASSERT_EQ(profile.string_table(func->name()), answers[i][j][k]);
            }
        }
    }
}

TEST(Symbolization, FunctionNameDemanglingAndPruning) {
    auto check = [](std::string dirtyMangled, std::string expected) {
        const auto cleanedUpDemangled = NPerforator::NSymbolize::DemangleFunctionName(
            NPerforator::NSymbolize::CleanupFunctionName(std::move(dirtyMangled))
        );
        ASSERT_EQ(expected, cleanedUpDemangled);
    };

    check("_Z3foov", "foo()");
    check("_ZN4llvm3foo3bar3bazEv", "llvm::foo::bar::baz()");
    check("_ZN4llvm4llvmEid", "llvm::llvm(int, double)");
    check("_ZN4llvm4llvmEid.llvm.123456", "llvm::llvm(int, double)");
    check("ZSTD_decodeLiteralsBlock.llvm.8240084484405978173", "ZSTD_decodeLiteralsBlock");
}
