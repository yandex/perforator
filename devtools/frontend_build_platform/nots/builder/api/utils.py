import os
import shutil
import stat
import subprocess
import sys

import library.python.archive as archive

from build.plugins.lib.nots.package_manager import (
    constants as pm_constants,
    PackageJson,
    utils as pm_utils,
)
from devtools.frontend_build_platform.libraries.logging import timeit


def eprint(*args, **kwargs):
    print(*args, file=sys.stderr, **kwargs)


@timeit
def extract_peer_tars(moddir_abs: str, visited: set[str] = set()):
    """Extracts all the output tars for the dependency tree, excluding root

    Args:
        moddir_abs: absolute path of the root module
        visited: set of paths we have visited already
    """
    pj = PackageJson.load(pm_utils.build_pj_path(moddir_abs))
    for dep_path in pj.get_workspace_dep_paths():
        extract_all_output_tars(dep_path, visited)


@timeit
def extract_all_output_tars(moddir_abs: str, visited: set[str] = set()):
    """Extracts all the output tars for the dependency tree, including root

    Args:
        moddir_abs: absolute path of the root module
        visited: set of paths we have visited already
    """
    if moddir_abs in visited:
        return

    visited.add(moddir_abs)
    try:
        _extract_output_tar(moddir_abs)
        extract_peer_tars(moddir_abs, visited)
    except Exception as e:
        eprint(f"could not extract output tar for {moddir_abs}: {e}")


@timeit
def _extract_output_tar(moddir_abs: str):
    """Extracts the output tar for a module

    Args:
        moddir_abs: absolute path of the module
    """
    output_tar_uuid_path = os.path.join(moddir_abs, pm_constants.OUTPUT_TAR_UUID_FILENAME)
    if not os.path.exists(output_tar_uuid_path):
        return

    with open(output_tar_uuid_path) as f:
        content = f.read()
        output_tar_filename = content.split(':', 1)[0]

    output_tar_path = os.path.join(moddir_abs, output_tar_filename)

    if not os.path.exists(output_tar_path):
        raise FileNotFoundError(output_tar_path)

    archive.extract_tar(output_tar_path, moddir_abs, fail_on_duplicates=False)


def __add_write_permissions(path):
    if not os.path.exists(path):
        eprint(f"Directory not exists: {path}")
        return

    dst_stat = os.stat(path)
    dst_mode = stat.S_IMODE(dst_stat.st_mode)
    upd_mode = dst_mode | stat.S_IWUSR | stat.S_IWGRP
    if dst_mode != upd_mode:
        try:
            os.chmod(path, upd_mode)
        except PermissionError:
            eprint(f"Can't update permissions for {path}")


def __copy_file_with_write_permissions(src, dst):
    os.makedirs(os.path.dirname(dst), exist_ok=True)
    shutil.copy(src, dst)
    __add_write_permissions(dst)


@timeit
def copy_if_not_exists(src: str, dst: str):
    """Copy file/directory skipping existing. Makes them writable."""
    if os.path.exists(dst):
        return

    if os.path.isdir(src):
        shutil.copytree(src, dst, ignore_dangling_symlinks=True, copy_function=__copy_file_with_write_permissions)

    if os.path.isfile(src):
        __copy_file_with_write_permissions(src, dst)


def recursive_copy_impl(src: str, dest: str, overwrite: bool, recurse_level=0):
    # just for avoiding extra tracing with @timeit decorator
    copy_fn = recursive_copy_impl if recurse_level >= 1 else recursive_copy

    __add_write_permissions(os.path.dirname(dest))

    if os.path.isdir(src):
        os.makedirs(dest, exist_ok=True)
        files = os.listdir(src)
        for f in files:
            copy_fn(os.path.join(src, f), os.path.join(dest, f), overwrite, recurse_level=recurse_level + 1)

    if os.path.isfile(src):
        if not os.path.exists(dest) or overwrite:
            __copy_file_with_write_permissions(src, dest)


@timeit
def recursive_copy(src, dest, overwrite=False, recurse_level=0):
    recursive_copy_impl(src, dest, overwrite, recurse_level=recurse_level)


@timeit
def simplify_colors(data):
    """
    Some tools use light-* colors instead of simple ones, this yet to be supported by ya make
    (refer FBP-999 for details)
    For now we can handle the light-* colors by transforming those into simple ones
    e.g. LIGHT_CYAN (96) -> CYAN (36)
    """
    from library.python import color

    for col in range(30, 38):
        high_col = col + 60
        data = data.replace(color.get_code(high_col), color.get_code(col))

    return data


@timeit
def popen(args: list[str], env: dict[str, str], cwd: str):
    p = subprocess.Popen(
        args,
        cwd=cwd,
        env=env,
        stderr=subprocess.PIPE,
        stdin=None,
        stdout=subprocess.PIPE,
        text=True,
    )
    stdout, stderr = p.communicate()
    return_code = p.returncode

    return return_code, stdout, stderr


def resolve_bin(cwd: str, package_name: str, bin_name: str = None) -> str:
    """
    Looks for the specified `bin_name` (or default) for the package
    :param package_name: Name of the package in `node_modules` dir
    :param bin_name: Custom "bin", defined in `package.json:bin` object
    :return: Full path to the script (.js file)
    """
    pj_path = os.path.join(
        cwd,
        pm_constants.NODE_MODULES_DIRNAME,
        package_name,
        pm_constants.PACKAGE_JSON_FILENAME,
    )
    pj = PackageJson.load(pj_path)
    bin_path = pj.get_bin_path(bin_name)

    assert bin_path is not None

    return os.path.normpath(os.path.join(cwd, pm_constants.NODE_MODULES_DIRNAME, package_name, bin_path))


def parse_opt_to_dict(opts: list[str]) -> dict[str, str]:
    result = {}
    for opt in opts:
        if "=" not in opt:
            raise AssertionError(f"ts_proto_opt should be in `key=value` format, got `{opt}`")

        key, value = opt.split("=", 1)
        result[key] = value
    return result


def dict_to_ts_proto_opt(d: dict[str, str]) -> str:
    return ','.join(f'{key}={value}' for key, value in d.items())
