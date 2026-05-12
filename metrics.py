#!/usr/bin/env python

from prometheus_client.parser import text_string_to_metric_families
import json
import sys

filename = sys.argv[1]

with open(filename, "r") as f:
    data = f.read()

for line in data.splitlines():
    start = "Collected profiler metrics "
    pos = line.find(start)
    if pos != -1:
        metrics = line[pos + len(start) :]
        break

metrics_json = json.loads(metrics)
metrics_text = metrics_json["metrics"]

# Convert to dict and just print key-value pairs
result_dict = {}
for family in text_string_to_metric_families(metrics_text):
    for sample in family.samples:
        if "lua_" in sample.name or "bpf_prog_stack_" in sample.name or "bpf_prog_dwarf_" in sample.name or "fp_error" in sample.name:
            result_dict[sample.name] = sample.value

no_process_info = result_dict.pop("bpf_prog_error_stage_collect_lua_stack_count_total", -1)
invocations = result_dict.pop("bpf_prog_lua_processed_stacks_count_total", -1)

print(f"Invoked: {invocations} times. +{no_process_info} times there were no process info")

print()

cached = result_dict.pop("bpf_prog_lua_valid_cache_count_total", -1)
cache_invalid = result_dict.pop("bpf_prog_lua_invalided_cache_count_total", -1)

print(f"{cached} times reused cached state, {cache_invalid} times invalidated")

not_in_luajit = result_dict.pop("bpf_prog_lua_not_in_luajit_binary_count_total", -1)
found_from_registers = result_dict.pop("bpf_prog_lua_global_state_found_count_total", -1)
not_found_from_registers = result_dict.pop("bpf_prog_lua_global_state_not_found_count_total", -1)

print(f"Found valid state {found_from_registers} times, {not_found_from_registers} times not found, {not_in_luajit} was not in luajit binary")

check_fail_cur_l = result_dict.pop("bpf_prog_lua_cur_l_is_null_count_total", -1)
invalid_cur_l = result_dict.pop("bpf_prog_lua_cur_l_read_fail_count_total", -1)
check_fail_g_eq_g = result_dict.pop("bpf_prog_lua_g_eq_g_mismatch_count_total", -1)

print(f"State checks fail reasons: {check_fail_cur_l} times due to null cur_l ({invalid_cur_l} failed reads), {check_fail_g_eq_g} times due to mismatch G(g->cur_L) and g")

if invocations != (cached + not_in_luajit + found_from_registers + not_found_from_registers):
    print(f"Mismatch of invocations and state checks {invocations} != {cached + not_in_luajit + found_from_registers + not_found_from_registers}")

print()

invalid_state = result_dict.pop("bpf_prog_lua_null_state_count_total", -1)

if invalid_state > 0:
    print(f"{invalid_state} times L or G was NULL. Must be 0!")

invalid_read = result_dict.pop("bpf_prog_lua_deref_error_count_total", -1)
print(f"{invalid_read} times failed to read some generic pointer")

symbol_cache_mismatch = result_dict.pop("bpf_prog_lua_cache_mismatch_count_total", -1)
print(f"{symbol_cache_mismatch} times cached symbol was different")

print()

frames = result_dict.pop("bpf_prog_lua_processed_frames_count_total", -1)
failed_frames = result_dict.pop("bpf_prog_lua_get_function_info_fail_count_total", -1)
failed_frames_reason_broken_frame = result_dict.pop("bpf_prog_lua_broken_frame_count_total", -1)
failed_frames_reason_frame_is_null = result_dict.pop("bpf_prog_lua_frame_is_null_count_total", -1)
failed_frames_reason_function_is_null = result_dict.pop("bpf_prog_lua_function_is_null_count_total", -1)

print(f"Processed {frames} frames successfully")
print(f"Failed {failed_frames}. {failed_frames_reason_broken_frame} times frame wasn't LJ_TFUNC, {failed_frames_reason_frame_is_null} times frame was NULL, {failed_frames_reason_function_is_null} times function was NULL, {failed_frames - failed_frames_reason_broken_frame - failed_frames_reason_frame_is_null - failed_frames_reason_function_is_null} times other")

go_collected_frames = result_dict.pop("lua_frame_collected_count_total", -1)

if go_collected_frames != (frames + failed_frames):
    print(f"Go collected different amount of frames! {go_collected_frames} != {frames + failed_frames}")

unwind_frames_total = result_dict.pop("bpf_prog_stack_frame_count_total", -1)
unwind_frames_using_fp = result_dict.pop("bpf_prog_stack_frame_fp_count_total", -1)
unwind_frames_using_dwarf = result_dict.pop("bpf_prog_stack_frame_dwarf_count_total", -1)
fp_fail = result_dict.pop("bpf_prog_fp_error_read_returnaddress_count_total", -1)

print()

print(f"Frames unwounded: {unwind_frames_total} ({unwind_frames_using_fp} by FP, {unwind_frames_using_dwarf} by DWARF)")
print(f"FP failed {fp_fail} times")

print()

print("Other:")

# Print the key-value pairs
for name, value in result_dict.items():
    print(name, value)
