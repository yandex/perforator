# TS_TSC()
TS_LIBRARY()
TS_LINT(lint)
# TS_FILES_GLOB(lib/components/**/*.css)
# RUN_JAVASCRIPT_AFTER_BUILD(scripts/copy-through.mjs)

TS_BUILD_SCRIPT(nots:build)
TS_BUILD_OUTPUTS(dist)

END()

RECURSE_FOR_TESTS(
tests
)
