/*===- TableGen'erated file -------------------------------------*- C++ -*-===*\
|*                                                                            *|
|* Target Library Info Source Fragment                                        *|
|*                                                                            *|
|* Automatically generated file, do not edit!                                 *|
|* From: TargetLibraryInfo.td                                                 *|
|*                                                                            *|
\*===----------------------------------------------------------------------===*/

#ifdef GET_TARGET_LIBRARY_INFO_ENUM
#undef GET_TARGET_LIBRARY_INFO_ENUM
enum LibFunc : unsigned {
  NotLibFunc = 0,
  LibFunc_msvc_new_int,
  LibFunc_msvc_new_int_nothrow,
  LibFunc_msvc_new_longlong,
  LibFunc_msvc_new_longlong_nothrow,
  LibFunc_msvc_delete_ptr32,
  LibFunc_msvc_delete_ptr32_nothrow,
  LibFunc_msvc_delete_ptr32_int,
  LibFunc_msvc_delete_ptr64,
  LibFunc_msvc_delete_ptr64_nothrow,
  LibFunc_msvc_delete_ptr64_longlong,
  LibFunc_msvc_new_array_int,
  LibFunc_msvc_new_array_int_nothrow,
  LibFunc_msvc_new_array_longlong,
  LibFunc_msvc_new_array_longlong_nothrow,
  LibFunc_msvc_delete_array_ptr32,
  LibFunc_msvc_delete_array_ptr32_nothrow,
  LibFunc_msvc_delete_array_ptr32_int,
  LibFunc_msvc_delete_array_ptr64,
  LibFunc_msvc_delete_array_ptr64_nothrow,
  LibFunc_msvc_delete_array_ptr64_longlong,
  LibFunc_under_IO_getc,
  LibFunc_under_IO_putc,
  LibFunc_ZdaPv,
  LibFunc_ZdaPvRKSt9nothrow_t,
  LibFunc_ZdaPvSt11align_val_t,
  LibFunc_ZdaPvSt11align_val_tRKSt9nothrow_t,
  LibFunc_ZdaPvj,
  LibFunc_ZdaPvjSt11align_val_t,
  LibFunc_ZdaPvm,
  LibFunc_ZdaPvmSt11align_val_t,
  LibFunc_ZdlPv,
  LibFunc_ZdlPvRKSt9nothrow_t,
  LibFunc_ZdlPvSt11align_val_t,
  LibFunc_ZdlPvSt11align_val_tRKSt9nothrow_t,
  LibFunc_ZdlPvj,
  LibFunc_ZdlPvjSt11align_val_t,
  LibFunc_ZdlPvm,
  LibFunc_ZdlPvmSt11align_val_t,
  LibFunc_Znaj,
  LibFunc_ZnajRKSt9nothrow_t,
  LibFunc_ZnajSt11align_val_t,
  LibFunc_ZnajSt11align_val_tRKSt9nothrow_t,
  LibFunc_Znam,
  LibFunc_Znam12__hot_cold_t,
  LibFunc_ZnamRKSt9nothrow_t,
  LibFunc_ZnamRKSt9nothrow_t12__hot_cold_t,
  LibFunc_ZnamSt11align_val_t,
  LibFunc_ZnamSt11align_val_t12__hot_cold_t,
  LibFunc_ZnamSt11align_val_tRKSt9nothrow_t,
  LibFunc_ZnamSt11align_val_tRKSt9nothrow_t12__hot_cold_t,
  LibFunc_Znwj,
  LibFunc_ZnwjRKSt9nothrow_t,
  LibFunc_ZnwjSt11align_val_t,
  LibFunc_ZnwjSt11align_val_tRKSt9nothrow_t,
  LibFunc_Znwm,
  LibFunc_Znwm12__hot_cold_t,
  LibFunc_ZnwmRKSt9nothrow_t,
  LibFunc_ZnwmRKSt9nothrow_t12__hot_cold_t,
  LibFunc_ZnwmSt11align_val_t,
  LibFunc_ZnwmSt11align_val_t12__hot_cold_t,
  LibFunc_ZnwmSt11align_val_tRKSt9nothrow_t,
  LibFunc_ZnwmSt11align_val_tRKSt9nothrow_t12__hot_cold_t,
  LibFunc_size_returning_new,
  LibFunc_size_returning_new_hot_cold,
  LibFunc_size_returning_new_aligned,
  LibFunc_size_returning_new_aligned_hot_cold,
  LibFunc_acos_finite,
  LibFunc_acosf_finite,
  LibFunc_acosh_finite,
  LibFunc_acoshf_finite,
  LibFunc_acoshl_finite,
  LibFunc_acosl_finite,
  LibFunc_asin_finite,
  LibFunc_asinf_finite,
  LibFunc_asinl_finite,
  LibFunc_atan2_finite,
  LibFunc_atan2f_finite,
  LibFunc_atan2l_finite,
  LibFunc_atanh_finite,
  LibFunc_atanhf_finite,
  LibFunc_atanhl_finite,
  LibFunc_atomic_load,
  LibFunc_atomic_store,
  LibFunc_cosh_finite,
  LibFunc_coshf_finite,
  LibFunc_coshl_finite,
  LibFunc_cospi,
  LibFunc_cospif,
  LibFunc_cxa_atexit,
  LibFunc_atexit,
  LibFunc_abort,
  LibFunc_exit,
  LibFunc_Exit,
  LibFunc_terminate,
  LibFunc_cxa_throw,
  LibFunc_cxa_guard_abort,
  LibFunc_cxa_guard_acquire,
  LibFunc_cxa_guard_release,
  LibFunc_exp10_finite,
  LibFunc_exp10f_finite,
  LibFunc_exp10l_finite,
  LibFunc_exp2_finite,
  LibFunc_exp2f_finite,
  LibFunc_exp2l_finite,
  LibFunc_exp_finite,
  LibFunc_expf_finite,
  LibFunc_expl_finite,
  LibFunc_dunder_isoc99_scanf,
  LibFunc_dunder_isoc99_sscanf,
  LibFunc___kmpc_alloc_shared,
  LibFunc___kmpc_free_shared,
  LibFunc_log10_finite,
  LibFunc_log10f_finite,
  LibFunc_log10l_finite,
  LibFunc_log2_finite,
  LibFunc_log2f_finite,
  LibFunc_log2l_finite,
  LibFunc_log_finite,
  LibFunc_logf_finite,
  LibFunc_logl_finite,
  LibFunc_memccpy_chk,
  LibFunc_memcpy_chk,
  LibFunc_memmove_chk,
  LibFunc_mempcpy_chk,
  LibFunc_memset_chk,
  LibFunc_nvvm_reflect,
  LibFunc_pow_finite,
  LibFunc_powf_finite,
  LibFunc_powl_finite,
  LibFunc_sincospi_stret,
  LibFunc_sincospif_stret,
  LibFunc_sinh_finite,
  LibFunc_sinhf_finite,
  LibFunc_sinhl_finite,
  LibFunc_sinpi,
  LibFunc_sinpif,
  LibFunc_small_fprintf,
  LibFunc_small_printf,
  LibFunc_small_sprintf,
  LibFunc_snprintf_chk,
  LibFunc_sprintf_chk,
  LibFunc_sqrt_finite,
  LibFunc_sqrtf_finite,
  LibFunc_sqrtl_finite,
  LibFunc_stpcpy_chk,
  LibFunc_stpncpy_chk,
  LibFunc_strcat_chk,
  LibFunc_strcpy_chk,
  LibFunc_dunder_strdup,
  LibFunc_strlcat_chk,
  LibFunc_strlcpy_chk,
  LibFunc_strlen_chk,
  LibFunc_strncat_chk,
  LibFunc_strncpy_chk,
  LibFunc_dunder_strndup,
  LibFunc_dunder_strtok_r,
  LibFunc_vsnprintf_chk,
  LibFunc_vsprintf_chk,
  LibFunc_abs,
  LibFunc_access,
  LibFunc_acos,
  LibFunc_acosf,
  LibFunc_acosh,
  LibFunc_acoshf,
  LibFunc_acoshl,
  LibFunc_acosl,
  LibFunc_aligned_alloc,
  LibFunc_asin,
  LibFunc_asinf,
  LibFunc_asinh,
  LibFunc_asinhf,
  LibFunc_asinhl,
  LibFunc_asinl,
  LibFunc_atan,
  LibFunc_atan2,
  LibFunc_atan2f,
  LibFunc_atan2l,
  LibFunc_atanf,
  LibFunc_atanh,
  LibFunc_atanhf,
  LibFunc_atanhl,
  LibFunc_atanl,
  LibFunc_atof,
  LibFunc_atoi,
  LibFunc_atol,
  LibFunc_atoll,
  LibFunc_bcmp,
  LibFunc_bcopy,
  LibFunc_bzero,
  LibFunc_cabs,
  LibFunc_cabsf,
  LibFunc_cabsl,
  LibFunc_calloc,
  LibFunc_cbrt,
  LibFunc_cbrtf,
  LibFunc_cbrtl,
  LibFunc_ceil,
  LibFunc_ceilf,
  LibFunc_ceill,
  LibFunc_chmod,
  LibFunc_chown,
  LibFunc_clearerr,
  LibFunc_closedir,
  LibFunc_copysign,
  LibFunc_copysignf,
  LibFunc_copysignl,
  LibFunc_cos,
  LibFunc_cosf,
  LibFunc_cosh,
  LibFunc_coshf,
  LibFunc_coshl,
  LibFunc_cosl,
  LibFunc_ctermid,
  LibFunc_erf,
  LibFunc_erff,
  LibFunc_erfl,
  LibFunc_tgamma,
  LibFunc_tgammaf,
  LibFunc_tgammal,
  LibFunc_execl,
  LibFunc_execle,
  LibFunc_execlp,
  LibFunc_execv,
  LibFunc_execvP,
  LibFunc_execve,
  LibFunc_execvp,
  LibFunc_execvpe,
  LibFunc_exp,
  LibFunc_exp10,
  LibFunc_exp10f,
  LibFunc_exp10l,
  LibFunc_exp2,
  LibFunc_exp2f,
  LibFunc_exp2l,
  LibFunc_expf,
  LibFunc_expl,
  LibFunc_expm1,
  LibFunc_expm1f,
  LibFunc_expm1l,
  LibFunc_fabs,
  LibFunc_fabsf,
  LibFunc_fabsl,
  LibFunc_fclose,
  LibFunc_fdopen,
  LibFunc_feof,
  LibFunc_ferror,
  LibFunc_fflush,
  LibFunc_ffs,
  LibFunc_ffsl,
  LibFunc_ffsll,
  LibFunc_fgetc,
  LibFunc_fgetc_unlocked,
  LibFunc_fgetpos,
  LibFunc_fgets,
  LibFunc_fgets_unlocked,
  LibFunc_fileno,
  LibFunc_fiprintf,
  LibFunc_flockfile,
  LibFunc_floor,
  LibFunc_floorf,
  LibFunc_floorl,
  LibFunc_fls,
  LibFunc_flsl,
  LibFunc_flsll,
  LibFunc_fmax,
  LibFunc_fmaxf,
  LibFunc_fmaxl,
  LibFunc_fmin,
  LibFunc_fminf,
  LibFunc_fminl,
  LibFunc_fmaximum_num,
  LibFunc_fmaximum_numf,
  LibFunc_fmaximum_numl,
  LibFunc_fminimum_num,
  LibFunc_fminimum_numf,
  LibFunc_fminimum_numl,
  LibFunc_fmod,
  LibFunc_fmodf,
  LibFunc_fmodl,
  LibFunc_fopen,
  LibFunc_fopen64,
  LibFunc_fork,
  LibFunc_fprintf,
  LibFunc_fputc,
  LibFunc_fputc_unlocked,
  LibFunc_fputs,
  LibFunc_fputs_unlocked,
  LibFunc_fread,
  LibFunc_fread_unlocked,
  LibFunc_free,
  LibFunc_frexp,
  LibFunc_frexpf,
  LibFunc_frexpl,
  LibFunc_fscanf,
  LibFunc_fseek,
  LibFunc_fseeko,
  LibFunc_fseeko64,
  LibFunc_fsetpos,
  LibFunc_fstat,
  LibFunc_fstat64,
  LibFunc_fstatvfs,
  LibFunc_fstatvfs64,
  LibFunc_ftell,
  LibFunc_ftello,
  LibFunc_ftello64,
  LibFunc_ftrylockfile,
  LibFunc_funlockfile,
  LibFunc_fwrite,
  LibFunc_fwrite_unlocked,
  LibFunc_getc,
  LibFunc_getc_unlocked,
  LibFunc_getchar,
  LibFunc_getchar_unlocked,
  LibFunc_getenv,
  LibFunc_getitimer,
  LibFunc_getlogin_r,
  LibFunc_getpwnam,
  LibFunc_gets,
  LibFunc_gettimeofday,
  LibFunc_htonl,
  LibFunc_htons,
  LibFunc_hypot,
  LibFunc_hypotf,
  LibFunc_hypotl,
  LibFunc_iprintf,
  LibFunc_isascii,
  LibFunc_isdigit,
  LibFunc_labs,
  LibFunc_lchown,
  LibFunc_ldexp,
  LibFunc_ldexpf,
  LibFunc_ldexpl,
  LibFunc_llabs,
  LibFunc_log,
  LibFunc_log10,
  LibFunc_log10f,
  LibFunc_log10l,
  LibFunc_log1p,
  LibFunc_log1pf,
  LibFunc_log1pl,
  LibFunc_log2,
  LibFunc_log2f,
  LibFunc_log2l,
  LibFunc_ilogb,
  LibFunc_ilogbf,
  LibFunc_ilogbl,
  LibFunc_logb,
  LibFunc_logbf,
  LibFunc_logbl,
  LibFunc_logf,
  LibFunc_logl,
  LibFunc_lstat,
  LibFunc_lstat64,
  LibFunc_malloc,
  LibFunc_memalign,
  LibFunc_memccpy,
  LibFunc_memchr,
  LibFunc_memcmp,
  LibFunc_memcpy,
  LibFunc_memmove,
  LibFunc_mempcpy,
  LibFunc_memrchr,
  LibFunc_memset,
  LibFunc_memset_pattern16,
  LibFunc_memset_pattern4,
  LibFunc_memset_pattern8,
  LibFunc_mkdir,
  LibFunc_mktime,
  LibFunc_modf,
  LibFunc_modff,
  LibFunc_modfl,
  LibFunc_nan,
  LibFunc_nanf,
  LibFunc_nanl,
  LibFunc_nearbyint,
  LibFunc_nearbyintf,
  LibFunc_nearbyintl,
  LibFunc_nextafter,
  LibFunc_nextafterf,
  LibFunc_nextafterl,
  LibFunc_nexttoward,
  LibFunc_nexttowardf,
  LibFunc_nexttowardl,
  LibFunc_ntohl,
  LibFunc_ntohs,
  LibFunc_open,
  LibFunc_open64,
  LibFunc_opendir,
  LibFunc_pclose,
  LibFunc_perror,
  LibFunc_popen,
  LibFunc_posix_memalign,
  LibFunc_pow,
  LibFunc_powf,
  LibFunc_powl,
  LibFunc_pread,
  LibFunc_printf,
  LibFunc_putc,
  LibFunc_putc_unlocked,
  LibFunc_putchar,
  LibFunc_putchar_unlocked,
  LibFunc_puts,
  LibFunc_pvalloc,
  LibFunc_pwrite,
  LibFunc_qsort,
  LibFunc_read,
  LibFunc_readlink,
  LibFunc_realloc,
  LibFunc_reallocf,
  LibFunc_reallocarray,
  LibFunc_realpath,
  LibFunc_remainder,
  LibFunc_remainderf,
  LibFunc_remainderl,
  LibFunc_remquo,
  LibFunc_remquof,
  LibFunc_remquol,
  LibFunc_fdim,
  LibFunc_fdimf,
  LibFunc_fdiml,
  LibFunc_remove,
  LibFunc_rename,
  LibFunc_rewind,
  LibFunc_rint,
  LibFunc_rintf,
  LibFunc_rintl,
  LibFunc_rmdir,
  LibFunc_round,
  LibFunc_roundeven,
  LibFunc_roundevenf,
  LibFunc_roundevenl,
  LibFunc_roundf,
  LibFunc_roundl,
  LibFunc_scalbln,
  LibFunc_scalblnf,
  LibFunc_scalblnl,
  LibFunc_scalbn,
  LibFunc_scalbnf,
  LibFunc_scalbnl,
  LibFunc_scanf,
  LibFunc_setbuf,
  LibFunc_setitimer,
  LibFunc_setvbuf,
  LibFunc_sin,
  LibFunc_sinf,
  LibFunc_sinh,
  LibFunc_sinhf,
  LibFunc_sinhl,
  LibFunc_sinl,
  LibFunc_sincos,
  LibFunc_sincosf,
  LibFunc_sincosl,
  LibFunc_siprintf,
  LibFunc_snprintf,
  LibFunc_sprintf,
  LibFunc_sqrt,
  LibFunc_sqrtf,
  LibFunc_sqrtl,
  LibFunc_sscanf,
  LibFunc_stat,
  LibFunc_stat64,
  LibFunc_statvfs,
  LibFunc_statvfs64,
  LibFunc_stpcpy,
  LibFunc_stpncpy,
  LibFunc_strcasecmp,
  LibFunc_strcat,
  LibFunc_strchr,
  LibFunc_strcmp,
  LibFunc_strcoll,
  LibFunc_strcpy,
  LibFunc_strcspn,
  LibFunc_strdup,
  LibFunc_strlcat,
  LibFunc_strlcpy,
  LibFunc_strlen,
  LibFunc_strncasecmp,
  LibFunc_strncat,
  LibFunc_strncmp,
  LibFunc_strncpy,
  LibFunc_strndup,
  LibFunc_strnlen,
  LibFunc_strpbrk,
  LibFunc_strrchr,
  LibFunc_strspn,
  LibFunc_strstr,
  LibFunc_strtod,
  LibFunc_strtof,
  LibFunc_strtok,
  LibFunc_strtok_r,
  LibFunc_strtol,
  LibFunc_strtold,
  LibFunc_strtoll,
  LibFunc_strtoul,
  LibFunc_strtoull,
  LibFunc_strxfrm,
  LibFunc_system,
  LibFunc_tan,
  LibFunc_tanf,
  LibFunc_tanh,
  LibFunc_tanhf,
  LibFunc_tanhl,
  LibFunc_tanl,
  LibFunc_times,
  LibFunc_tmpfile,
  LibFunc_tmpfile64,
  LibFunc_toascii,
  LibFunc_trunc,
  LibFunc_truncf,
  LibFunc_truncl,
  LibFunc_uname,
  LibFunc_ungetc,
  LibFunc_unlink,
  LibFunc_unsetenv,
  LibFunc_utime,
  LibFunc_utimes,
  LibFunc_valloc,
  LibFunc_vec_calloc,
  LibFunc_vec_free,
  LibFunc_vec_malloc,
  LibFunc_vec_realloc,
  LibFunc_vfprintf,
  LibFunc_vfscanf,
  LibFunc_vprintf,
  LibFunc_vscanf,
  LibFunc_vsnprintf,
  LibFunc_vsprintf,
  LibFunc_vsscanf,
  LibFunc_wcslen,
  LibFunc_write,
  NumLibFuncs,
  End_LibFunc = NumLibFuncs,
  Begin_LibFunc = LibFunc_msvc_new_int,
};
#endif

