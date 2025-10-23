TS_TSC()
TS_FILES_GLOB(lib/components/**/*.css)
RUN_JAVASCRIPT_AFTER_BUILD(scripts/copy-through.mjs)

END()

RECURSE_FOR_TESTS(
tests
)
