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

#ifndef LLVM_IR_INTRINSIC_BPF_ENUMS_H
#define LLVM_IR_INTRINSIC_BPF_ENUMS_H
namespace llvm::Intrinsic {
enum BPFIntrinsics : unsigned {
// Enum values for intrinsics.
    bpf_btf_type_id = 4187,                           // llvm.bpf.btf.type.id (IntrinsicsBPF.td:26)
    bpf_compare,                               // llvm.bpf.compare (IntrinsicsBPF.td:37)
    bpf_getelementptr_and_load,                // llvm.bpf.getelementptr.and.load (IntrinsicsBPF.td:40)
    bpf_getelementptr_and_store,               // llvm.bpf.getelementptr.and.store (IntrinsicsBPF.td:59)
    bpf_load_byte,                             // llvm.bpf.load.byte (IntrinsicsBPF.td:15)
    bpf_load_half,                             // llvm.bpf.load.half (IntrinsicsBPF.td:17)
    bpf_load_word,                             // llvm.bpf.load.word (IntrinsicsBPF.td:19)
    bpf_passthrough,                           // llvm.bpf.passthrough (IntrinsicsBPF.td:35)
    bpf_preserve_enum_value,                   // llvm.bpf.preserve.enum.value (IntrinsicsBPF.td:32)
    bpf_preserve_field_info,                   // llvm.bpf.preserve.field.info (IntrinsicsBPF.td:23)
    bpf_preserve_type_info,                    // llvm.bpf.preserve.type.info (IntrinsicsBPF.td:29)
    bpf_pseudo,                                // llvm.bpf.pseudo (IntrinsicsBPF.td:21)
}; // enum
} // namespace llvm::Intrinsic
#endif


#ifdef __GNUC__
#pragma GCC diagnostic pop
#endif
