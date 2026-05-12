# TS_TSC()
TS_LIBRARY()
# TS_FILES_GLOB(lib/components/**/*.css)
# RUN_JAVASCRIPT_AFTER_BUILD(scripts/copy-through.mjs)

USE_LEGACY_PNPM_VIRTUAL_STORE()
TS_BUILD_SCRIPT(nots:build)
TS_BUILD_OUTPUTS(dist)

END()

RECURSE_FOR_TESTS(
tests
)
