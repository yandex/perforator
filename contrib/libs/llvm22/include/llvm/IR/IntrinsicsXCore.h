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

#ifndef LLVM_IR_INTRINSIC_XCORE_ENUMS_H
#define LLVM_IR_INTRINSIC_XCORE_ENUMS_H
namespace llvm::Intrinsic {
enum XCOREIntrinsics : unsigned {
// Enum values for intrinsics.
    xcore_bitrev = 16071,                              // llvm.xcore.bitrev (IntrinsicsXCore.td:15)
    xcore_checkevent,                          // llvm.xcore.checkevent (IntrinsicsXCore.td:100)
    xcore_chkct,                               // llvm.xcore.chkct (IntrinsicsXCore.td:53)
    xcore_clre,                                // llvm.xcore.clre (IntrinsicsXCore.td:102)
    xcore_clrpt,                               // llvm.xcore.clrpt (IntrinsicsXCore.td:69)
    xcore_clrsr,                               // llvm.xcore.clrsr (IntrinsicsXCore.td:36)
    xcore_crc32,                               // llvm.xcore.crc32 (IntrinsicsXCore.td:20)
    xcore_crc8,                                // llvm.xcore.crc8 (IntrinsicsXCore.td:17)
    xcore_edu,                                 // llvm.xcore.edu (IntrinsicsXCore.td:82)
    xcore_eeu,                                 // llvm.xcore.eeu (IntrinsicsXCore.td:81)
    xcore_endin,                               // llvm.xcore.endin (IntrinsicsXCore.td:91)
    xcore_freer,                               // llvm.xcore.freer (IntrinsicsXCore.td:40)
    xcore_geted,                               // llvm.xcore.geted (IntrinsicsXCore.td:33)
    xcore_getet,                               // llvm.xcore.getet (IntrinsicsXCore.td:34)
    xcore_getid,                               // llvm.xcore.getid (IntrinsicsXCore.td:27)
    xcore_getps,                               // llvm.xcore.getps (IntrinsicsXCore.td:29)
    xcore_getr,                                // llvm.xcore.getr (IntrinsicsXCore.td:39)
    xcore_getst,                               // llvm.xcore.getst (IntrinsicsXCore.td:105)
    xcore_getts,                               // llvm.xcore.getts (IntrinsicsXCore.td:71)
    xcore_in,                                  // llvm.xcore.in (IntrinsicsXCore.td:42)
    xcore_inct,                                // llvm.xcore.inct (IntrinsicsXCore.td:45)
    xcore_initcp,                              // llvm.xcore.initcp (IntrinsicsXCore.td:116)
    xcore_initdp,                              // llvm.xcore.initdp (IntrinsicsXCore.td:118)
    xcore_initlr,                              // llvm.xcore.initlr (IntrinsicsXCore.td:114)
    xcore_initpc,                              // llvm.xcore.initpc (IntrinsicsXCore.td:112)
    xcore_initsp,                              // llvm.xcore.initsp (IntrinsicsXCore.td:110)
    xcore_inshr,                               // llvm.xcore.inshr (IntrinsicsXCore.td:63)
    xcore_int,                                 // llvm.xcore.int (IntrinsicsXCore.td:43)
    xcore_mjoin,                               // llvm.xcore.mjoin (IntrinsicsXCore.td:109)
    xcore_msync,                               // llvm.xcore.msync (IntrinsicsXCore.td:107)
    xcore_out,                                 // llvm.xcore.out (IntrinsicsXCore.td:47)
    xcore_outct,                               // llvm.xcore.outct (IntrinsicsXCore.td:51)
    xcore_outshr,                              // llvm.xcore.outshr (IntrinsicsXCore.td:65)
    xcore_outt,                                // llvm.xcore.outt (IntrinsicsXCore.td:49)
    xcore_peek,                                // llvm.xcore.peek (IntrinsicsXCore.td:89)
    xcore_setc,                                // llvm.xcore.setc (IntrinsicsXCore.td:61)
    xcore_setclk,                              // llvm.xcore.setclk (IntrinsicsXCore.td:83)
    xcore_setd,                                // llvm.xcore.setd (IntrinsicsXCore.td:59)
    xcore_setev,                               // llvm.xcore.setev (IntrinsicsXCore.td:79)
    xcore_setps,                               // llvm.xcore.setps (IntrinsicsXCore.td:31)
    xcore_setpsc,                              // llvm.xcore.setpsc (IntrinsicsXCore.td:87)
    xcore_setpt,                               // llvm.xcore.setpt (IntrinsicsXCore.td:67)
    xcore_setrdy,                              // llvm.xcore.setrdy (IntrinsicsXCore.td:85)
    xcore_setsr,                               // llvm.xcore.setsr (IntrinsicsXCore.td:35)
    xcore_settw,                               // llvm.xcore.settw (IntrinsicsXCore.td:75)
    xcore_setv,                                // llvm.xcore.setv (IntrinsicsXCore.td:77)
    xcore_sext,                                // llvm.xcore.sext (IntrinsicsXCore.td:23)
    xcore_ssync,                               // llvm.xcore.ssync (IntrinsicsXCore.td:108)
    xcore_syncr,                               // llvm.xcore.syncr (IntrinsicsXCore.td:73)
    xcore_testct,                              // llvm.xcore.testct (IntrinsicsXCore.td:55)
    xcore_testwct,                             // llvm.xcore.testwct (IntrinsicsXCore.td:57)
    xcore_waitevent,                           // llvm.xcore.waitevent (IntrinsicsXCore.td:95)
    xcore_zext,                                // llvm.xcore.zext (IntrinsicsXCore.td:25)
}; // enum
} // namespace llvm::Intrinsic
#endif


#ifdef __GNUC__
#pragma GCC diagnostic pop
#endif
