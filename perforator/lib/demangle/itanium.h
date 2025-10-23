#pragma once

#include "demangle.h"

#include <optional>
#include <string>


namespace NPerforator::NDemangle::NPrivate {

std::optional<std::string> TryItaniumDemangle(std::string_view str, DemangleOptions options);

} // namespace NPerforator::NDemangle::NPrivate
