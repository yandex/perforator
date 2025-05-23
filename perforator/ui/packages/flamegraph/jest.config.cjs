module.exports = {
    moduleFileExtensions: ['js', 'json', 'ts'],
    rootDir: 'lib',
    collectCoverageFrom: ['**/*.(t|j)s'],
    coverageDirectory: '../coverage',
    testEnvironment: 'node',
    testRegex: '\\.test.(ts|js)$',
    transform: {
        '^.+\\.(t|j)sx?$': '@swc/jest',
    },
};
