module.exports = {
    root: true,
    env: {browser: true, es2020: true},
    extends: [
        'eslint:recommended',
        'plugin:react-hooks/recommended',
        'plugin:@typescript-eslint/base'
    ],
    plugins: ["@typescript-eslint", 'react-refresh'],
    ignorePatterns: ['dist', '.eslintrc.cjs'],
    parser: '@typescript-eslint/parser',
    rules: {
        'react-refresh/only-export-components': [
            'error',
            {allowConstantExport: true},
        ],
        "no-unused-vars": "off",
        'eol-last': 'error',
        "@typescript-eslint/no-unused-vars": ["error"]
    },
    overrides: [
        {
            files: ['e2e/**/*.ts', 'playwright.config.ts', 'playwright.ts'],
            env: {node: true, browser: false},
        },
    ],
    root: true
}
