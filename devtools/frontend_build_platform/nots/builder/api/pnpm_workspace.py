import copy
import os

from build.plugins.lib.nots.package_manager.common_config import load_tier0_settings, select_pnpm_settings

try:
    import ymakeyaml as yaml
except Exception:
    import yaml


def pnpm_defaults():
    return load_tier0_settings()["pnpm"]["defaults"]


class PnpmWorkspace(object):
    @classmethod
    def load(cls, path):
        ws = cls(path)
        ws.read()

        return ws

    def __init__(self, path):
        if not os.path.isabs(path):
            raise TypeError("Absolute path required, given: {}".format(path))

        self.path = path
        # NOTE: pnpm requires relative workspace paths.
        self.packages = set()
        self.catalogs = {}
        self.settings = {}
        self.runtime_settings = {}

    def read(self):
        with open(self.path) as f:
            parsed = yaml.load(f, Loader=yaml.CSafeLoader) or {}
            self.packages = set(parsed.get("packages", []))
            self.catalogs = parsed.get("catalogs", {})
            self.settings = select_pnpm_settings(parsed)

    def write(self, path=None):
        if not path:
            path = self.path

        with open(path, "w") as f:
            data = copy.deepcopy(self.settings)
            data.update(self.runtime_settings)
            data["packages"] = sorted(self.packages)
            if self.catalogs:
                data["catalogs"] = self.catalogs
            yaml.dump(data, f, Dumper=yaml.CSafeDumper)

    def get_paths(self, base_path=None, ignore_self=False):
        """
        Returns absolute paths of the workspace packages.
        :param base_path: base path to resolve relative dep paths
        :type base_path: str
        :param ignore_self: whether path of the current module will be excluded (if present)
        :type ignore_self: bool
        :rtype: list of str
        """
        if base_path is None:
            base_path = os.path.dirname(self.path)

        return [
            os.path.normpath(os.path.join(base_path, pkg_path))
            for pkg_path in self.packages
            if not ignore_self or pkg_path != "."
        ]

    def set_from_package_json(self, package_json):
        """
        Sets packages to "workspace" deps from given package.json.
        :param package_json: package.json of workspace
        :type package_json: PackageJson
        """
        if os.path.dirname(package_json.path) != os.path.dirname(self.path):
            raise TypeError(
                "package.json should be in workspace directory {}, given: {}".format(
                    os.path.dirname(self.path), package_json.path
                )
            )

        self.packages = set()
        self.settings = select_pnpm_settings(package_json.data.get("pnpm"))

    def merge(self, ws):
        """Import peer catalogs without inheriting peer settings or workspace entries."""
        for group, entries in ws.catalogs.items():
            if group in self.catalogs and self.catalogs[group] != entries:
                raise ValueError("Conflicting catalog group: {}".format(group))
            self.catalogs[group] = copy.deepcopy(entries)
