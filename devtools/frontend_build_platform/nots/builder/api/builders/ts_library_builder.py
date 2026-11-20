import copy
import json
import os
import shutil
import tempfile
import textwrap
import time
from dataclasses import dataclass

import click
from build.plugins.lib.nots.package_manager import (
    PackageJson,
    constants as pm_constants,
    utils as pm_utils,
)
from devtools.frontend_build_platform.libraries.logging import get_logger, timeit

from .base_builder import BaseBuilder
from ..package_manager import validate_package_json
from ..models import BaseBuildersOptions, BuildError
from ..ram_disk import RamDisk, RamDiskUsage
from ..utils import copy_files_with_exclusions, get_built_workspace_dep_paths, popen

from ..create_node_modules import (
    bundle_workspace_node_modules,
    create_node_modules,
    NodeModulesBuildContext,
    restore_node_modules_layer,
)

logger = get_logger(__name__)


@dataclass
class BaseTsLibraryBuilderOptions(BaseBuildersOptions):
    outputs: list[str]
    """output directories for the bundler"""

    exclude_globs: list[str]
    """globs to exclude files when copy from CURDIR to BINDIR"""


@dataclass
class TsLibraryBuilderOptions(BaseTsLibraryBuilderOptions):
    build_script: str
    """name of a script from package.json#scripts"""

    build_command: str | None = None
    """inline command to execute as build_script instead of package.json#scripts"""


