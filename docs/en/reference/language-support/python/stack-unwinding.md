# Python Stack Unwinding

Having a current `*PyThreadState` pointer a top executing frame is retrieved using the `current_frame` field and the frame chain is traversed using the `previous` field.

The process of passing symbols from the eBPF context to the user space is not trivial. Copying symbol names on each frame processing is avoided as it is not efficient.

[Python Symbolization](./symbolization.md) section describes how python symbols are handled in the eBPF program.
