from dataclasses import dataclass

from .base_builder import BaseLegacyBuilder
from ..models import CommonBuildersOptions


@dataclass
class PackageBuilderOptions(CommonBuildersOptions):
    pass


class PackageBuilder(BaseLegacyBuilder):
    def _build(self):
        pass
