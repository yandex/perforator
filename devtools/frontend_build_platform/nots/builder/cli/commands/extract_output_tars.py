import os
import shutil

import yatest.common

from build.plugins.lib.nots.package_manager import utils as pm_utils
from devtools.frontend_build_platform.nots.builder.api import extract_all_output_tars


def run_extract_output_tars_recipe(argv: list[str]) -> None:
    moddir_build_path = yatest.common.build_path(argv[0])
    moddir_source_path = yatest.common.source_path(argv[0])

    if not os.path.exists(pm_utils.build_pj_path(moddir_build_path)):
        shutil.copyfile(
            pm_utils.build_pj_path(moddir_source_path),
            pm_utils.build_pj_path(moddir_build_path),
        )

    extract_all_output_tars(moddir_build_path)
