TS_VITE()

SRCS(
    index.html
)

TS_TYPECHECK()

TS_ESLINT_CONFIG(.eslintrc.js)

TS_CONFIG(tsconfig.json)

TS_STYLELINT(.stylelintrc)

END()

RECURSE_FOR_TESTS(
  tests
)
