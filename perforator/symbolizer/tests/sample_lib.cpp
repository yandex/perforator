#include "sample_lib.h"

// clang++ -shared -O3 -fPIC -g -o libsample.so sample_lib.cpp

size_t aaaa(size_t val) {
    if (val % 5 == 2) {
        return 90;
    }

    return 42;
}

size_t __attribute((noinline)) bbbb(size_t x) {
    if (x % 3 == 0) {
        return aaaa(x);
    }

    return 42;
}
