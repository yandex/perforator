# Merging Python Stack with Native Stack

The native and Python stacks are collected separately for the same perf event. Afterwards these stacks are merged into a single stack for better visualization and analysis, because looking at two stacks simultaneously is not convenient.

## Stub Frames

When C code starts evaluating Python code through CPython API, it pushes a stub frame. Each `_PyInterpreterFrame` structure contains the `owner` field, which stores the `python_frame_owner` enum value.

```c
enum python_frame_owner : u8 {
    FRAME_OWNED_BY_THREAD = 0,
    FRAME_OWNED_BY_GENERATOR = 1,
    FRAME_OWNED_BY_FRAME_OBJECT = 2,
    FRAME_OWNED_BY_CSTACK = 3,
};
```

If the value is equal to `FRAME_OWNED_BY_CSTACK`, then the frame is a stub frame.

A stub frame is a delimiter between the native and Python stacks. This frame is pushed onto the native stack in the `_PyEval_EvalFrameDefault` function.

## Algorithm

The Python user stack is divided into segments each one starting with a stub frame. Also, segments of the native stack with CPython are extracted using `_PyEval_EvalFrameDefault` as a delimiter. The functions starting with the `_Py` or `Py` prefix are considered to be CPython internal implementation.

These stack segments should map one-to-one with each other, but there are some exceptions:

* `_PyEval_EvalFrameDefault` has started executing on top of the native stack but has not finished pushing the stub Python frame yet.
* The native stack contains entries like `PyImport_ImportModule`. Python importlib may drop its own frames from the native stack.

The first case is handled easily, while the second case is more complex and is ignored for now.
