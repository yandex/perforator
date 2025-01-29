# ELF Parsing

## Python Version

There are multiple ways to parse the Python version from an ELF file that we utilize in Perforator.

### `Py_Version` Symbol

There is a `Py_Version` symbol in the ELF file. This is a global variable that stores 4 bytes of the Python version: the first byte is the major version, the second byte is the minor version, the third byte is the micro version, and the fourth byte is the release level.

### Disassemble `Py_GetVersion` Function

For CPython versions earlier than 3.11, we can disassemble the `Py_GetVersion` function to get the version.

There is this line of code inside the `Py_GetVersion` function:

```
PyOS_snprintf(version, sizeof(version), buildinfo_format, PY_VERSION, Py_GetBuildInfo(), Py_GetCompiler());
```

Perforator extracts the 4th argument, which is a pointer to a constant global string with the Python version in the `.rodata` section.

Then, we can read the version as a string from this address in the binary.

## `_PyRuntime` Global Variable

There is a `_PyRuntime` symbol in the `.dynsym` section which can be used to obtain the address of the `_PyRuntime` global variable.

## Disassembling `PyThreadState_GetCurrent`

The `PyThreadState_GetCurrent` function can be disassembled to get the offset of `_PyThreadState_Current` in the Thread Local Image for [Python Thread State collection](./threadstate.md).
