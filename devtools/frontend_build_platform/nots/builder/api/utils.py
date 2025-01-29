import os
import shutil
import stat
import subprocess

import library.python.archive as archive
from build.plugins.lib.nots.package_manager import (
    constants as pm_constants,
    PackageJson,
    utils as pm_utils,
)
from devtools.frontend_build_platform.libraries.logging import timeit


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
    _extract_output_tar(moddir_abs)
    extract_peer_tars(moddir_abs, visited)


@timeit
def _extract_output_tar(moddir_abs: str):
    """Extracts the output tar for a module

    Args:
        moddir_abs: absolute path of the module
    """
    output_tar_path = os.path.join(moddir_abs, pm_constants.OUTPUT_TAR_FILENAME)
    if not os.path.exists(output_tar_path):
        return

    archive.extract_tar(output_tar_path, moddir_abs, fail_on_duplicates=False)


@timeit
def __copy_file_with_write_permissions(src, dst):
    os.makedirs(os.path.dirname(dst), exist_ok=True)

    shutil.copy(src, dst)

    dst_stat = os.stat(dst)
    dst_mode = stat.S_IMODE(dst_stat.st_mode)
    upd_mode = dst_mode | stat.S_IWUSR | stat.S_IWGRP
    if dst_mode != upd_mode:
        os.chmod(dst, upd_mode)


@timeit
def copy_if_not_exists(src: str, dst: str):
    """Copy file/directory skipping existing. Makes them writable."""
    if os.path.exists(dst):
        return

    if os.path.isdir(src):
        shutil.copytree(src, dst, ignore_dangling_symlinks=True, copy_function=__copy_file_with_write_permissions)

    if os.path.isfile(src):
        __copy_file_with_write_permissions(src, dst)


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
