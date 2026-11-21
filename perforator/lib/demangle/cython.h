#pragma once

#include <optional>
#include <string>
#include <string_view>

namespace NPerforator::NDemangle::NPrivate {

    std::optional<std::string> TryCythonDemangle(std::string_view value);

} // namespace NPerforator::NDemangle::NPrivate
