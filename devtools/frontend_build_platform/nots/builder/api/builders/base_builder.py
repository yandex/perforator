import json
import os
import shutil
import stat
import sys
import textwrap
from abc import ABCMeta, abstractmethod
from six import add_metaclass, iteritems

import click
import library.python.archive as archive
from build.plugins.lib.nots.package_manager import (
    constants as pm_constants,
    PackageJson,
    utils as pm_utils,
)
from build.plugins.lib.nots.typescript import TsConfig
from devtools.frontend_build_platform.libraries.logging import timeit
from ..models import BuildError, CommonBuildersOptions
from ..utils import copy_if_not_exists, extract_peer_tars, popen, resolve_bin


@add_metaclass(ABCMeta)
class BaseBuilder(object):
    @staticmethod
    @timeit
    def load_ts_config(ts_config_file: str, sources_path: str) -> TsConfig:
        ts_config_curdir = os.path.normpath(os.path.join(sources_path, ts_config_file))
        ts_config = TsConfig.load(ts_config_curdir)

        pj = PackageJson.load(pm_utils.build_pj_path(sources_path))
        ts_config.inline_extend(pj.get_dep_paths_by_names())

        return ts_config

    @staticmethod
    @timeit
    def bundle_dirs(output_dirs: list[str], build_path: str, bundle_path: str):
        if not output_dirs:
            raise RuntimeError("Please define `output_dirs`")

        paths_to_pack = []
        for output_dir in output_dirs:
            arcname = output_dir[2:] if output_dir.startswith("./") else output_dir
            paths_to_pack.append((os.path.join(build_path, output_dir), arcname))

        archive.tar(set(paths_to_pack), bundle_path, compression_filter=None, compression_level=None, fixed_mtime=0)

    @timeit
    def __init__(
        self,
        options: CommonBuildersOptions,
        # TODO consider using self.options.output_dir or removing CommonBundlersOptions.output_dir at all
        output_dirs: list[str],
        # TODO consider supporting multiple ts_config_path?
        ts_config_path: str,
        copy_package_json=True,
        external_dependencies=None,
    ):
        """
        :param output_dirs: output directory names
        :type output_dirs: str
        :param ts_config_path: path to tsconfig.json (in srcdir)
        :type ts_config_path: str
        :param copy_package_json: whether package.json should be copied to build path
        :type copy_package_json: bool
        :param external_dependencies: external dependencies which will be linked to node_modules/ (mapping name -> path)
        :type external_dependencies: dict
        """
        self.options = options
        self.output_dirs = output_dirs
        self.ts_config_path = ts_config_path
        self.copy_package_json = copy_package_json
        self.external_dependencies = external_dependencies

    @property
    def ts_config_binpath(self) -> str:
        """tsconfig.json in $BINDIR (with expanding 'extends')"""
        return os.path.join(self.options.bindir, self.ts_config_path)

    @timeit
    def resolve_bin(self, package_name: str, bin_name: str = None) -> str:
        """
        Looks for the specified `bin_name` (or default) for the package
        :param package_name: Name of the package in `node_modules` dir
        :param bin_name: Custom "bin", defined in `package.json:bin` object
        :return: Full path to the script (.js file)
        """
        return resolve_bin(self.options.bindir, package_name, bin_name)

    @timeit
    def build(self):
        self._copy_package_json()
        self._prepare_dependencies()
        self._copy_src_files_to_bindir()

        self._build()

    @timeit
    def _get_copy_ignore_list(self) -> set[str]:
        return {
            # IDE's
            ".idea",
            ".vscode",
            # Output dirs
            "dist",
            pm_constants.BUILD_DIRNAME,
            pm_constants.BUNDLE_DIRNAME,
            # Dependencies
            pm_constants.NODE_MODULES_DIRNAME,
            pm_constants.PNPM_LOCKFILE_FILENAME,
            # ya-make artefacts
            pm_constants.NODE_MODULES_WORKSPACE_BUNDLE_FILENAME,
            "output.tar",  # TODO FBP-1978
            pm_constants.OUTPUT_TAR_UUID_FILENAME,
            # Other
            ".traces",
            "a.yaml",
            self.ts_config_path,  # Will be generated inside the builder (merged all the `extends`)
        }.union(self.output_dirs)

    @timeit
    def _copy_src_files_to_bindir(self):
        ignore_list = self._get_copy_ignore_list()

        for entry in os.scandir(self.options.curdir):
            if entry.name in ignore_list:
                continue

            dst = os.path.normpath(os.path.join(self.options.bindir, entry.name))
            copy_if_not_exists(entry.path, dst)

    @timeit
    def _copy_package_json(self):
        if not self.copy_package_json:
            return

        shutil.copyfile(
            pm_utils.build_pj_path(self.options.curdir),
            pm_utils.build_pj_path(self.options.bindir),
        )

    @timeit
    def __extract_peer_tars(self, *args, **kwargs):
        return extract_peer_tars(*args, **kwargs)

    @timeit
    def _prepare_dependencies(self):
        self.__extract_peer_tars(self.options.bindir)

        if self.external_dependencies:
            self._link_external_dependencies()

    @timeit
    def _link_external_dependencies(self):
        nm_path = pm_utils.build_nm_path(self.options.bindir)
        try:
            os.makedirs(nm_path)
        except OSError:
            pass

        # Don't want to use `os.makedirs(exists_ok=True)` here (we don't want to skip all "file exists" errors).
        scope_paths = set()

        for name, src in iteritems(self.external_dependencies):
            dst = os.path.join(nm_path, name)
            scope_path = os.path.dirname(dst)
            if scope_path and scope_path not in scope_paths:
                os.mkdir(scope_path)
                scope_paths.add(scope_path)

            os.symlink(src, dst, target_is_directory=True)

    @abstractmethod
    def _output_macro(self) -> str | None:
        pass

    @abstractmethod
    def _config_filename(self) -> str:
        pass

    @timeit
    def _assert_output_dirs_exists(self):
        for output_dir in self.output_dirs:
            if os.path.exists(os.path.join(self.options.bindir, output_dir)):
                continue

            output_dir_styled = click.style(output_dir, fg="green")
            missing = click.style("missing", fg="red", bold=True)
            config_filename = click.style(self._config_filename(), fg="blue")
            message = f"""
                We expected to get output directory '{output_dir_styled}' but it is {missing}.
                Probably, you set another output directory in {config_filename}.
            """

            output_macro = self._output_macro()
            if output_macro:
                output_macro_styled = click.style(output_macro + "(output_dir)", fg="green", bold=True)
                message += f"            Add macro {output_macro_styled} to ya.make to configure your output directory."

            raise BuildError(self.options.command, 1, "", textwrap.dedent(message))

    @timeit
    def _load_ts_config(self):
        return self.load_ts_config(self.ts_config_path, self.options.curdir)

    @timeit
    def _create_bin_tsconfig(self):
        ts_config = self._load_ts_config()

        opts = ts_config.get_or_create_compiler_options()
        opts["skipLibCheck"] = True

        ts_config.write(self.ts_config_binpath, indent=2)

    @abstractmethod
    def _get_script_path(self) -> str:
        """
        Should return path to the build script (.js file)
        """
        pass

    @abstractmethod
    def _get_exec_args(self) -> list[str]:
        """
        Should return arguments for the build script
        """
        pass

    @timeit
    def _get_vcs_info_env(self, vcs_info_file: str) -> dict[str, str]:
        """convert vcs_info.json to environment variables (as dict)"""
        assert vcs_info_file

        vcs_info_path = os.path.join(self.options.bindir, vcs_info_file)
        with open(vcs_info_path) as f:
            data = json.load(f)

        def get_env_name(field: str) -> str:
            return f'VCS_INFO_{field.upper().replace("-", "_")}'

        return {get_env_name(k): str(v) for k, v in data.items()}

    @timeit
    def _get_envs(self):
        env = {}

        if self.options.vcs_info:
            env.update(self._get_vcs_info_env(self.options.vcs_info))

        # MODDIR is persistent API for users. Do not change without project changes.
        # Other variables is not persistent and can not be exposed to users application
        # See contract documentation: https://docs.yandex-team.ru/ya-make/manual/common/vars
        env['MODDIR'] = self.options.moddir

        # Set directory with the `node` executable as the PATH
        env['PATH'] = os.path.dirname(self.options.nodejs_bin)

        env['NODE_PATH'] = pm_utils.build_nm_path(self.options.bindir)

        for pair in self.options.env:
            key, value = pair.split("=", 1)
            env[key] = value

        return env

    @timeit
    def _exec(self):
        args = [self.options.nodejs_bin, self._get_script_path()] + self._get_exec_args()
        env = self._get_envs()

        if self.options.verbose:
            sys.stderr.write("\n")
            export = click.style("export", fg="green")
            for key, value in env.items():
                escaped_value = value.replace('"', '\\"').replace("$", "\\$")
                sys.stderr.write(f'{export} {key}="{escaped_value}"\n')

            sys.stderr.write(
                f"cd {click.style(self.options.bindir, fg='cyan')} && {click.style(' '.join(args), fg='magenta')}\n\n"
            )

        return_code, stdout, stderr = popen(args, env=env, cwd=self.options.bindir)

        if self.options.verbose:
            if stdout:
                sys.stderr.write(f"_exec stdout:\n{click.style(stdout, fg='green')}\n")
            if stderr:
                sys.stderr.write(f"_exec stderr:\n{click.style(stderr, fg='yellow')}\n")

        if return_code != 0:
            raise BuildError(self.options.command, return_code, stdout, stderr)

    @timeit
    def _make_bins_executable(self):
        pj = PackageJson.load(pm_utils.build_pj_path(self.options.curdir))
        for bin_tool in pj.bins_iter():
            bin_path = os.path.join(self.options.bindir, bin_tool)
            bin_stat = os.stat(bin_path)
            os.chmod(bin_path, bin_stat.st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)

    @timeit
    def bundle(self):
        return self.bundle_dirs(self.output_dirs, self.options.bindir, self.options.output_file)

    @timeit
    def _build(self):
        # Pre-operations
        self._create_bin_tsconfig()

        # Action (building)
        self._exec()

        # Post-operations
        self._assert_output_dirs_exists()
        self._make_bins_executable()
