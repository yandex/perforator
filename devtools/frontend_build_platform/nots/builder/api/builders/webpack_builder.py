from dataclasses import dataclass

from devtools.frontend_build_platform.libraries.logging import timeit
from .base_builder import BaseTsBuilder
from ..models import CommonBundlersOptions


@dataclass
class WebpackBuilderOptions(CommonBundlersOptions):
    pass


class WebpackBuilder(BaseTsBuilder):
    options: WebpackBuilderOptions

    @timeit
    def __init__(
        self,
        options: WebpackBuilderOptions,
        ts_config_path: str,
    ):
        super(WebpackBuilder, self).__init__(
            options=options,
            output_dirs=options.output_dirs,
            ts_config_path=ts_config_path,
        )

    @timeit
    def _get_script_path(self):
        return self.get_bin_path("webpack")

    @timeit
    def _get_exec_args(self) -> list[str]:
        return ["--config", self._config_filename(), "--color"]

    def _output_macro(self):
        return "TS_WEBPACK_OUTPUT"

    def _config_filename(self):
        return self.options.bundler_configs[0]
