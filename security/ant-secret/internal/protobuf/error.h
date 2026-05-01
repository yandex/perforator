#pragma once

#include <util/string/builder.h>


namespace NAnt::NProtobuf {

class TError {
public:
    enum class ECode {
        None = 0,
        BufferOverflow,
        MalformedVarint,
        InvalidWireType,
        UnexpectedEOF
    };

public:
    TError(ECode code) : Code(code) {}

    bool operator==(const TError& other) const {
        return this->Code == other.Code;
    };

    bool operator==(const ECode& other) const {
        return this->Code == other;
    };

    ECode Code{ECode::None};

    static constexpr ECode None = ECode::None;
    static constexpr ECode BufferOverflow = ECode::BufferOverflow;
    static constexpr ECode MalformedVarint = ECode::MalformedVarint;
    static constexpr ECode InvalidWireType = ECode::InvalidWireType;
    static constexpr ECode UnexpectedEOF = ECode::UnexpectedEOF;
};

inline const char* ToString(TError::ECode code) {
    switch (code) {
        case TError::ECode::None:
            return "None";
        case TError::ECode::BufferOverflow:
            return "BufferOverflow";
        case TError::ECode::MalformedVarint:
            return "MalformedVarint";
        case TError::ECode::InvalidWireType:
            return "InvalidWireType";
        case TError::ECode::UnexpectedEOF:
           return "UnexpectedEOF";
        default:
            return "Unknown";
    }
}

IOutputStream& operator<<(IOutputStream& out, const TError& err) {
    return out << "ProtoError{" << ToString(err.Code) << "}";
}

} // namespace NAnt::NProtobuf
