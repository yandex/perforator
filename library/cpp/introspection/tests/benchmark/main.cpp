#include <library/cpp/testing/gbenchmark/benchmark.h>
#include <library/cpp/introspection/hash_ops.h>
#include <library/cpp/introspection/tests/data.h>
#include <util/string/cast.h>

Y_GENERATE_T_HASH_AND_EQUALS(NIntrospectionTest, TChar)
Y_GENERATE_T_HASH_AND_EQUALS(NIntrospectionTest, TCompositePrimitive)
Y_GENERATE_T_HASH_AND_EQUALS(NIntrospectionTest, THuge)

namespace {
    using TChar = NIntrospectionTest::TChar;
    using TCompositePrimitive = NIntrospectionTest::TCompositePrimitive;
    using THuge = NIntrospectionTest::THuge;
}

static void SmallHash(benchmark::State& state) {
    ui64 bigNum = 1e10;

    TCompositePrimitive primitive;

    for (auto _ : state) {
        bigNum += 4000;

        primitive.M1 = bigNum % std::numeric_limits<char>::max();
        primitive.M2 = bigNum % std::numeric_limits<ui32>::max();
        primitive.M3 = bigNum / 2;
        primitive.M4 = ToString(bigNum);
        primitive.M5 = bigNum % std::numeric_limits<i64>::max();
        primitive.M6 = bigNum / 1e5;
        primitive.M7.M1 = bigNum % std::numeric_limits<char>::max();

        THash<TCompositePrimitive>()(primitive);
    }
}
BENCHMARK(SmallHash);

static void HugeHash(benchmark::State& state) {
    ui64 bigNum = 1e10;

    THuge huge;

    for (auto _ : state) {
        bigNum += 4000;

        huge.M1 = bigNum / 2;
        huge.M10 = bigNum / 3;
        huge.M21 = bigNum / 4;
        huge.M34 = bigNum / 5;
        huge.M45 = bigNum / 6;
        huge.M50 = bigNum / 7;
        huge.M66 = bigNum / 8;
        huge.M72 = bigNum / 9;
        huge.M83 = bigNum / 10;
        huge.M97 = bigNum / 11;
        huge.M100 = bigNum / 12;

        THash<THuge>()(huge);
    }
}
BENCHMARK(HugeHash);

static void SmallEquals(benchmark::State& state) {
    ui64 bigNum = 1e10;

    TCompositePrimitive primitive;
    const TCompositePrimitive something {
        .M1 = 1,
        .M2 = 2,
        .M3 = 3,
        .M4 = "fff",
        .M5 = 5,
        .M6 = 6,
        .M7 = {
            .M1 = 7
        }
    };

    for (auto _ : state) {
        bigNum += 4000;

        primitive.M1 = bigNum % std::numeric_limits<char>::max();
        primitive.M2 = bigNum % std::numeric_limits<ui32>::max();
        primitive.M3 = bigNum / 2;
        primitive.M4 = ToString(bigNum);
        primitive.M5 = bigNum % std::numeric_limits<i64>::max();
        primitive.M6 = bigNum / 1e5;
        primitive.M7.M1 = bigNum % std::numeric_limits<char>::max();

        Y_UNUSED(primitive == something);
    }
}
BENCHMARK(SmallEquals);

static void HugeEquals(benchmark::State& state) {
    ui64 bigNum = 1e10;

    THuge huge;
    const THuge something = std::invoke([]() {
        THuge result;

        result.M1 = 326;
        result.M75 = 4;
        result.M100 = 90;

        return result;
    });

    for (auto _ : state) {
        bigNum += 4000;

        huge.M1 = bigNum / 2;
        huge.M10 = bigNum / 3;
        huge.M21 = bigNum / 4;
        huge.M34 = bigNum / 5;
        huge.M45 = bigNum / 6;
        huge.M50 = bigNum / 7;
        huge.M66 = bigNum / 8;
        huge.M72 = bigNum / 9;
        huge.M83 = bigNum / 10;
        huge.M97 = bigNum / 11;
        huge.M100 = bigNum / 12;

        Y_UNUSED(huge == something);
    }
}
BENCHMARK(HugeEquals);

BENCHMARK_MAIN();
