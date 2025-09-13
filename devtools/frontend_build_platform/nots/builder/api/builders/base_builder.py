import json
import os
import stat
import sys
import textwrap
from abc import ABCMeta, abstractmethod
from six import add_metaclass

import click
import library.python.archive as archive
from build.plugins.lib.nots.package_manager import (
    constants as pm_constants,
    PackageJson,
    utils as pm_utils,
)
from build.plugins.lib.nots.typescript import TsConfig
from devtools.frontend_build_platform.libraries.logging import timeit
from ..models import BuildError, CommonBuildersOptions, CommonTsBuildersOptions
from ..utils import recursive_copy, extract_peer_tars, popen, resolve_bin


@add_metaclass(ABCMeta)
class BaseBuilder(object):
    @staticmethod
    @timeit
    def bundle_dirs(output_dirs: list[str], build_path: str, bundle_path: str):
        if not output_dirs:
            raise RuntimeError("Please define `output_dirs`")

        paths_to_pack = {}
        for output_dir in output_dirs:
            arcname = os.path.normpath(output_dir)
            path_to_pack = os.path.normpath(os.path.join(build_path, output_dir))
            paths_to_pack[path_to_pack] = arcname

        archive.tar(
            list(paths_to_pack.items()), bundle_path, compression_filter=None, compression_level=None, fixed_mtime=0
        )

    def __init__(self, options: CommonBuildersOptions, copy_package_json=True):
        self.options = options
        self.copy_package_json = copy_package_json

    @timeit
    def build(self):
        self._prepare_bindir()
        self._build()
        self._run_javascript_after_build()

    @abstractmethod
    def _build(self):
        pass

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
            # ya-make artifacts
            pm_constants.NODE_MODULES_WORKSPACE_BUNDLE_FILENAME,
            "output.tar",  # TODO FBP-1978
            pm_constants.OUTPUT_TAR_UUID_FILENAME,
            # Other
            "a.yaml",
            self.options.after_build_outdir,
        }

    @timeit
    def _copy_src_files_to_bindir(self):
        ignore_list = self._get_copy_ignore_list()
        for entry in os.scandir(self.options.curdir):
            if entry.name in ignore_list:
                continue

            dst = os.path.normpath(os.path.join(self.options.bindir, entry.name))
            recursive_copy(entry.path, dst)

    @timeit
    def _copy_package_json(self):
        recursive_copy(
            pm_utils.build_pj_path(self.options.curdir),
            pm_utils.build_pj_path(self.options.bindir),
        )

    @timeit
    def _exec_nodejs_script(self, script_path: str, script_args: list[str], env: dict):
        args = [self.options.nodejs_bin, script_path] + script_args

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
    def _get_envs(self):
        env = {}

        # MODDIR is persistent API for users. Do not change without project changes.
        # Other variables is not persistent and can not be exposed to users application
        # See contract documentation: https://docs.yandex-team.ru/ya-make/manual/common/vars
        env['MODDIR'] = self.options.moddir

        # Set directory with the `node` executable as the PATH
        env['PATH'] = os.path.dirname(self.options.nodejs_bin)

        env['NODE_PATH'] = pm_utils.build_nm_path(self.options.bindir)

        return env

    @timeit
    def _run_javascript_after_build(self):
        if not self.options.with_after_build:
            return

        self._exec_nodejs_script(
            script_path=self.options.after_build_js,
            script_args=self.options.after_build_args.split("<~~~>"),
            env=self._get_envs(),
        )

    @timeit
    def _prepare_bindir(self):
        if self.copy_package_json:
            self._copy_package_json()
        self._prepare_dependencies()
        self._copy_src_files_to_bindir()

    @timeit
    def _prepare_dependencies(self):
        self.__extract_peer_tars(self.options.bindir)

    @timeit
    def __extract_peer_tars(self, *args, **kwargs):
        return extract_peer_tars(*args, **kwargs)


@add_metaclass(ABCMeta)
class BaseTsBuilder(BaseBuilder):
    @staticmethod
    @timeit
    def load_ts_config(ts_config_file: str, sources_path: str) -> TsConfig:
        ts_config_curdir = os.path.normpath(os.path.join(sources_path, ts_config_file))
        ts_config = TsConfig.load(ts_config_curdir, sources_path)

        pj = PackageJson.load(pm_utils.build_pj_path(sources_path))
        ts_config.inline_extend(pj.get_dep_paths_by_names())

        return ts_config

    @timeit
    def __init__(
        self,
        options: CommonTsBuildersOptions,
        # TODO consider using self.options.output_dir or removing CommonBundlersOptions.output_dir at all
        output_dirs: list[str],
        # TODO consider supporting multiple ts_config_path?
        ts_config_path: str,
        copy_package_json=True,
    ):
        """
        :param output_dirs: output directory names
        :type output_dirs: str
        :param ts_config_path: path to tsconfig.json (in srcdir)
        :type ts_config_path: str
        :param copy_package_json: whether package.json should be copied to build path
        :type copy_package_json: bool
        """
        super(BaseTsBuilder, self).__init__(options, copy_package_json)
        self.options = options  # this is for type hints to understand real options' type
        self.output_dirs = output_dirs
        self.ts_config_path = ts_config_path

    def _get_copy_ignore_list(self) -> set[str]:
        ignored = super(BaseTsBuilder, self)._get_copy_ignore_list()
        return ignored.union(self.output_dirs + [self.ts_config_path])

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
    def _prepare_bindir(self):
        super(BaseTsBuilder, self)._prepare_bindir()
        self._create_bin_tsconfig()

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
        env = super(BaseTsBuilder, self)._get_envs()

        if self.options.vcs_info:
            env.update(self._get_vcs_info_env(self.options.vcs_info))

        for pair in self.options.env:
            key, value = pair.split("=", 1)
            env[key] = value

        return env

    @timeit
    def _make_bins_executable(self):
        pj = PackageJson.load(pm_utils.build_pj_path(self.options.bindir))
        for bin_tool in pj.bins_iter():
            bin_path = os.path.join(self.options.bindir, bin_tool)
            bin_stat = os.stat(bin_path)
            os.chmod(bin_path, bin_stat.st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)

    @timeit
    def bundle(self):
        output_dirs = self.output_dirs

        if self.options.with_after_build and self.options.after_build_outdir:
            output_dirs.append(self.options.after_build_outdir)

        return self.bundle_dirs(output_dirs, self.options.bindir, self.options.output_file)

    @timeit
    def _build(self):
        # Action (building)
        self._exec_nodejs_script(
            script_path=self._get_script_path(),
            script_args=self._get_exec_args(),
            env=self._get_envs(),
        )

        # Post-operations
        self._assert_output_dirs_exists()
        self._make_bins_executable()
