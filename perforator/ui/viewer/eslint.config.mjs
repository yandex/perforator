import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import tseslint from '@typescript-eslint/eslint-plugin';
import tsParser from '@typescript-eslint/parser';


const disabledBrowserGlobals = Object.fromEntries(
    Object.keys(globals.browser).map((name) => [name, 'off']),
);

export default [
    {
        ignores: [
            'dist/**',
            'vite.config.ts',
        ],
    },
    js.configs.recommended,
    {
        files: ['**/*.{ts,tsx}'],
        languageOptions: {
            parser: tsParser,
            ecmaVersion: 2020,
            sourceType: 'module',
            parserOptions: {
                ecmaFeatures: {
                    jsx: true,
                },
            },
            globals: globals.browser,
        },
        plugins: {
            '@typescript-eslint': tseslint,
            'react-hooks': reactHooks,
            'react-refresh': reactRefresh,
        },
        rules: {
            '@typescript-eslint/no-unused-vars': ['error'],
            'eol-last': 'error',
            'no-unused-vars': 'off',
            'react-hooks/exhaustive-deps': 'warn',
            'react-hooks/rules-of-hooks': 'error',
            'react-refresh/only-export-components': [
                'error',
                { allowConstantExport: true },
            ],
        },
    },
    {
        files: ['e2e/**/*.ts', 'playwright.config.ts', 'playwright.ts'],
        languageOptions: {
            globals: {
                ...disabledBrowserGlobals,
                ...globals.node,
            },
        },
    },
];
