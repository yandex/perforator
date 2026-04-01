"""
Shared library for extracting struct field offsets from language runtimes.

Pipeline: download source → configure (arbitrary commands) → compile offsets.c → parse output → JSON.
Used by PHP and Python offset extraction scripts via PEERDIR.
"""

import json
import os
import shutil
import subprocess
import sys


BLUE = "\033[34m"
RED = "\033[31m"
GREEN = "\033[32m"
RESET = "\033[0m"


def log_info(msg):
    print(f"{BLUE}[INFO] {msg}{RESET}", file=sys.stderr)


def log_error(msg):
    print(f"{RED}[ERROR] {msg}{RESET}", file=sys.stderr)


def log_success(msg):
    print(f"{GREEN}[SUCCESS] {msg}{RESET}", file=sys.stderr)


def download_source(repo, tag, dest_dir, force=False):
    """Clone a git repo at a specific tag. Returns True on success."""
    if os.path.isdir(dest_dir) and not force:
        log_info(f"Using cached source: {dest_dir}")
        return True

    if os.path.isdir(dest_dir):
        shutil.rmtree(dest_dir)

    log_info(f"Cloning {repo} (tag: {tag})...")
    result = subprocess.run(
        ["git", "clone", "--depth", "1", "--branch", tag, repo, dest_dir],
        capture_output=True, text=True,
    )
    if result.returncode != 0:
        log_error(f"Clone failed: {result.stderr}")
        return False

    log_success(f"Downloaded to {dest_dir}")
    return True


def configure_source(source_dir, configure_commands):
    """Run a sequence of configure commands (each is a list of args)."""
    for cmd in configure_commands:
        log_info(f"Running: {' '.join(cmd)}")
        result = subprocess.run(cmd, cwd=source_dir, capture_output=True, text=True)
        if result.returncode != 0:
            log_error(f"Command failed: {result.stderr}")
            return False

    log_success("Configured")
    return True


def build_and_run(source_dir, include_dirs, offsets_c):
    """Compile offsets.c and run it. Returns {struct: {field: offset}} or None."""
    executable = offsets_c + ".bin"

    if os.path.exists(executable):
        os.remove(executable)

    abs_includes = [
        source_dir if d == "." else os.path.join(source_dir, d)
        for d in include_dirs
    ]
    compile_cmd = ["gcc", "-o", executable]
    for d in abs_includes:
        compile_cmd.append(f"-I{d}")
    compile_cmd.append(offsets_c)

    log_info(f"Compiling: {' '.join(compile_cmd)}")
    result = subprocess.run(compile_cmd, capture_output=True, text=True)
    if result.returncode != 0:
        log_error(f"Compilation failed:\n{result.stderr}")
        return None

    log_info("Running offsets extractor...")
    result = subprocess.run([executable], capture_output=True, text=True)

    if os.path.exists(executable):
        os.remove(executable)

    if result.returncode != 0:
        log_error(f"Execution failed:\n{result.stderr}")
        return None

    offsets = {}
    for line in result.stdout.strip().split("\n"):
        parts = line.strip().split()
        if len(parts) == 3:
            offsets.setdefault(parts[0], {})[parts[1]] = int(parts[2])

    return offsets


def run(repo, tag, offsets_c, include_dirs, configure_commands=None,
        cache_dir="~/.offset_sources", force=False,
        output_format="json"):
    """Full pipeline: download → configure → build → print offsets."""
    safe_tag = tag.replace("/", "_")
    source_dir = os.path.join(os.path.expanduser(cache_dir), safe_tag)
    os.makedirs(os.path.dirname(source_dir), exist_ok=True)

    if not download_source(repo, tag, source_dir, force):
        sys.exit(1)

    if not configure_source(source_dir, configure_commands or []):
        sys.exit(1)

    offsets = build_and_run(source_dir, include_dirs, os.path.abspath(offsets_c))
    if not offsets:
        sys.exit(1)

    if output_format == "json":
        print(json.dumps(offsets, indent=2))
    else:
        for struct, fields in offsets.items():
            for field_name, offset in fields.items():
                print(f"{struct}.{field_name}: {offset}")

    log_success("Done!")