class TsLibraryBuilder(BaseBuilder):
    def build(self):
        if (
            getattr(self.options, "prepared_deps_moddir", None)
            and self.options.hermetic_node_modules
            and not os.path.isfile(self.options.node_modules_layer)
        ):
            # A TS_LIBRARY_FOR target can inherit dependencies from a parent
            # that opts out of hermetic node_modules.
            self.options.hermetic_node_modules = False
        node_modules_restored = self._restore_node_modules_to_ram_disk()
        self._prepare_bindir()

        if self.options.hermetic_node_modules:
            if not node_modules_restored:
                restore_node_modules_layer(self.options)
            node_modules_context = NodeModulesBuildContext(peer_paths=())
        else:
            node_modules_context = create_node_modules(self.options)

        self._build()

        bundle_workspace_node_modules(self.options, node_modules_context)

    def __init__(self, options: TsLibraryBuilderOptions):
        self._ram_disk_usage = RamDiskUsage.NONE
        self._ram_disk = None
        self._original_bindir = None
        self._original_output_file = None
        if options.hermetic_node_modules:
            self._ram_disk = RamDisk.from_env()

        super(TsLibraryBuilder, self).__init__(options)
        self.options = options  # for type hints

    def _restore_node_modules_to_ram_disk(self) -> bool:
        if not self.options.hermetic_node_modules or self._ram_disk is None:
            return False

        self._ram_disk_usage = RamDiskUsage.NODE_MODULES
        validate_package_json(PackageJson.load(pm_utils.build_pj_path(self.options.bindir)))
        free_before_restore = shutil.disk_usage(self._ram_disk.root).free
        options = copy.copy(self.options)
        options.use_ram_disk = True
        restore_node_modules_layer(options)
        free_after_restore = shutil.disk_usage(self._ram_disk.root).free
        node_modules_size = max(0, free_before_restore - free_after_restore)

        source_exclude_globs = options.exclude_globs + [f"{output}/**/*" for output in options.outputs]
        build_exclude_globs = [
            f"{pm_constants.NODE_MODULES_DIRNAME}/**/*",
            os.path.basename(options.output_file),
            *[f"{output}/**/*" for output in options.outputs],
        ]
        self._ram_disk_usage = self._ram_disk.select_usage(
            options.curdir,
            options.bindir,
            source_exclude_globs,
            build_exclude_globs,
            workspace_bundle_reserve=node_modules_size if options.nm_bundle else 0,
        )
        if self._ram_disk_usage == RamDiskUsage.FULL_BUILD:
            self._original_bindir = options.bindir
            self._original_output_file = options.output_file
            options.bindir = self._ram_disk.path(options.arcadia_build_root, options.bindir)
            options.output_file = self._ram_disk.path(options.arcadia_build_root, options.output_file)

        self.options = options
        return True

    @timeit
    def _get_pack_files(self) -> list[str]:
        if getattr(self.options, "prepared_deps_moddir", None):
            # An auxiliary build has its own outputs; the parent's publish files
            # describe its main build and must not constrain this archive.
            return self.options.outputs
        return super()._get_pack_files()

    @timeit
    def bundle(self):
        os.makedirs(os.path.dirname(os.path.abspath(self.options.output_file)), exist_ok=True)
        result = super().bundle()

        if self._ram_disk_usage == RamDiskUsage.FULL_BUILD:
            assert self._original_bindir is not None
            assert self._original_output_file is not None
            os.makedirs(os.path.dirname(self._original_output_file), exist_ok=True)
            shutil.copyfile(self.options.output_file, self._original_output_file)
            if self.options.nm_bundle:
                shutil.copyfile(
                    pm_utils.build_nm_bundle_path(self.options.bindir),
                    pm_utils.build_nm_bundle_path(self._original_bindir),
                )

        return result

    def cleanup(self):
        if self._ram_disk_usage != RamDiskUsage.NONE:
            assert self._ram_disk is not None
            self._ram_disk.cleanup()

    @timeit
    def _prepare_bindir(self):
        """Prepare bindir by extracting dependencies and copying source files"""
        if self._ram_disk_usage == RamDiskUsage.FULL_BUILD:
            assert self._original_bindir is not None
            assert self._original_output_file is not None
            exclude_globs = [
                f"{pm_constants.NODE_MODULES_DIRNAME}/**/*",
                os.path.basename(self._original_output_file),
                *[f"{output}/**/*" for output in self.options.outputs],
            ]
            copy_files_with_exclusions(self._original_bindir, self.options.bindir, exclude_globs)
            os.makedirs(self.options.bindir, exist_ok=True)
        else:
            super()._prepare_bindir()

        exclude_globs = self.options.exclude_globs + [
            pm_constants.PACKAGE_JSON_FILENAME,
            pm_constants.PNPM_LOCKFILE_FILENAME,
            *[f"{o}/**/*" for o in self.options.outputs],
        ]
        copy_files_with_exclusions(self.options.curdir, self.options.bindir, exclude_globs)
        if self._ram_disk_usage == RamDiskUsage.FULL_BUILD:
            self._copy_workspace_peer_package_jsons()

    def _copy_workspace_peer_package_jsons(self):
        bindir = getattr(self, "_original_bindir", None) or self.options.bindir
        package_json_path = pm_utils.build_pj_path(bindir)
        package_json = PackageJson.load(package_json_path)

        # pnpm validates the full local dependency graph even for production installs.
        # Follow paths without validating names: build manifests may use generated names.
        pending = [(bindir, package_json)]
        visited = {os.path.normpath(bindir)}
        while pending:
            module_dir, manifest = pending.pop()
            for peer_build_path in sorted(get_built_workspace_dep_paths(manifest, module_dir)):
                if peer_build_path in visited:
                    continue
                visited.add(peer_build_path)
                source_package_json = pm_utils.build_pj_path(peer_build_path)
                pending.append((peer_build_path, PackageJson.load(source_package_json)))
                assert self._ram_disk is not None
                ram_package_json = self._ram_disk.path(self.options.arcadia_build_root, source_package_json)
                os.makedirs(os.path.dirname(ram_package_json), exist_ok=True)
                shutil.copyfile(source_package_json, ram_package_json)

    @timeit
    def _run_build_script(self):
        """Execute node --run <build_script> in bindir"""
        package_json_path = None
        package_json_backup_path = None
        inline_package_json = None
        build_command = self.options.build_command
        auxiliary_inline = bool(build_command and getattr(self.options, "prepared_deps_moddir", None))
        if self.options.build_command and not auxiliary_inline:
            package_json_path = pm_utils.build_pj_path(self.options.bindir)
            with open(package_json_path, "rb") as package_json_file:
                package_json = json.load(package_json_file)
            package_json.setdefault("scripts", {})[self.options.build_script] = self.options.build_command
            inline_package_json = json.dumps(package_json).encode("utf-8")
        elif not auxiliary_inline:
            with open(pm_utils.build_pj_path(self.options.bindir), "rb") as package_json_file:
                package_json = json.load(package_json_file)
            build_command = package_json.get("scripts", {}).get(self.options.build_script)

        args = ['node', '--run', self.options.build_script]
        env = self._get_envs()
        if auxiliary_inline:
            # Keep the prepared manifest untouched while providing the same
            # script environment as node --run for the inline command.
            args = ["/bin/sh", "-c", build_command]
            env["NODE_RUN_SCRIPT_NAME"] = self.options.build_script
            env["NODE_RUN_PACKAGE_JSON_PATH"] = pm_utils.build_pj_path(self.options.bindir)
            ancestor = os.path.abspath(self.options.bindir)
            bin_paths = []
            while True:
                bin_dir = os.path.join(pm_utils.build_nm_path(ancestor), ".bin")
                if os.path.isdir(bin_dir):
                    bin_paths.append(bin_dir)
                parent = os.path.dirname(ancestor)
                if parent == ancestor:
                    break
                ancestor = parent
            env["PATH"] = os.pathsep.join([*bin_paths, env["PATH"]])

        try:
            if package_json_path is not None and inline_package_json is not None:
                backup_fd, backup_path = tempfile.mkstemp(prefix="package-json-", suffix=".backup")
                os.close(backup_fd)
                try:
                    shutil.copyfile(package_json_path, backup_path)
                except Exception:
                    os.unlink(backup_path)
                    raise
                package_json_backup_path = backup_path
                with open(package_json_path, "wb") as package_json_file:
                    package_json_file.write(inline_package_json)
            started_at = time.monotonic()
            try:
                return_code, stdout, stderr = popen(
                    args,
                    env=env,
                    cwd=self.options.bindir,
                    verbose=self.options.verbose,
                    command_comment=build_command,
                )
            finally:
                logger.debug(
                    "Package.json script %r completed in %.3f s",
                    self.options.build_script,
                    time.monotonic() - started_at,
                )
        finally:
            if package_json_path is not None and package_json_backup_path is not None:
                shutil.copyfile(package_json_backup_path, package_json_path)
                os.unlink(package_json_backup_path)

        if return_code != 0:
            raise BuildError(self.options.command, return_code, stdout, stderr)

    @timeit
    def _assert_output_dirs_exists(self):
        """Verify all output directories exist and are not empty"""
        for output_dir in self.options.outputs:
            output_path = os.path.join(self.options.bindir, output_dir)

            if not os.path.exists(output_path):
                output_dir_styled = click.style(output_dir, fg="green")
                missing = click.style("missing", fg="red", bold=True)
                build_outputs_macro = click.style("BUILD_OUTPUTS", fg="green", bold=True)
                message = f"""
                    We expected to get output directory '{output_dir_styled}' but it is {missing}.
                    Probably, the build script didn't create this directory.
                    Check the {build_outputs_macro} macro in ya.make to ensure it matches your build script output.
                """
                raise BuildError(self.options.command, 1, "", textwrap.dedent(message))

            if os.path.isdir(output_path) and not os.listdir(output_path):
                output_dir_styled = click.style(output_dir, fg="green")
                empty = click.style("empty", fg="red", bold=True)
                message = f"""
                    Output directory '{output_dir_styled}' exists but is {empty}.
                    The build script may have failed to generate output files.
                """
                raise BuildError(self.options.command, 1, "", textwrap.dedent(message))

    @timeit
    def _build(self):
        """Execute the build process"""
        self._run_build_script()
        self._assert_output_dirs_exists()
        self._make_bins_executable()
