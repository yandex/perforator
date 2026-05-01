#pragma once
#include "error.h"

#include <functional>

#include <security/ant-secret/internal/string_utils/common.h>
#include <google/protobuf/io/coded_stream.h>


namespace NAnt::NProtobuf {
namespace {
    enum WireType: int {
        WIRETYPE_VARINT = 0,
        WIRETYPE_FIXED64 = 1,
        WIRETYPE_LENGTH_DELIMITED = 2,
        WIRETYPE_START_GROUP = 3,
        WIRETYPE_END_GROUP = 4,
        WIRETYPE_FIXED32 = 5,
    };
}

class TVisitor {
public:
    using TConsumer = std::function<void(size_t offset, TStringBuf value)>;

    explicit TVisitor(TConsumer consumer)
        : Consumer_(consumer)
    {}

    TError VisitWire(TStringBuf in) {
        return this->VisitWire(reinterpret_cast<const uint8_t*>(in.data()), in.size());
    }

    TError VisitWire(const uint8_t* data, size_t size) {
        google::protobuf::io::CodedInputStream input(data, static_cast<int>(size));
        return this->VisitMessage(input, 0);
    }

private:
    TError VisitMessage(google::protobuf::io::CodedInputStream& input, int offset) {
        while (input.BytesUntilLimit() > 0) {
            ui32 tag = input.ReadTag();
            if (tag == 0) {
                return TError::UnexpectedEOF;
            }

            ui32 wireType = tag & 0x07;
            switch (wireType) {
                case WIRETYPE_LENGTH_DELIMITED: {
                    ui32 length;
                    if (!input.ReadVarint32(&length)) {
                        return TError::MalformedVarint;
                    }

                    if (length == 0) {
                        break;
                    }

                    size_t curOffset = offset + input.CurrentPosition();

                    const void* ptr = nullptr;
                    int len = 0;
                    if (!input.GetDirectBufferPointer(&ptr, &len) || len < static_cast<int>(length)) {
                        return TError::BufferOverflow;
                    }

                    TStringBuf data(static_cast<const char*>(ptr), length);
                    input.Skip(length);

                    // Try to recurse: if parsing succeeds, it's an embedded message
                    google::protobuf::io::CodedInputStream sub(reinterpret_cast<const uint8_t*>(data.data()), length);
                    auto err = this->VisitMessage(sub, curOffset);
                    // TODO(buglloc): Do we really need to check UTF-8? Is this really critical for our case?
                    if (err != TError::None && NStringUtils::IsValidUTF8(data)) {
                        // If it fails: treat it as string
                        this->Consumer_(curOffset, data);
                    }

                    break;
                }
                case WIRETYPE_VARINT: {
                    ui64 dummy;
                    if (!input.ReadVarint64(&dummy)) {
                        return TError::MalformedVarint;
                    }
                    break;
                }
                case WIRETYPE_FIXED32: {
                    ui32 dummy;
                    if (!input.ReadLittleEndian32(&dummy)) {
                        return TError::MalformedVarint;
                    }
                    break;
                }
                case WIRETYPE_FIXED64: {
                    ui64 dummy;
                    if (!input.ReadLittleEndian64(&dummy)) {
                        return TError::MalformedVarint;
                    }
                    break;
                }
                case WIRETYPE_START_GROUP: {
                    // TODO(buglloc): support groups?
                    return TError::InvalidWireType;
                }
                case WIRETYPE_END_GROUP: {
                    // TODO(buglloc): support groups?
                    return TError::InvalidWireType;
                }
                default:
                    return TError::InvalidWireType;
            }
        }

        return TError::None;
    }

private:
    TConsumer Consumer_;
};

} // namespace NAnt::NProtobuf
