import os

from build.plugins.lib.nots.typescript.ts_config import TsConfig
from devtools.frontend_build_platform.libraries.logging import timeit

from .tsc_builder import TscBuilder, TscBuilderOptions


class TsProtoAutoTscBuilder(TscBuilder):
    options: TscBuilderOptions

    @timeit
    def __init__(
        self,
        options: TscBuilderOptions,
        ts_config: TsConfig,
    ):
        super(TsProtoAutoTscBuilder, self).__init__(options=options, ts_config=ts_config)
        self.ts_config_path = os.path.relpath(ts_config.path, options.bindir)

    def _create_bin_tsconfig(self):
        # for TS_PROTO_AUTO we already have our tsconfigs prepared in bindir
        pass
