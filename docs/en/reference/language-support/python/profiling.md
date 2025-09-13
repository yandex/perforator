# Python Profiling

Perforator supports stack unwinding for most of the CPython releases used in practice.

| СPython Version | Support Status | Requirements for the executable | Note |
|----------------|----------------|-------------|-------|
| 3.12 - 3.13           | ✅             | Vanilla build | 
| 3.11           | ✅             | Vanilla build and glibc libpthread.so 2.4+ linked | Python and native stack merging not adapted yet
| 3.x (<= 3.10) | ✅             | Vanilla build and glibc libpthread.so 2.4+ linked |
| 2.x (>= 2.4)         | ✅             | Vanilla build and glibc libpthread.so 2.4+ linked |
| Cython         | ✅              |  |

See [ELF Parsing Requirements](./parse_elf.md#requirements-for-elf-cpython-binary) for detailed binary requirements.

## Problem

The native stack unwinding algorithm allows to collect stacks of different compiled programming languages in an eBPF program. However, trying to collect a Python process stack with the same algorithm will result in only seeing CPython runtime frames that are called to execute the user's code. To collect the user's Python stack, a different algorithm is needed. It traverses Python's internal structures and extracts valuable information about the execution.

## Algorithm

Each native thread is mapped to one `PyThreadState` structure that contains information about the corresponding Python thread. From this structure, we can extract information about the current executing frame of user code - the `struct _PyInterpreterFrame *current_frame` or `struct _frame* frame` field is responsible for this. In Python 3.11 to 3.12 versions, there is a proxy field `_PyCFrame *cframe`. The `_PyCFrame` structure also contains the `struct _PyInterpreterFrame *current_frame` field.

Having the top executing user frame (`struct _PyInterpreterFrame` or `struct _frame`) the stack can be collected. Frame structure contains the code object field (`f_code` or `f_executable`) that stores a pointer to the `PyCodeObject` structure, which can be utilized to extract the symbol name and line number. Also, there is a pointer to the previous frame in the given frame structure.

With all this knowledge the eBPF algorithm can be divided into these phases:

1. [Extract the corresponding `*PyThreadState`](./threadstate.md)
2. [Retrieve current top frame from `*PyThreadState`](./stack-unwinding.md)
3. [Walk the stack frames collecting symbol names](./symbolization.md)
4. [Symbolize frames in user space](./merging.md)

To follow all the steps the hardcode of the offsets of certain fields in CPython internal structures is needed. These offsets are not exported by CPython until Python 3.13. [The necessary information is extracted from the CPython ELF file.](./parse_elf.md)

The phases of the algorithm are described in the following sections.
