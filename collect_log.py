from collections import defaultdict
from os import system
from re import L
import subprocess
import sys

def get_function_name(executable_path, address):
    """
    Calls addr2line to retrieve the function name for a given address.

    Args:
        executable_path (str): Path to the executable file with debug info.
        address (str): The memory address (e.g., '0x4005bdc').

    Returns:
        str: The function name or None if an error occurs.
    """
    # Arguments for the addr2line command:
    # -f: print function names
    # -C: demangle C++ names
    # -e <executable_path>: specify the executable file
    command = ["addr2line", "-f", "-C", "-e", executable_path, address]

    try:
        # Use subprocess.run to execute the command and capture output
        result = subprocess.run(
            command,
            capture_output=True,
            text=True, # Decode output as a string (Python 3.5+)
            check=True # Raise an exception if the command fails
        )

        # The output will have the function name on the first line
        # and the file/line info on the second.
        output_lines = result.stdout.strip().split('\n')
        if output_lines:
            function_name = output_lines[0]
            # addr2line uses '??' for unknown function names
            if function_name == '??':
                return None
            return function_name

    except FileNotFoundError:
        print(f"Error: 'addr2line' command not found. Is it installed and in your PATH?", file=sys.stderr)
    except subprocess.CalledProcessError as e:
        print(f"Error running addr2line: {e.stderr}", file=sys.stderr)
    except Exception as e:
        print(f"An unexpected error occurred: {e}", file=sys.stderr)

    return None

lbase_and_rips = defaultdict(set)
lbase = None
current_address = None



with open('./bpf.log', 'r') as file:
    for linen, line in enumerate(file):
        if 'CURRENT rip:' in line:
            parts = line.split('CURRENT rip:')
            if len(parts) > 1:
                current_address = hex(int(parts[1].strip(), base=16) - int("00005fb63f4c5000", base=16))
        elif 'L->base=' in line and current_address is not None:
            parts = line.split('L->base=')
            if len(parts) > 1:
                lbase = parts[1].strip()
        elif '!!! bad frame function' in line and current_address and lbase:
            # if lbase == "00005fb673a2c030" and current_address == hex(0x00005fb63f4ecb10 - 0x00005fb63f4c5000):
                # print("line: ", linen)
            lbase_and_rips[lbase].add(current_address)
            current_address = None
            lbase = None

# Print or use the collected addresses
for lbase, rip_addresses in lbase_and_rips.items():
    print(f"L->base: {lbase}")

    for rip in rip_addresses:
        print((get_function_name("/usr/local/bin/luajit", rip), rip), end = ", ")

    print()
