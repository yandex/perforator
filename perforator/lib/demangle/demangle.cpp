#include "demangle.h"
#include "itanium.h"
#include "rustc.h"

#include <util/generic/deque.h>
#include <util/generic/scope.h>
#include <util/generic/yexception.h>

#include <llvm/Demangle/Demangle.h>

#include <contrib/libs/re2/re2/re2.h>


namespace NPerforator::NDemangle {

////////////////////////////////////////////////////////////////////////////////

static std::optional<std::string> OwnMallocedString(char* ptr) {
    if (!ptr) {
        return std::nullopt;
    }

    Y_DEFER {
        std::free(ptr);
    };

    return std::string{ptr};
}

// Note that this (v0) scheme is experimental, and in 2025 Rust uses "legacy"
// mangling scheme by default. Legacy scheme is based on Itanium with a few
// additions like escaping of unsupported characters. We handle such cases in
// @TryItaniumDemangle.
static std::optional<std::string> TryRustV0Demangle(std::string_view value) {
    if (!value.starts_with("_R")) {
        return std::nullopt;
    }
    return OwnMallocedString(llvm::rustDemangle(value));
}

static std::optional<std::string> TryDLangDemangle(std::string_view value) {
    if (!value.starts_with("_D")) {
        return std::nullopt;
    }
    return OwnMallocedString(llvm::dlangDemangle(value));
}

// Try to remove some well-known vendor-specific suffixes.
static std::string CleanupNonMangledName(std::string&& name) {
    static const auto toErase = [] () {
        // re2::RE2 is not movable.
        std::deque<re2::RE2> patterns;
        Y_ENSURE(patterns.emplace_back(R"(\.llvm\.[0-9a-f]+)").ok());
        Y_ENSURE(patterns.emplace_back(R"(\.isra\..*)").ok());
        return patterns;
    }();

    for (const auto& pattern : toErase) {
         Y_ENSURE(pattern.ok(), "Failed to compile regex");
         re2::RE2::Replace(&name, pattern, "");
    }

    return std::move(name);
}

// This piece of code is inspired by llvm::demangle source.
// The main problem is that we need to use low-level Itanium demangler to drop
// vendor-specific suffixes.
std::string Demangle(std::string name, NDemangle::DemangleOptions options) {
    if (auto res = NPrivate::TryItaniumDemangle(name, options)) {
        return MaybePostprocessLegacyRustSymbol(std::move(*res));
    }
    if (auto res = TryRustV0Demangle(name)) {
        return *res;
    }
    if (auto res = TryDLangDemangle(name)) {
        return *res;
    }
    if (char* ptr = llvm::microsoftDemangle(name, nullptr, nullptr)) {
        Y_DEFER {
            std::free(ptr);
        };

        return std::string{ptr};
    }

    // The symbol is non-mangled or has unknown mangling scheme.
    if (options.DropVendorSpecificSuffix) {
        return CleanupNonMangledName(std::move(name));
    } else {
        return name;
    }
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NDemangle