#ifdef GET_TARGET_LIBRARY_INFO_STRING_TABLE
#undef GET_TARGET_LIBRARY_INFO_STRING_TABLE

#ifdef __GNUC__
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Woverlength-strings"
#endif
 constexpr char StandardNamesStrTableStorage[] =
  "\0"
  "??2@YAPAXI@Z\0"
  "??2@YAPAXIABUnothrow_t@std@@@Z\0"
  "??2@YAPEAX_K@Z\0"
  "??2@YAPEAX_KAEBUnothrow_t@std@@@Z\0"
  "??3@YAXPAX@Z\0"
  "??3@YAXPAXABUnothrow_t@std@@@Z\0"
  "??3@YAXPAXI@Z\0"
  "??3@YAXPEAX@Z\0"
  "??3@YAXPEAXAEBUnothrow_t@std@@@Z\0"
  "??3@YAXPEAX_K@Z\0"
  "??_U@YAPAXI@Z\0"
  "??_U@YAPAXIABUnothrow_t@std@@@Z\0"
  "??_U@YAPEAX_K@Z\0"
  "??_U@YAPEAX_KAEBUnothrow_t@std@@@Z\0"
  "??_V@YAXPAX@Z\0"
  "??_V@YAXPAXABUnothrow_t@std@@@Z\0"
  "??_V@YAXPAXI@Z\0"
  "??_V@YAXPEAX@Z\0"
  "??_V@YAXPEAXAEBUnothrow_t@std@@@Z\0"
  "??_V@YAXPEAX_K@Z\0"
  "_IO_getc\0"
  "_IO_putc\0"
  "_ZdaPv\0"
  "_ZdaPvRKSt9nothrow_t\0"
  "_ZdaPvSt11align_val_t\0"
  "_ZdaPvSt11align_val_tRKSt9nothrow_t\0"
  "_ZdaPvj\0"
  "_ZdaPvjSt11align_val_t\0"
  "_ZdaPvm\0"
  "_ZdaPvmSt11align_val_t\0"
  "_ZdlPv\0"
  "_ZdlPvRKSt9nothrow_t\0"
  "_ZdlPvSt11align_val_t\0"
  "_ZdlPvSt11align_val_tRKSt9nothrow_t\0"
  "_ZdlPvj\0"
  "_ZdlPvjSt11align_val_t\0"
  "_ZdlPvm\0"
  "_ZdlPvmSt11align_val_t\0"
  "_Znaj\0"
  "_ZnajRKSt9nothrow_t\0"
  "_ZnajSt11align_val_t\0"
  "_ZnajSt11align_val_tRKSt9nothrow_t\0"
  "_Znam\0"
  "_Znam12__hot_cold_t\0"
  "_ZnamRKSt9nothrow_t\0"
  "_ZnamRKSt9nothrow_t12__hot_cold_t\0"
  "_ZnamSt11align_val_t\0"
  "_ZnamSt11align_val_t12__hot_cold_t\0"
  "_ZnamSt11align_val_tRKSt9nothrow_t\0"
  "_ZnamSt11align_val_tRKSt9nothrow_t12__hot_cold_t\0"
  "_Znwj\0"
  "_ZnwjRKSt9nothrow_t\0"
  "_ZnwjSt11align_val_t\0"
  "_ZnwjSt11align_val_tRKSt9nothrow_t\0"
  "_Znwm\0"
  "_Znwm12__hot_cold_t\0"
  "_ZnwmRKSt9nothrow_t\0"
  "_ZnwmRKSt9nothrow_t12__hot_cold_t\0"
  "_ZnwmSt11align_val_t\0"
  "_ZnwmSt11align_val_t12__hot_cold_t\0"
  "_ZnwmSt11align_val_tRKSt9nothrow_t\0"
  "_ZnwmSt11align_val_tRKSt9nothrow_t12__hot_cold_t\0"
  "__size_returning_new\0"
  "__size_returning_new_hot_cold\0"
  "__size_returning_new_aligned\0"
  "__size_returning_new_aligned_hot_cold\0"
  "__acos_finite\0"
  "__acosf_finite\0"
  "__acosh_finite\0"
  "__acoshf_finite\0"
  "__acoshl_finite\0"
  "__acosl_finite\0"
  "__asin_finite\0"
  "__asinf_finite\0"
  "__asinl_finite\0"
  "__atan2_finite\0"
  "__atan2f_finite\0"
  "__atan2l_finite\0"
  "__atanh_finite\0"
  "__atanhf_finite\0"
  "__atanhl_finite\0"
  "__atomic_load\0"
  "__atomic_store\0"
  "__cosh_finite\0"
  "__coshf_finite\0"
  "__coshl_finite\0"
  "__cospi\0"
  "__cospif\0"
  "__cxa_atexit\0"
  "atexit\0"
  "abort\0"
  "exit\0"
  "_Exit\0"
  "_ZSt9terminatev\0"
  "__cxa_throw\0"
  "__cxa_guard_abort\0"
  "__cxa_guard_acquire\0"
  "__cxa_guard_release\0"
  "__exp10_finite\0"
  "__exp10f_finite\0"
  "__exp10l_finite\0"
  "__exp2_finite\0"
  "__exp2f_finite\0"
  "__exp2l_finite\0"
  "__exp_finite\0"
  "__expf_finite\0"
  "__expl_finite\0"
  "__isoc99_scanf\0"
  "__isoc99_sscanf\0"
  "__kmpc_alloc_shared\0"
  "__kmpc_free_shared\0"
  "__log10_finite\0"
  "__log10f_finite\0"
  "__log10l_finite\0"
  "__log2_finite\0"
  "__log2f_finite\0"
  "__log2l_finite\0"
  "__log_finite\0"
  "__logf_finite\0"
  "__logl_finite\0"
  "__memccpy_chk\0"
  "__memcpy_chk\0"
  "__memmove_chk\0"
  "__mempcpy_chk\0"
  "__memset_chk\0"
  "__nvvm_reflect\0"
  "__pow_finite\0"
  "__powf_finite\0"
  "__powl_finite\0"
  "__sincospi_stret\0"
  "__sincospif_stret\0"
  "__sinh_finite\0"
  "__sinhf_finite\0"
  "__sinhl_finite\0"
  "__sinpi\0"
  "__sinpif\0"
  "__small_fprintf\0"
  "__small_printf\0"
  "__small_sprintf\0"
  "__snprintf_chk\0"
  "__sprintf_chk\0"
  "__sqrt_finite\0"
  "__sqrtf_finite\0"
  "__sqrtl_finite\0"
  "__stpcpy_chk\0"
  "__stpncpy_chk\0"
  "__strcat_chk\0"
  "__strcpy_chk\0"
  "__strdup\0"
  "__strlcat_chk\0"
  "__strlcpy_chk\0"
  "__strlen_chk\0"
  "__strncat_chk\0"
  "__strncpy_chk\0"
  "__strndup\0"
  "__strtok_r\0"
  "__vsnprintf_chk\0"
  "__vsprintf_chk\0"
  "abs\0"
  "access\0"
  "acos\0"
  "acosf\0"
  "acosh\0"
  "acoshf\0"
  "acoshl\0"
  "acosl\0"
  "aligned_alloc\0"
  "asin\0"
  "asinf\0"
  "asinh\0"
  "asinhf\0"
  "asinhl\0"
  "asinl\0"
  "atan\0"
  "atan2\0"
  "atan2f\0"
  "atan2l\0"
  "atanf\0"
  "atanh\0"
  "atanhf\0"
  "atanhl\0"
  "atanl\0"
  "atof\0"
  "atoi\0"
  "atol\0"
  "atoll\0"
  "bcmp\0"
  "bcopy\0"
  "bzero\0"
  "cabs\0"
  "cabsf\0"
  "cabsl\0"
  "calloc\0"
  "cbrt\0"
  "cbrtf\0"
  "cbrtl\0"
  "ceil\0"
  "ceilf\0"
  "ceill\0"
  "chmod\0"
  "chown\0"
  "clearerr\0"
  "closedir\0"
  "copysign\0"
  "copysignf\0"
  "copysignl\0"
  "cos\0"
  "cosf\0"
  "cosh\0"
  "coshf\0"
  "coshl\0"
  "cosl\0"
  "ctermid\0"
  "erf\0"
  "erff\0"
  "erfl\0"
  "tgamma\0"
  "tgammaf\0"
  "tgammal\0"
  "execl\0"
  "execle\0"
  "execlp\0"
  "execv\0"
  "execvP\0"
  "execve\0"
  "execvp\0"
  "execvpe\0"
  "exp\0"
  "exp10\0"
  "exp10f\0"
  "exp10l\0"
  "exp2\0"
  "exp2f\0"
  "exp2l\0"
  "expf\0"
  "expl\0"
  "expm1\0"
  "expm1f\0"
  "expm1l\0"
  "fabs\0"
  "fabsf\0"
  "fabsl\0"
  "fclose\0"
  "fdopen\0"
  "feof\0"
  "ferror\0"
  "fflush\0"
  "ffs\0"
  "ffsl\0"
  "ffsll\0"
  "fgetc\0"
  "fgetc_unlocked\0"
  "fgetpos\0"
  "fgets\0"
  "fgets_unlocked\0"
  "fileno\0"
  "fiprintf\0"
  "flockfile\0"
  "floor\0"
  "floorf\0"
  "floorl\0"
  "fls\0"
  "flsl\0"
  "flsll\0"
  "fmax\0"
  "fmaxf\0"
  "fmaxl\0"
  "fmin\0"
  "fminf\0"
  "fminl\0"
  "fmaximum_num\0"
  "fmaximum_numf\0"
  "fmaximum_numl\0"
  "fminimum_num\0"
  "fminimum_numf\0"
  "fminimum_numl\0"
  "fmod\0"
  "fmodf\0"
  "fmodl\0"
  "fopen\0"
  "fopen64\0"
  "fork\0"
  "fprintf\0"
  "fputc\0"
  "fputc_unlocked\0"
  "fputs\0"
  "fputs_unlocked\0"
  "fread\0"
  "fread_unlocked\0"
  "free\0"
  "frexp\0"
  "frexpf\0"
  "frexpl\0"
  "fscanf\0"
  "fseek\0"
  "fseeko\0"
  "fseeko64\0"
  "fsetpos\0"
  "fstat\0"
  "fstat64\0"
  "fstatvfs\0"
  "fstatvfs64\0"
  "ftell\0"
  "ftello\0"
  "ftello64\0"
  "ftrylockfile\0"
  "funlockfile\0"
  "fwrite\0"
  "fwrite_unlocked\0"
  "getc\0"
  "getc_unlocked\0"
  "getchar\0"
  "getchar_unlocked\0"
  "getenv\0"
  "getitimer\0"
  "getlogin_r\0"
  "getpwnam\0"
  "gets\0"
  "gettimeofday\0"
  "htonl\0"
  "htons\0"
  "hypot\0"
  "hypotf\0"
  "hypotl\0"
  "iprintf\0"
  "isascii\0"
  "isdigit\0"
  "labs\0"
  "lchown\0"
  "ldexp\0"
  "ldexpf\0"
  "ldexpl\0"
  "llabs\0"
  "log\0"
  "log10\0"
  "log10f\0"
  "log10l\0"
  "log1p\0"
  "log1pf\0"
  "log1pl\0"
  "log2\0"
  "log2f\0"
  "log2l\0"
  "ilogb\0"
  "ilogbf\0"
  "ilogbl\0"
  "logb\0"
  "logbf\0"
  "logbl\0"
  "logf\0"
  "logl\0"
  "lstat\0"
  "lstat64\0"
  "malloc\0"
  "memalign\0"
  "memccpy\0"
  "memchr\0"
  "memcmp\0"
  "memcpy\0"
  "memmove\0"
  "mempcpy\0"
  "memrchr\0"
  "memset\0"
  "memset_pattern16\0"
  "memset_pattern4\0"
  "memset_pattern8\0"
  "mkdir\0"
  "mktime\0"
  "modf\0"
  "modff\0"
  "modfl\0"
  "nan\0"
  "nanf\0"
  "nanl\0"
  "nearbyint\0"
  "nearbyintf\0"
  "nearbyintl\0"
  "nextafter\0"
  "nextafterf\0"
  "nextafterl\0"
  "nexttoward\0"
  "nexttowardf\0"
  "nexttowardl\0"
  "ntohl\0"
  "ntohs\0"
  "open\0"
  "open64\0"
  "opendir\0"
  "pclose\0"
  "perror\0"
  "popen\0"
  "posix_memalign\0"
  "pow\0"
  "powf\0"
  "powl\0"
  "pread\0"
  "printf\0"
  "putc\0"
  "putc_unlocked\0"
  "putchar\0"
  "putchar_unlocked\0"
  "puts\0"
  "pvalloc\0"
  "pwrite\0"
  "qsort\0"
  "read\0"
  "readlink\0"
  "realloc\0"
  "reallocf\0"
  "reallocarray\0"
  "realpath\0"
  "remainder\0"
  "remainderf\0"
  "remainderl\0"
  "remquo\0"
  "remquof\0"
  "remquol\0"
  "fdim\0"
  "fdimf\0"
  "fdiml\0"
  "remove\0"
  "rename\0"
  "rewind\0"
  "rint\0"
  "rintf\0"
  "rintl\0"
  "rmdir\0"
  "round\0"
  "roundeven\0"
  "roundevenf\0"
  "roundevenl\0"
  "roundf\0"
  "roundl\0"
  "scalbln\0"
  "scalblnf\0"
  "scalblnl\0"
  "scalbn\0"
  "scalbnf\0"
  "scalbnl\0"
  "scanf\0"
  "setbuf\0"
  "setitimer\0"
  "setvbuf\0"
  "sin\0"
  "sinf\0"
  "sinh\0"
  "sinhf\0"
  "sinhl\0"
  "sinl\0"
  "sincos\0"
  "sincosf\0"
  "sincosl\0"
  "siprintf\0"
  "snprintf\0"
  "sprintf\0"
  "sqrt\0"
  "sqrtf\0"
  "sqrtl\0"
  "sscanf\0"
  "stat\0"
  "stat64\0"
  "statvfs\0"
  "statvfs64\0"
  "stpcpy\0"
  "stpncpy\0"
  "strcasecmp\0"
  "strcat\0"
  "strchr\0"
  "strcmp\0"
  "strcoll\0"
  "strcpy\0"
  "strcspn\0"
  "strdup\0"
  "strlcat\0"
  "strlcpy\0"
  "strlen\0"
  "strncasecmp\0"
  "strncat\0"
  "strncmp\0"
  "strncpy\0"
  "strndup\0"
  "strnlen\0"
  "strpbrk\0"
  "strrchr\0"
  "strspn\0"
  "strstr\0"
  "strtod\0"
  "strtof\0"
  "strtok\0"
  "strtok_r\0"
  "strtol\0"
  "strtold\0"
  "strtoll\0"
  "strtoul\0"
  "strtoull\0"
  "strxfrm\0"
  "system\0"
  "tan\0"
  "tanf\0"
  "tanh\0"
  "tanhf\0"
  "tanhl\0"
  "tanl\0"
  "times\0"
  "tmpfile\0"
  "tmpfile64\0"
  "toascii\0"
  "trunc\0"
  "truncf\0"
  "truncl\0"
  "uname\0"
  "ungetc\0"
  "unlink\0"
  "unsetenv\0"
  "utime\0"
  "utimes\0"
  "valloc\0"
  "vec_calloc\0"
  "vec_free\0"
  "vec_malloc\0"
  "vec_realloc\0"
  "vfprintf\0"
  "vfscanf\0"
  "vprintf\0"
  "vscanf\0"
  "vsnprintf\0"
  "vsprintf\0"
  "vsscanf\0"
  "wcslen\0"
  "write\0"
  ;
