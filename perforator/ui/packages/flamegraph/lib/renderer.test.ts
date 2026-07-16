import { describe, expect, it } from '@jest/globals';

import type { FormatNode } from './models/Profile';
import { getNodeTitleFull } from './node-title';
import { FlamegraphOffseter } from './renderer';
import { getCanvasTitleFull, getStatusTitleFull, renderTitleFull } from './title';


const stringTable = ['all', 'child1', 'child2', 'child3', 'child4', '@[kernel]'];

const node: FormatNode = {
    parentIndex: -1,
    eventCount: 100,
    baseEventCount: 200,
    baseSampleCount: 4,
    sampleCount: 1,
    textId: 0,
};

const childOne: FormatNode = {
    parentIndex: 0,
    eventCount: 50,
    baseEventCount: 40,
    baseSampleCount: 2,
    sampleCount: 1,
    textId: 1,
};

const childTwo: FormatNode = {
    parentIndex: 0,
    eventCount: 25,
    baseEventCount: 10,
    baseSampleCount: 1,
    sampleCount: 1,
    textId: 2,
};

const childThree: FormatNode = {
    parentIndex: 1,
    eventCount: 2,
    baseEventCount: 10,
    baseSampleCount: 1,
    sampleCount: 1,
    textId: 3,
    inlined: true,
};

const childFour: FormatNode = {
    parentIndex: 1,
    eventCount: 2,
    baseEventCount: 10,
    baseSampleCount: 1,
    sampleCount: 1,
    textId: 4,
    file: 5,
};

function readString(id?: number) {
    if (id === undefined) { return '';}
    return stringTable[id];
}

describe('flamegraph titles', () => {
    const getNodeTitle = getNodeTitleFull.bind(null, readString, (s) => s, false);

    const renderTitle = renderTitleFull.bind(null, n => n.eventCount, n => n.sampleCount, getNodeTitle, false, false);

    const getStatusTitle = getStatusTitleFull('cycles', renderTitle);

    const getCanvasTitle = getCanvasTitleFull('cycles', renderTitle);
    it('should correctly render canvas title', () => {
        expect(getCanvasTitle(childOne, null, node)).toMatchSnapshot();
        expect(getCanvasTitle(childTwo, null, node)).toMatchSnapshot();
        expect(getCanvasTitle(childThree, null, node)).toMatchSnapshot();
        expect(getCanvasTitle(childFour, null, node)).toMatchSnapshot();
        expect(getCanvasTitle(childTwo, childOne, node)).toMatchSnapshot();
    });
    it('should correctly render status title', () => {
        expect(getStatusTitle(childOne, null, node)).toMatchSnapshot();
        expect(getStatusTitle(childTwo, null, node)).toMatchSnapshot();
        expect(getCanvasTitle(childThree, null, node)).toMatchSnapshot();
        expect(getCanvasTitle(childFour, null, node)).toMatchSnapshot();
        expect(getStatusTitle(childTwo, childOne, node)).toMatchSnapshot();
    });
});

describe('flamegraph titles for diffs', () => {
    const getNodeTitle = getNodeTitleFull.bind(null, readString, (s) => s, false);

    const renderTitle = renderTitleFull.bind(null, n => n.eventCount, n => n.sampleCount, getNodeTitle, true, false);

    const getStatusTitle = getStatusTitleFull('cycles', renderTitle);

    const getCanvasTitle = getCanvasTitleFull('cycles', renderTitle);

    it('should correctly render canvas title', () => {
        expect(getCanvasTitle(childOne, null, node)).toMatchSnapshot();
        expect(getCanvasTitle(childTwo, null, node)).toMatchSnapshot();
        expect(getCanvasTitle(childThree, null, node)).toMatchSnapshot();
        expect(getCanvasTitle(childFour, null, node)).toMatchSnapshot();
        expect(getCanvasTitle(childTwo, childOne, node)).toMatchSnapshot();
    });
    it('should correctly render status title', () => {
        expect(getStatusTitle(childOne, null, node)).toMatchSnapshot();
        expect(getStatusTitle(childTwo, null, node)).toMatchSnapshot();
        expect(getCanvasTitle(childThree, null, node)).toMatchSnapshot();
        expect(getCanvasTitle(childFour, null, node)).toMatchSnapshot();
        expect(getStatusTitle(childTwo, childOne, node)).toMatchSnapshot();
    });
});

describe('FlamegraphOffseter filters', () => {
    it('keeps matched stacks filtered when one of duplicate matches is omitted', () => {
        const rows: FormatNode[][] = [
            [
                { parentIndex: -1, textId: 0, eventCount: 100, sampleCount: 100 },
            ],
            [
                { parentIndex: 0, textId: 1, eventCount: 40, sampleCount: 40 },
                { parentIndex: 0, textId: 1, eventCount: 30, sampleCount: 30 },
                { parentIndex: 0, textId: 2, eventCount: 30, sampleCount: 30 },
            ],
        ];
        const offsetter = new FlamegraphOffseter(rows, { reverse: false, levelHeight: 20 });
        const matchedCoordinates = [1, 0, 1, 1];

        offsetter.prerenderOffsets(100, [0, 0], [], matchedCoordinates);
        expect(rows[0][0].omittedEventCount).toBe(30);

        offsetter.prerenderOffsets(100, [0, 0], [[1, 0]], matchedCoordinates);

        expect(rows[0][0].omittedEventCount).toBe(70);
        expect(rows[1][0].omittedEventCount).toBe(40);
        expect(rows[1][1].omittedEventCount).toBe(0);
        expect(rows[1][2].omittedEventCount).toBe(30);
    });
});
