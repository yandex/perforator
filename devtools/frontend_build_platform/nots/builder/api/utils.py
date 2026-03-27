import os
import shutil
import stat
import subprocess
import sys

import click
import library.python.archive as archive
import libarchive

from build.plugins.lib.nots.package_manager import (
    constants as pm_constants,
    PackageJson,
    utils as pm_utils,
)
from devtools.frontend_build_platform.libraries.logging import timeit
from .globs import GlobMatcher


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

    pj_exists = os.path.exists(os.path.join(moddir_abs, pm_constants.PACKAGE_JSON_FILENAME))

    def pj_filter(e: libarchive.Entry):
        # extract package.json if it does not exist yet
        should_extract = e.pathname != pm_constants.PACKAGE_JSON_FILENAME or not pj_exists
        return should_extract

    archive.extract_tar(output_tar_path, moddir_abs, fail_on_duplicates=False, entry_filter=pj_filter)


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
def copy_files_with_exclusions(src_dir: str, dst_dir: str, exclude_globs: list[str]):
    """
    Copy files from src_dir to dst_dir, excluding files matching exclude_globs patterns.

    Args:
        src_dir: Source directory path
        dst_dir: Destination directory path
        exclude_globs: List of glob patterns to exclude (supports *, **, (a|b|c) alternation)
    """
    # Create glob matcher from exclusion patterns
    matcher = GlobMatcher(exclude_globs)

    # Walk through src_dir recursively
    for root, dirs, files in os.walk(src_dir):
        # Calculate relative path from src_dir
        rel_root = os.path.relpath(root, src_dir)
        if rel_root == '.':
            rel_root = ''

        # Filter directories to avoid walking into excluded ones
        dirs_to_remove = []
        for dir_name in dirs:
            rel_dir_path = os.path.join(rel_root, dir_name) if rel_root else dir_name
            # Check if directory would be excluded using matches_whole_dir
            if matcher.matches_whole_dir(rel_dir_path):
                dirs_to_remove.append(dir_name)

        # Remove excluded directories from dirs list to prevent os.walk from descending
        for dir_name in dirs_to_remove:
            dirs.remove(dir_name)

        # Create destination directory once for all files in current directory
        if files:
            dst_subdir = os.path.join(dst_dir, rel_root) if rel_root else dst_dir
            os.makedirs(dst_subdir, exist_ok=True)

        # Copy files that are not excluded
        for file_name in files:
            rel_file_path = os.path.join(rel_root, file_name) if rel_root else file_name

            # Check if file matches any exclusion pattern
            if matcher.matches(rel_file_path):
                continue

            # Copy file to dst_dir maintaining directory structure
            src_path = os.path.join(root, file_name)
            dst_path = os.path.join(dst_dir, rel_file_path)
            if not os.path.exists(dst_path):
                shutil.copy(src_path, dst_path)
                __add_write_permissions(dst_path)


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


def print_cmd_info(cmd: list[str], cwd: str, env: dict[str, str]):
    sys.stderr.write("\n")
    export = click.style("export", fg="green")
    for key, value in env.items():
        escaped_value = value.replace('"', '\\"').replace("$", "\\$")
        sys.stderr.write(f'{export} {key}="{escaped_value}"\n')

    sys.stderr.write(f"cd {click.style(cwd, fg='cyan')} && {click.style(' '.join(cmd), fg='magenta')}\n\n")


def print_cmd_output(stdout: str, stderr: str):
    if stdout:
        sys.stderr.write(f"{click.style('_exec stdout:', fg='green')}\n{stdout}\n")
    if stderr:
        sys.stderr.write(f"{click.style('_exec stderr:', fg='yellow')}\n{stderr}\n")


@timeit
def popen(args: list[str], env: dict[str, str], cwd: str, verbose: bool = False):
    if verbose:
        print_cmd_info(args, cwd, env)

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

    if verbose:
        print_cmd_output(stdout, stderr)

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


@timeit
def bundle_fs_entries(dirs_and_files: list[str], build_path: str, bundle_path: str):
    if not dirs_and_files:
        raise RuntimeError("Please define `output_dirs`")

    paths_to_pack = {}
    for dir_or_file in dirs_and_files:
        arcname = os.path.normpath(dir_or_file)
        path_to_pack = os.path.normpath(os.path.join(build_path, dir_or_file))
        paths_to_pack[path_to_pack] = arcname

    archive.tar(
        list(paths_to_pack.items()), bundle_path, compression_filter=None, compression_level=None, fixed_mtime=0
    )