#ifdef __GNUC__
#pragma GCC diagnostic pop
#endif

const llvm::StringTable
TargetLibraryInfoImpl::StandardNamesStrTable = StandardNamesStrTableStorage;

const llvm::StringTable::Offset TargetLibraryInfoImpl::StandardNamesOffsets[531] = {
  0, //
  1, // ??2@YAPAXI@Z
  14, // ??2@YAPAXIABUnothrow_t@std@@@Z
  45, // ??2@YAPEAX_K@Z
  60, // ??2@YAPEAX_KAEBUnothrow_t@std@@@Z
  94, // ??3@YAXPAX@Z
  107, // ??3@YAXPAXABUnothrow_t@std@@@Z
  138, // ??3@YAXPAXI@Z
  152, // ??3@YAXPEAX@Z
  166, // ??3@YAXPEAXAEBUnothrow_t@std@@@Z
  199, // ??3@YAXPEAX_K@Z
  215, // ??_U@YAPAXI@Z
  229, // ??_U@YAPAXIABUnothrow_t@std@@@Z
  261, // ??_U@YAPEAX_K@Z
  277, // ??_U@YAPEAX_KAEBUnothrow_t@std@@@Z
  312, // ??_V@YAXPAX@Z
  326, // ??_V@YAXPAXABUnothrow_t@std@@@Z
  358, // ??_V@YAXPAXI@Z
  373, // ??_V@YAXPEAX@Z
  388, // ??_V@YAXPEAXAEBUnothrow_t@std@@@Z
  422, // ??_V@YAXPEAX_K@Z
  439, // _IO_getc
  448, // _IO_putc
  457, // _ZdaPv
  464, // _ZdaPvRKSt9nothrow_t
  485, // _ZdaPvSt11align_val_t
  507, // _ZdaPvSt11align_val_tRKSt9nothrow_t
  543, // _ZdaPvj
  551, // _ZdaPvjSt11align_val_t
  574, // _ZdaPvm
  582, // _ZdaPvmSt11align_val_t
  605, // _ZdlPv
  612, // _ZdlPvRKSt9nothrow_t
  633, // _ZdlPvSt11align_val_t
  655, // _ZdlPvSt11align_val_tRKSt9nothrow_t
  691, // _ZdlPvj
  699, // _ZdlPvjSt11align_val_t
  722, // _ZdlPvm
  730, // _ZdlPvmSt11align_val_t
  753, // _Znaj
  759, // _ZnajRKSt9nothrow_t
  779, // _ZnajSt11align_val_t
  800, // _ZnajSt11align_val_tRKSt9nothrow_t
  835, // _Znam
  841, // _Znam12__hot_cold_t
  861, // _ZnamRKSt9nothrow_t
  881, // _ZnamRKSt9nothrow_t12__hot_cold_t
  915, // _ZnamSt11align_val_t
  936, // _ZnamSt11align_val_t12__hot_cold_t
  971, // _ZnamSt11align_val_tRKSt9nothrow_t
  1006, // _ZnamSt11align_val_tRKSt9nothrow_t12__hot_cold_t
  1055, // _Znwj
  1061, // _ZnwjRKSt9nothrow_t
  1081, // _ZnwjSt11align_val_t
  1102, // _ZnwjSt11align_val_tRKSt9nothrow_t
  1137, // _Znwm
  1143, // _Znwm12__hot_cold_t
  1163, // _ZnwmRKSt9nothrow_t
  1183, // _ZnwmRKSt9nothrow_t12__hot_cold_t
  1217, // _ZnwmSt11align_val_t
  1238, // _ZnwmSt11align_val_t12__hot_cold_t
  1273, // _ZnwmSt11align_val_tRKSt9nothrow_t
  1308, // _ZnwmSt11align_val_tRKSt9nothrow_t12__hot_cold_t
  1357, // __size_returning_new
  1378, // __size_returning_new_hot_cold
  1408, // __size_returning_new_aligned
  1437, // __size_returning_new_aligned_hot_cold
  1475, // __acos_finite
  1489, // __acosf_finite
  1504, // __acosh_finite
  1519, // __acoshf_finite
  1535, // __acoshl_finite
  1551, // __acosl_finite
  1566, // __asin_finite
  1580, // __asinf_finite
  1595, // __asinl_finite
  1610, // __atan2_finite
  1625, // __atan2f_finite
  1641, // __atan2l_finite
  1657, // __atanh_finite
  1672, // __atanhf_finite
  1688, // __atanhl_finite
  1704, // __atomic_load
  1718, // __atomic_store
  1733, // __cosh_finite
  1747, // __coshf_finite
  1762, // __coshl_finite
  1777, // __cospi
  1785, // __cospif
  1794, // __cxa_atexit
  1807, // atexit
  1814, // abort
  1820, // exit
  1825, // _Exit
  1831, // _ZSt9terminatev
  1847, // __cxa_throw
  1859, // __cxa_guard_abort
  1877, // __cxa_guard_acquire
  1897, // __cxa_guard_release
  1917, // __exp10_finite
  1932, // __exp10f_finite
  1948, // __exp10l_finite
  1964, // __exp2_finite
  1978, // __exp2f_finite
  1993, // __exp2l_finite
  2008, // __exp_finite
  2021, // __expf_finite
  2035, // __expl_finite
  2049, // __isoc99_scanf
  2064, // __isoc99_sscanf
  2080, // __kmpc_alloc_shared
  2100, // __kmpc_free_shared
  2119, // __log10_finite
  2134, // __log10f_finite
  2150, // __log10l_finite
  2166, // __log2_finite
  2180, // __log2f_finite
  2195, // __log2l_finite
  2210, // __log_finite
  2223, // __logf_finite
  2237, // __logl_finite
  2251, // __memccpy_chk
  2265, // __memcpy_chk
  2278, // __memmove_chk
  2292, // __mempcpy_chk
  2306, // __memset_chk
  2319, // __nvvm_reflect
  2334, // __pow_finite
  2347, // __powf_finite
  2361, // __powl_finite
  2375, // __sincospi_stret
  2392, // __sincospif_stret
  2410, // __sinh_finite
  2424, // __sinhf_finite
  2439, // __sinhl_finite
  2454, // __sinpi
  2462, // __sinpif
  2471, // __small_fprintf
  2487, // __small_printf
  2502, // __small_sprintf
  2518, // __snprintf_chk
  2533, // __sprintf_chk
  2547, // __sqrt_finite
  2561, // __sqrtf_finite
  2576, // __sqrtl_finite
  2591, // __stpcpy_chk
  2604, // __stpncpy_chk
  2618, // __strcat_chk
  2631, // __strcpy_chk
  2644, // __strdup
  2653, // __strlcat_chk
  2667, // __strlcpy_chk
  2681, // __strlen_chk
  2694, // __strncat_chk
  2708, // __strncpy_chk
  2722, // __strndup
  2732, // __strtok_r
  2743, // __vsnprintf_chk
  2759, // __vsprintf_chk
  2774, // abs
  2778, // access
  2785, // acos
  2790, // acosf
  2796, // acosh
  2802, // acoshf
  2809, // acoshl
  2816, // acosl
  2822, // aligned_alloc
  2836, // asin
  2841, // asinf
  2847, // asinh
  2853, // asinhf
  2860, // asinhl
  2867, // asinl
  2873, // atan
  2878, // atan2
  2884, // atan2f
  2891, // atan2l
  2898, // atanf
  2904, // atanh
  2910, // atanhf
  2917, // atanhl
  2924, // atanl
  2930, // atof
  2935, // atoi
  2940, // atol
  2945, // atoll
  2951, // bcmp
  2956, // bcopy
  2962, // bzero
  2968, // cabs
  2973, // cabsf
  2979, // cabsl
  2985, // calloc
  2992, // cbrt
  2997, // cbrtf
  3003, // cbrtl
  3009, // ceil
  3014, // ceilf
  3020, // ceill
  3026, // chmod
  3032, // chown
  3038, // clearerr
  3047, // closedir
  3056, // copysign
  3065, // copysignf
  3075, // copysignl
  3085, // cos
  3089, // cosf
  3094, // cosh
  3099, // coshf
  3105, // coshl
  3111, // cosl
  3116, // ctermid
  3124, // erf
  3128, // erff
  3133, // erfl
  3138, // tgamma
  3145, // tgammaf
  3153, // tgammal
  3161, // execl
  3167, // execle
  3174, // execlp
  3181, // execv
  3187, // execvP
  3194, // execve
  3201, // execvp
  3208, // execvpe
  3216, // exp
  3220, // exp10
  3226, // exp10f
  3233, // exp10l
  3240, // exp2
  3245, // exp2f
  3251, // exp2l
  3257, // expf
  3262, // expl
  3267, // expm1
  3273, // expm1f
  3280, // expm1l
  3287, // fabs
  3292, // fabsf
  3298, // fabsl
  3304, // fclose
  3311, // fdopen
  3318, // feof
  3323, // ferror
  3330, // fflush
  3337, // ffs
  3341, // ffsl
  3346, // ffsll
  3352, // fgetc
  3358, // fgetc_unlocked
  3373, // fgetpos
  3381, // fgets
  3387, // fgets_unlocked
  3402, // fileno
  3409, // fiprintf
  3418, // flockfile
  3428, // floor
  3434, // floorf
  3441, // floorl
  3448, // fls
  3452, // flsl
  3457, // flsll
  3463, // fmax
  3468, // fmaxf
  3474, // fmaxl
  3480, // fmin
  3485, // fminf
  3491, // fminl
  3497, // fmaximum_num
  3510, // fmaximum_numf
  3524, // fmaximum_numl
  3538, // fminimum_num
  3551, // fminimum_numf
  3565, // fminimum_numl
  3579, // fmod
  3584, // fmodf
  3590, // fmodl
  3596, // fopen
  3602, // fopen64
  3610, // fork
  3615, // fprintf
  3623, // fputc
  3629, // fputc_unlocked
  3644, // fputs
  3650, // fputs_unlocked
  3665, // fread
  3671, // fread_unlocked
  3686, // free
  3691, // frexp
  3697, // frexpf
  3704, // frexpl
  3711, // fscanf
  3718, // fseek
  3724, // fseeko
  3731, // fseeko64
  3740, // fsetpos
  3748, // fstat
  3754, // fstat64
  3762, // fstatvfs
  3771, // fstatvfs64
  3782, // ftell
  3788, // ftello
  3795, // ftello64
  3804, // ftrylockfile
  3817, // funlockfile
  3829, // fwrite
  3836, // fwrite_unlocked
  3852, // getc
  3857, // getc_unlocked
  3871, // getchar
  3879, // getchar_unlocked
  3896, // getenv
  3903, // getitimer
  3913, // getlogin_r
  3924, // getpwnam
  3933, // gets
  3938, // gettimeofday
  3951, // htonl
  3957, // htons
  3963, // hypot
  3969, // hypotf
  3976, // hypotl
  3983, // iprintf
  3991, // isascii
  3999, // isdigit
  4007, // labs
  4012, // lchown
  4019, // ldexp
  4025, // ldexpf
  4032, // ldexpl
  4039, // llabs
  4045, // log
  4049, // log10
  4055, // log10f
  4062, // log10l
  4069, // log1p
  4075, // log1pf
  4082, // log1pl
  4089, // log2
  4094, // log2f
  4100, // log2l
  4106, // ilogb
  4112, // ilogbf
  4119, // ilogbl
  4126, // logb
  4131, // logbf
  4137, // logbl
  4143, // logf
  4148, // logl
  4153, // lstat
  4159, // lstat64
  4167, // malloc
  4174, // memalign
  4183, // memccpy
  4191, // memchr
  4198, // memcmp
  4205, // memcpy
  4212, // memmove
  4220, // mempcpy
  4228, // memrchr
  4236, // memset
  4243, // memset_pattern16
  4260, // memset_pattern4
  4276, // memset_pattern8
  4292, // mkdir
  4298, // mktime
  4305, // modf
  4310, // modff
  4316, // modfl
  4322, // nan
  4326, // nanf
  4331, // nanl
  4336, // nearbyint
  4346, // nearbyintf
  4357, // nearbyintl
  4368, // nextafter
  4378, // nextafterf
  4389, // nextafterl
  4400, // nexttoward
  4411, // nexttowardf
  4423, // nexttowardl
  4435, // ntohl
  4441, // ntohs
  4447, // open
  4452, // open64
  4459, // opendir
  4467, // pclose
  4474, // perror
  4481, // popen
  4487, // posix_memalign
  4502, // pow
  4506, // powf
  4511, // powl
  4516, // pread
  4522, // printf
  4529, // putc
  4534, // putc_unlocked
  4548, // putchar
  4556, // putchar_unlocked
  4573, // puts
  4578, // pvalloc
  4586, // pwrite
  4593, // qsort
  4599, // read
  4604, // readlink
  4613, // realloc
  4621, // reallocf
  4630, // reallocarray
  4643, // realpath
  4652, // remainder
  4662, // remainderf
  4673, // remainderl
  4684, // remquo
  4691, // remquof
  4699, // remquol
  4707, // fdim
  4712, // fdimf
  4718, // fdiml
  4724, // remove
  4731, // rename
  4738, // rewind
  4745, // rint
  4750, // rintf
  4756, // rintl
  4762, // rmdir
  4768, // round
  4774, // roundeven
  4784, // roundevenf
  4795, // roundevenl
  4806, // roundf
  4813, // roundl
  4820, // scalbln
  4828, // scalblnf
  4837, // scalblnl
  4846, // scalbn
  4853, // scalbnf
  4861, // scalbnl
  4869, // scanf
  4875, // setbuf
  4882, // setitimer
  4892, // setvbuf
  4900, // sin
  4904, // sinf
  4909, // sinh
  4914, // sinhf
  4920, // sinhl
  4926, // sinl
  4931, // sincos
  4938, // sincosf
  4946, // sincosl
  4954, // siprintf
  4963, // snprintf
  4972, // sprintf
  4980, // sqrt
  4985, // sqrtf
  4991, // sqrtl
  4997, // sscanf
  5004, // stat
  5009, // stat64
  5016, // statvfs
  5024, // statvfs64
  5034, // stpcpy
  5041, // stpncpy
  5049, // strcasecmp
  5060, // strcat
  5067, // strchr
  5074, // strcmp
  5081, // strcoll
  5089, // strcpy
  5096, // strcspn
  5104, // strdup
  5111, // strlcat
  5119, // strlcpy
  5127, // strlen
  5134, // strncasecmp
  5146, // strncat
  5154, // strncmp
  5162, // strncpy
  5170, // strndup
  5178, // strnlen
  5186, // strpbrk
  5194, // strrchr
  5202, // strspn
  5209, // strstr
  5216, // strtod
  5223, // strtof
  5230, // strtok
  5237, // strtok_r
  5246, // strtol
  5253, // strtold
  5261, // strtoll
  5269, // strtoul
  5277, // strtoull
  5286, // strxfrm
  5294, // system
  5301, // tan
  5305, // tanf
  5310, // tanh
  5315, // tanhf
  5321, // tanhl
  5327, // tanl
  5332, // times
  5338, // tmpfile
  5346, // tmpfile64
  5356, // toascii
  5364, // trunc
  5370, // truncf
  5377, // truncl
  5384, // uname
  5390, // ungetc
  5397, // unlink
  5404, // unsetenv
  5413, // utime
  5419, // utimes
  5426, // valloc
  5433, // vec_calloc
  5444, // vec_free
  5453, // vec_malloc
  5464, // vec_realloc
  5476, // vfprintf
  5485, // vfscanf
  5493, // vprintf
  5501, // vscanf
  5508, // vsnprintf
  5518, // vsprintf
  5527, // vsscanf
  5535, // wcslen
  5542, // write
};
const uint8_t TargetLibraryInfoImpl::StandardNamesSizeTable[531] = {
  0,
  12,
  30,
  14,
  33,
  12,
  30,
  13,
  13,
  32,
  15,
  13,
  31,
  15,
  34,
  13,
  31,
  14,
  14,
  33,
  16,
  8,
  8,
  6,
  20,
  21,
  35,
  7,
  22,
  7,
  22,
  6,
  20,
  21,
  35,
  7,
  22,
  7,
  22,
  5,
  19,
  20,
  34,
  5,
  19,
  19,
  33,
  20,
  34,
  34,
  48,
  5,
  19,
  20,
  34,
  5,
  19,
  19,
  33,
  20,
  34,
  34,
  48,
  20,
  29,
  28,
  37,
  13,
  14,
  14,
  15,
  15,
  14,
  13,
  14,
  14,
  14,
  15,
  15,
  14,
  15,
  15,
  13,
  14,
  13,
  14,
  14,
  7,
  8,
  12,
  6,
  5,
  4,
  5,
  15,
  11,
  17,
  19,
  19,
  14,
  15,
  15,
  13,
  14,
  14,
  12,
  13,
  13,
  14,
  15,
  19,
  18,
  14,
  15,
  15,
  13,
  14,
  14,
  12,
  13,
  13,
  13,
  12,
  13,
  13,
  12,
  14,
  12,
  13,
  13,
  16,
  17,
  13,
  14,
  14,
  7,
  8,
  15,
  14,
  15,
  14,
  13,
  13,
  14,
  14,
  12,
  13,
  12,
  12,
  8,
  13,
  13,
  12,
  13,
  13,
  9,
  10,
  15,
  14,
  3,
  6,
  4,
  5,
  5,
  6,
  6,
  5,
  13,
  4,
  5,
  5,
  6,
  6,
  5,
  4,
  5,
  6,
  6,
  5,
  5,
  6,
  6,
  5,
  4,
  4,
  4,
  5,
  4,
  5,
  5,
  4,
  5,
  5,
  6,
  4,
  5,
  5,
  4,
  5,
  5,
  5,
  5,
  8,
  8,
  8,
  9,
  9,
  3,
  4,
  4,
  5,
  5,
  4,
  7,
  3,
  4,
  4,
  6,
  7,
  7,
  5,
  6,
  6,
  5,
  6,
  6,
  6,
  7,
  3,
  5,
  6,
  6,
  4,
  5,
  5,
  4,
  4,
  5,
  6,
  6,
  4,
  5,
  5,
  6,
  6,
  4,
  6,
  6,
  3,
  4,
  5,
  5,
  14,
  7,
  5,
  14,
  6,
  8,
  9,
  5,
  6,
  6,
  3,
  4,
  5,
  4,
  5,
  5,
  4,
  5,
  5,
  12,
  13,
  13,
  12,
  13,
  13,
  4,
  5,
  5,
  5,
  7,
  4,
  7,
  5,
  14,
  5,
  14,
  5,
  14,
  4,
  5,
  6,
  6,
  6,
  5,
  6,
  8,
  7,
  5,
  7,
  8,
  10,
  5,
  6,
  8,
  12,
  11,
  6,
  15,
  4,
  13,
  7,
  16,
  6,
  9,
  10,
  8,
  4,
  12,
  5,
  5,
  5,
  6,
  6,
  7,
  7,
  7,
  4,
  6,
  5,
  6,
  6,
  5,
  3,
  5,
  6,
  6,
  5,
  6,
  6,
  4,
  5,
  5,
  5,
  6,
  6,
  4,
  5,
  5,
  4,
  4,
  5,
  7,
  6,
  8,
  7,
  6,
  6,
  6,
  7,
  7,
  7,
  6,
  16,
  15,
  15,
  5,
  6,
  4,
  5,
  5,
  3,
  4,
  4,
  9,
  10,
  10,
  9,
  10,
  10,
  10,
  11,
  11,
  5,
  5,
  4,
  6,
  7,
  6,
  6,
  5,
  14,
  3,
  4,
  4,
  5,
  6,
  4,
  13,
  7,
  16,
  4,
  7,
  6,
  5,
  4,
  8,
  7,
  8,
  12,
  8,
  9,
  10,
  10,
  6,
  7,
  7,
  4,
  5,
  5,
  6,
  6,
  6,
  4,
  5,
  5,
  5,
  5,
  9,
  10,
  10,
  6,
  6,
  7,
  8,
  8,
  6,
  7,
  7,
  5,
  6,
  9,
  7,
  3,
  4,
  4,
  5,
  5,
  4,
  6,
  7,
  7,
  8,
  8,
  7,
  4,
  5,
  5,
  6,
  4,
  6,
  7,
  9,
  6,
  7,
  10,
  6,
  6,
  6,
  7,
  6,
  7,
  6,
  7,
  7,
  6,
  11,
  7,
  7,
  7,
  7,
  7,
  7,
  7,
  6,
  6,
  6,
  6,
  6,
  8,
  6,
  7,
  7,
  7,
  8,
  7,
  6,
  3,
  4,
  4,
  5,
  5,
  4,
  5,
  7,
  9,
  7,
  5,
  6,
  6,
  5,
  6,
  6,
  8,
  5,
  6,
  6,
  10,
  8,
  10,
  11,
  8,
  7,
  7,
  6,
  9,
  8,
  7,
  6,
  5,
};
#endif

