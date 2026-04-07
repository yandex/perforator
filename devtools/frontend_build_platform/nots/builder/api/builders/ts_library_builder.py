import os
import textwrap
from dataclasses import dataclass

import click
from devtools.frontend_build_platform.libraries.logging import timeit

from .base_builder import BaseBuilder
from ..models import BaseBuildersOptions, BuildError
from ..utils import copy_files_with_exclusions, popen

from ..create_node_modules import create_node_modules


@dataclass
class TsLibraryBuilderOptions(BaseBuildersOptions):
    outputs: list[str]
    """output directories for the bundler"""

    build_script: str
    """name of a script from package.json#scripts"""

    exclude_globs: list[str]
    """globs to exclude files when copy from CURDIR to BINDIR"""


class TsLibraryBuilder(BaseBuilder):
    def build(self, args: TsLibraryBuilderOptions):
        self._prepare_bindir()

        create_node_modules(args)

        self._build()

    def __init__(self, options: TsLibraryBuilderOptions):
        super(TsLibraryBuilder, self).__init__(options)
        self.options = options  # for type hints

    @timeit
    def _prepare_bindir(self):
        """Prepare bindir by extracting dependencies and copying source files"""
        super()._prepare_bindir()
        exclude_globs = self.options.exclude_globs + [f"{o}/**/*" for o in self.options.outputs]
        copy_files_with_exclusions(self.options.curdir, self.options.bindir, exclude_globs)

    @timeit
    def _run_build_script(self):
        """Execute node --run <build_script> in bindir"""
        args = [self.options.nodejs_bin, '--run', self.options.build_script]
        env = self._get_envs()

        return_code, stdout, stderr = popen(args, env=env, cwd=self.options.bindir, verbose=self.options.verbose)

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
