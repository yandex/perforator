import json
import os
import stat
import sys
from abc import ABCMeta, abstractmethod
from six import add_metaclass

import click
from build.plugins.lib.nots.package_manager import (
    constants as pm_constants,
    PackageJson,
    utils as pm_utils,
)
from devtools.frontend_build_platform.libraries.logging import timeit
from ..models import BuildError, BaseBuildersOptions
from ..utils import (
    copy_writable_file,
    extract_peer_tars,
    popen,
    bundle_fs_entries,
)

npmignore_content = """__tarball__
.pnpm
pnpm-workspace.yaml
"""


@add_metaclass(ABCMeta)
class BaseBuilder(object):
    def __init__(self, options: BaseBuildersOptions):
        self.options = options

    def build(self):
        self._prepare_bindir()
        self._build()

    @timeit
    def bundle(self):
        """Create output archive from files listed by pnpm pack"""
        file_paths = self._get_pack_files()
        bundle_fs_entries(
            file_paths, self.options.bindir, self.options.output_file, getattr(self.options, "output_prefix", "")
        )

    @timeit
    def _get_pack_files(self) -> list[str]:
        """Run pnpm pack --json and return list of files to include in archive"""
        # Create or update .npmignore file
        npmignore_path = os.path.join(self.options.bindir, '.npmignore')

        with open(npmignore_path, 'w') as f:
            f.write(npmignore_content)

        args = [
            self.options.nodejs_bin,
            self.options.pm_script,
            'pack',
            '--json',
            '--dry-run',
            '--config.ignoreScripts=true',
        ]

        pj = PackageJson.load(pm_utils.build_pj_path(self.options.bindir))

        # todo: FBP-2979
        is_pj_need_update = False
        if 'name' not in pj.data:
            parts = self.options.moddir.split('/')
            pj.data['name'] = f'@{parts[0]}/{parts[-1]}'
            is_pj_need_update = True
        if 'version' not in pj.data:
            pj.data['version'] = '0.0.0'
            is_pj_need_update = True
        if 'files' not in pj.data:
            files = []
            if hasattr(self.options, 'output_dirs') and self.options.output_dirs is not None:
                files.extend(self.options.output_dirs)
            if hasattr(self.options, 'outputs') and self.options.outputs is not None:
                files.extend(self.options.outputs)

            pj.data['files'] = files
            is_pj_need_update = True

        if is_pj_need_update:
            pj.write()

        return_code, stdout, stderr = popen(args, env=self._get_envs(), cwd=self.options.bindir)

        if return_code != 0:
            raise BuildError(self.options.command, return_code, stdout, stderr)

        # Parse JSON output
        pack_data = json.loads(stdout)

        # Extract file paths from json['files'][]['path']
        files = [file_entry['path'] for file_entry in pack_data['files']]

        publish_config = pj.data.get('publishConfig', {})
        publish_directory = publish_config.get('directory')
        if publish_directory and files:
            files = [os.path.join(publish_directory, f) for f in files]

        files.append('.npmignore')

        return files

    def _prepare_bindir(self):
        self._prepare_dependencies()

    @timeit
    def __extract_peer_tars(self, *args, **kwargs):
        return extract_peer_tars(*args, **kwargs)

    @abstractmethod
    def _build(self): ...

    @timeit
    def _prepare_dependencies(self):
        package_json_path = pm_utils.build_pj_path(self.options.bindir)
        required_paths = [package_json_path, pm_utils.build_lockfile_path(self.options.bindir)]
        missing_paths = [path for path in required_paths if not os.path.exists(path)]
        if missing_paths:
            raise BuildError(
                self.options.command,
                1,
                "",
                "TS_PREPARE_DEPS did not materialize required files: {}".format(
                    ", ".join(os.path.basename(path) for path in missing_paths)
                ),
            )
        copy_writable_file(package_json_path, package_json_path)
        self.__extract_peer_tars(self.options.bindir, build_root=self.options.arcadia_build_root)

    def _get_base_env(self, extra_paths: list[str] = []) -> dict[str, str]:
        env = {}

        # MODDIR is persistent API for users. Do not change without project changes.
        # Other variables is not persistent and can not be exposed to users application
        # See contract documentation: https://docs.yandex-team.ru/ya-make/manual/common/vars
        env['MODDIR'] = self.options.moddir

        # Set directory with the `node` executable as the PATH
        env_paths = [os.path.dirname(self.options.nodejs_bin)] + extra_paths
        if self.options.bun_bin:
            env_paths.insert(0, os.path.dirname(self.options.bun_bin))
        env['PATH'] = os.pathsep.join(env_paths)

        bindir_node_modules_path = os.path.join(self.options.bindir, pm_constants.NODE_MODULES_DIRNAME)
        node_path = [
            os.path.join(
                pm_utils.build_vs_store_path(self.options.arcadia_build_root, self.options.moddir),
                pm_constants.NODE_MODULES_DIRNAME,
            ),
            # TODO: remove - no longer needed
            os.path.join(
                bindir_node_modules_path, pm_constants.VIRTUAL_STORE_DIRNAME, pm_constants.NODE_MODULES_DIRNAME
            ),
            os.path.join(self.options.bindir, pm_constants.VIRTUAL_STORE_DIRNAME, pm_constants.NODE_MODULES_DIRNAME),
            bindir_node_modules_path,
        ]

        env['NODE_PATH'] = os.pathsep.join(node_path)

        if self.options.ld_library_path:
            env['LD_LIBRARY_PATH'] = self.options.ld_library_path

        return env

    def _get_vcs_info_env(self, vcs_info_file: str) -> dict[str, str]:
        """convert vcs_info.json to environment variables (as dict)"""
        assert vcs_info_file

        vcs_info_path = os.path.join(self.options.bindir, vcs_info_file)
        with open(vcs_info_path) as f:
            data = json.load(f)

        def get_env_name(field: str) -> str:
            return f'VCS_INFO_{field.upper().replace("-", "_")}'

        return {get_env_name(k): str(v) for k, v in data.items()}

    def _get_user_defined_env(self) -> dict[str, str]:
        env = {}
        for pair in self.options.env:
            key, value = pair.split("=", 1)
            env[key] = value
        return env

    @timeit
    def _get_envs(self, extra_paths: list[str] = []) -> dict[str, str]:
        env = self._get_base_env(extra_paths)

        if self.options.vcs_info:
            env.update(self._get_vcs_info_env(self.options.vcs_info))

        if self.options.env:
            env.update(self._get_user_defined_env())

        return env

    @timeit
    def _make_bins_executable(self):
        pj = PackageJson.load(pm_utils.build_pj_path(self.options.bindir))
        for bin_tool in pj.bins_iter():
            bin_path = os.path.join(self.options.bindir, bin_tool)
            try:
                bin_stat = os.stat(bin_path)
                os.chmod(bin_path, bin_stat.st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)
            except FileNotFoundError:
                sys.stderr.write(f"Bin file does not exist: {click.style(bin_tool, fg='red')}\n")
