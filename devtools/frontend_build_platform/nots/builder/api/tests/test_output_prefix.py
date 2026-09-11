import json
import os
import tarfile

import pytest

from devtools.frontend_build_platform.nots.builder.api.utils import (
    bundle_fs_entries,
    extract_output_tar,
    normalize_output_prefix,
)


@pytest.mark.parametrize("prefix", ["", "devtools/example", "code/service"])
def test_output_prefix_roundtrip(tmp_path, prefix):
    source = tmp_path / "source"
    source.mkdir()
    for name in ("dist", "types"):
        (source / name).mkdir()
        (source / name / "index.js").write_text(name)
    (source / "package.json").write_text('{"name":"peer"}')
    (source / "dist" / "link.js").symlink_to("index.js")
    os.link(source / "dist" / "index.js", source / "dist" / "hard.js")
    destination = tmp_path / "peer"
    destination.mkdir()
    bundle = destination / "output.tar"
    bundle_fs_entries(["dist", "types", "package.json"], str(source), str(bundle), prefix)
    with tarfile.open(bundle) as archive:
        assert f"{prefix}/dist/index.js".lstrip('/') in archive.getnames()
        assert f"{prefix}/types/index.js".lstrip('/') in archive.getnames()
    (destination / "output.tar.uuid").write_text(json.dumps({"outputTar": {"prefix": prefix}}))
    # Existing build inputs must not be overwritten by the peer archive.
    (destination / "package.json").write_text("existing")
    extract_output_tar(str(destination))
    assert (destination / "dist" / "index.js").read_text() == "dist"
    assert (destination / "types" / "index.js").read_text() == "types"
    assert (destination / "package.json").read_text() == "existing"
    assert os.readlink(destination / "dist" / "link.js") == "index.js"
    assert (destination / "dist" / "link.js").read_text() == "dist"
    assert (destination / "dist" / "hard.js").read_text() == "dist"


@pytest.mark.parametrize(
    "metadata",
    [None, "legacy uuid", '{"outputTar":{"sha256":"old"}}']
    + [json.dumps({"outputTar": value}) for value in (None, [], "invalid", 42)]
    + [json.dumps(value) for value in (None, [], "invalid", 42)],
)
def test_legacy_output(tmp_path, metadata):
    source = tmp_path / "source"
    source.mkdir()
    (source / "file").write_text("old archive")
    bundle_fs_entries(["file"], str(source), str(tmp_path / "output.tar"))
    if metadata is not None:
        (tmp_path / "output.tar.uuid").write_text(metadata)
    extract_output_tar(str(tmp_path))
    assert (tmp_path / "file").read_text() == "old archive"


@pytest.mark.parametrize("raw,expected", [("/code/service/", "code/service"), ("a//b", "a/b"), ("/", ""), ("", "")])
def test_normalize_prefix(raw, expected):
    assert normalize_output_prefix(raw) == expected


@pytest.mark.parametrize("prefix", ["../outside", "app/../outside", "app\\dir", "app\nfile"])
def test_invalid_prefix(prefix):
    with pytest.raises(ValueError):
        normalize_output_prefix(prefix)
