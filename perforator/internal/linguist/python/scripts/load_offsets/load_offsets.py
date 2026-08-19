#!/usr/bin/env python3
"""
Script to extract offsets for multiple Python versions and save them to JSON files.

This script:
1. Gets a list of CPython versions between a start and end version
2. For each version, runs extract_offsets.py to get the offset data
3. Saves the data to a JSON file in the specified output directory
"""

import argparse
import json
import logging
import os
import re
import subprocess
import sys
from typing import List, Optional, Tuple

# Set up logging
logging.basicConfig(
    level=logging.INFO,
    format='[%(levelname)s] %(message)s',
    stream=sys.stderr
)


def parse_arguments():
    parser = argparse.ArgumentParser(description="Extract offsets for multiple Python versions")
    parser.add_argument(
        "--start-version",
        type=str,
        required=True,
        help="Start version (e.g., 3.11.0)"
    )
    parser.add_argument(
        "--end-version",
        type=str,
        help="End version (e.g., 3.13.0). If not specified, extracts up to the latest version."
    )
    parser.add_argument(
        "--output-dir",
        type=str,
        default=None,
        help="Directory to save the JSON files (default: perforator/internal/linguist/python/offsets/offsets relative to arc root)"
    )
    parser.add_argument(
        "--cpython-repo",
        type=str,
        default="https://github.com/python/cpython.git",
        help="URL of the CPython git repository"
    )
    return parser.parse_args()


def parse_version(version_str: str) -> Tuple[int, int, int]:
    """Parse a version string into a tuple of (major, minor, micro)."""
    # Check for X.Y.Z format
    match = re.match(r"(\d+)\.(\d+)\.(\d+)", version_str)
    if match:
        return tuple(map(int, match.groups()))

    # Check for X.Y format
    match = re.match(r"(\d+)\.(\d+)", version_str)
    if match:
        major, minor = map(int, match.groups())
        return (major, minor, 0)

    raise ValueError(f"Invalid version format: {version_str}. Expected format: X.Y.Z or X.Y")


def version_to_str(version: Tuple[int, int, int]) -> str:
    """Convert a version tuple to a string."""
    return ".".join(map(str, version))


def get_python_versions(start_version: str, end_version: Optional[str], repo_url: str) -> List[str]:
    """Get a list of Python versions between start_version and end_version."""
    start = parse_version(start_version)
    end = parse_version(end_version) if end_version else None

    # Get all tags from the CPython repository
    logging.info("Fetching tags from CPython repository...")
    try:
        # Use git ls-remote to avoid cloning the entire repository
        cmd = ["git", "ls-remote", "--tags", repo_url]
        output = subprocess.check_output(cmd, universal_newlines=True)
    except subprocess.CalledProcessError as e:
        logging.error(f"Failed to fetch tags: {e}")
        sys.exit(1)

    # Parse the tags to find Python version tags
    version_pattern = re.compile(r"refs/tags/v(\d+\.\d+(?:\.\d+)?)$")
    versions = []

    for line in output.splitlines():
        match = version_pattern.search(line)
        if match:
            version_str = match.group(1)
            try:
                version = parse_version(version_str)

                # Check if the version is within the specified range
                if version >= start and (end is None or version <= end):
                    versions.append(version_str)
            except ValueError:
                # Skip invalid version formats
                logging.warning(f"Skipping invalid version format: {version_str}")
                continue

    # Sort versions
    versions.sort(key=parse_version)

    if not versions:
        logging.error(f"No Python versions found between {start_version} and {end_version or 'latest'}")
        sys.exit(1)

    logging.info(f"Found {len(versions)} Python versions to process")
    logging.info(f"Versions: {', '.join(versions)}")
    return versions


def extract_offsets(version: str, extract_offsets_binary_path: str, offsets_c_path: str) -> dict:
    """Run extract_offsets.py for the given Python version and return the results as a dictionary."""
    logging.info(f"Extracting offsets for Python {version}...")

    try:
        cmd = [
            extract_offsets_binary_path,
            "--cpython-version", version,
            "--offsets-c", offsets_c_path,
            "--output-format", "json"
        ]
        output = subprocess.check_output(cmd, universal_newlines=True)
        return json.loads(output)
    except subprocess.CalledProcessError as e:
        logging.error(f"Failed to extract offsets for Python {version}: {e}")
        logging.error(f"Command output: {e.output if hasattr(e, 'output') else 'No output'}")
        sys.exit(1)
    except json.JSONDecodeError as e:
        logging.error(f"Failed to parse JSON output for Python {version}: {e}")
        sys.exit(1)


def save_offsets(version: str, offsets: dict, output_dir: str):
    """Save the offsets to a JSON file."""
    # Ensure the output directory exists
    os.makedirs(output_dir, exist_ok=True)

    # Create the output filename
    filename = f"cpython-{version}-offsets.json"
    filepath = os.path.join(output_dir, filename)

    logging.info(f"Saving offsets to {filepath}...")

    try:
        with open(filepath, "w") as f:
            json.dump(offsets, f, indent=2)
        logging.info(f"Saved offsets for Python {version}")
    except IOError as e:
        logging.error(f"Failed to save offsets for Python {version}: {e}")
        sys.exit(1)


def main():
    args = parse_arguments()

    try:
        arc_root = subprocess.check_output(["arc", "root"], universal_newlines=True).strip()
    except subprocess.CalledProcessError as e:
        logging.error(f"Failed to get arc root: {e}")
        sys.exit(1)

    extract_offsets_binary_path = os.path.join(
        arc_root,
        "perforator", "internal", "linguist", "python", "scripts", "extract_offsets", "extract_offsets"
    )
    offsets_c_path = os.path.join(
        arc_root,
        "perforator", "internal", "linguist", "python", "scripts", "extract_offsets", "offsets.c"
    )

    output_dir = args.output_dir
    if output_dir is None:
        output_dir = os.path.join(arc_root, "perforator", "internal", "linguist", "python", "offsets", "offsets")

    # Get the list of Python versions to process
    versions = get_python_versions(args.start_version, args.end_version, args.cpython_repo)

    # Process each version
    for i, version in enumerate(versions):
        logging.info(f"Processing version {version} ({i+1}/{len(versions)})")

        # Extract the offsets
        offsets = extract_offsets(version, extract_offsets_binary_path, offsets_c_path)

        # Save the offsets to a JSON file
        save_offsets(version, offsets, output_dir)

    logging.info(f"Successfully processed {len(versions)} Python versions")


if __name__ == "__main__":
    main()
