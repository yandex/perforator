GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.0.3-0.20250322232337-35a7c28c31ee)

SRCS(
    go_above_118.go
    go_above_19.go
    reflect2.go
    reflect2_kind.go
    relfect2_mips64x.s
    relfect2_mipsx.s
    relfect2_ppc64x.s
    safe_field.go
    safe_map.go
    safe_slice.go
    safe_struct.go
    safe_type.go
    type_map.go
    unsafe_array.go
    unsafe_eface.go
    unsafe_field.go
    unsafe_iface.go
    unsafe_link.go
    unsafe_map.go
    unsafe_ptr.go
    unsafe_slice.go
    unsafe_struct.go
    unsafe_type.go
)

IF (ARCH_X86_64)
    SRCS(
        reflect2_amd64.s
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        relfect2_arm64.s
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        relfect2_arm.s
    )
ENDIF()

END()