#ifdef GET_TARGET_LIBRARY_INFO_IMPL_DECL
#undef GET_TARGET_LIBRARY_INFO_IMPL_DECL
LLVM_ABI static const llvm::StringTable StandardNamesStrTable;
LLVM_ABI static const llvm::StringTable::Offset StandardNamesOffsets[531];
LLVM_ABI static const uint8_t StandardNamesSizeTable[531];
#endif

#ifdef GET_TARGET_LIBRARY_INFO_SIGNATURE_TABLE
#undef GET_TARGET_LIBRARY_INFO_SIGNATURE_TABLE
enum FuncArgTypeID : char {
  NoFuncArgType = 0,
  Void,
  Bool,
  Int16,
  Int32,
  Int,
  IntPlus,
  Long,
  IntX,
  Int64,
  LLong,
  SizeT,
  SSizeT,
  Flt,
  Dbl,
  LDbl,
  Floating,
  Ptr,
  Struct,
  Ellip,
  Same,
};
static const FuncArgTypeID SignatureTable[] = {
  /* 0 */ Ptr, Long, Long, Bool, NoFuncArgType,
  /* 5 */ Ptr, Long, Bool, NoFuncArgType,
  /* 9 */ Ptr, Long, Long, Ptr, Bool, NoFuncArgType,
  /* 15 */ Ptr, Long, Ptr, Bool, NoFuncArgType,
  /* 20 */ Dbl, Dbl, Dbl, NoFuncArgType,
  /* 24 */ Int, Dbl, NoFuncArgType,
  /* 27 */ Int, Ptr, Int, Ellip, NoFuncArgType,
  /* 32 */ Int, Ptr, Ellip, NoFuncArgType,
  /* 36 */ Int, Ptr, Ptr, Ellip, NoFuncArgType,
  /* 41 */ Int, Ptr, Int, SizeT, Ptr, Ellip, NoFuncArgType,
  /* 48 */ Int, Ptr, SizeT, Int, SizeT, Ptr, Ellip, NoFuncArgType,
  /* 56 */ Int, Ptr, SizeT, Ptr, Ellip, NoFuncArgType,
  /* 62 */ Flt, Flt, Flt, NoFuncArgType,
  /* 66 */ Int, Flt, NoFuncArgType,
  /* 69 */ Dbl, Dbl, Int, NoFuncArgType,
  /* 73 */ Flt, Flt, Int, NoFuncArgType,
  /* 77 */ Void, Ptr, Int, Int, NoFuncArgType,
  /* 82 */ Int, Ptr, Int64, Int, NoFuncArgType,
  /* 87 */ Int, Ptr, IntX, Int, NoFuncArgType,
  /* 92 */ LDbl, LDbl, Int, NoFuncArgType,
  /* 96 */ Int, Ptr, Long, Int, NoFuncArgType,
  /* 101 */ Int, Ptr, Int, NoFuncArgType,
  /* 105 */ LLong, Ptr, Ptr, Int, NoFuncArgType,
  /* 110 */ Long, Ptr, Ptr, Int, NoFuncArgType,
  /* 115 */ Void, SizeT, Ptr, Ptr, Int, NoFuncArgType,
  /* 121 */ Void, Ptr, Int, NoFuncArgType,
  /* 125 */ Void, Int, NoFuncArgType,
  /* 128 */ Int16, Int16, NoFuncArgType,
  /* 131 */ Int32, Int32, NoFuncArgType,
  /* 134 */ Void, Ptr, IntPlus, NoFuncArgType,
  /* 138 */ SSizeT, Int, Ptr, SizeT, IntPlus, NoFuncArgType,
  /* 144 */ Int, Ptr, IntX, IntX, NoFuncArgType,
  /* 149 */ Int, Ptr, IntX, NoFuncArgType,
  /* 153 */ Dbl, Dbl, LDbl, NoFuncArgType,
  /* 157 */ Flt, Flt, LDbl, NoFuncArgType,
  /* 161 */ Int, LDbl, NoFuncArgType,
  /* 164 */ LDbl, LDbl, LDbl, NoFuncArgType,
  /* 168 */ Int, LLong, NoFuncArgType,
  /* 171 */ LLong, LLong, NoFuncArgType,
  /* 174 */ Void, Ptr, LLong, NoFuncArgType,
  /* 178 */ Dbl, Dbl, Long, NoFuncArgType,
  /* 182 */ Flt, Flt, Long, NoFuncArgType,
  /* 186 */ Int, Long, NoFuncArgType,
  /* 189 */ LDbl, LDbl, Long, NoFuncArgType,
  /* 193 */ Void, Ptr, Long, Long, NoFuncArgType,
  /* 198 */ Void, Ptr, Long, NoFuncArgType,
  /* 202 */ Dbl, Dbl, Dbl, Ptr, NoFuncArgType,
  /* 207 */ Flt, Flt, Flt, Ptr, NoFuncArgType,
  /* 212 */ Ptr, Int, Int, Ptr, NoFuncArgType,
  /* 217 */ Ptr, Ptr, Int, Ptr, NoFuncArgType,
  /* 222 */ Int64, Ptr, NoFuncArgType,
  /* 225 */ Void, Ptr, IntPlus, Ptr, NoFuncArgType,
  /* 230 */ LDbl, LDbl, LDbl, Ptr, NoFuncArgType,
  /* 235 */ Ptr, LLong, Ptr, NoFuncArgType,
  /* 239 */ Ptr, Long, Long, Ptr, NoFuncArgType,
  /* 244 */ Ptr, Long, Ptr, NoFuncArgType,
  /* 248 */ Void, Dbl, Ptr, Ptr, NoFuncArgType,
  /* 253 */ Void, Flt, Ptr, Ptr, NoFuncArgType,
  /* 258 */ Int, Int, Ptr, Ptr, NoFuncArgType,
  /* 263 */ Void, LDbl, Ptr, Ptr, NoFuncArgType,
  /* 268 */ Int, Ptr, Ptr, Ptr, NoFuncArgType,
  /* 273 */ Ptr, Ptr, Ptr, Ptr, NoFuncArgType,
  /* 278 */ Void, Ptr, Ptr, Ptr, NoFuncArgType,
  /* 283 */ Int, Ptr, Int, SizeT, Ptr, Ptr, NoFuncArgType,
  /* 290 */ Int, Ptr, SizeT, Int, SizeT, Ptr, Ptr, NoFuncArgType,
  /* 298 */ Int, Ptr, SizeT, Ptr, Ptr, NoFuncArgType,
  /* 304 */ Void, Ptr, Ptr, NoFuncArgType,
  /* 308 */ SizeT, Ptr, SizeT, SizeT, Ptr, NoFuncArgType,
  /* 314 */ Void, Ptr, SizeT, SizeT, Ptr, NoFuncArgType,
  /* 320 */ Void, Ptr, NoFuncArgType,
  /* 323 */ Long, Same, NoFuncArgType,
  /* 326 */ Floating, Same, Same, NoFuncArgType,
  /* 330 */ Int, Ptr, Ptr, Int, SizeT, NoFuncArgType,
  /* 336 */ Ptr, Ptr, Ptr, Int, SizeT, NoFuncArgType,
  /* 342 */ SSizeT, Int, Ptr, SizeT, NoFuncArgType,
  /* 347 */ Int, Ptr, Ptr, SizeT, NoFuncArgType,
  /* 352 */ Ptr, Ptr, Ptr, SizeT, NoFuncArgType,
  /* 357 */ SSizeT, Ptr, Ptr, SizeT, NoFuncArgType,
  /* 362 */ SizeT, Ptr, Ptr, SizeT, NoFuncArgType,
  /* 367 */ Void, Ptr, Ptr, SizeT, NoFuncArgType,
  /* 372 */ SizeT, Ptr, SizeT, NoFuncArgType,
  /* 376 */ Void, Ptr, SizeT, NoFuncArgType,
  /* 380 */ Ptr, Ptr, Ptr, Int, SizeT, SizeT, NoFuncArgType,
  /* 387 */ Int, Ptr, SizeT, SizeT, NoFuncArgType,
  /* 392 */ Ptr, Ptr, Ptr, SizeT, SizeT, NoFuncArgType,
  /* 398 */ SizeT, Ptr, Ptr, SizeT, SizeT, NoFuncArgType,
  /* 404 */ Void, NoFuncArgType,
};
static const uint16_t SignatureOffset[] = {
  404, //
  102, // msvc_new_int
  218, // msvc_new_int_nothrow
  175, // msvc_new_longlong
  235, // msvc_new_longlong_nothrow
  320, // msvc_delete_ptr32
  304, // msvc_delete_ptr32_nothrow
  121, // msvc_delete_ptr32_int
  320, // msvc_delete_ptr64
  304, // msvc_delete_ptr64_nothrow
  174, // msvc_delete_ptr64_longlong
  102, // msvc_new_array_int
  218, // msvc_new_array_int_nothrow
  175, // msvc_new_array_longlong
  235, // msvc_new_array_longlong_nothrow
  320, // msvc_delete_array_ptr32
  304, // msvc_delete_array_ptr32_nothrow
  121, // msvc_delete_array_ptr32_int
  320, // msvc_delete_array_ptr64
  304, // msvc_delete_array_ptr64_nothrow
  174, // msvc_delete_array_ptr64_longlong
  214, // under_IO_getc
  213, // under_IO_putc
  320, // ZdaPv
  304, // ZdaPvRKSt9nothrow_t
  134, // ZdaPvSt11align_val_t
  225, // ZdaPvSt11align_val_tRKSt9nothrow_t
  121, // ZdaPvj
  77, // ZdaPvjSt11align_val_t
  198, // ZdaPvm
  193, // ZdaPvmSt11align_val_t
  320, // ZdlPv
  304, // ZdlPvRKSt9nothrow_t
  134, // ZdlPvSt11align_val_t
  225, // ZdlPvSt11align_val_tRKSt9nothrow_t
  121, // ZdlPvj
  77, // ZdlPvjSt11align_val_t
  198, // ZdlPvm
  193, // ZdlPvmSt11align_val_t
  102, // Znaj
  218, // ZnajRKSt9nothrow_t
  78, // ZnajSt11align_val_t
  212, // ZnajSt11align_val_tRKSt9nothrow_t
  199, // Znam
  5, // Znam12__hot_cold_t
  244, // ZnamRKSt9nothrow_t
  15, // ZnamRKSt9nothrow_t12__hot_cold_t
  194, // ZnamSt11align_val_t
  0, // ZnamSt11align_val_t12__hot_cold_t
  239, // ZnamSt11align_val_tRKSt9nothrow_t
  9, // ZnamSt11align_val_tRKSt9nothrow_t12__hot_cold_t
  102, // Znwj
  218, // ZnwjRKSt9nothrow_t
  78, // ZnwjSt11align_val_t
  212, // ZnwjSt11align_val_tRKSt9nothrow_t
  199, // Znwm
  5, // Znwm12__hot_cold_t
  244, // ZnwmRKSt9nothrow_t
  15, // ZnwmRKSt9nothrow_t12__hot_cold_t
  194, // ZnwmSt11align_val_t
  0, // ZnwmSt11align_val_t12__hot_cold_t
  239, // ZnwmSt11align_val_tRKSt9nothrow_t
  9, // ZnwmSt11align_val_tRKSt9nothrow_t12__hot_cold_t
  4, // size_returning_new
  4, // size_returning_new_hot_cold
  4, // size_returning_new_aligned
  4, // size_returning_new_aligned_hot_cold
  21, // acos_finite
  63, // acosf_finite
  21, // acosh_finite
  63, // acoshf_finite
  165, // acoshl_finite
  165, // acosl_finite
  21, // asin_finite
  63, // asinf_finite
  165, // asinl_finite
  20, // atan2_finite
  62, // atan2f_finite
  164, // atan2l_finite
  21, // atanh_finite
  63, // atanhf_finite
  165, // atanhl_finite
  115, // atomic_load
  115, // atomic_store
  21, // cosh_finite
  63, // coshf_finite
  165, // coshl_finite
  21, // cospi
  63, // cospif
  268, // cxa_atexit
  214, // atexit
  404, // abort
  125, // exit
  125, // Exit
  404, // terminate
  278, // cxa_throw
  320, // cxa_guard_abort
  214, // cxa_guard_acquire
  320, // cxa_guard_release
  21, // exp10_finite
  63, // exp10f_finite
  165, // exp10l_finite
  21, // exp2_finite
  63, // exp2f_finite
  165, // exp2l_finite
  21, // exp_finite
  63, // expf_finite
  165, // expl_finite
  32, // dunder_isoc99_scanf
  36, // dunder_isoc99_sscanf
  344, // __kmpc_alloc_shared
  376, // __kmpc_free_shared
  21, // log10_finite
  63, // log10f_finite
  165, // log10l_finite
  21, // log2_finite
  63, // log2f_finite
  165, // log2l_finite
  21, // log_finite
  63, // logf_finite
  165, // logl_finite
  380, // memccpy_chk
  392, // memcpy_chk
  392, // memmove_chk
  392, // mempcpy_chk
  381, // memset_chk
  214, // nvvm_reflect
  20, // pow_finite
  62, // powf_finite
  164, // powl_finite
  4, // sincospi_stret
  4, // sincospif_stret
  21, // sinh_finite
  63, // sinhf_finite
  165, // sinhl_finite
  21, // sinpi
  63, // sinpif
  36, // small_fprintf
  32, // small_printf
  36, // small_sprintf
  48, // snprintf_chk
  41, // sprintf_chk
  21, // sqrt_finite
  63, // sqrtf_finite
  165, // sqrtl_finite
  352, // stpcpy_chk
  392, // stpncpy_chk
  352, // strcat_chk
  352, // strcpy_chk
  250, // dunder_strdup
  398, // strlcat_chk
  398, // strlcpy_chk
  372, // strlen_chk
  392, // strncat_chk
  392, // strncpy_chk
  348, // dunder_strndup
  273, // dunder_strtok_r
  290, // vsnprintf_chk
  283, // vsprintf_chk
  79, // abs
  101, // access
  21, // acos
  63, // acosf
  21, // acosh
  63, // acoshf
  165, // acoshl
  165, // acosl
  388, // aligned_alloc
  21, // asin
  63, // asinf
  21, // asinh
  63, // asinhf
  165, // asinhl
  165, // asinl
  21, // atan
  20, // atan2
  62, // atan2f
  164, // atan2l
  63, // atanf
  21, // atanh
  63, // atanhf
  165, // atanhl
  165, // atanl
  204, // atof
  214, // atoi
  241, // atol
  236, // atoll
  347, // bcmp
  367, // bcopy
  376, // bzero
  4, // cabs
  4, // cabsf
  4, // cabsl
  388, // calloc
  21, // cbrt
  63, // cbrtf
  165, // cbrtl
  21, // ceil
  63, // ceilf
  165, // ceill
  149, // chmod
  144, // chown
  320, // clearerr
  214, // closedir
  20, // copysign
  62, // copysignf
  164, // copysignl
  21, // cos
  63, // cosf
  21, // cosh
  63, // coshf
  165, // coshl
  165, // cosl
  250, // ctermid
  21, // erf
  63, // erff
  165, // erfl
  21, // tgamma
  63, // tgammaf
  165, // tgammal
  36, // execl
  36, // execle
  36, // execlp
  259, // execv
  268, // execvP
  268, // execve
  259, // execvp
  268, // execvpe
  21, // exp
  21, // exp10
  63, // exp10f
  165, // exp10l
  21, // exp2
  63, // exp2f
  165, // exp2l
  63, // expf
  165, // expl
  21, // expm1
  63, // expm1f
  165, // expm1l
  21, // fabs
  63, // fabsf
  165, // fabsl
  214, // fclose
  218, // fdopen
  214, // feof
  214, // ferror
  214, // fflush
  79, // ffs
  186, // ffsl
  168, // ffsll
  214, // fgetc
  214, // fgetc_unlocked
  259, // fgetpos
  217, // fgets
  217, // fgets_unlocked
  214, // fileno
  36, // fiprintf
  320, // flockfile
  21, // floor
  63, // floorf
  165, // floorl
  79, // fls
  186, // flsl
  168, // flsll
  326, // fmax
  326, // fmaxf
  326, // fmaxl
  326, // fmin
  326, // fminf
  326, // fminl
  326, // fmaximum_num
  326, // fmaximum_numf
  326, // fmaximum_numl
  326, // fminimum_num
  326, // fminimum_numf
  326, // fminimum_numl
  20, // fmod
  62, // fmodf
  164, // fmodl
  269, // fopen
  269, // fopen64
  71, // fork
  36, // fprintf
  213, // fputc
  213, // fputc_unlocked
  259, // fputs
  259, // fputs_unlocked
  308, // fread
  308, // fread_unlocked
  320, // free
  203, // frexp
  208, // frexpf
  231, // frexpl
  36, // fscanf
  96, // fseek
  87, // fseeko
  82, // fseeko64
  259, // fsetpos
  213, // fstat
  213, // fstat64
  213, // fstatvfs
  213, // fstatvfs64
  241, // ftell
  227, // ftello
  222, // ftello64
  214, // ftrylockfile
  320, // funlockfile
  308, // fwrite
  308, // fwrite_unlocked
  214, // getc
  214, // getc_unlocked
  71, // getchar
  71, // getchar_unlocked
  250, // getenv
  213, // getitimer
  343, // getlogin_r
  250, // getpwnam
  250, // gets
  259, // gettimeofday
  131, // htonl
  128, // htons
  20, // hypot
  62, // hypotf
  164, // hypotl
  32, // iprintf
  79, // isascii
  79, // isdigit
  323, // labs
  144, // lchown
  69, // ldexp
  73, // ldexpf
  92, // ldexpl
  171, // llabs
  21, // log
  21, // log10
  63, // log10f
  165, // log10l
  21, // log1p
  63, // log1pf
  165, // log1pl
  21, // log2
  63, // log2f
  165, // log2l
  24, // ilogb
  66, // ilogbf
  161, // ilogbl
  21, // logb
  63, // logbf
  165, // logbl
  63, // logf
  165, // logl
  259, // lstat
  259, // lstat64
  344, // malloc
  388, // memalign
  336, // memccpy
  331, // memchr
  347, // memcmp
  352, // memcpy
  352, // memmove
  352, // mempcpy
  331, // memrchr
  331, // memset
  367, // memset_pattern16
  367, // memset_pattern4
  367, // memset_pattern8
  149, // mkdir
  227, // mktime
  203, // modf
  208, // modff
  231, // modfl
  204, // nan
  209, // nanf
  232, // nanl
  21, // nearbyint
  63, // nearbyintf
  165, // nearbyintl
  20, // nextafter
  62, // nextafterf
  164, // nextafterl
  153, // nexttoward
  157, // nexttowardf
  164, // nexttowardl
  131, // ntohl
  128, // ntohs
  27, // open
  27, // open64
  250, // opendir
  214, // pclose
  320, // perror
  269, // popen
  387, // posix_memalign
  20, // pow
  62, // powf
  164, // powl
  138, // pread
  32, // printf
  213, // putc
  213, // putc_unlocked
  79, // putchar
  79, // putchar_unlocked
  214, // puts
  344, // pvalloc
  138, // pwrite
  314, // qsort
  342, // read
  357, // readlink
  348, // realloc
  348, // reallocf
  393, // reallocarray
  269, // realpath
  20, // remainder
  62, // remainderf
  164, // remainderl
  202, // remquo
  207, // remquof
  230, // remquol
  20, // fdim
  62, // fdimf
  164, // fdiml
  214, // remove
  259, // rename
  320, // rewind
  21, // rint
  63, // rintf
  165, // rintl
  214, // rmdir
  21, // round
  21, // roundeven
  63, // roundevenf
  165, // roundevenl
  63, // roundf
  165, // roundl
  178, // scalbln
  182, // scalblnf
  189, // scalblnl
  69, // scalbn
  73, // scalbnf
  92, // scalbnl
  32, // scanf
  304, // setbuf
  258, // setitimer
  330, // setvbuf
  21, // sin
  63, // sinf
  21, // sinh
  63, // sinhf
  165, // sinhl
  165, // sinl
  248, // sincos
  253, // sincosf
  263, // sincosl
  36, // siprintf
  56, // snprintf
  36, // sprintf
  21, // sqrt
  63, // sqrtf
  165, // sqrtl
  36, // sscanf
  259, // stat
  259, // stat64
  259, // statvfs
  259, // statvfs64
  269, // stpcpy
  352, // stpncpy
  259, // strcasecmp
  269, // strcat
  106, // strchr
  259, // strcmp
  259, // strcoll
  269, // strcpy
  286, // strcspn
  250, // strdup
  362, // strlcat
  362, // strlcpy
  311, // strlen
  347, // strncasecmp
  352, // strncat
  347, // strncmp
  352, // strncpy
  348, // strndup
  372, // strnlen
  269, // strpbrk
  106, // strrchr
  286, // strspn
  269, // strstr
  249, // strtod
  254, // strtof
  269, // strtok
  273, // strtok_r
  110, // strtol
  264, // strtold
  105, // strtoll
  110, // strtoul
  105, // strtoull
  362, // strxfrm
  214, // system
  21, // tan
  63, // tanf
  21, // tanh
  63, // tanhf
  165, // tanhl
  165, // tanl
  227, // times
  205, // tmpfile
  205, // tmpfile64
  79, // toascii
  21, // trunc
  63, // truncf
  165, // truncl
  214, // uname
  213, // ungetc
  214, // unlink
  214, // unsetenv
  259, // utime
  259, // utimes
  344, // valloc
  388, // vec_calloc
  320, // vec_free
  344, // vec_malloc
  348, // vec_realloc
  268, // vfprintf
  268, // vfscanf
  259, // vprintf
  259, // vscanf
  298, // vsnprintf
  268, // vsprintf
  268, // vsscanf
  311, // wcslen
  342, // write
};
#endif

