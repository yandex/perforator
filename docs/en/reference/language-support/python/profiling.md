# Python Profiling

Perforator supports stack unwinding for the latest releases of Python - 3.12 and 3.13 executing with CPython.

Cython is not supported yet. Previous versions of CPython will be supported soon.

## Problem

The native stack unwinding algorithm allows to collect stacks of different compiled programming languages in an eBPF program. However, trying to collect a Python process stack with the same algorithm will result in only seeing CPython runtime frames that are called to execute the user's code. To collect the user's Python stack, a different algorithm is needed. It traverses Python's internal structures and extracts valuable information about the execution.

## Algorithm

Each native thread is mapped to one `PyThreadState` structure that contains information about the corresponding Python thread. From this structure, we can extract information about the current executing frame of user code - the `struct _PyInterpreterFrame *current_frame;` field is responsible for this. In Python 3.11 to 3.12 versions, there is a proxy field `_PyCFrame *cframe`. The `_PyCFrame` structure also contains the `struct _PyInterpreterFrame *current_frame` field.

Having the top executing user frame, which is represented by the `_PyInterpreterFrame` structure, the stack can be collected. `_PyInterpreterFrame` structure contains the `f_code` or `f_executable` field that stores a pointer to the `PyCodeObject` structure, which can be utilized to extract the symbol name and line number. Also, there is a field `struct _PyInterpreterFrame *previous` pointing to the previous frame.

With all this knowledge the eBPF algorithm can be divided into these phases:

1. [Extract the corresponding `*PyThreadState`](./threadstate.md)
2. [Retrieve `current_frame` from `*PyThreadState`](./stack-unwinding.md)
3. [Walk the stack frames collecting symbol names](./symbolization.md)
4. [Symbolize frames in user space](./merging.md)

To follow all the steps the hardcode of the offsets of certain fields in CPython internal structures is needed. These offsets are not exported by CPython until Python 3.13. [The necessary information is extracted from the CPython ELF file.](./parse_elf.md)

The phases of the algorithm are described in the following sections.
