import os
from dataclasses import dataclass

from build.plugins.lib.nots.typescript.ts_config import TsConfig
from devtools.frontend_build_platform.libraries.logging import timeit

from .base_builder import BaseBuilder
from ..models import CommonBuildersOptions


@dataclass
class TscBuilderOptions(CommonBuildersOptions):
    pass


class TscBuilder(BaseBuilder):
    options: TscBuilderOptions

    @timeit
    def __init__(
        self,
        options: TscBuilderOptions,
        ts_config: TsConfig,
    ):
        super(TscBuilder, self).__init__(
            options=options,
            output_dirs=[ts_config.compiler_option("outDir")],
            ts_config_path=os.path.relpath(ts_config.path, options.curdir),
        )

    def bundle(self):
        """
        Should not bundle itself, see FBP-868
        """
        pass

    @timeit
    def _get_script_path(self) -> str:
        return self.resolve_bin("typescript", "tsc")

    @timeit
    def _get_exec_args(self) -> list[str]:
        return ["--project", self._config_filename(), "--incremental", "false", "--composite", "false", "--pretty"]

    @timeit
    def _get_copy_ignore_list(self) -> set[str]:
        ignore_list = super()._get_copy_ignore_list()

        ignore_list.update(set(self.options.tsconfigs))

        return ignore_list

    def _output_macro(self):
        return None

    def _config_filename(self):
        return self.ts_config_path

    def _run_javascript_after_build(self):
        pass

    def run_javascript_after_build(self):
        super()._run_javascript_after_build()
