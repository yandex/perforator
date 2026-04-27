from dataclasses import dataclass
import sys

from .ts_library_builder import BaseTsLibraryBuilderOptions, TsLibraryBuilder


@dataclass
class PackageBuilderOptions(BaseTsLibraryBuilderOptions):
    pass


class PackageBuilder(TsLibraryBuilder):
    def _run_build_script(self):
        if self.options.verbose:
            sys.stderr.write("\nTS_PACKAGE does not have build script\n")
