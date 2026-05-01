#pragma once

#include <util/datetime/base.h>
#include <util/stream/printf.h>
#include <util/stream/str.h>
#include <util/string/escape.h>

namespace NSecurityHelpers {

enum ELogPriority {
    TLOG_EMERG = 0 /* "EMERG" */,
    TLOG_ALERT = 1 /* "ALERT" */,
    TLOG_CRIT = 2 /* "CRITICAL_INFO" */,
    TLOG_ERR = 3 /* "ERROR" */,
    TLOG_WARNING = 4 /* "WARNING" */,
    TLOG_NOTICE = 5 /* "NOTICE" */,
    TLOG_INFO = 6 /* "INFO" */,
    TLOG_DEBUG = 7 /* "DEBUG" */,
};

class TLogger : public TNonCopyable {
public:
    explicit TLogger(IOutputStream& out, ELogPriority level = TLOG_INFO, bool colorized = false)
        : level(level), out(out), colorized(colorized) {
    }

    static TLogger& Instance(ELogPriority level = TLOG_INFO) {
        bool colorized = false;
#if defined(OS_LINUX)
        colorized = isatty(STDERR_FILENO) == 1;
#endif
        static TLogger logger(Cerr, level, colorized);
        return logger;
    }

    bool Colorized() {
        return colorized;
    }

    void ForceColorized() {
        colorized = true;
    }

    void Write(ELogPriority prio, const TString& message) {
        if (prio > level) {
            return;
        }

        ui32 color = 0;
        if (colorized) {
            color = LogLevelToColor(prio);
        }

        TStringStream stream;
        if (color > 0) {
            stream
                << "\x1b[" << color << "m" << LogLevelToString(prio) << "\x1b[0m";
        } else {
            stream
                << LogLevelToString(prio);
        }

        stream
            << "[" << Now().FormatLocalTime("%Y-%m-%d %H:%M:%S") << "] ";

        stream
            << message;
        if (message.size() < 39) {
            for (size_t i = 0; i < 39 - message.size(); ++i) {
                stream << " ";
            }
        }
        stream << " ";

        stream << Endl;
        out << stream.Str();
    }

    template<typename Tk, typename Tv, typename... Args>
    void Write(ELogPriority prio, const TString& message, const Tk& key, const Tv& val, const Args& ... args) {
        if (prio > level) {
            return;
        }

        ui32 color = 0;
        if (colorized) {
            color = LogLevelToColor(prio);
        }

        TStringStream stream;
        if (color > 0) {
            stream
                << "\x1b[" << color << "m" << LogLevelToString(prio) << "\x1b[0m";
        } else {
            stream
                << LogLevelToString(prio);
        }

        stream
            << "[" << Now().FormatLocalTime("%Y-%m-%d %H:%M:%S") << "] ";

        stream
            << message;
        if (message.size() < 39) {
            for (size_t i = 0; i < 39 - message.size(); ++i) {
                stream << " ";
            }
        }
        stream << " ";

        PrintArgTo(stream, color, key, val, args...);
        stream << Endl;
        out << stream.Str();
    }

public:
    ELogPriority level;

protected:
    IOutputStream& out;
    bool colorized;

protected:
    template<typename Tk, typename Tv>
    void PrintArgTo(IOutputStream& out, ui32 color, const Tk& key, const Tv& val) {
        if (!key || !val) {
            return;
        }

        if (color > 0) {
            out
                << "\x1b[" << color << "m" << key << "\x1b[0m=\"" << EscapeC(val) << "\" ";
        } else {
            out
                << key << "=\"" << EscapeC(val) << "\" ";
        }
    }

    template<typename Tk, typename Tv, typename... Args>
    void PrintArgTo(IOutputStream& out, ui32 color, const Tk& key, const Tv& val, const Args& ... args) {
        PrintArgTo(out, color, key, val);
        PrintArgTo(out, color, args...);
    }

    inline TString LogLevelToString(ELogPriority prio) {
        switch (prio) {
            case TLOG_EMERG:
                return "EMRG";
            case TLOG_ALERT:
                return "ALRT";
            case TLOG_CRIT:
                return "CRIT";
            case TLOG_ERR:
                return "EROR";
            case TLOG_WARNING:
                return "WARN";
            case TLOG_NOTICE:
                return "NOTE";
            case TLOG_INFO:
                return "INFO";
            case TLOG_DEBUG:
                return "DBUG";
            default:
                return "XXXX";
        }
    }

    inline ui32 LogLevelToColor(ELogPriority prio) {
        switch (prio) {
            case TLOG_EMERG:
                return 35;
            case TLOG_ALERT:
                return 35;
            case TLOG_CRIT:
                return 35;
            case TLOG_ERR:
                return 31;
            case TLOG_WARNING:
                return 33;
            case TLOG_NOTICE:
                return 32;
            case TLOG_INFO:
                return 32;
            case TLOG_DEBUG:
                return 36;
            default:
                return 0;
        }
    }

};

template<typename... Args>
static void LogDebug(const TString& message, Args... args) {
    auto& instance = TLogger::Instance();
    instance.Write(TLOG_DEBUG, message, args...);
}

template<typename... Args>
static void LogInfo(const TString& message, Args... args) {
    auto& instance = TLogger::Instance();
    if (TLOG_INFO > instance.level) {
        return;
    }
    instance.Write(TLOG_INFO, message, args...);
}

template<typename... Args>
static void LogWarn(const TString& message, Args... args) {
    auto& instance = TLogger::Instance();
    if (TLOG_WARNING > instance.level) {
        return;
    }
    instance.Write(TLOG_WARNING, message, args...);
}

template<typename... Args>
static void LogErr(const TString& message, Args... args) {
    auto& instance = TLogger::Instance();
    if (TLOG_ERR > instance.level) {
        return;
    }
    instance.Write(TLOG_ERR, message, args...);
}

}  // namespace NSecurityHelpers
