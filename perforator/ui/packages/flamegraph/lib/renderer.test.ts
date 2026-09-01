import * as fc from 'fast-check';

import { describe, expect, it } from '@jest/globals';

import { cloneRows, forEachNode, profileRowsArbitrary } from './__test-helpers__/profileRowsArbitrary';
import { toDenseCoordinates } from './densely-packed';
import type { FormatNode } from './models/Profile';
import { getNodeTitleFull } from './node-title';
import { type Coordinate, FlamegraphOffseter } from './renderer';
import { getCanvasTitleFull, getStatusTitleFull, renderTitleFull } from './title';


const CANVAS_WIDTH = 1_000;

const stringTable = ['all', 'child1', 'child2', 'child3', 'child4', '@[kernel]'];

const rootNode: FormatNode = {
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

interface RenderStateSeed {
    canvasWidth: number;
    initialCoordinateSeed: number;
    omittedCoordinateSeeds: number[];
    keepCoordinateSeeds: number[] | null;
    shouldReverseDiff: boolean;
}

interface RenderState {
    canvasWidth: number;
    initialCoordinates: Coordinate;
    omittedCoordinates: Coordinate[];
    keepCoordinates: number[] | null;
    shouldReverseDiff: boolean;
}

interface RenderModel {
    sourceRows: FormatNode[][];
}

interface RenderReal {
    offsetter: FlamegraphOffseter;
}

const renderStateSeedArbitrary: fc.Arbitrary<RenderStateSeed> = fc.record({
    canvasWidth: fc.integer({ min: 1, max: 2_000 }),
    initialCoordinateSeed: fc.nat(),
    omittedCoordinateSeeds: fc.array(fc.nat(), { maxLength: 12 }),
    keepCoordinateSeeds: fc.option(fc.array(fc.nat(), { maxLength: 12 }), { nil: null }),
    shouldReverseDiff: fc.boolean(),
});

class PrerenderOffsetsCommand implements fc.Command<RenderModel, RenderReal> {
    private readonly stateSeed: RenderStateSeed;

    constructor(stateSeed: RenderStateSeed) {
        this.stateSeed = stateSeed;
    }

    check(): boolean {
        return true;
    }

    run(model: RenderModel, real: RenderReal): void {
        assertMatchesFreshRender(real.offsetter, model.sourceRows, makeRenderState(model.sourceRows, this.stateSeed));
    }

    toString(): string {
        return `prerenderOffsets(${JSON.stringify(this.stateSeed)})`;
    }
}

const renderCommandsArbitrary = fc.commands<RenderModel, RenderReal>([
    renderStateSeedArbitrary.map((stateSeed) => new PrerenderOffsetsCommand(stateSeed)),
], { maxCommands: 12 });

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
        expect(getCanvasTitle(childOne, null, rootNode)).toMatchSnapshot();
        expect(getCanvasTitle(childTwo, null, rootNode)).toMatchSnapshot();
        expect(getCanvasTitle(childThree, null, rootNode)).toMatchSnapshot();
        expect(getCanvasTitle(childFour, null, rootNode)).toMatchSnapshot();
        expect(getCanvasTitle(childTwo, childOne, rootNode)).toMatchSnapshot();
    });
    it('should correctly render status title', () => {
        expect(getStatusTitle(childOne, null, rootNode)).toMatchSnapshot();
        expect(getStatusTitle(childTwo, null, rootNode)).toMatchSnapshot();
        expect(getCanvasTitle(childThree, null, rootNode)).toMatchSnapshot();
        expect(getCanvasTitle(childFour, null, rootNode)).toMatchSnapshot();
        expect(getStatusTitle(childTwo, childOne, rootNode)).toMatchSnapshot();
    });
});

