#pragma once

#include <string>


namespace NPerforator::NDemangle {

// Apply Rust "legacy" extensions to the Itanium mangling scheme.
std::string MaybePostprocessLegacyRustSymbol(std::string&& str);

} // namespace NPerforator::NDemangle
