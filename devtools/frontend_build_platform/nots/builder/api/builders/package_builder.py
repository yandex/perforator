from dataclasses import dataclass
import sys

from .ts_library_builder import BaseTsLibraryBuilderOptions, TsLibraryBuilder


@dataclass
class PackageBuilderOptions(BaseTsLibraryBuilderOptions):
    pass


class PackageBuilder(TsLibraryBuilder):
    def _run_build_script(self):
        sys.stderr.write("\nTS_PACKAGE does not have build script\n")
