module.exports = {
    moduleFileExtensions: ['js', 'json', 'ts'],
    rootDir: 'lib',
    collectCoverageFrom: ['**/*.(t|j)s'],
    coverageDirectory: '../coverage',
    setupFiles: ['<rootDir>/../fast-check.setup.cjs'],
    testEnvironment: 'node',
    testRegex: '\\.test.(ts|js)$',
    transformIgnorePatterns: [
        'node_modules/.pnpm/virtual-store/(?!(culori))',
    ],
    transform: {
        '^.+\\.(t|j)sx?$': '@swc/jest',
    },
};
