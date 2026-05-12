import re
from collections import defaultdict

# Initialize a dictionary to store line numbers and their counts
line_counts = defaultdict[int, int](int)
line_infos = defaultdict[int, set[str]](set[str])
source_code_info = defaultdict[str, set[int]](set[int])

current_source_info = None

# Read the log file
with open("./log.log", "r") as file:
    for line in file:
        # Match lines containing the pattern
        match = re.search(
            r'Kernel verifier rejected the program.*?"log": "(.*)\"}', line
        )

        if not match:
            continue

        # Extract the log message from the matched group
        log_message = match.group(1)

        # Extract number from the beginning of log_message
        number_match = re.match(r'^(\d+)', log_message)

        if number_match:
            # Extract the line number from the matched group
            line_number = int(number_match.group(1))
            # Increment the count for this line number
            line_infos[line_number].add(log_message)
            line_counts[line_number] += 1

            if current_source_info:
                source_code_info[current_source_info].add(line_number)
        elif log_message.startswith(";"):
            current_source_info = log_message.removeprefix("; ")

# Sort the dictionary by value in descending order and print
def source_line_sorter(lines):
    total_count = 0

    for line in lines[1]:
        total_count += line_counts[line]

    return total_count

sorted_source_code_info = dict(
    sorted(source_code_info.items(), key=source_line_sorter, reverse=True)
)

for k, v in sorted_source_code_info.items():
    print(f"`{k}` instructions:")

    for instruction in v:
        print(f"\t#{instruction} visited {line_counts[instruction]} times")

        for line in sorted(line_infos[instruction]):
            print("\t\t", line.strip())

