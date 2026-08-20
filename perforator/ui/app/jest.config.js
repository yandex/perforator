module.exports = {
    moduleFileExtensions: ['js', 'json', 'ts', 'tsx'],
    rootDir: 'src',
    collectCoverageFrom: ['**/*.{ts,tsx,js}'],
    coverageDirectory: '../coverage',
    testEnvironment: 'jsdom',
    testRegex: '\\.test\\.(ts|tsx|js)$',
    setupFilesAfterEnv: ['<rootDir>/test/setup.ts'],
    transform: {
        '^.+\\.(t|j)sx?$': ['@swc/jest', {
            jsc: {
                transform: {
                    react: {
                        runtime: 'automatic',
                    },
                },
            },
        }],
    },
    moduleNameMapper: {
        '^src/(.*)$': '<rootDir>/$1',
        '\\.(css|scss)$': '<rootDir>/test/styleMock.js',
    },
};
