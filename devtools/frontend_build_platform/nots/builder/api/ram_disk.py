import logging
import os
import shutil
from dataclasses import dataclass
from enum import Enum

import libarchive

from .globs import GlobMatcher

logger = logging.getLogger(__name__)

BUILD_OUTPUT_RESERVE_BYTES = 500 * 1024 * 1024


class RamDiskUsage(Enum):
    NONE = 'none'
    NODE_MODULES = 'node_modules'
    FULL_BUILD = 'full_build'


@dataclass(frozen=True)
class RamDisk:
    root: str

    @classmethod
    def from_env(cls):
        root = os.environ.get('DISTBUILD_RAM_DISK_PATH')
        return cls(root) if root else None

    def path(self, build_root: str, path: str) -> str:
        try:
            _path_inside_root(path, self.root)
            return path
        except ValueError:
            return os.path.join(self.root, _path_inside_root(path, build_root))

    def pnpm_store_path(self) -> str:
        return os.path.join(self.root, 'pnpm-ca-store')

    def select_usage(
        self,
        source_path: str,
        build_path: str,
        node_modules_layer: str,
        source_exclude_globs: list[str],
        build_exclude_globs: list[str],
        include_workspace_bundle: bool = False,
    ) -> RamDiskUsage:
        return select_ram_disk_usage(
            self.root,
            source_path,
            build_path,
            node_modules_layer,
            source_exclude_globs,
            build_exclude_globs,
            include_workspace_bundle,
        )

    def cleanup(self) -> None:
        if not os.path.isdir(self.root):
            return

        with os.scandir(self.root) as entries:
            for entry in entries:
                try:
                    if entry.is_symlink() or not entry.is_dir():
                        os.unlink(entry.path)
                    else:
                        shutil.rmtree(entry.path)
                except OSError:
                    logger.warning('Failed to clean RAM disk path %s', entry.path, exc_info=True)


def _path_inside_root(path: str, root: str) -> str:
    relative_path = os.path.relpath(path, root)
    if relative_path == os.pardir or relative_path.startswith(os.pardir + os.sep):
        raise ValueError('path must be inside the build root: {}'.format(path))
    return relative_path


def _allocated_size(size: int, block_size: int) -> int:
    return ((max(size, 1) + block_size - 1) // block_size) * block_size


def _tree_size(path: str, exclude_globs: list[str], block_size: int) -> int:
    matcher = GlobMatcher(exclude_globs)
    total = 0

    for root, dirs, files in os.walk(path):
        relative_root = os.path.relpath(root, path)
        if relative_root == '.':
            relative_root = ''

        dirs[:] = [
            directory for directory in dirs if not matcher.matches_whole_dir(os.path.join(relative_root, directory))
        ]

        for filename in files:
            source = os.path.join(root, filename)
            relative_path = os.path.join(relative_root, filename) if relative_root else filename
            if os.path.islink(source) or matcher.matches(relative_path):
                continue
            total += _allocated_size(os.path.getsize(source), block_size)

    return total


def _archive_size(path: str, block_size: int) -> int:
    with libarchive.Archive(path, mode='rb') as entries:
        return sum(_allocated_size(entry.size, block_size) for entry in entries)


def select_ram_disk_usage(
    ram_disk_path: str,
    source_path: str,
    build_path: str,
    node_modules_layer: str,
    source_exclude_globs: list[str],
    build_exclude_globs: list[str],
    include_workspace_bundle: bool = False,
) -> RamDiskUsage:
    try:
        block_size = os.statvfs(ram_disk_path).f_frsize
        source_size = _tree_size(source_path, source_exclude_globs, block_size)
        build_size = _tree_size(build_path, build_exclude_globs, block_size)
        node_modules_size = _archive_size(node_modules_layer, block_size)
        workspace_bundle_size = node_modules_size if include_workspace_bundle else 0
        full_build_required = (
            source_size + build_size + node_modules_size + workspace_bundle_size + BUILD_OUTPUT_RESERVE_BYTES
        )
        available = shutil.disk_usage(ram_disk_path).free
    except Exception:
        logger.exception('Failed to estimate RAM disk capacity; using build directory')
        return RamDiskUsage.NONE

    logger.info(
        'RAM disk capacity: full_build_required=%d, node_modules_required=%d, available=%d '
        '(source=%d, build_inputs=%d, node_modules=%d, workspace_bundle=%d, '
        'build_output_reserve=%d)',
        full_build_required,
        node_modules_size,
        available,
        source_size,
        build_size,
        node_modules_size,
        workspace_bundle_size,
        BUILD_OUTPUT_RESERVE_BYTES,
    )
    if full_build_required <= available:
        return RamDiskUsage.FULL_BUILD
    if node_modules_size <= available:
        return RamDiskUsage.NODE_MODULES
    return RamDiskUsage.NONE
