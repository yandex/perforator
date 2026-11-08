from functools import cache
import os
from pathlib import Path
import subprocess
import tempfile

from library.python import resource


@cache
def _tool_directory():
    binary = resource.find("nots/builder/nots-tar")
    if binary is None:
        raise RuntimeError("The builder is missing its embedded nots-tar executable")
    directory = tempfile.TemporaryDirectory(prefix="nots-tar-")
    executable = Path(directory.name) / "nots-tar"
    executable.write_bytes(binary)
    executable.chmod(0o700)
    return directory


def _run(*args, inputs=None):
    directory = _tool_directory()
    result = subprocess.run(
        [os.path.join(directory.name, "nots-tar"), *args],
        input=inputs if inputs is not None else b"",
        stderr=subprocess.PIPE,
    )
    if result.returncode:
        raise RuntimeError(f"nots-tar {args[0]} failed: {os.fsdecode(result.stderr).strip()}")


def pack(paths, destination, *, layer=False):
    inputs = bytearray()
    for source, archive_path in paths:
        inputs.extend(os.fsencode(source))
        inputs.append(0)
        inputs.extend(os.fsencode(archive_path))
        inputs.append(0)
    _run("pack-layer" if layer else "pack-tar", destination, inputs=inputs)


def unpack_tar(source, destination, prefix, *, keep_existing=False):
    options = ["--keep-existing"] if keep_existing else []
    _run("unpack-tar", source, destination, prefix, *options)


def unpack_layer(source, destination):
    parent = os.path.dirname(os.path.abspath(destination))
    os.makedirs(parent, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix=".nots-layer-", dir=parent) as temporary:
        output = os.path.join(temporary, "node_modules")
        _run("unpack-layer", source, output)
        os.replace(output, destination)
