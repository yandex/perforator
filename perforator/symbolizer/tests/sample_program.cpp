#include "sample_lib.h"

// clang++ -g -O3 sample_program.cpp -o sample_program.elf -L. -lsample -Wl,-rpath,/home/pashaguskov/arcadia/perforator/symbolizer/tests

bool __attribute__ ((noinline)) bar(int x) {
    std::cout << "bar(" << x << ")" << std::endl;

    return (x % 2 == 0);
}

bool __attribute__ ((noinline)) foo(int val) {
    std::cout << "foo(" << val << ")" << std::endl;

    return bar(val);
}

inline bool boo(int val) {
    if (val % 3 == 0) {
        return foo(val);
    }

    return bar(val);
}

int main() {
    for (size_t i = 0; ; ++i) {
        foo(i);
        boo(i);
        aaaa(i);
        bbbb(i);
    }

    return 0;
}
