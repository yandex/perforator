#include "cython.h"

#include <contrib/libs/icu/common/punycode.h>
#include <library/cpp/containers/stack_vector/stack_vec.h>
#include <util/charset/utf8.h>
#include <util/system/compiler.h>
#include <unicode/uchar.h>
#include <unicode/ustring.h>

#include <algorithm>
#include <charconv>
#include <utility>

namespace NPerforator::NDemangle::NPrivate {

    [[nodiscard]] static bool IsCythonIdentifierStart(std::string_view value) {
        if (value.empty()) {
            return false;
        }
        const auto* p = reinterpret_cast<const unsigned char*>(value.data());
        const auto* end = p + value.size();
        wchar32 rune;
        if (ReadUTF8CharAndAdvance(rune, p, end) != RECODE_OK) {
            return false;
        }
        if (rune < 128) {
            return rune == '_' || (rune >= 'a' && rune <= 'z') || (rune >= 'A' && rune <= 'Z');
        }
        return u_hasBinaryProperty(rune, UCHAR_XID_START);
    }

    [[nodiscard]] static bool HasCythonIdentifierCharacters(std::string_view value) {
        const auto* p = reinterpret_cast<const unsigned char*>(value.data());
        const auto* end = p + value.size();
        while (p < end) {
            wchar32 rune;
            if (ReadUTF8CharAndAdvance(rune, p, end) != RECODE_OK) {
                return false;
            }
            if (rune < 128) {
                if (rune != '_' && !(rune >= 'a' && rune <= 'z') &&
                    !(rune >= 'A' && rune <= 'Z') && !(rune >= '0' && rune <= '9')) {
                    return false;
                }
            } else if (!u_hasBinaryProperty(rune, UCHAR_XID_CONTINUE)) {
                return false;
            }
        }
        return true;
    }

    [[nodiscard]] static std::optional<std::size_t> ConsumeCythonNumber(std::string_view& value) {
        if (value.empty() || value.front() == '0') {
            return std::nullopt;
        }
        std::size_t number = 0;
        const auto [end, error] = std::from_chars(value.data(), value.data() + value.size(), number);
        if (error != std::errc{}) {
            return std::nullopt;
        }
        value.remove_prefix(end - value.data());
        return number;
    }

    [[nodiscard]] static std::optional<std::string_view> ReadCythonScope(std::string_view& value) {
        auto remainder = value;
        const auto parsedLength = ConsumeCythonNumber(remainder);
        if (!parsedLength.has_value()) {
            return std::nullopt;
        }
        const auto* start = reinterpret_cast<const unsigned char*>(remainder.data());
        const auto* p = start;
        const auto* end = start + remainder.size();
        for (std::size_t i = 0; i < *parsedLength; ++i) {
            wchar32 rune;
            if (p == end || ReadUTF8CharAndAdvance(rune, p, end) != RECODE_OK) {
                return std::nullopt;
            }
        }
        const auto length = static_cast<std::size_t>(p - start);
        if (p == end || *p != '_' || !IsCythonIdentifierStart(remainder) ||
            !HasCythonIdentifierCharacters(remainder.substr(0, length))) {
            return std::nullopt;
        }
        value = remainder.substr(length + 1);
        return remainder.substr(0, length);
    }

    [[nodiscard]] static std::string_view FindCythonFunction(std::string_view value, bool hasDiscriminator) {
        if (!ReadCythonScope(value)) {
            return {};
        }
        std::string_view function;
        while (!value.empty()) {
            auto candidate = value;
            if (hasDiscriminator && !IsCythonIdentifierStart(candidate)) {
                if (!ConsumeCythonNumber(candidate).has_value()) {
                    break;
                }
            }
            if (!candidate.empty() && IsCythonIdentifierStart(candidate)) {
                // A scope and a function discriminator can encode different names identically.
                if (!function.empty()) {
                    return {};
                }
                function = candidate;
            }
            if (!ReadCythonScope(value)) {
                break;
            }
        }
        return function;
    }

