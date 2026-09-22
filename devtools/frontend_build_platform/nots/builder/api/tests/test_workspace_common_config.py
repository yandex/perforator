import pytest

from devtools.frontend_build_platform.nots.builder.api.pnpm_workspace import PnpmWorkspace


def test_workspace_roundtrip_and_transitive_merge(tmp_path):
    peer = PnpmWorkspace(str(tmp_path / "peer/pnpm-workspace.yaml"))
    peer.catalogs = {"project/common": {"colors": "1.4.0"}}
    peer.common_config_sources = {"project/common.yaml": ["project/common"]}
    peer.packages = {"."}
    parent = PnpmWorkspace(str(tmp_path / "pnpm-workspace.yaml"))
    parent.merge(peer)
    parent.merge(peer)
    parent.write()
    restored = PnpmWorkspace.load(parent.path)
    assert restored.catalogs == peer.catalogs
    assert restored.common_config_sources == peer.common_config_sources
    conflicting = PnpmWorkspace(str(tmp_path / "another/pnpm-workspace.yaml"))
    conflicting.common_config_sources = {"project/other.yaml": []}
    with pytest.raises(ValueError, match="Conflicting common configs"):
        restored.merge(conflicting)


def test_conflicting_groups_in_nested_directories(tmp_path):
    parent = PnpmWorkspace(str(tmp_path / "pnpm-workspace.yaml"))
    parent.common_config_sources = {"project/common.yaml": ["project/nested/common"]}
    peer = PnpmWorkspace(str(tmp_path / "peer/pnpm-workspace.yaml"))
    peer.common_config_sources = {"project/nested/common.yaml": ["project/nested/common"]}
    with pytest.raises(ValueError, match="Conflicting catalog groups"):
        parent.merge(peer)
