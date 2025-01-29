#pragma once

#ifdef __cplusplus
namespace NPerforator::NThreadLocal {
#endif

enum perforator_tls_magic_bytes {
    PERFORATOR_TLS_MAGIC_BYTE_0 = 0x7e,
    PERFORATOR_TLS_MAGIC_BYTE_1 = 0x6f,
    PERFORATOR_TLS_MAGIC_BYTE_2 = 0x06,
    PERFORATOR_TLS_MAGIC_BYTE_3 = 0xa7,
    PERFORATOR_TLS_MAGIC_BYTE_4 = 0x06,
    PERFORATOR_TLS_MAGIC_BYTE_5 = 0x04,
    PERFORATOR_TLS_MAGIC_BYTE_6 = 0xa6
};

#ifdef __cplusplus
} // namespace NPerforator::NThreadLocal
#endif