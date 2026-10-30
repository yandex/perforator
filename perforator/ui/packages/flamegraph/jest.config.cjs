module.exports = {
    moduleFileExtensions: ['js', 'json', 'ts', 'tsx'],
    rootDir: 'lib',
    collectCoverageFrom: ['**/*.{ts,tsx,js}'],
    coverageDirectory: '../coverage',
    setupFiles: ['<rootDir>/../fast-check.setup.cjs'],
    testEnvironment: 'node',
    testRegex: '\\.test.(ts|tsx|js)$',
    transformIgnorePatterns: [
        'node_modules/.pnpm/virtual-store/(?!(culori))',
    ],
    moduleNameMapper: {
        '\\.(css|scss)$': '<rootDir>/__test-helpers__/styleMock.js',
    },
    transform: {
        '^.+\\.(t|j)sx?$': '@swc/jest',
    },
};
