#pragma once

#include <string>

namespace NPerforator::NDemangle {

////////////////////////////////////////////////////////////////////////////////

struct DemangleOptions {
    // Itanium ABI supports vendor-specific suffixes in the mangled names
    // like .llvm.some-hash or .isra.1234. Such names can lead to unnecessary
    // diffs in performance profiles. So this option allows to omit them.
    // See https://itanium-cxx-abi.github.io/cxx-abi/abi.html#mangling-structure.
    bool DropVendorSpecificSuffix = true;
};

std::string Demangle(std::string name, DemangleOptions options = {});

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NDemangle
