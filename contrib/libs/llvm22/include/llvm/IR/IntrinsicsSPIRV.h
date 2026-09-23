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

#ifndef LLVM_IR_INTRINSIC_SPV_ENUMS_H
#define LLVM_IR_INTRINSIC_SPV_ENUMS_H
namespace llvm::Intrinsic {
enum SPVIntrinsics : unsigned {
// Enum values for intrinsics.
    spv_all = 12936,                                   // llvm.spv.all (IntrinsicsSPIRV.td:81)
    spv_alloca,                                // llvm.spv.alloca (IntrinsicsSPIRV.td:41)
    spv_alloca_array,                          // llvm.spv.alloca.array (IntrinsicsSPIRV.td:42)
    spv_any,                                   // llvm.spv.any (IntrinsicsSPIRV.td:82)
    spv_assign_aliasing_decoration,            // llvm.spv.assign.aliasing.decoration (IntrinsicsSPIRV.td:206)
    spv_assign_decoration,                     // llvm.spv.assign.decoration (IntrinsicsSPIRV.td:19)
    spv_assign_fpmaxerror_decoration,          // llvm.spv.assign.fpmaxerror.decoration (IntrinsicsSPIRV.td:209)
    spv_assign_name,                           // llvm.spv.assign.name (IntrinsicsSPIRV.td:18)
    spv_assign_ptr_type,                       // llvm.spv.assign.ptr.type (IntrinsicsSPIRV.td:17)
    spv_assign_type,                           // llvm.spv.assign.type (IntrinsicsSPIRV.td:16)
    spv_assume,                                // llvm.spv.assume (IntrinsicsSPIRV.td:47)
    spv_bitcast,                               // llvm.spv.bitcast (IntrinsicsSPIRV.td:34)
    spv_cmpxchg,                               // llvm.spv.cmpxchg (IntrinsicsSPIRV.td:39)
    spv_const_composite,                       // llvm.spv.const.composite (IntrinsicsSPIRV.td:33)
    spv_cross,                                 // llvm.spv.cross (IntrinsicsSPIRV.td:83)
    spv_ddx,                                   // llvm.spv.ddx (IntrinsicsSPIRV.td:139)
    spv_ddx_coarse,                            // llvm.spv.ddx.coarse (IntrinsicsSPIRV.td:141)
    spv_ddx_fine,                              // llvm.spv.ddx.fine (IntrinsicsSPIRV.td:143)
    spv_ddy,                                   // llvm.spv.ddy (IntrinsicsSPIRV.td:140)
    spv_ddy_coarse,                            // llvm.spv.ddy.coarse (IntrinsicsSPIRV.td:142)
    spv_ddy_fine,                              // llvm.spv.ddy.fine (IntrinsicsSPIRV.td:144)
    spv_degrees,                               // llvm.spv.degrees (IntrinsicsSPIRV.td:84)
    spv_discard,                               // llvm.spv.discard (IntrinsicsSPIRV.td:138)
    spv_distance,                              // llvm.spv.distance (IntrinsicsSPIRV.td:85)
    spv_dot4add_i8packed,                      // llvm.spv.dot4add.i8packed (IntrinsicsSPIRV.td:118)
    spv_dot4add_u8packed,                      // llvm.spv.dot4add.u8packed (IntrinsicsSPIRV.td:119)
    spv_expect,                                // llvm.spv.expect (IntrinsicsSPIRV.td:48)
    spv_extractelt,                            // llvm.spv.extractelt (IntrinsicsSPIRV.td:31)
    spv_extractv,                              // llvm.spv.extractv (IntrinsicsSPIRV.td:29)
    spv_faceforward,                           // llvm.spv.faceforward (IntrinsicsSPIRV.td:86)
    spv_fdot,                                  // llvm.spv.fdot (IntrinsicsSPIRV.td:106)
    spv_firstbitlow,                           // llvm.spv.firstbitlow (IntrinsicsSPIRV.td:175)
    spv_firstbitshigh,                         // llvm.spv.firstbitshigh (IntrinsicsSPIRV.td:174)
    spv_firstbituhigh,                         // llvm.spv.firstbituhigh (IntrinsicsSPIRV.td:173)
    spv_flattened_thread_id_in_group,          // llvm.spv.flattened.thread.id.in.group (IntrinsicsSPIRV.td:80)
    spv_frac,                                  // llvm.spv.frac (IntrinsicsSPIRV.td:87)
    spv_fwidth,                                // llvm.spv.fwidth (IntrinsicsSPIRV.td:145)
    spv_generic_cast_to_ptr_explicit,          // llvm.spv.generic.cast.to.ptr.explicit (IntrinsicsSPIRV.td:212)
    spv_gep,                                   // llvm.spv.gep (IntrinsicsSPIRV.td:26)
    spv_global_offset,                         // llvm.spv.global.offset (IntrinsicsSPIRV.td:68)
    spv_global_size,                           // llvm.spv.global.size (IntrinsicsSPIRV.td:67)
    spv_group_id,                              // llvm.spv.group.id (IntrinsicsSPIRV.td:64)
    spv_group_memory_barrier_with_group_sync,  // llvm.spv.group.memory.barrier.with.group.sync (IntrinsicsSPIRV.td:136)
    spv_init_global,                           // llvm.spv.init.global (IntrinsicsSPIRV.td:23)
    spv_inline_asm,                            // llvm.spv.inline.asm (IntrinsicsSPIRV.td:44)
    spv_insertelt,                             // llvm.spv.insertelt (IntrinsicsSPIRV.td:32)
    spv_insertv,                               // llvm.spv.insertv (IntrinsicsSPIRV.td:30)
    spv_isinf,                                 // llvm.spv.isinf (IntrinsicsSPIRV.td:88)
    spv_isnan,                                 // llvm.spv.isnan (IntrinsicsSPIRV.td:90)
    spv_length,                                // llvm.spv.length (IntrinsicsSPIRV.td:94)
    spv_lerp,                                  // llvm.spv.lerp (IntrinsicsSPIRV.td:92)
    spv_lifetime_end,                          // llvm.spv.lifetime.end (IntrinsicsSPIRV.td:56)
    spv_lifetime_start,                        // llvm.spv.lifetime.start (IntrinsicsSPIRV.td:51)
    spv_load,                                  // llvm.spv.load (IntrinsicsSPIRV.td:27)
    spv_loop_merge,                            // llvm.spv.loop.merge (IntrinsicsSPIRV.td:37)
    spv_nclamp,                                // llvm.spv.nclamp (IntrinsicsSPIRV.td:148)
    spv_normalize,                             // llvm.spv.normalize (IntrinsicsSPIRV.td:95)
    spv_num_subgroups,                         // llvm.spv.num.subgroups (IntrinsicsSPIRV.td:72)
    spv_num_workgroups,                        // llvm.spv.num.workgroups (IntrinsicsSPIRV.td:69)
    spv_ptrcast,                               // llvm.spv.ptrcast (IntrinsicsSPIRV.td:35)
    spv_pushconstant_getpointer,               // llvm.spv.pushconstant.getpointer (IntrinsicsSPIRV.td:188)
    spv_radians,                               // llvm.spv.radians (IntrinsicsSPIRV.td:135)
    spv_reflect,                               // llvm.spv.reflect (IntrinsicsSPIRV.td:96)
    spv_refract,                               // llvm.spv.refract (IntrinsicsSPIRV.td:97)
    spv_resource_counterhandlefrombinding,     // llvm.spv.resource.counterhandlefrombinding (IntrinsicsSPIRV.td:168)
    spv_resource_counterhandlefromimplicitbinding,  // llvm.spv.resource.counterhandlefromimplicitbinding (IntrinsicsSPIRV.td:164)
    spv_resource_getdimensions_x,              // llvm.spv.resource.getdimensions.x (IntrinsicsSPIRV.td:181)
    spv_resource_getpointer,                   // llvm.spv.resource.getpointer (IntrinsicsSPIRV.td:184)
    spv_resource_handlefrombinding,            // llvm.spv.resource.handlefrombinding (IntrinsicsSPIRV.td:154)
    spv_resource_handlefromimplicitbinding,    // llvm.spv.resource.handlefromimplicitbinding (IntrinsicsSPIRV.td:159)
    spv_resource_load_typedbuffer,             // llvm.spv.resource.load.typedbuffer (IntrinsicsSPIRV.td:197)
    spv_resource_nonuniformindex,              // llvm.spv.resource.nonuniformindex (IntrinsicsSPIRV.td:191)
    spv_resource_store_typedbuffer,            // llvm.spv.resource.store.typedbuffer (IntrinsicsSPIRV.td:202)
    spv_resource_updatecounter,                // llvm.spv.resource.updatecounter (IntrinsicsSPIRV.td:177)
    spv_rsqrt,                                 // llvm.spv.rsqrt (IntrinsicsSPIRV.td:102)
    spv_saturate,                              // llvm.spv.saturate (IntrinsicsSPIRV.td:103)
    spv_sclamp,                                // llvm.spv.sclamp (IntrinsicsSPIRV.td:147)
    spv_sdot,                                  // llvm.spv.sdot (IntrinsicsSPIRV.td:110)
    spv_selection_merge,                       // llvm.spv.selection.merge (IntrinsicsSPIRV.td:38)
    spv_sign,                                  // llvm.spv.sign (IntrinsicsSPIRV.td:134)
    spv_smoothstep,                            // llvm.spv.smoothstep (IntrinsicsSPIRV.td:104)
    spv_step,                                  // llvm.spv.step (IntrinsicsSPIRV.td:105)
    spv_store,                                 // llvm.spv.store (IntrinsicsSPIRV.td:28)
    spv_subgroup_id,                           // llvm.spv.subgroup.id (IntrinsicsSPIRV.td:74)
    spv_subgroup_local_invocation_id,          // llvm.spv.subgroup.local.invocation.id (IntrinsicsSPIRV.td:76)
    spv_subgroup_max_size,                     // llvm.spv.subgroup.max.size (IntrinsicsSPIRV.td:78)
    spv_subgroup_size,                         // llvm.spv.subgroup.size (IntrinsicsSPIRV.td:70)
    spv_switch,                                // llvm.spv.switch (IntrinsicsSPIRV.td:36)
    spv_thread_id,                             // llvm.spv.thread.id (IntrinsicsSPIRV.td:63)
    spv_thread_id_in_group,                    // llvm.spv.thread.id.in.group (IntrinsicsSPIRV.td:65)
    spv_track_constant,                        // llvm.spv.track.constant (IntrinsicsSPIRV.td:22)
    spv_uclamp,                                // llvm.spv.uclamp (IntrinsicsSPIRV.td:146)
    spv_udot,                                  // llvm.spv.udot (IntrinsicsSPIRV.td:114)
    spv_undef,                                 // llvm.spv.undef (IntrinsicsSPIRV.td:43)
    spv_unpackhalf2x16,                        // llvm.spv.unpackhalf2x16 (IntrinsicsSPIRV.td:216)
    spv_unreachable,                           // llvm.spv.unreachable (IntrinsicsSPIRV.td:40)
    spv_unref_global,                          // llvm.spv.unref.global (IntrinsicsSPIRV.td:24)
    spv_value_md,                              // llvm.spv.value.md (IntrinsicsSPIRV.td:20)
    spv_wave_active_countbits,                 // llvm.spv.wave.active.countbits (IntrinsicsSPIRV.td:120)
    spv_wave_all,                              // llvm.spv.wave.all (IntrinsicsSPIRV.td:121)
    spv_wave_any,                              // llvm.spv.wave.any (IntrinsicsSPIRV.td:122)
    spv_wave_ballot,                           // llvm.spv.wave.ballot (IntrinsicsSPIRV.td:123)
    spv_wave_get_lane_count,                   // llvm.spv.wave.get.lane.count (IntrinsicsSPIRV.td:132)
    spv_wave_is_first_lane,                    // llvm.spv.wave.is.first.lane (IntrinsicsSPIRV.td:130)
    spv_wave_readlane,                         // llvm.spv.wave.readlane (IntrinsicsSPIRV.td:131)
    spv_wave_reduce_max,                       // llvm.spv.wave.reduce.max (IntrinsicsSPIRV.td:126)
    spv_wave_reduce_min,                       // llvm.spv.wave.reduce.min (IntrinsicsSPIRV.td:127)
    spv_wave_reduce_sum,                       // llvm.spv.wave.reduce.sum (IntrinsicsSPIRV.td:129)
    spv_wave_reduce_umax,                      // llvm.spv.wave.reduce.umax (IntrinsicsSPIRV.td:125)
    spv_wave_reduce_umin,                      // llvm.spv.wave.reduce.umin (IntrinsicsSPIRV.td:128)
    spv_workgroup_size,                        // llvm.spv.workgroup.size (IntrinsicsSPIRV.td:66)
}; // enum
} // namespace llvm::Intrinsic
#endif


#ifdef __GNUC__
#pragma GCC diagnostic pop
#endif