    [[nodiscard]] static std::optional<std::string> DecodeCythonScope(std::string_view value) {
        std::string decoded;
        bool found = false;
        while (!value.empty()) {
            const auto component = ReadCythonScope(value);
            if (!component) {
                break;
            }
            if (!decoded.empty()) {
                decoded += '.';
            }
            decoded += *component;
            if (!value.empty() && IsCythonIdentifierStart(value) &&
                HasCythonIdentifierCharacters(value)) {
                if (found) {
                    return std::nullopt;
                }
                found = true;
            }
        }
        if (!found || value.empty() || !IsCythonIdentifierStart(value) ||
            !HasCythonIdentifierCharacters(value)) {
            return std::nullopt;
        }
        decoded += '.';
        decoded += value;
        return decoded;
    }

    // Keep Unicode scratch buffers out of the ordinary-name stack frame.
    [[nodiscard]] static Y_NO_INLINE std::optional<std::string> DecodeCythonUnicode(std::string_view value) {
        constexpr std::u16string_view prefix = u"__pyx_";
        const auto suffix = value.substr(sizeof("__pyx_U") - 1);
        // ICU rejects Punycode longer than 2000 characters; avoid allocating its scratch buffers.
        constexpr size_t maxPunycodeLength = 2000;
        if (suffix.size() > maxPunycodeLength - prefix.size()) {
            return std::nullopt;
        }
        TStackVec<char16_t, 64> punycode(prefix.begin(), prefix.end());
        punycode.insert(punycode.end(), suffix.begin(), suffix.end());
        *std::find(punycode.rbegin(), punycode.rend(), u'_') = u'-';
        TStackVec<char16_t, 64> wide(punycode.size());
        UErrorCode error = U_ZERO_ERROR;
        const auto wideLength = u_strFromPunycode(punycode.data(), punycode.size(),
                                                  wide.data(), wide.size(), nullptr, &error);
        if (U_FAILURE(error)) {
            return std::nullopt;
        }
        wide.resize(wideLength);
        if (std::none_of(wide.begin(), wide.end(), [](char16_t c) { return c >= 128; })) {
            return std::nullopt;
        }
        if (wide.size() <= 64) {
            // ICU checks decode overflow; below its guaranteed encoder limit, only digit case can differ (RFC 3492).
            const auto digits = std::find(punycode.rbegin(), punycode.rend(), u'-').base();
            if (std::any_of(digits, punycode.end(), [](char16_t c) { return c >= u'A' && c <= u'Z'; })) {
                return std::nullopt;
            }
        } else {
            TStackVec<char16_t, 64> canonical(punycode.size());
            error = U_ZERO_ERROR;
            const auto canonicalLength = u_strToPunycode(wide.data(), wide.size(),
                                                         canonical.data(), canonical.size(), nullptr, &error);
            if (U_FAILURE(error)) {
                return std::nullopt;
            }
            canonical.resize(canonicalLength);
            if (canonical != punycode) {
                return std::nullopt;
            }
        }
        std::string decoded(wide.size() * 3, char{});
        int32_t decodedLength = 0;
        error = U_ZERO_ERROR;
        u_strToUTF8(decoded.data(), decoded.size(), &decodedLength,
                    wide.data(), wide.size(), &error);
        if (U_FAILURE(error)) {
            return std::nullopt;
        }
        decoded.resize(decodedLength);
        if (decoded.starts_with("__pyx_U") || decoded == value) {
            return std::nullopt;
        }
        return decoded;
    }

    [[nodiscard]] static std::optional<std::string> DemangleCythonGenerator(std::string_view value) {
        const auto separator = value.rfind('_');
        if (separator == std::string_view::npos) {
            return std::nullopt;
        }
        const auto identity = value.substr(separator + 1);
        const auto marker = identity.find("generator");
        if (marker == std::string_view::npos || marker == 0) {
            return std::nullopt;
        }
        for (char c : identity.substr(0, marker)) {
            if (c < '0' || c > '9') {
                return std::nullopt;
            }
        }
        for (char c : identity.substr(marker + sizeof("generator") - 1)) {
            if (c < '0' || c > '9') {
                return std::nullopt;
            }
        }
        auto scopes = value.substr(0, separator + 1);
        std::string result;
        while (!scopes.empty()) {
            const auto component = ReadCythonScope(scopes);
            if (!component) {
                return std::nullopt;
            }
            result += *component;
            result += '.';
        }
        if (result.empty()) {
            return std::nullopt;
        }
        result += identity;
        result += " [generator body]";
        return result;
    }

