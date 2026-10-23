#pragma once

// Canonical language identifiers shared by the BPF wire ABI and the Go/C++
// runtime resolver ABI. Keep the values in this header only.
enum {
    PERFORATOR_LANGUAGE_PYTHON = 0,
    PERFORATOR_LANGUAGE_PHP = 1,
    PERFORATOR_LANGUAGE_JVM = 2,
    PERFORATOR_LANGUAGE_LUA = 3,
    PERFORATOR_LANGUAGE_COUNT = 4,
};
