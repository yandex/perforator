#include "options.h"

namespace NRe2Utils {
    RE2::Options DefaultSetOptions() {
        RE2::Options opts = DefaultReOptions();
        opts.set_never_capture(true);

        return opts;
    }

    RE2::Options DefaultReOptions() {
        RE2::Options opts(RE2::DefaultOptions);
        opts.set_log_errors(false);
        opts.set_encoding(re2::RE2::Options::Encoding::EncodingLatin1);
        return opts;
    }
}
