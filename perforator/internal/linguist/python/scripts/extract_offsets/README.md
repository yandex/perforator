# Extract CPython Structure Field Offsets

This tool extracts the offsets of fields within CPython internal structures. It compiles and runs a C program that uses the `offsetof` macro to determine field offsets, ensuring the offsets are accurate for the specified CPython version.

## Requirements

- Python 3.x
- A C compiler (gcc)
- Git (for downloading CPython source code)

## Files

- `extract_offsets.py`: The main Python script
- `offsets.c`: C program that calculates the field offsets

## Usage

```bash
python extract_offsets.py [options]
```

### Arguments

- `--output-format`: Format for the output (choices: json, dict, plain; default: json)
- `--cpython-version`: **(Required)** Specific CPython version to use (e.g., 3.13.1, 3.12.1, 3.11.0, etc.). Must be an exact version tag.
- `--force-download`: Force re-download of CPython source even if it exists
- `--cpython-cache-dir`: Directory to store downloaded CPython sources (default: ~/.cpython_sources)
- `--target-arch`: Target architecture for CPython configure (e.g., x86_64, aarch64)
- `--libpthread-path`: Path to libpthread library for linking
- `--configure-opts`: Additional configure options (as a single quoted string)

### Examples

Extract offsets for a specific Python version:

```bash
python extract_offsets.py --cpython-version 3.13.1
```

Use a custom cache directory:

```bash
python extract_offsets.py --cpython-version 3.12.1 --cpython-cache-dir /path/to/cache
```

Use plain text output format:

```bash
python extract_offsets.py --output-format plain
```

## How it works

1. The script requires a specific CPython version tag:
   - Clones the CPython repository from GitHub to the specified cache directory with the exact version tag
   - Configures the CPython source to generate the necessary header files
   - Uses the source code include paths for building
   - Fails with an error if the exact version tag does not exist

2. It compiles a standalone C program (`offsets.c`) that calculates structure field offsets using offsetof()

3. The C program is executed, and it outputs the offsets in a simple text format

4. Finally, the script parses the output and formats the results in the specified format

## Version Compatibility

The script is tested and compatible with:
- Python 3.11.x
- Python 3.12.x
- Python 3.13.x

Field names and structures may differ between versions. The script handles these differences using version-specific code.

## Available Structures and Fields

The script currently provides offsets for the following structures and fields:

- **PyThreadState**
  - next
  - prev
  - native_thread_id
  - cframe (used before Python 3.13)
  - current_frame (added in Python 3.13)

- **_PyCFrame** (removed in Python 3.13)
  - current_frame

- **_PyInterpreterFrame**
  - f_code (Python 3.11-3.12 only)
  - f_executable (Python 3.13+, replaces f_code)
  - previous
  - owner (relevant only for CPython 3.12+)

- **_PyRuntimeState**
  - interpreters.main

- **PyInterpreterState**
  - threads.head
  - next

- **PyASCIIObject**
  - length
  - state
  - data
  - ascii_bit
  - compact_bit
  - static_bit

- **PyCodeObject**
  - co_filename
  - co_qualname (Python 3.11+)
  - co_firstlineno

## Key Features

- **Version-specific offsets**: Can extract offsets for any CPython version, not just the running one
- **Exact version matching**: Requires exact version tags with no fallbacks to similar versions
- **Caching**: Downloaded CPython sources are cached (default: ~/.cpython_sources)
- **Custom cache location**: Specify where to store downloaded CPython sources
- **Auto-detection of include paths**: Uses the correct include paths for the downloaded source
- **Always fresh builds**: The C program is completely rebuilt on every run to ensure latest changes are used
- **Version compatibility**: Gracefully handles missing structures or fields in different CPython versions
- **Detailed diagnostics**: Provides helpful error messages for common issues
- **Simple usage**: No need to specify structure-field pairs - gets all available offsets

## Error Handling

If the C program cannot be built, the script will:

1. Attempt to diagnose the issue and provide helpful error messages
2. Check for the presence of necessary include paths
3. Exit with an error code if the compilation fails

## Troubleshooting

If you encounter build errors:

1. Ensure you have Git installed for downloading CPython source

2. Ensure the `offsets.c` file exists in the same directory as the script

3. Ensure you have a compatible C compiler:
   - Linux: GCC
   - macOS: Clang
   - Windows: MSVC

4. If you get an error that the CPython version tag was not found:
   - Make sure you're specifying a valid version (e.g., 3.13.1, 3.12.1, 3.11.0)
   - Check available tags at https://github.com/python/cpython/tags
   - Note that the script requires an exact version match

5. If you have permission issues with the default cache directory:
   - Specify a custom cache directory: `--cpython-cache-dir /path/to/writable/directory`

6. If the compilation fails with "Python.h file not found" or similar errors:
   - The script may not be finding the CPython include paths correctly
   - Try running with verbose mode to see the include paths being used

## Extending

To add support for additional CPython structures:

1. Add new structure offset calculations to `offsets.c` in the `main()` function
2. Include any necessary headers
3. Use version checks (`PYVER_AT_LEAST`, `PYVER_BEFORE`) for version-specific structures or fields

The script will automatically rebuild the C program every time it runs. 