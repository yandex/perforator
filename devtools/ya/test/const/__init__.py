from build.plugins.lib import test_const as _test_const
from build.plugins.lib.test_const import *  # noqa: F401, F403

# Numeric suite metric: a positive value confirms failed preparation; 0 confirms completion.
TEST_PREPARATION_FAILED = 'test_preparation_failed'

# RESOURCES_LIBRARY targets: these add descriptors, not build-result nodes.
TEST_RESOURCE_TARGETS = (
    'build/platform/test_tool',
    'build/external_resources/go_tools',
    'build/external_resources/flake8_py2',
    'build/external_resources/flake8_py3',
    'build/platform/java/jstyle_lib',
)
TEST_RESOURCE_NAMES = (
    _test_const.TEST_TOOL_HOST,
    _test_const.TEST_TOOL_TARGET,
    _test_const.XCODE_TOOLS_RESOURCE,
    _test_const.GO_TOOLS_RESOURCE,
    _test_const.FLAKE8_PY2_RESOURCE,
    _test_const.FLAKE8_PY3_RESOURCE,
    _test_const.JSTYLE_RUNNER_LIB,
)
