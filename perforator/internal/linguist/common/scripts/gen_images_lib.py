import argparse
import os
import subprocess
import sys
from pathlib import Path
from typing import Callable, Dict


def build_docker_image(
    version: str,
    image_tag: str,
    arch: str,
    dockerfile_path: Path,
    build_args: Dict[str, str],
    runtime: str = "docker",
    language_name: str = "interpreter",
) -> bool:
    print(f"Building Docker image for {language_name} {version} ({arch}) using {runtime}...")

    if not dockerfile_path.exists():
        print(f"Error: Dockerfile not found at {dockerfile_path}")
        return False

    platform = None
    if arch == "x86_64":
        platform = "linux/amd64"
    elif arch == "arm64":
        platform = "linux/arm64"

    build_cmd = [
        runtime,
        "build",
        "-t",
        image_tag,
        "-f",
        str(dockerfile_path),
        "--network=host",
    ]
    for key, value in build_args.items():
        build_cmd += ["--build-arg", f"{key}={value}"]
    if platform:
        build_cmd += ["--platform", platform]
    build_cmd.append(str(dockerfile_path.parent))

    try:
        subprocess.run(build_cmd, check=True)
        print(f"Successfully built Docker image: {image_tag}")
        return True
    except subprocess.CalledProcessError as e:
        print(f"Error building Docker image: {e}")
        return False


def save_docker_image(image_tag: str, output_path: str, runtime: str = "docker") -> bool:
    print(f"Saving Docker image {image_tag} to {output_path} using {runtime}...")
    try:
        subprocess.run([runtime, "save", "-o", output_path, image_tag], check=True)
        print(f"Successfully saved Docker image to: {output_path}")
        return True
    except subprocess.CalledProcessError as e:
        print(f"Error saving Docker image: {e}")
        return False


def upload_to_registry(image_tag: str, registry_url: str, runtime: str = "docker") -> bool:
    print(f"Uploading Docker image to registry {registry_url}...")
    remote_image = f"{registry_url}/{image_tag}"
    print(f"Tagging image {image_tag} as {remote_image}...")
    try:
        subprocess.run([runtime, "tag", image_tag, remote_image], check=True)
    except subprocess.CalledProcessError as e:
        print(f"Error tagging image: {e}")
        return False
    print(f"Pushing image {remote_image}...")
    try:
        subprocess.run([runtime, "push", remote_image], check=True)
        print("Successfully pushed Docker image")
        return True
    except subprocess.CalledProcessError as e:
        print(f"Error pushing image: {e}")
        return False


def cleanup_docker_image(image_tag: str, runtime: str = "docker") -> None:
    print(f"Cleaning up local Docker image: {image_tag}")
    try:
        subprocess.run([runtime, "rmi", image_tag], check=True, capture_output=True)
        print("Successfully removed local Docker image")
    except subprocess.CalledProcessError as e:
        print(f"Warning: Could not remove local Docker image: {e}")


def run_main(
    language_name: str,
    image_prefix: str,
    version_arg_name: str,
    validate_version: Callable[[str], bool],
    get_build_args: Callable[[str], Dict[str, str]],
    description: str,
    dockerfile_path: Path,
) -> None:
    """
    Generic main() for interpreter Docker image builders.

    Args:
        language_name:    Human-readable name, e.g. "Python" or "PHP".
        image_prefix:     Docker image name prefix, e.g. "python" or "php".
        version_arg_name: Name of the positional CLI argument, e.g. "python_version".
        validate_version: Returns True if the version string is acceptable.
        get_build_args:   Returns Docker --build-arg dict for a given version string.
        description:      argparse description string.
        dockerfile_path:  Absolute path to the Dockerfile (pass Path(__file__).parent / "Dockerfile").
    """
    parser = argparse.ArgumentParser(description=description)
    parser.add_argument(version_arg_name, help=f"{language_name} version to build (e.g., 3.11.7)")
    parser.add_argument("--image-name", default=None, help=f"Docker image name (default: {image_prefix}-{{version}})")
    parser.add_argument("--output-dir", default="./", help="Directory to save Docker image tar file (default: ./)")
    parser.add_argument("--upload", action="store_true", help="Upload to Registry")
    parser.add_argument(
        "--registry-url", default=None, help="Registry URL to push the image to (required if --upload is set)"
    )
    parser.add_argument("--arch", default="x86_64", help="Target architecture for the Docker image (default: x86_64)")
    parser.add_argument("--podman", action="store_true", help="Use podman instead of docker")

    args = parser.parse_args()
    version = getattr(args, version_arg_name)
    runtime = "podman" if args.podman else "docker"

    if not validate_version(version):
        print(f"Error: Invalid {language_name} version format: {version}")
        sys.exit(1)

    image_tag = args.image_name or f"{image_prefix}-{version.replace('.', '-')}"
    output_path = os.path.join(args.output_dir, f"{image_tag}.tar")

    try:
        if not build_docker_image(
            version, image_tag, args.arch, dockerfile_path, get_build_args(version), runtime, language_name
        ):
            print("Failed to build Docker image")
            sys.exit(1)

        if args.upload:
            if not args.registry_url:
                print("Error: --registry-url is required when --upload is set")
                sys.exit(1)
            if not upload_to_registry(image_tag, args.registry_url, runtime):
                print("Failed to upload Docker image")
                sys.exit(1)
            print("Image has been uploaded to registry.")
        else:
            if not save_docker_image(image_tag, output_path, runtime):
                print("Failed to save Docker image")
                sys.exit(1)
            print(f"\nSuccess! Docker image for {language_name} {version} is ready.")
            print(f"Local tar file: {output_path}")

    except KeyboardInterrupt:
        print("\nOperation interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"Unexpected error: {e}")
        sys.exit(1)
    finally:
        cleanup_docker_image(image_tag, runtime)
