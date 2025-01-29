from dataclasses import dataclass

from devtools.frontend_build_platform.libraries.logging import timeit

from .base_builder import BaseBuilder
from ..models import CommonBundlersOptions


@dataclass
class ViteBuilderOptions(CommonBundlersOptions):
    pass


class ViteBuilder(BaseBuilder):
    options: ViteBuilderOptions

    @timeit
    def __init__(
        self,
        options: ViteBuilderOptions,
        ts_config_path: str,
    ):
        super(ViteBuilder, self).__init__(
            options=options,
            output_dirs=options.output_dirs,
            ts_config_path=ts_config_path,
        )

    @timeit
    def _get_script_path(self):
        return self.resolve_bin("vite")

    @timeit
    def _get_exec_args(self) -> list[str]:
        return ["build", "--config", self._config_filename()]

    def _output_macro(self):
        return "TS_VITE_OUTPUT"

    def _config_filename(self):
        return self.options.bundler_config