    [[nodiscard]] static std::optional<std::string> DemangleCythonQualifiedName(std::string_view value) {
        static constexpr std::pair<std::string_view, std::string_view> qualifiedPrefixes[] = {
            {"__pyx_convert__from_py_struct____pyx_t_", "from_py"},
            {"__pyx_convert__to_py_struct____pyx_t_", "to_py"},
            {"__pyx_convert__from_py_union____pyx_t_", "from_py"},
            {"__pyx_convert__to_py_union____pyx_t_", "to_py"},
            {"__pyx_getprop_", "get"},
            {"__pyx_setprop_", "set"},
            {"__pyx_tp_new__initialisation_", "tp_new__initialisation"},
            {"__pyx_tp_new_vectorcall_", "tp_new_vectorcall"},
            {"__pyx_tp_vectorcall_", "tp_vectorcall"},
            {"__pyx_tp_new_", "tp_new"},
            {"__pyx_tp_dealloc_", "tp_dealloc"},
            {"__pyx_tp_traverse_", "tp_traverse"},
            {"__pyx_tp_clear_", "tp_clear"},
            {"__pyx_tp_finalize_", "tp_finalize"},
            {"__pyx_tp_getattro_", "tp_getattro"},
            {"__pyx_tp_setattro_", "tp_setattro"},
            {"__pyx_tp_richcompare_", "tp_richcompare"},
            {"__pyx_tp_descr_get_", "tp_descr_get"},
            {"__pyx_tp_descr_set_", "tp_descr_set"},
            {"__pyx_tp_init_", "tp_init"},
            {"__pyx_tp_repr_", "tp_repr"},
            {"__pyx_tp_str_", "tp_str"},
            {"__pyx_tp_hash_", "tp_hash"},
            {"__pyx_tp_call_", "tp_call"},
            {"__pyx_tp_iter_", "tp_iter"},
            {"__pyx_tp_iternext_", "tp_iternext"},
            {"__pyx_sq_item_", "sq_item"},
            {"__pyx_sq_length_", "sq_length"},
            {"__pyx_sq_concat_", "sq_concat"},
            {"__pyx_sq_repeat_", "sq_repeat"},
            {"__pyx_sq_contains_", "sq_contains"},
            {"__pyx_sq_ass_item_", "sq_ass_item"},
            {"__pyx_mp_length_", "mp_length"},
            {"__pyx_mp_subscript_", "mp_subscript"},
            {"__pyx_mp_ass_subscript_", "mp_ass_subscript"},
            {"__pyx_nb_add_", "nb_add"},
            {"__pyx_nb_subtract_", "nb_subtract"},
            {"__pyx_nb_multiply_", "nb_multiply"},
            {"__pyx_nb_remainder_", "nb_remainder"},
            {"__pyx_nb_divmod_", "nb_divmod"},
            {"__pyx_nb_power_", "nb_power"},
            {"__pyx_nb_negative_", "nb_negative"},
            {"__pyx_nb_positive_", "nb_positive"},
            {"__pyx_nb_absolute_", "nb_absolute"},
            {"__pyx_nb_bool_", "nb_bool"},
            {"__pyx_nb_invert_", "nb_invert"},
            {"__pyx_nb_lshift_", "nb_lshift"},
            {"__pyx_nb_rshift_", "nb_rshift"},
            {"__pyx_nb_and_", "nb_and"},
            {"__pyx_nb_xor_", "nb_xor"},
            {"__pyx_nb_or_", "nb_or"},
            {"__pyx_nb_int_", "nb_int"},
            {"__pyx_nb_float_", "nb_float"},
            {"__pyx_nb_index_", "nb_index"},
            {"__pyx_nb_inplace_add_", "nb_inplace_add"},
            {"__pyx_nb_inplace_subtract_", "nb_inplace_subtract"},
            {"__pyx_nb_inplace_multiply_", "nb_inplace_multiply"},
            {"__pyx_nb_inplace_remainder_", "nb_inplace_remainder"},
            {"__pyx_nb_inplace_power_", "nb_inplace_power"},
            {"__pyx_nb_inplace_lshift_", "nb_inplace_lshift"},
            {"__pyx_nb_inplace_rshift_", "nb_inplace_rshift"},
            {"__pyx_nb_inplace_and_", "nb_inplace_and"},
            {"__pyx_nb_inplace_xor_", "nb_inplace_xor"},
            {"__pyx_nb_inplace_or_", "nb_inplace_or"},
            {"__pyx_nb_floor_divide_", "nb_floor_divide"},
            {"__pyx_nb_true_divide_", "nb_true_divide"},
            {"__pyx_nb_inplace_floor_divide_", "nb_inplace_floor_divide"},
            {"__pyx_nb_inplace_true_divide_", "nb_inplace_true_divide"},
            {"__pyx_nb_matrix_multiply_", "nb_matrix_multiply"},
            {"__pyx_nb_inplace_matrix_multiply_", "nb_inplace_matrix_multiply"},
        };
        for (const auto& [prefix, label] : qualifiedPrefixes) {
            if (value.starts_with(prefix)) {
                if (auto scope = DecodeCythonScope(value.substr(prefix.size()))) {
                    scope->append(" [");
                    scope->append(label);
                    scope->append("]");
                    return scope;
                }
                return std::nullopt;
            }
        }
        return std::nullopt;
    }

