import io
import os
import tarfile
import tempfile

import pytest
import yatest.common


@pytest.fixture
def nots_tar():
    binary = yatest.common.binary_path("devtools/frontend_build_platform/nots/builder/tar/nots-tar")

    def run(*args, input_data=b"", check_exit_code=True):
        with tempfile.TemporaryFile() as manifest:
            manifest.write(input_data)
            manifest.seek(0)
            return yatest.common.execute(
                [binary, *(os.fspath(arg) for arg in args)],
                stdin=manifest,
                timeout=30,
                check_exit_code=check_exit_code,
            )

    return run


def manifest(entries):
    return b"".join(os.fsencode(value) + b"\0" for entry in entries for value in entry)


@pytest.mark.parametrize("pack_command", ["pack-tar", "pack-layer"])
def test_archive_is_reproducible_and_round_trips(nots_tar, tmp_path, pack_command):
    files = {f"bin/file-{i:03}": str(i).encode() for i in range(140)}
    files["bin/file-000"] = b"#!/bin/sh\n"
    files["large"] = b"payload\n" * (140 * 1024)
    files["empty"] = b""

    layer = pack_command == "pack-layer"
    prefix = "." if layer else "prefix/node_modules"
    extra_name = "notes with space\nand newline"
    archives = []
    for run in range(2):
        workspace = tmp_path / f"run-{run}"
        root = workspace / "node_modules"
        (root / "bin").mkdir(parents=True)
        (root / "bin").chmod(0o750)
        for name in (files if run == 0 else reversed(files)):
            path = root / name
            path.write_bytes(files[name])
            os.utime(path, (100 + run, 200 + run))
        (root / "bin/file-000").chmod(0o751)
        os.link(root / "bin/file-000", root / "hard")
        (root / "sym").symlink_to("bin/file-000")
        extra = workspace / extra_name
        extra.write_bytes(b"separate manifest entry")
        os.utime(extra, (100 + run, 200 + run))

        archive = workspace / "archive"
        entries = [(root, prefix), (extra, extra_name if layer else f"prefix/{extra_name}")]
        nots_tar(pack_command, archive, input_data=manifest(entries if run == 0 else reversed(entries)))
        archives.append(archive)

    assert archives[0].read_bytes() == archives[1].read_bytes()

    destination = tmp_path / "destination"
    if layer:
        with archives[0].open("rb") as stream:
            assert stream.read(4) == b"\x28\xb5\x2f\xfd"
        nots_tar("unpack-layer", archives[0], destination)
        restored = destination
    else:
        with tarfile.open(archives[0], "r:") as archive:
            assert archive.extractfile("prefix/node_modules/bin/file-000").read() == files["bin/file-000"]
            assert all(entry.mtime == 0 for entry in archive.getmembers())
        destination.mkdir()
        (destination / extra_name).write_bytes(b"existing")
        nots_tar("unpack-tar", archives[0], destination, "prefix", "--keep-existing")
        restored = destination / "node_modules"

    for name, contents in files.items():
        path = restored / name
        assert path.read_bytes() == contents
        assert path.stat().st_mtime_ns == 0
    assert (restored / "bin").stat().st_mode & 0o777 == 0o750
    assert (restored / "bin/file-000").stat().st_mode & 0o777 == 0o751
    assert os.readlink(restored / "sym") == "bin/file-000"
    assert os.path.samefile(restored / "bin/file-000", restored / "hard")
    assert (destination / extra_name).read_bytes() == (b"separate manifest entry" if layer else b"existing")


@pytest.mark.parametrize("suffix", [b"\0", b"\0file"], ids=["missing-path", "missing-final-nul"])
def test_pack_requires_complete_nul_delimited_pairs(nots_tar, tmp_path, suffix):
    source = tmp_path / "file"
    source.write_bytes(b"payload")
    archive = tmp_path / "output.tar"

    result = nots_tar("pack-tar", archive, input_data=os.fsencode(source) + suffix, check_exit_code=False)

    assert result.exit_code != 0
    assert b"Incomplete archive input" in result.std_err
    assert not archive.exists()


def test_failed_pack_preserves_existing_archive(nots_tar, tmp_path):
    archive = tmp_path / "output.tar"
    archive.write_bytes(b"existing archive")

    result = nots_tar("pack-tar", archive, input_data=manifest([(tmp_path / "missing", "file")]), check_exit_code=False)

    assert result.exit_code != 0
    assert b"Cannot stat" in result.std_err
    assert archive.read_bytes() == b"existing archive"
    assert list(tmp_path.iterdir()) == [archive]


def test_unpack_tar_rejects_parent_traversal(nots_tar, tmp_path):
    archive = tmp_path / "archive.tar"
    with tarfile.open(archive, "w", format=tarfile.GNU_FORMAT) as stream:
        entry = tarfile.TarInfo("../outside")
        entry.size = 1
        stream.addfile(entry, io.BytesIO(b"x"))
    destination = tmp_path / "destination"
    destination.mkdir()

    result = nots_tar("unpack-tar", archive, destination, "", check_exit_code=False)

    assert result.exit_code != 0
    assert b"Parent traversal" in result.std_err
    assert not (tmp_path / "outside").exists()
