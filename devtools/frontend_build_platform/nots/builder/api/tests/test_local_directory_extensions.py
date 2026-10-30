import json
import os

import pytest

from devtools.frontend_build_platform.nots.builder.api.package_manager import PackageManager
from devtools.frontend_build_platform.nots.builder.api.pnpm_workspace import PnpmWorkspace, pnpm_defaults


@pytest.mark.parametrize("extension_group", ["dependencies", "peerDependencies"])
def test_frozen_install_accepts_extensions_of_local_directory_dependencies(tmp_path, extension_group):
    project = tmp_path / "app"
    project.mkdir()
    (project / "package.json").write_text(
        json.dumps(
            {
                "name": "app",
                "version": "1.0.0",
                "dependencies": {"peer": "workspace:../peer", "host": "workspace:../host"},
            }
        )
    )
    for name in ["peer", "host"]:
        directory = tmp_path / name
        directory.mkdir()
        (directory / "package.json").write_text(json.dumps({"name": name, "version": "1.0.0"}))
    workspace = PnpmWorkspace(str(project / "pnpm-workspace.yaml"))
    workspace.runtime_settings = pnpm_defaults()
    host_specifier = "workspace:../host" if extension_group == "dependencies" else "1.0.0"
    workspace.settings = {"packageExtensions": {"peer@1.0.0": {extension_group: {"host": host_specifier}}}}
    workspace.write()
    store = tmp_path / "store"
    store.mkdir()
    package_manager = PackageManager(
        build_root=str(tmp_path),
        build_path=str(project),
        sources_path=str(project),
        nodejs_bin_path=os.path.join(os.environ["NOTS_TEST_NODEJS_ROOT"], "node"),
        pnpm_root=os.environ["NOTS_TEST_PNPM_ROOT"],
    )
    package_manager._exec_command(
        ["install", "--lockfile-only", "--offline", "--store-dir", str(store)], cwd=str(project)
    )
    package_manager._exec_command(
        ["install", "--frozen-lockfile", "--offline", "--store-dir", str(store)], cwd=str(project)
    )
    assert (project / "node_modules" / "peer" / "package.json").is_file()
