from dataclasses import dataclass

from devtools.frontend_build_platform.libraries.logging import timeit

from .base_builder import BaseBuilder
from ..models import CommonBundlersOptions


@dataclass
class NextBuilderOptions(CommonBundlersOptions):
    ts_next_command: str
    """Use specific build command"""


class NextBuilder(BaseBuilder):
    options: NextBuilderOptions

    @timeit
    def __init__(
        self,
        options: NextBuilderOptions,
        ts_config_path: str,
    ):
        super(NextBuilder, self).__init__(
            options=options, output_dirs=options.output_dirs, ts_config_path=ts_config_path
        )

    @timeit
    def _get_script_path(self) -> str:
        return self.resolve_bin("next")

    @timeit
    def _get_exec_args(self) -> list[str]:
        return [self.options.ts_next_command]

    def _output_macro(self):
        return "TS_NEXT_OUTPUT"

    def _config_filename(self):
        return self.options.bundler_config