describe('flamegraph titles for diffs', () => {
    const getNodeTitle = getNodeTitleFull.bind(null, readString, (s) => s, false);

    const renderTitle = renderTitleFull.bind(null, n => n.eventCount, n => n.sampleCount, getNodeTitle, true, false);

    const getStatusTitle = getStatusTitleFull('cycles', renderTitle);

    const getCanvasTitle = getCanvasTitleFull('cycles', renderTitle);

    it('should correctly render canvas title', () => {
        expect(getCanvasTitle(childOne, null, rootNode)).toMatchSnapshot();
        expect(getCanvasTitle(childTwo, null, rootNode)).toMatchSnapshot();
        expect(getCanvasTitle(childThree, null, rootNode)).toMatchSnapshot();
        expect(getCanvasTitle(childFour, null, rootNode)).toMatchSnapshot();
        expect(getCanvasTitle(childTwo, childOne, rootNode)).toMatchSnapshot();
    });
    it('should correctly render status title', () => {
        expect(getStatusTitle(childOne, null, rootNode)).toMatchSnapshot();
        expect(getStatusTitle(childTwo, null, rootNode)).toMatchSnapshot();
        expect(getCanvasTitle(childThree, null, rootNode)).toMatchSnapshot();
        expect(getCanvasTitle(childFour, null, rootNode)).toMatchSnapshot();
        expect(getStatusTitle(childTwo, childOne, rootNode)).toMatchSnapshot();
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

    it('clears omission flags from zero-count nodes when filters change', () => {
        const rows: FormatNode[][] = [[
            { parentIndex: -1, textId: 0, eventCount: 0, sampleCount: 0 },
        ]];
        const offsetter = createOffsetter(rows);

        offsetter.prerenderOffsets(100, [0, 0], [[0, 0]]);
        expect(rows[0][0].omittedNode).toBe(true);

        offsetter.prerenderOffsets(100, [0, 0]);
        expect(rows[0][0].omittedNode).toBe(false);
    });

    it('recomputes keep-only-found when generated omissions change', () => {
        fc.assert(fc.property(
            profileRowsArbitrary(),
            fc.nat(),
            ({ rows: generatedRows }, coordinateSeed) => {
                const eligibleKeepCoordinates = getAllCoordinates(generatedRows).filter(([h, i]) => {
                    return h > 0 && generatedRows[h][i].eventCount > 0 &&
                        generatedRows[0][0].eventCount > generatedRows[h][i].eventCount;
                });
                fc.pre(eligibleKeepCoordinates.length > 0);

                const keepCoordinate = eligibleKeepCoordinates[coordinateSeed % eligibleKeepCoordinates.length];
                const keepCoordinates = toDenseCoordinates([keepCoordinate]);
                const reusedRows = cloneRows(generatedRows);
                const reusedOffsetter = createOffsetter(reusedRows);

                assertMatchesFreshRender(reusedOffsetter, generatedRows, {
                    canvasWidth: CANVAS_WIDTH,
                    initialCoordinates: [0, 0],
                    omittedCoordinates: [],
                    keepCoordinates,
                    shouldReverseDiff: false,
                });
                assertMatchesFreshRender(reusedOffsetter, generatedRows, {
                    canvasWidth: CANVAS_WIDTH,
                    initialCoordinates: [0, 0],
                    omittedCoordinates: [keepCoordinate],
                    keepCoordinates,
                    shouldReverseDiff: false,
                });
            },
        ));
    });

    it('matches fresh renders throughout generated filter and viewport changes', () => {
        fc.assert(fc.property(
            profileRowsArbitrary(),
            renderCommandsArbitrary,
            ({ rows: generatedRows }, commands) => {
                const reusedRows = cloneRows(generatedRows);
                const reusedOffsetter = createOffsetter(reusedRows);

                fc.modelRun(
                    () => ({ model: { sourceRows: generatedRows }, real: { offsetter: reusedOffsetter } }),
                    commands,
                );
                expect(readWireFields(reusedRows)).toEqual(readWireFields(generatedRows));
            },
        ));
    });

    it('lays out generated rows contiguously and resolves visible frame centers', () => {
        fc.assert(fc.property(
            profileRowsArbitrary(),
            fc.boolean(),
            ({ rows: generatedRows }, shouldReverseDiff) => {
                const root = generatedRows[0][0];
                const rootCount = shouldReverseDiff ? root.baseEventCount ?? 0 : root.eventCount;
                fc.pre(rootCount > 0);

                const renderedRows = cloneRows(generatedRows);
                const offsetter = createOffsetter(renderedRows);
                offsetter.prerenderOffsets(CANVAS_WIDTH, [0, 0], [], null, shouldReverseDiff);

                expect(renderedRows[0][0].x).toBe(0);
                forEachNode(renderedRows, (node, h, i) => {
                    expect(Number.isFinite(node.x)).toBe(true);
                    expect(node.x).toBeGreaterThanOrEqual(0);

                    const width = offsetter.countWidth(node);
                    expect(Number.isFinite(width)).toBe(true);
                    expect(width).toBeGreaterThanOrEqual(0);
                    expect(width).toBeLessThanOrEqual(CANVAS_WIDTH);

                    if (offsetter.visibleNode(node) && width > 0) {
                        expect(offsetter.getCoordsByPositionWithKnownHeight(h, node.x! + width / 2)).toEqual({ h, i });
                    }
                });

                assertContiguousChildren(renderedRows, offsetter);
                expect(readWireFields(renderedRows)).toEqual(readWireFields(generatedRows));
            },
        ));
    });
});

function createOffsetter(rows: FormatNode[][]) {
    return new FlamegraphOffseter(rows, { reverse: false, levelHeight: 20 });
}

function makeRenderState(rows: FormatNode[][], seed: RenderStateSeed): RenderState {
    const coordinates = getAllCoordinates(rows);
    const initialCoordinates = coordinates[seed.initialCoordinateSeed % coordinates.length];
    const omittedCoordinates = makeOmissionAntichain(rows, coordinates, seed.omittedCoordinateSeeds);
    const keepCoordinates = seed.keepCoordinateSeeds === null
        ? null
        : toDenseCoordinates(getUniqueSeededCoordinates(coordinates, seed.keepCoordinateSeeds));

    return {
        canvasWidth: seed.canvasWidth,
        initialCoordinates,
        omittedCoordinates,
        keepCoordinates,
        shouldReverseDiff: seed.shouldReverseDiff,
    };
}

function assertMatchesFreshRender(
    reusedOffsetter: FlamegraphOffseter,
    sourceRows: FormatNode[][],
    state: RenderState,
) {
    const reusedDepth = prerender(reusedOffsetter, state);
    const freshRows = cloneRows(sourceRows);
    const freshOffsetter = createOffsetter(freshRows);
    const freshDepth = prerender(freshOffsetter, state);

    expect(reusedDepth).toBe(freshDepth);
    expect(readRenderState(reusedOffsetter)).toEqual(readRenderState(freshOffsetter));
    expect(readWireFields(reusedOffsetter.rows)).toEqual(readWireFields(sourceRows));
}

function prerender(offsetter: FlamegraphOffseter, state: RenderState) {
    return offsetter.prerenderOffsets(
        state.canvasWidth,
        state.initialCoordinates,
        state.omittedCoordinates,
        state.keepCoordinates,
        state.shouldReverseDiff,
    );
}

function readRenderState(offsetter: FlamegraphOffseter) {
    return {
        currentNodeCoords: offsetter.currentNodeCoords,
        framesWindow: offsetter.rows.map((_, h) => ({
            has: offsetter.hasFramesWindowFor(h),
            left: offsetter.getFramesWindowLeft(h),
            right: offsetter.getFramesWindowRight(h),
        })),
        nodes: offsetter.rows.map((row, h) => row.map((frame, i) => ({
            x: isInFramesWindow(offsetter, h, i) ? frame.x : undefined,
            omittedEventCount: frame.omittedEventCount ?? 0,
            omittedSampleCount: frame.omittedSampleCount ?? 0,
            omittedNode: Boolean(frame.omittedNode),
            eventCountWidth: offsetter.countEventCountWidth(frame),
            sampleCountWidth: offsetter.countSampleCountWidth(frame),
            width: offsetter.countWidth(frame),
            visible: offsetter.visibleNode(frame),
        }))),
    };
}

function isInFramesWindow(offsetter: FlamegraphOffseter, h: number, i: number) {
    const left = offsetter.getFramesWindowLeft(h);
    const right = offsetter.getFramesWindowRight(h);
    return left !== undefined && right !== undefined && i >= left && i <= right;
}

function readWireFields(rows: FormatNode[][]) {
    return rows.map((row) => row.map((node) => ({
        parentIndex: node.parentIndex,
        textId: node.textId,
        eventCount: node.eventCount,
        sampleCount: node.sampleCount,
        baseEventCount: node.baseEventCount,
        baseSampleCount: node.baseSampleCount,
    })));
}

function getAllCoordinates(rows: FormatNode[][]): Coordinate[] {
    return rows.flatMap((row, h) => row.map((_, i): Coordinate => [h, i]));
}

function getUniqueSeededCoordinates(coordinates: Coordinate[], seeds: number[]) {
    const result: Coordinate[] = [];
    const seen = new Set<number>();
    for (const seed of seeds) {
        const coordinateIndex = seed % coordinates.length;
        if (!seen.has(coordinateIndex)) {
            seen.add(coordinateIndex);
            result.push(coordinates[coordinateIndex]);
        }
    }
    return result;
}

function makeOmissionAntichain(rows: FormatNode[][], coordinates: Coordinate[], seeds: number[]) {
    const result: Coordinate[] = [];
    for (const coordinate of getUniqueSeededCoordinates(coordinates, seeds)) {
        if (result.every((selected) => !isAncestor(rows, selected, coordinate) && !isAncestor(rows, coordinate, selected))) {
            result.push(coordinate);
        }
    }
    return result;
}

function isAncestor(rows: FormatNode[][], candidateAncestor: Coordinate, coordinate: Coordinate) {
    const [ancestorH, ancestorI] = candidateAncestor;
    let [h, i] = coordinate;
    while (h > ancestorH) {
        i = rows[h][i].parentIndex;
        h--;
    }
    return h === ancestorH && i === ancestorI;
}

function assertContiguousChildren(rows: FormatNode[][], offsetter: FlamegraphOffseter) {
    for (let h = 1; h < rows.length; h++) {
        const nextOffsetByParent = new Map<number, number>();
        for (const node of rows[h]) {
            const parent = rows[h - 1][node.parentIndex];
            const expectedX = nextOffsetByParent.get(node.parentIndex) ?? parent.x!;
            expect(node.x).toBeCloseTo(expectedX, 8);

            if (offsetter.visibleNode(node)) {
                nextOffsetByParent.set(node.parentIndex, expectedX + offsetter.countWidth(node));
            }
        }

        for (const [parentIndex, childrenRightEdge] of nextOffsetByParent) {
            const parent = rows[h - 1][parentIndex];
            expect(childrenRightEdge).toBeLessThanOrEqual(parent.x! + offsetter.countWidth(parent) + 1e-8);
        }
    }
}