    [[nodiscard]] static std::optional<std::string> DemangleCythonFunction(std::string_view value) {
        if (value.starts_with("__pyx_fuse_")) {
            value.remove_prefix(sizeof("__pyx_fuse_") - 1);
            while (true) {
                const auto end = value.find('_');
                if (end == std::string_view::npos || end == 0 ||
                    (end > 1 && value.front() == '0')) {
                    return std::nullopt;
                }
                for (char c : value.substr(0, end)) {
                    if (c < '0' || c > '9') {
                        return std::nullopt;
                    }
                }
                value.remove_prefix(end);
                if (value.starts_with("__pyx_")) {
                    break;
                }
                value.remove_prefix(1);
            }
        }
        static constexpr std::pair<std::string_view, bool> prefixes[] = {
            {"__pyx_pf_", true},
            {"__pyx_pw_", true},
            {"__pyx_f_", false},
        };
        for (const auto& [prefix, hasDiscriminator] : prefixes) {
            if (!value.starts_with(prefix)) {
                continue;
            }
            value.remove_prefix(prefix.size());
            if (!HasCythonIdentifierCharacters(value)) {
                return std::nullopt;
            }
            const auto function = FindCythonFunction(value, hasDiscriminator);
            if (!function.empty()) {
                return std::string{function};
            }
            return std::nullopt;
        }
        return std::nullopt;
    }

    [[nodiscard]] std::optional<std::string> TryCythonDemangle(std::string_view value) {
        if (!value.starts_with("__pyx_") && !value.starts_with("___pyx_")) {
            return std::nullopt;
        }
        if (value.starts_with("___pyx_")) {
            value.remove_prefix(1);
        }
        if (value.starts_with("__pyx_U")) {
            if (auto decoded = DecodeCythonUnicode(value)) {
                return TryCythonDemangle(*decoded);
            }
            return std::nullopt;
        }
        if (value.starts_with("__pyx_specialmethod_")) {
            value.remove_prefix(sizeof("__pyx_specialmethod_") - 1);
            if (!value.starts_with("__pyx_pw_")) {
                return std::nullopt;
            }
            return TryCythonDemangle(value);
        }
        if (value.starts_with("__pyx_lambda_funcdef_")) {
            value.remove_prefix(sizeof("__pyx_lambda_funcdef_") - 1);
            if (auto scope = DecodeCythonScope(value)) {
                return *scope + " [lambda]";
            }
            if (IsCythonIdentifierStart(value) && HasCythonIdentifierCharacters(value)) {
                return std::string{value} + " [lambda]";
            }
            return std::nullopt;
        }
        if (value.starts_with("__pyx_pymod_exec_")) {
            value.remove_prefix(sizeof("__pyx_pymod_exec_") - 1);
            if (IsCythonIdentifierStart(value) && HasCythonIdentifierCharacters(value)) {
                return std::string{value} + " [module exec]";
            }
            return std::nullopt;
        }
        if (value.starts_with("__pyx_gb_")) {
            return DemangleCythonGenerator(value.substr(sizeof("__pyx_gb_") - 1));
        }
        if (auto function = DemangleCythonFunction(value)) {
            return function;
        }
        return DemangleCythonQualifiedName(value);
    }

} // namespace NPerforator::NDemangle::NPrivate
