#pragma once

#ifdef __GNUC__
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wunused-parameter"
#endif

/*===- TableGen'erated file -------------------------------------*- C++ -*-===*\
|*                                                                            *|
|* Intrinsic Function Source Fragment                                         *|
|*                                                                            *|
|* Automatically generated file, do not edit!                                 *|
|*                                                                            *|
\*===----------------------------------------------------------------------===*/

#ifndef LLVM_IR_INTRINSIC_R600_ENUMS_H
#define LLVM_IR_INTRINSIC_R600_ENUMS_H
namespace llvm::Intrinsic {
enum R600Intrinsics : unsigned {
// Enum values for intrinsics.
    r600_cube = 11848,                                 // llvm.r600.cube (IntrinsicsAMDGPU.td:97)
    r600_ddx,                                  // llvm.r600.ddx (IntrinsicsAMDGPU.td:145)
    r600_ddy,                                  // llvm.r600.ddy (IntrinsicsAMDGPU.td:146)
    r600_dot4,                                 // llvm.r600.dot4 (IntrinsicsAMDGPU.td:148)
    r600_group_barrier,                        // llvm.r600.group.barrier (IntrinsicsAMDGPU.td:73)
    r600_implicitarg_ptr,                      // llvm.r600.implicitarg.ptr (IntrinsicsAMDGPU.td:77)
    r600_kill,                                 // llvm.r600.kill (IntrinsicsAMDGPU.td:152)
    r600_rat_store_typed,                      // llvm.r600.rat.store.typed (IntrinsicsAMDGPU.td:82)
    r600_read_global_size_x,                   // llvm.r600.read.global.size.x (IntrinsicsAMDGPU.td:53)
    r600_read_global_size_y,                   // llvm.r600.read.global.size.y (IntrinsicsAMDGPU.td:55)
    r600_read_global_size_z,                   // llvm.r600.read.global.size.z (IntrinsicsAMDGPU.td:57)
    r600_read_local_size_x,                    // llvm.r600.read.local.size.x (IntrinsicsAMDGPU.td:46)
    r600_read_local_size_y,                    // llvm.r600.read.local.size.y (IntrinsicsAMDGPU.td:47)
    r600_read_local_size_z,                    // llvm.r600.read.local.size.z (IntrinsicsAMDGPU.td:48)
    r600_read_ngroups_x,                       // llvm.r600.read.ngroups.x (IntrinsicsAMDGPU.td:53)
    r600_read_ngroups_y,                       // llvm.r600.read.ngroups.y (IntrinsicsAMDGPU.td:55)
    r600_read_ngroups_z,                       // llvm.r600.read.ngroups.z (IntrinsicsAMDGPU.td:57)
    r600_read_tgid_x,                          // llvm.r600.read.tgid.x (IntrinsicsAMDGPU.td:53)
    r600_read_tgid_y,                          // llvm.r600.read.tgid.y (IntrinsicsAMDGPU.td:55)
    r600_read_tgid_z,                          // llvm.r600.read.tgid.z (IntrinsicsAMDGPU.td:57)
    r600_read_tidig_x,                         // llvm.r600.read.tidig.x (IntrinsicsAMDGPU.td:53)
    r600_read_tidig_y,                         // llvm.r600.read.tidig.y (IntrinsicsAMDGPU.td:55)
    r600_read_tidig_z,                         // llvm.r600.read.tidig.z (IntrinsicsAMDGPU.td:57)
    r600_recipsqrt_clamped,                    // llvm.r600.recipsqrt.clamped (IntrinsicsAMDGPU.td:93)
    r600_recipsqrt_ieee,                       // llvm.r600.recipsqrt.ieee (IntrinsicsAMDGPU.td:89)
    r600_store_stream_output,                  // llvm.r600.store.stream.output (IntrinsicsAMDGPU.td:101)
    r600_store_swizzle,                        // llvm.r600.store.swizzle (IntrinsicsAMDGPU.td:133)
    r600_tex,                                  // llvm.r600.tex (IntrinsicsAMDGPU.td:137)
    r600_texc,                                 // llvm.r600.texc (IntrinsicsAMDGPU.td:138)
    r600_txb,                                  // llvm.r600.txb (IntrinsicsAMDGPU.td:141)
    r600_txbc,                                 // llvm.r600.txbc (IntrinsicsAMDGPU.td:142)
    r600_txf,                                  // llvm.r600.txf (IntrinsicsAMDGPU.td:143)
    r600_txl,                                  // llvm.r600.txl (IntrinsicsAMDGPU.td:139)
    r600_txlc,                                 // llvm.r600.txlc (IntrinsicsAMDGPU.td:140)
    r600_txq,                                  // llvm.r600.txq (IntrinsicsAMDGPU.td:144)
}; // enum
} // namespace llvm::Intrinsic
#endif


#ifdef __GNUC__
#pragma GCC diagnostic pop
#endif
