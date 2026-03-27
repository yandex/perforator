from dataclasses import dataclass

from devtools.frontend_build_platform.libraries.logging import timeit

from .base_builder import BaseLegacyBuilder
from ..models import CommonBuildersOptions
from ..utils import bundle_fs_entries


@dataclass
class PackageBuilderOptions(CommonBuildersOptions):
    pass


class PackageBuilder(BaseLegacyBuilder):
    @timeit
    def bundle(self):
        """Create output archive from files listed by pnpm pack"""
        file_paths = self._get_pack_files()
        # if self.options.with_after_build and self.options.after_build_outdir:
        #     file_paths.append(self.options.after_build_outdir)
        bundle_fs_entries(file_paths, self.options.bindir, self.options.output_file)

    def _build(self):
        pass
