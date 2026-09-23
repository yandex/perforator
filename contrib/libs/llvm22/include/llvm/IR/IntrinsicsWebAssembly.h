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

#ifndef LLVM_IR_INTRINSIC_WASM_ENUMS_H
#define LLVM_IR_INTRINSIC_WASM_ENUMS_H
namespace llvm::Intrinsic {
enum WASMIntrinsics : unsigned {
// Enum values for intrinsics.
    wasm_alltrue = 14310,                              // llvm.wasm.alltrue (IntrinsicsWebAssembly.td:221)
    wasm_anytrue,                              // llvm.wasm.anytrue (IntrinsicsWebAssembly.td:217)
    wasm_avgr_unsigned,                        // llvm.wasm.avgr.unsigned (IntrinsicsWebAssembly.td:209)
    wasm_bitmask,                              // llvm.wasm.bitmask (IntrinsicsWebAssembly.td:225)
    wasm_bitselect,                            // llvm.wasm.bitselect (IntrinsicsWebAssembly.td:213)
    wasm_catch,                                // llvm.wasm.catch (IntrinsicsWebAssembly.td:147)
    wasm_dot,                                  // llvm.wasm.dot (IntrinsicsWebAssembly.td:229)
    wasm_extadd_pairwise_signed,               // llvm.wasm.extadd.pairwise.signed (IntrinsicsWebAssembly.td:257)
    wasm_extadd_pairwise_unsigned,             // llvm.wasm.extadd.pairwise.unsigned (IntrinsicsWebAssembly.td:261)
    wasm_extract_lane_f16x8,                   // llvm.wasm.extract.lane.f16x8 (IntrinsicsWebAssembly.td:358)
    wasm_get_ehselector,                       // llvm.wasm.get.ehselector (IntrinsicsWebAssembly.td:140)
    wasm_get_exception,                        // llvm.wasm.get.exception (IntrinsicsWebAssembly.td:138)
    wasm_landingpad_index,                     // llvm.wasm.landingpad.index (IntrinsicsWebAssembly.td:154)
    wasm_loadf16_f32,                          // llvm.wasm.loadf16.f32 (IntrinsicsWebAssembly.td:344)
    wasm_lsda,                                 // llvm.wasm.lsda (IntrinsicsWebAssembly.td:159)
    wasm_memory_atomic_notify,                 // llvm.wasm.memory.atomic.notify (IntrinsicsWebAssembly.td:179)
    wasm_memory_atomic_wait32,                 // llvm.wasm.memory.atomic.wait32 (IntrinsicsWebAssembly.td:167)
    wasm_memory_atomic_wait64,                 // llvm.wasm.memory.atomic.wait64 (IntrinsicsWebAssembly.td:173)
    wasm_memory_grow,                          // llvm.wasm.memory.grow (IntrinsicsWebAssembly.td:24)
    wasm_memory_size,                          // llvm.wasm.memory.size (IntrinsicsWebAssembly.td:22)
    wasm_narrow_signed,                        // llvm.wasm.narrow.signed (IntrinsicsWebAssembly.td:234)
    wasm_narrow_unsigned,                      // llvm.wasm.narrow.unsigned (IntrinsicsWebAssembly.td:238)
    wasm_pmax,                                 // llvm.wasm.pmax (IntrinsicsWebAssembly.td:252)
    wasm_pmin,                                 // llvm.wasm.pmin (IntrinsicsWebAssembly.td:248)
    wasm_q15mulr_sat_signed,                   // llvm.wasm.q15mulr.sat.signed (IntrinsicsWebAssembly.td:243)
    wasm_ref_is_null_exn,                      // llvm.wasm.ref.is_null.exn (IntrinsicsWebAssembly.td:42)
    wasm_ref_is_null_extern,                   // llvm.wasm.ref.is_null.extern (IntrinsicsWebAssembly.td:36)
    wasm_ref_is_null_func,                     // llvm.wasm.ref.is_null.func (IntrinsicsWebAssembly.td:39)
    wasm_ref_null_exn,                         // llvm.wasm.ref.null.exn (IntrinsicsWebAssembly.td:34)
    wasm_ref_null_extern,                      // llvm.wasm.ref.null.extern (IntrinsicsWebAssembly.td:30)
    wasm_ref_null_func,                        // llvm.wasm.ref.null.func (IntrinsicsWebAssembly.td:32)
    wasm_ref_test_func,                        // llvm.wasm.ref.test.func (IntrinsicsWebAssembly.td:46)
    wasm_relaxed_dot_bf16x8_add_f32,           // llvm.wasm.relaxed.dot.bf16x8.add.f32 (IntrinsicsWebAssembly.td:333)
    wasm_relaxed_dot_i8x16_i7x16_add_signed,   // llvm.wasm.relaxed.dot.i8x16.i7x16.add.signed (IntrinsicsWebAssembly.td:328)
    wasm_relaxed_dot_i8x16_i7x16_signed,       // llvm.wasm.relaxed.dot.i8x16.i7x16.signed (IntrinsicsWebAssembly.td:323)
    wasm_relaxed_laneselect,                   // llvm.wasm.relaxed.laneselect (IntrinsicsWebAssembly.td:279)
    wasm_relaxed_madd,                         // llvm.wasm.relaxed.madd (IntrinsicsWebAssembly.td:270)
    wasm_relaxed_max,                          // llvm.wasm.relaxed.max (IntrinsicsWebAssembly.td:293)
    wasm_relaxed_min,                          // llvm.wasm.relaxed.min (IntrinsicsWebAssembly.td:289)
    wasm_relaxed_nmadd,                        // llvm.wasm.relaxed.nmadd (IntrinsicsWebAssembly.td:274)
    wasm_relaxed_q15mulr_signed,               // llvm.wasm.relaxed.q15mulr.signed (IntrinsicsWebAssembly.td:318)
    wasm_relaxed_swizzle,                      // llvm.wasm.relaxed.swizzle (IntrinsicsWebAssembly.td:284)
    wasm_relaxed_trunc_signed,                 // llvm.wasm.relaxed.trunc.signed (IntrinsicsWebAssembly.td:298)
    wasm_relaxed_trunc_signed_zero,            // llvm.wasm.relaxed.trunc.signed.zero (IntrinsicsWebAssembly.td:308)
    wasm_relaxed_trunc_unsigned,               // llvm.wasm.relaxed.trunc.unsigned (IntrinsicsWebAssembly.td:303)
    wasm_relaxed_trunc_unsigned_zero,          // llvm.wasm.relaxed.trunc.unsigned.zero (IntrinsicsWebAssembly.td:313)
    wasm_replace_lane_f16x8,                   // llvm.wasm.replace.lane.f16x8 (IntrinsicsWebAssembly.td:362)
    wasm_rethrow,                              // llvm.wasm.rethrow (IntrinsicsWebAssembly.td:134)
    wasm_shuffle,                              // llvm.wasm.shuffle (IntrinsicsWebAssembly.td:193)
    wasm_splat_f16x8,                          // llvm.wasm.splat.f16x8 (IntrinsicsWebAssembly.td:354)
    wasm_storef16_f32,                         // llvm.wasm.storef16.f32 (IntrinsicsWebAssembly.td:349)
    wasm_swizzle,                              // llvm.wasm.swizzle (IntrinsicsWebAssembly.td:189)
    wasm_table_copy,                           // llvm.wasm.table.copy (IntrinsicsWebAssembly.td:76)
    wasm_table_fill_exnref,                    // llvm.wasm.table.fill.exnref (IntrinsicsWebAssembly.td:97)
    wasm_table_fill_externref,                 // llvm.wasm.table.fill.externref (IntrinsicsWebAssembly.td:89)
    wasm_table_fill_funcref,                   // llvm.wasm.table.fill.funcref (IntrinsicsWebAssembly.td:93)
    wasm_table_get_exnref,                     // llvm.wasm.table.get.exnref (IntrinsicsWebAssembly.td:69)
    wasm_table_get_externref,                  // llvm.wasm.table.get.externref (IntrinsicsWebAssembly.td:63)
    wasm_table_get_funcref,                    // llvm.wasm.table.get.funcref (IntrinsicsWebAssembly.td:66)
    wasm_table_grow_exnref,                    // llvm.wasm.table.grow.exnref (IntrinsicsWebAssembly.td:86)
    wasm_table_grow_externref,                 // llvm.wasm.table.grow.externref (IntrinsicsWebAssembly.td:80)
    wasm_table_grow_funcref,                   // llvm.wasm.table.grow.funcref (IntrinsicsWebAssembly.td:83)
    wasm_table_set_exnref,                     // llvm.wasm.table.set.exnref (IntrinsicsWebAssembly.td:59)
    wasm_table_set_externref,                  // llvm.wasm.table.set.externref (IntrinsicsWebAssembly.td:53)
    wasm_table_set_funcref,                    // llvm.wasm.table.set.funcref (IntrinsicsWebAssembly.td:56)
    wasm_table_size,                           // llvm.wasm.table.size (IntrinsicsWebAssembly.td:74)
    wasm_throw,                                // llvm.wasm.throw (IntrinsicsWebAssembly.td:132)
    wasm_tls_align,                            // llvm.wasm.tls.align (IntrinsicsWebAssembly.td:377)
    wasm_tls_base,                             // llvm.wasm.tls.base (IntrinsicsWebAssembly.td:382)
    wasm_tls_size,                             // llvm.wasm.tls.size (IntrinsicsWebAssembly.td:372)
    wasm_trunc_saturate_signed,                // llvm.wasm.trunc.saturate.signed (IntrinsicsWebAssembly.td:118)
    wasm_trunc_saturate_unsigned,              // llvm.wasm.trunc.saturate.unsigned (IntrinsicsWebAssembly.td:121)
    wasm_trunc_signed,                         // llvm.wasm.trunc.signed (IntrinsicsWebAssembly.td:107)
    wasm_trunc_unsigned,                       // llvm.wasm.trunc.unsigned (IntrinsicsWebAssembly.td:110)
}; // enum
} // namespace llvm::Intrinsic
#endif


#ifdef __GNUC__
#pragma GCC diagnostic pop
#endif
