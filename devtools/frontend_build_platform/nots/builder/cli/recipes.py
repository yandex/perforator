from library.python.testing.recipe import declare_recipe

from devtools.frontend_build_platform.nots.builder.cli.commands.extract_node_modules import (
    run_extract_node_modules_recipe,
)
from devtools.frontend_build_platform.nots.builder.cli.commands.extract_output_tars import (
    run_extract_output_tars_recipe,
)
from devtools.frontend_build_platform.nots.builder.cli.commands.install_node_modules import (
    run_install_node_modules_recipe,
)

RECIPE_HANDLERS = {
    "extract-node-modules": run_extract_node_modules_recipe,
    "extract-output-tars": run_extract_output_tars_recipe,
    "install-node-modules": run_install_node_modules_recipe,
}


def is_recipe_invocation(argv: list[str]) -> bool:
    for marker in ("start", "stop"):
        if marker in argv:
            index = argv.index(marker)
            return index + 1 < len(argv) and argv[index + 1] in RECIPE_HANDLERS
    return False


def _start(argv: list[str]) -> None:
    command, command_args = argv[0], argv[1:]
    RECIPE_HANDLERS[command](command_args)


def _stop(argv: list[str]) -> None:
    return


def run_recipe() -> None:
    declare_recipe(_start, _stop)
