#pragma once

#include <util/generic/strbuf.h>

namespace NHelpDesk {
    bool IsRobotPassword(const TStringBuf token);

    bool IsRobotPasswordV2(const TStringBuf token);
}
