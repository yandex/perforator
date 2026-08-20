const { randomInt } = require('node:crypto');

const fc = require('fast-check');


const AUTOCHECK_SEED = 0x5eedc0de;
const AUTOCHECK_NUM_RUNS = 200;
const RANDOM_NUM_RUNS = 2_000;

const randomSeedRequested = process.env.FC_SEED === 'random';
const seed = randomSeedRequested
    ? randomInt(0x1_0000_0000) - 0x8000_0000
    : Number(process.env.FC_SEED ?? AUTOCHECK_SEED);
const numRuns = Number(process.env.FC_NUM_RUNS ?? (randomSeedRequested ? RANDOM_NUM_RUNS : AUTOCHECK_NUM_RUNS));

if (!Number.isInteger(seed) || seed < -0x8000_0000 || seed >= 0x8000_0000) {
    throw new Error('FC_SEED must be a signed 32-bit integer or "random"');
}
if (!Number.isInteger(numRuns) || numRuns <= 0) {
    throw new Error('FC_NUM_RUNS must be a positive integer');
}

fc.configureGlobal({ numRuns, seed });
