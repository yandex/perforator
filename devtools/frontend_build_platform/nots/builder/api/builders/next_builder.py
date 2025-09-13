import itertools
import os
import shutil
from dataclasses import dataclass

from devtools.frontend_build_platform.libraries.logging import timeit

from .base_builder import BaseTsBuilder
from ..models import CommonBundlersOptions


@dataclass
class NextBuilderOptions(CommonBundlersOptions):
    ts_next_command: str
    """Use specific build command"""


class NextBuilder(BaseTsBuilder):
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
        return self.options.ts_next_command.split(' ')

    def _output_macro(self):
        return "TS_NEXT_OUTPUT"

    def _config_filename(self):
        return self.options.bundler_config

    def _move_cache_out_of_bundle(self):
        """
        Move `cache` directory from the `.next` (output dir).

        `libray.archive` doesn't have options to ignore/exclude some directories/patterns.
        So, it's a workaround to ignore `.next/cache` directory bundling.
        """
        for output_dir_name, ignore_item in itertools.product(self.output_dirs, ['cache', 'trace']):
            ignore_src = os.path.join(self.options.bindir, output_dir_name, ignore_item)
            ignore_dst = os.path.join(self.options.bindir, f'{output_dir_name}.{ignore_item}')

            if os.path.exists(ignore_src):
                shutil.rmtree(ignore_dst, ignore_errors=True)
                shutil.move(ignore_src, ignore_dst)

    def bundle(self):
        self._move_cache_out_of_bundle()

        return super().bundle()
