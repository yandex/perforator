import pytest

from devtools.frontend_build_platform.nots.builder.cli import recipes


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
