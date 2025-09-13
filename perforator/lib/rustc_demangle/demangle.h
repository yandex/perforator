#pragma once

#include <string>


namespace NPerforator::NDemangle {

std::string MaybeDemangleRustcName(std::string&& str);

} // namespace NPerforator::NDemangle
