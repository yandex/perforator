import typing
from dataclasses import dataclass

from library.python import color

from build.plugins.lib.nots.typescript.ts_errors import TsError
from .utils import simplify_colors


@dataclass
class BaseOptions:
    """
    This class and its descendants are used only for the type system
    """

    # Arcadia paths
    arcadia_root: str
    """source root ($ARCADIA_ROOT, $S)"""
    arcadia_build_root: str
    """build root ($ARCADIA_BUILD_ROOT, $B)"""
    moddir: str
    """module path ($MODDIR)"""

    # Essential
    nodejs_bin: str
    """path to nodejs bin"""
    pm_script: str
    """path to package manager script to run `install` command"""
    pm_type: str
    """type of package manager (pnpm or npm)"""
    yatool_prebuilder_path: str | None
    """optional path to `@yatool/prebuilder` script"""

    command: str
    """builder `command` argument, used only in log messages"""

    # Flags
    local_cli = False
    """Is run locally (from `nots`) or on the distbuild"""

    bundle = True
    """Bundle the result into a tar archive"""

    trace = False
    """storing execution time, build the Chrome Tools compatible trace file"""

    verbose = False
    """write to logs (stderr)"""

    # Calculated options
    node_modules_bundle: str
    """path to node_modules.tar bundle, calculated"""

    bindir: str
    """module build path ($BINDIR), calculated"""

    curdir: str
    """module sources path ($CURDIR), calculated"""

    # Methods
    def func(self, args: typing.Self):
        """execute action for the command"""
        pass


@dataclass
class CommonBuildersOptions(BaseOptions):
    output_file: str
    """Absolute path to `<module_name>.output.tar`, expecting to be after building"""

    tsconfigs: list[str]
    """list of the tsconfig files. For bundlers only the first record used."""

    vcs_info: str | None
    """
    path to json file with VCS details.
    See https://docs.yandex-team.ru/frontend-in-arcadia/references/macros#vcs-info-file
    """

    env: list[str]
    """Environment variables lint in VAR format"""


@dataclass
class CommonBundlersOptions(CommonBuildersOptions):
    output_dirs: list[str]
    """output directories for the bundler"""

    bundler_config_path: str
    """path to the bundler config (vite.config.ts, webpack.config.js, etc...)"""

    bundler_config: str
    """path relative to curdir (vite.config.ts, webpack.config.js, etc...)"""


class BuildError(TsError):
    def __init__(self, command: str, code: int, stdout: str, stderr: str):
        self.command = command
        self.code = code
        self.stdout = stdout
        self.stderr = stderr

        messages = [color.colored(f"{command} exited with code {code}", color='red')]
        if stdout:
            messages.append(simplify_colors(stdout))
        if stderr:
            messages.append(simplify_colors(stderr))

        super(BuildError, self).__init__("\n".join(messages))
