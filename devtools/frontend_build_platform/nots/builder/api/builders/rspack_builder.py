from dataclasses import dataclass

from devtools.frontend_build_platform.libraries.logging import timeit
from .base_builder import BaseTsBuilder
from ..models import CommonBundlersOptions


@dataclass
class RspackBuilderOptions(CommonBundlersOptions):
    pass


class RspackBuilder(BaseTsBuilder):
    options: RspackBuilderOptions

    @timeit
    def __init__(
        self,
        options: RspackBuilderOptions,
        ts_config_path: str,
    ):
        super(RspackBuilder, self).__init__(
            options=options,
            output_dirs=options.output_dirs,
            ts_config_path=ts_config_path,
        )

    @timeit
    def _get_script_path(self):
        return self.resolve_bin("@rspack/cli", "rspack")

    @timeit
    def _get_exec_args(self) -> list[str]:
        return ["--config", self._config_filename()]

    def _output_macro(self):
        return "TS_RSPACK_OUTPUT"

    def _config_filename(self):
        return self.options.bundler_config
