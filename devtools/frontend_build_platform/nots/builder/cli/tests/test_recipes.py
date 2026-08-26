import pytest

from devtools.frontend_build_platform.nots.builder.cli import recipes
from devtools.frontend_build_platform.nots.builder.cli.commands import extract_node_modules


@pytest.mark.parametrize(
    "argv",
    [
        ["start", "install-node-modules"],
        ["--build-root", "/build", "start", "extract-node-modules", "project", "node_modules.layer"],
        ["stop", "extract-output-tars"],
    ],
)
def test_is_recipe_invocation(argv):
    assert recipes.is_recipe_invocation(argv)


@pytest.mark.parametrize(
    "argv",
    [
        [],
        ["--arcadia-root", "/arcadia", "build-library"],
        ["start", "unknown-command"],
        ["start"],
    ],
)
def test_is_not_recipe_invocation(argv):
    assert not recipes.is_recipe_invocation(argv)


@pytest.mark.parametrize("command", sorted(recipes.RECIPE_HANDLERS))
def test_recipe_start_dispatches_command(monkeypatch, command):
    called = []
    monkeypatch.setitem(recipes.RECIPE_HANDLERS, command, lambda argv: called.append((command, argv)))

    recipes._start([command, "argument"])

    assert called == [(command, ["argument"])]


def test_extract_node_modules_recipe_prepares_package_json(monkeypatch):
    prepared = []
    monkeypatch.setattr(extract_node_modules.yatest.common, "build_path", lambda path="": "/build/" + path)
    monkeypatch.setattr(extract_node_modules.yatest.common, "source_path", lambda: "/source")
    monkeypatch.setattr(extract_node_modules.yatest.common, "global_resources", lambda: {})
    monkeypatch.setattr(extract_node_modules, "resolve_recipe_arg", lambda arg, *args: arg)
    monkeypatch.setattr(extract_node_modules, "prepare_package_json", prepared.append)
    monkeypatch.setattr(extract_node_modules.os.path, "exists", lambda path: False)

    extract_node_modules.run_extract_node_modules_recipe(["project", "node_modules.layer"])

    assert prepared == ["/build/project"]
