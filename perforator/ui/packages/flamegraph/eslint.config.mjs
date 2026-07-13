import baseConfig from '@gravity-ui/eslint-config';
import globals from 'globals';
import importHelpers from 'eslint-plugin-import-helpers';
import react from 'eslint-plugin-react';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import simpleImportSort from 'eslint-plugin-simple-import-sort';
import tseslint from '@typescript-eslint/eslint-plugin';


const IMPORT_GROUPS = [
    ['^react$'],
    ['^[a-z]'],
    ['^@'],
    ['^@gravity-ui'],
    ['^src'],
    ['^\\.\\./'],
    ['^\\./'],
    ['^\\u0000'],
    ['^.*\\.(css|scss)$'],
];

export default [
    {
        ignores: [
            'dist/**',
            'src/generated/**',
        ],
    },
    ...baseConfig,
    {
        languageOptions: {
            ecmaVersion: 'latest',
            sourceType: 'module',
            globals: {
                ...globals.browser,
                ...globals.node,
            },
        },
        plugins: {
            'import-helpers': importHelpers,
            react,
            'react-hooks': reactHooks,
            'react-refresh': reactRefresh,
            'simple-import-sort': simpleImportSort,
        },
        rules: {
            'camelcase': ['error', { 'ignoreImports': true }],
            'comma-dangle': ['error', 'always-multiline'],
            'comma-spacing': 'error',
            'consistent-return': 'error',
            'eol-last': 'error',
            'import/newline-after-import': ['error', { 'count': 2 }],
            'indent': ['error', 4],
            'keyword-spacing': 'error',
            'no-console': ['error', { allow: ['info', 'warn', 'error'] }],
            'no-implicit-coercion': 'error',
            'no-multi-spaces': ['error', { ignoreEOLComments: true }],
            'no-multiple-empty-lines': 'error',
            'no-trailing-spaces': 'error',
            'object-curly-spacing': ['error', 'always'],
            'quotes': ['error', 'single'],
            'react-hooks/exhaustive-deps': 'warn',
            'react-hooks/rules-of-hooks': 'error',
            'react-refresh/only-export-components': 'error',
            'react/jsx-curly-spacing': ['error', { 'when': 'never', 'children': true }],
            'semi': 'error',
            'simple-import-sort/imports': ['error', { 'groups': IMPORT_GROUPS }],
            'sort-imports': 'off',
            'space-before-blocks': 'error',
            'space-infix-ops': 'error',
        },
    },
    {
        files: ['**/*.{ts,tsx}'],
        plugins: {
            '@typescript-eslint': tseslint,
        },
        rules: {
            '@typescript-eslint/consistent-type-imports': 'error',
            '@typescript-eslint/no-explicit-any': 'off',
            '@typescript-eslint/no-non-null-assertion': 'off',
            '@typescript-eslint/no-shadow': 'error',
        },
    },
];
