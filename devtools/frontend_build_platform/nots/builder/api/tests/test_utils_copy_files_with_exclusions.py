import os
import stat
import tempfile

import pytest
from click.utils import strip_ansi

from devtools.frontend_build_platform.nots.builder.api.utils import (
    copy_files_with_exclusions,
    copy_writable_file,
    print_cmd_info,
)

TEST_FILE_STRUCTURE = {
    'src': {
        'common': {
            'index.ts': 'export const foo = 1;',
            'utils.ts': 'export const bar = 2;',
            'utils.test.ts': 'test("bar", () => {});',
        },
        'index.ts': 'export const foo = 1;',
        'index.test.ts': 'test("foo", () => {});',
        'utils.ts': 'export const bar = 2;',
    },
    'build': {
        'output.js': 'console.log("built");',
    },
    '.idea': {
        'workspace.xml': '<xml/>',
    },
    '.vscode': {
        'settings.json': '{}',
    },
    'node_modules': {
        'package.json': '{}',
    },
    'package.json': '{"name": "test"}',
    'README.md': '# Test',
}


def test_print_cmd_info_appends_command_comment(capsys):
    print_cmd_info(["/node", "--run", "nots:build"], "/build/module", {}, "next build\n--config config.js")

    assert "cd /build/module && /node --run nots:build # (next build --config config.js)" in strip_ansi(
        capsys.readouterr().err
    )


def test_copy_writable_file_does_not_hardlink_source(tmp_path):
    src = tmp_path / 'src-package.json'
    dst = tmp_path / 'dst-package.json'
    src.write_text('{"name": "original"}')
    src.chmod(0o444)
    os.link(src, dst)

    copy_writable_file(src, dst)

    assert dst.read_text() == src.read_text()
    assert dst.stat().st_ino != src.stat().st_ino
    assert dst.stat().st_mode & stat.S_IWUSR

    dst.write_text('{"name": "updated"}')
    assert src.read_text() == '{"name": "original"}'


def create_test_file_structure(base_dir, structure=None):
    """
    Create a test file structure from a dictionary.

    Args:
        base_dir: Base directory to create structure in
        structure: Dictionary where keys are paths and values are either:
                  - str: file content
                  - dict: nested directory structure
    """
    if structure is None:
        structure = TEST_FILE_STRUCTURE

    for name, content in structure.items():
        path = os.path.join(base_dir, name)

        if isinstance(content, dict):
            # It's a directory - create it and recurse
            os.makedirs(path, exist_ok=True)
            create_test_file_structure(path, content)
        else:
            # It's a file - create parent directory and write content
            os.makedirs(os.path.dirname(path), exist_ok=True)
            with open(path, 'w') as f:
                f.write(content)


@pytest.fixture
def mock_file_operations(monkeypatch):
    """
    Fixture that mocks file copy operations and tracks copied files.

    Returns:
        tuple: (src_dir, dst_dir, copied_files_list)
    """
    with tempfile.TemporaryDirectory() as tmpdir:
        src_dir = os.path.join(tmpdir, 'src')
        dst_dir = os.path.join(tmpdir, 'dst')
        os.makedirs(src_dir)
        os.makedirs(dst_dir)

        create_test_file_structure(src_dir)

        copied_files = []

        def mock_copy(src, dst):
            # Record relative path from dst_dir
            rel_path = os.path.relpath(dst, dst_dir)
            copied_files.append(rel_path)

        def mock_add_write_permissions(path):
            pass

        # Monkeypatch file operations.
        import devtools.frontend_build_platform.nots.builder.api.utils as utils_module
        import shutil as shutil_module

        monkeypatch.setattr(shutil_module, 'copy', mock_copy)
        monkeypatch.setattr(utils_module, 'add_write_permissions', mock_add_write_permissions)

        yield src_dir, dst_dir, copied_files


def test_copy_files_with_simple_exclusions(mock_file_operations):
    """Test that simple patterns like *.test.ts are excluded"""
    src_dir, dst_dir, copied_files = mock_file_operations

    # Act
    copy_files_with_exclusions(src_dir, dst_dir, ['*.test.ts', 'src/**/*.test.ts'])

    # Assert
    expected_files = [
        'src/common/index.ts',
        'src/common/utils.ts',
        'src/index.ts',
        'src/utils.ts',
        'build/output.js',
        '.idea/workspace.xml',
        '.vscode/settings.json',
        'node_modules/package.json',
        'package.json',
        'README.md',
    ]
    assert sorted(copied_files) == sorted(expected_files)


def test_copy_files_with_recursive_patterns(mock_file_operations):
    """Test that recursive patterns like build/**/* are excluded"""
    src_dir, dst_dir, copied_files = mock_file_operations

    # Act
    copy_files_with_exclusions(src_dir, dst_dir, ['build/**/*', 'node_modules/**/*'])

    # Assert
    expected_files = [
        'src/common/index.ts',
        'src/common/utils.ts',
        'src/common/utils.test.ts',
        'src/index.ts',
        'src/index.test.ts',
        'src/utils.ts',
        '.idea/workspace.xml',
        '.vscode/settings.json',
        'package.json',
        'README.md',
    ]
    assert sorted(copied_files) == sorted(expected_files)


def test_copy_files_with_alternation_patterns(mock_file_operations):
    """Test that alternation patterns like (.idea|.vscode)/**/* are excluded"""
    src_dir, dst_dir, copied_files = mock_file_operations

    # Act
    copy_files_with_exclusions(src_dir, dst_dir, ['(.idea|.vscode)/**/*'])

    # Assert
    expected_files = [
        'src/common/index.ts',
        'src/common/utils.ts',
        'src/common/utils.test.ts',
        'src/index.ts',
        'src/index.test.ts',
        'src/utils.ts',
        'build/output.js',
        'node_modules/package.json',
        'package.json',
        'README.md',
    ]
    assert sorted(copied_files) == sorted(expected_files)


def test_copy_files_includes_non_excluded(mock_file_operations):
    """Test that files not matching exclusion patterns are copied"""
    src_dir, dst_dir, copied_files = mock_file_operations

    # Act
    copy_files_with_exclusions(src_dir, dst_dir, ['**/*.test.ts', 'build/**/*', '(.idea|.vscode)/**/*'])

    # Assert
    expected_files = [
        'src/common/index.ts',
        'src/common/utils.ts',
        'src/index.ts',
        'src/utils.ts',
        'node_modules/package.json',
        'package.json',
        'README.md',
    ]
    assert sorted(copied_files) == sorted(expected_files)


def test_copy_files_with_empty_exclusions(mock_file_operations):
    """Test that all files are copied when no exclusions are specified"""
    src_dir, dst_dir, copied_files = mock_file_operations

    # Act
    copy_files_with_exclusions(src_dir, dst_dir, [])

    # Assert - all files should be copied
    expected_files = [
        'src/common/index.ts',
        'src/common/utils.ts',
        'src/common/utils.test.ts',
        'src/index.ts',
        'src/index.test.ts',
        'src/utils.ts',
        'build/output.js',
        '.idea/workspace.xml',
        '.vscode/settings.json',
        'node_modules/package.json',
        'package.json',
        'README.md',
    ]
    assert sorted(copied_files) == sorted(expected_files)
