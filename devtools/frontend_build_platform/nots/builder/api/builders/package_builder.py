from dataclasses import dataclass
import json
import os
import sys

from .ts_library_builder import BaseTsLibraryBuilderOptions, TsLibraryBuilder


@dataclass
class PackageBuilderOptions(BaseTsLibraryBuilderOptions):
    pass


class PackageBuilder(TsLibraryBuilder):
    def build(self):
        if self.options.nm_bundle:
            return super().build()

        self._prepare_bindir()
        self._build()

    def _get_pack_files(self):
        if self.options.nm_bundle:
            return super()._get_pack_files()

        package_json_path = os.path.join(self.options.bindir, "package.json")
        with open(package_json_path, "rb") as package_json_file:
            original_package_json = package_json_file.read()

        package_json = json.loads(original_package_json)
        # pnpm pack resolves workspace protocols through node_modules even in dry-run mode,
        # but dependency-free TS_PACKAGE builds only need the resulting file list.
        has_workspace_protocol = False
        for section in ("dependencies", "devDependencies", "peerDependencies", "optionalDependencies"):
            dependencies = package_json.get(section, {})
            for name, version in dependencies.items():
                if isinstance(version, str) and version.startswith("workspace:"):
                    dependencies[name] = "*"
                    has_workspace_protocol = True

        if not has_workspace_protocol:
            return super()._get_pack_files()

        with open(package_json_path, "w", encoding="utf-8") as package_json_file:
            json.dump(package_json, package_json_file)

        try:
            return super()._get_pack_files()
        finally:
            with open(package_json_path, "wb") as package_json_file:
                package_json_file.write(original_package_json)

    def _run_build_script(self):
        if self.options.verbose:
            sys.stderr.write("\nTS_PACKAGE does not have build script\n")
