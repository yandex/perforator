/* eslint-disable @typescript-eslint/member-ordering */
/* eslint-disable no-console */

// Copyright The async-profiler authors
// SPDX-License-Identifier: Apache-2.0
//
// This code is based on the flamegraph from the beautiful async-profiler.
// See https://github.com/async-profiler/async-profiler/blob/d1498a6c7fda7c5987caf5e301c3de1deb9743c5/src/res/flame.html.
// alterations from the licensed code
// * rewritten into typescript
// * changed the format of the individual nodes: each one does not have it x coordinate
// * added dark mode with automatic darkening
// * rewritten the render logic with higher order functions
// * added different titles for hover and status

import { darken, DARKEN_FACTOR, diffcolor } from './colors';
import type { DenselyPackedCoordinates } from './densely-packed';
import { getDenseH, getDenseI, getDenseLength } from './densely-packed';
import { hugenum } from './flame-utils';
import type { FormatNode, ProfileData } from './models/Profile';
import { getNodeTitleFull } from './node-title';
import { pct } from './pct';
import type { GetStateFromQuery, SetStateFromQuery } from './query-utils';
import { parseStacks, stringifyStacks } from './query-utils';
import { shorten } from './shorten/shorten';
import { getStatusTitleFull, renderTitleFull } from './title';


const dw = Math.floor(255 * (1 - DARKEN_FACTOR)).toString(16);
// for dark theme
const WHITE_TEXT_COLOR = `#${dw}${dw}${dw}`;

const minVisibleWidth = 1e-2;


export type I = number;
export type H = number;

export type Coordinate = [H, I];
export type Interval = [I, I];

export type QueryKeys =
    | 'flamegraphReverse'
    | 'flamegraphQuery'
    | 'flamegraphExclude'
    | 'omittedIndexes'
    | 'keepOnlyFound'
    | 'exactMatch'
    | 'caseInsensitive'
    | 'frameDepth'
    | 'tab'
    | 'topQuery'
    | 'leftHeavy'
    | 'flameBase'
    | 'framePos';
export type RenderFlamegraphOptions = {
    getState: GetStateFromQuery<QueryKeys>;
    setState: SetStateFromQuery<QueryKeys>;
    theme: 'dark' | 'light';
    isDiff: boolean;
    shortenFrameTexts: 'true' | 'false' | 'hover';
    onFinishRendering?: (opts?: {textNodesCount: number; delta: number; exceededLimit: boolean}) => void;
    searchPattern: RegExp | null;
    excludeSearchPattern: RegExp | null;
    reverse: boolean;
    disableHighlightRender: boolean;
    shouldScroll: boolean;
    scrollParent: HTMLElement;
    foundCoords: DenselyPackedCoordinates | null;
}

function makeByHDense(coords: DenselyPackedCoordinates): Record<H, Set<I>> {
    const res: Record<H, Set<I>> = {};
    const length = getDenseLength(coords);
    for (let n = 0; n < length; n++) {
        const h = getDenseH(coords, n);
        const i = getDenseI(coords, n);
        if (!(res[h] instanceof Set)) {
            res[h] = new Set();
        }
        res[h].add(i);
    }
    return res;
}

interface RenderOpts {
    pattern?: RegExp | null;
    excludePattern?: RegExp | null;
}


type RenderFlamegraphType = (
    flamegraphContainer: HTMLDivElement,
    profileData: ProfileData,
    fg: FlamegraphOffseter,
    options: RenderFlamegraphOptions,
) => () => void;

/**
 * Continuous array keeping rendering borders for each level
 * Uses even indexes for left border and odd indexes for right border
 * For level h: left border at index h*2, right border at index h*2+1
 * everything up the subtree is filled before rendering
 * everything below the subtree is filled during render
 */
export type FramesWindow = number[];

const getSamplesFn = (node?: FormatNode) => {
    return node?.sampleCount ?? 0;
};

const getBaseSamplesFn = (node?: FormatNode) => {
    return node?.baseSampleCount ?? 0;
};

const getEventsFn = (node?: FormatNode) => {
    return node?.eventCount ?? 0;
};

const getBaseEventsFn = (node?: FormatNode) => {
    return node?.baseEventCount ?? 0;
};

export class FlamegraphOffseter {
    currentNodeCoords: Coordinate = [0, 0];
    rows: FormatNode[][];


    private framesWindow: FramesWindow;
    private canvasWidth: number | undefined;
    private widthRatio: number | undefined;
    private minVisibleEv: number | undefined;
    private reverse: boolean;
    levelHeight: number;
    private shouldReverseDiff = false;
    private maxVerticalRow: number | undefined;

    private prevOmittedOffsetCoordinates: Coordinate[] | undefined;
    private prevKeepFoundCoordinates: DenselyPackedCoordinates | null | undefined;
    private prevShouldReverseDiff: boolean | undefined;

    getEvents: (node?: FormatNode) => number;
    getSamples: (node: FormatNode) => number;

    constructor(rows: ProfileData['rows'], options: { reverse: boolean; levelHeight: number }) {
        this.rows = rows;
        this.reverse = options.reverse;
        this.levelHeight = options.levelHeight;
        this.maxVerticalRow = this.rows.length;
        this.getEvents = getEventsFn;
        this.getSamples = getSamplesFn;
    }
    fillFramesWindow([hmax, imax]: Coordinate): FramesWindow {
        const res: FramesWindow = [];
        let nextParentIndex = imax;

        for (let h = Math.min(hmax, this.rows.length - 1); h >= 0; h--) {
            const row = this.rows[h];
            res[h * 2] = nextParentIndex;     // left border at even index
            res[h * 2 + 1] = nextParentIndex; // right border at odd index
            // will be assigned -1 on the last iteration (root)
            // we do not care about it because it will not be assigned anywhere else
            nextParentIndex = row[nextParentIndex].parentIndex;
        }

        return res;
    }

    getFramesWindowLeft(h: number): number | undefined {
        return this?.framesWindow?.[h * 2];
    }

    getFramesWindowRight(h: number): number | undefined {
        return this.framesWindow?.[h * 2 + 1];
    }

    hasFramesWindowFor(h: number) {
        return this?.framesWindow?.[h * 2] !== undefined;
    }

    calcTopOffset(h: number) {
        return this.reverse ? h * this.levelHeight : (this.maxVerticalRow * this.levelHeight) - (h + 1) * this.levelHeight;
    }

    backpropagateOmittedEventCount(omittedOffsetCoordinates: Coordinate[]) {
        for (const [h, i] of omittedOffsetCoordinates) {
            const node = this.rows[h][i];
            let currentH = h;
            let currentI = i;
            const eventCountToOmit = (this.getEvents(node)) - (node.omittedEventCount ?? 0);
            const sampleCountToOmit = (this.getSamples(node)) - (node.omittedSampleCount ?? 0);
            while (currentH >= 0) {
                const currentNode = this.rows[currentH][currentI];
                currentNode.omittedEventCount = (currentNode.omittedEventCount ?? 0) + eventCountToOmit;
                currentNode.omittedSampleCount = (currentNode.omittedSampleCount ?? 0) + sampleCountToOmit;
                if (currentNode.omittedEventCount === (this.getEvents(currentNode))) {
                    currentNode.omittedNode = true;
                }
                currentH--;
                currentI = currentNode.parentIndex;
            }
        }
        let omittedParentIndexes: Set<number> = new Set();
        let nextOmittedParentIndexes: Set<number> = new Set();
        for (let h = 0; h < this.rows.length; h++) {
            const row = this.rows[h];
            for (let i = 0; i < row.length; i++) {
                const node = row[i];
                if (node.omittedNode && node.omittedEventCount && node.omittedSampleCount) {
                    nextOmittedParentIndexes.add(i);
                }
                if (omittedParentIndexes.has(node.parentIndex)) {
                    node.omittedNode = true;
                    node.omittedEventCount = (this.getEvents(node));
                    node.omittedSampleCount = (this.getSamples(node));
                    nextOmittedParentIndexes.add(i);
                }
            }
            omittedParentIndexes = nextOmittedParentIndexes;
            nextOmittedParentIndexes = new Set();
        }
    }

    // omit everything
    // then start deleting everything we want to keep
    backpropagateKeepOnlyFound(keepCoordinates: DenselyPackedCoordinates) {
        for (let h = 0; h < this.rows.length; h++) {
            for (let i = 0; i < this.rows[h].length; i++) {
                const node = this.rows[h][i];
                node.omittedEventCount = (this.getEvents(node));
                node.omittedSampleCount = (this.getSamples(node));
            }
        }

        let maxH = 0;
        const length = getDenseLength(keepCoordinates);
        for (let n = 0; n < length; n++) {
            const h = getDenseH(keepCoordinates, n);
            if (h > maxH) {
                maxH = h;
            }
        }

        const keptCoordinatesByHs = makeByHDense(keepCoordinates);

        for (let h = maxH; h >= 0; h--) {
            const row = this.rows[h];
            const keptCoordinates = keptCoordinatesByHs[h];
            for (let i = 0; i < row.length; i++) {
                const node = row[i];


                if (keptCoordinates?.has?.(i)) {
                    node.omittedEventCount = 0;
                    node.omittedSampleCount = 0;
                }
                if (node.omittedEventCount !== (this.getEvents(node)) && node.omittedSampleCount !== node.sampleCount) {
                    if (node.parentIndex !== -1) {
                        const parentNode = this.rows[h - 1][node.parentIndex];
                        parentNode.omittedEventCount! -= ((this.getEvents(node)) - (node.omittedEventCount ?? 0));
                        parentNode.omittedSampleCount! -= (node.sampleCount - (node.omittedSampleCount ?? 0));

                    }
                }
            }
        }

        let keptParentIndexes: Set<number> = new Set();
        let nextKeptParentIndexes: Set<number> = new Set();
        for (let h = 0; h < this.rows.length; h++) {
            const row = this.rows[h];
            const keptCoordinates = keptCoordinatesByHs[h];
            // const parentKeptCoordinates = keptCoordinatesByHs[h - 1];
            for (let i = 0; i < row.length; i++) {
                const node = row[i];
                if (keptCoordinates?.has?.(i)) {
                    nextKeptParentIndexes.add(i);
                }
                if (keptParentIndexes.has(node.parentIndex)) {
                    node.omittedEventCount = 0;
                    node.omittedSampleCount = 0;
                    nextKeptParentIndexes.add(i);
                }
            }
            keptParentIndexes = nextKeptParentIndexes;
            nextKeptParentIndexes = new Set();
        }
    }

    private clearOmittedEventCount() {
        if ('omittedEventCount' in this.rows[0][0] && this.rows[0][0].omittedEventCount > 0) {
            for (let h = 0; h < this.rows.length; h++) {
                const row = this.rows[h];
                for (let i = 0; i < row.length; i++) {
                    const node = row[i];
                    if (node.omittedEventCount) {
                        node.omittedEventCount = undefined;
                    }
                    if (node.omittedSampleCount) {
                        node.omittedSampleCount = undefined;
                    }
                    if (node.omittedNode) {
                        node.omittedNode = false;
                    }
                }
            }
        }
    }

    private areDenseCoordinateArraysEqual(a: DenselyPackedCoordinates | null | undefined, b: DenselyPackedCoordinates | null | undefined): boolean {
        if (a === b) {
            return true;
        }
        if (!a || !b) {
            return false;
        }
        if (a.length !== b.length) {
            return false;
        }

        for (let i = 0; i < a.length; i++) {
            if (a[i] !== b[i]) {
                return false;
            }
        }
        return true;
    }

    private areCoordinateArraysEqual(a: Coordinate[] | null | undefined, b: Coordinate[] | null | undefined): boolean {
        if (a === b) {
            return true;
        }
        if (!a || !b) {
            return false;
        }
        if (a.length !== b.length) {
            return false;
        }

        for (let i = 0; i < a.length; i++) {
            if (a[i][0] !== b[i][0] || a[i][1] !== b[i][1]) {
                return false;
            }
        }
        return true;
    }

    // eslint-disable-next-line complexity
    prerenderOffsets(
        canvasWidth: number,
        initialCoordinates: Coordinate,
        omittedOffsetCoordinates: Coordinate[] = [],
        keepFoundCoordinates: DenselyPackedCoordinates | null = null,
        shouldReverseDiff = false,
        visitors: Array<{run: (node: FormatNode) => void}> = [],
    ) {
        // Check if we need to recalculate omitted/kept nodes
        const omittedChanged = !this.areCoordinateArraysEqual(this.prevOmittedOffsetCoordinates, omittedOffsetCoordinates);
        const keepFoundChanged = !this.areDenseCoordinateArraysEqual(this.prevKeepFoundCoordinates, keepFoundCoordinates);
        const shouldReverseDiffChanged = this.prevShouldReverseDiff !== shouldReverseDiff;

        // Only clear and recalculate if relevant parameters have changed
        if (omittedChanged || keepFoundChanged || shouldReverseDiffChanged) {
            this.clearOmittedEventCount();
        }

        this.shouldReverseDiff = shouldReverseDiff;
        this.canvasWidth = canvasWidth;
        this.currentNodeCoords = initialCoordinates;
        const [initialH, initialI] = initialCoordinates;

        if (shouldReverseDiff) {
            this.getEvents = getBaseEventsFn;
            this.getSamples = getBaseSamplesFn;
        } else {
            this.getEvents = getEventsFn;
            this.getSamples = getSamplesFn;
        }

        this.framesWindow = this.fillFramesWindow(initialCoordinates);

        // Only run expensive operations if parameters changed
        if (keepFoundChanged && keepFoundCoordinates) {
            this.backpropagateKeepOnlyFound(keepFoundCoordinates);
        }
        if (omittedChanged && omittedOffsetCoordinates && omittedOffsetCoordinates.length) {
            this.backpropagateOmittedEventCount(omittedOffsetCoordinates);
        }

        // Update cached parameters
        this.prevOmittedOffsetCoordinates = omittedOffsetCoordinates;
        this.prevKeepFoundCoordinates = keepFoundCoordinates;
        this.prevShouldReverseDiff = shouldReverseDiff;
        const root = this.rows[initialH][initialI];
        this.widthRatio = (this.getEvents(root) - (root.omittedEventCount ?? 0)) / canvasWidth!;
        this.minVisibleEv = minVisibleWidth * this.widthRatio;

        let maxDrawableLayerDepth = 0;

        for (let h = 0; h < this.rows.length; h++) {
            const leftBorder = this.getFramesWindowLeft(h) ?? 0;
            const rightBorder = this.getFramesWindowRight(h) ?? this.rows[h].length - 1;

            const parentLeftBorder = this.getFramesWindowLeft(h - 1);
            const parentRightBorder = this.getFramesWindowRight(h - 1);
            let prevParentIndex: number | null = -2;
            let currentOffset = 0;
            const row = this.rows[h];
            const updateFrameWindows = this.createUpdateWindow(h);
            let shouldDrawLayer = false;
            for (let i = leftBorder; i <= rightBorder; i++) {
                let shouldDrawFrame: boolean | undefined;
                if (h === 0) {
                    shouldDrawFrame = true;
                } else if (parentLeftBorder === undefined || parentRightBorder === undefined) {
                    shouldDrawFrame = true;
                } else {
                    const parentIndex = this.rows[h][i].parentIndex;
                    shouldDrawFrame = parentLeftBorder <= parentIndex &&
                    parentRightBorder >= parentIndex;
                }
                if (!shouldDrawFrame) {
                    continue;
                }
                updateFrameWindows(i);
                for (let vi = 0; vi < visitors.length; vi++) {
                    const visitor = visitors[vi];
                    visitor.run(this.rows[h][i]);
                }
                const isVisible = this.visibleNode(this.rows[h][i]);

                const node = row[i];

                // can ignore when we know parents
                // node.parentIndex === null means root
                if (node.parentIndex !== prevParentIndex && h !== 0) {
                    const parent = this.rows[h - 1][node.parentIndex].x;
                    prevParentIndex = node.parentIndex;
                    currentOffset = parent;
                }
                node.x = currentOffset;
                if (isVisible) {
                    const width = this.countWidth(node);
                    currentOffset += width;
                    shouldDrawLayer = true;
                }
            }

            if (shouldDrawLayer) {
                maxDrawableLayerDepth = h;
            }

            if (!this.hasFramesWindowFor(h)) {
                break;
            }
        }
        this.maxVerticalRow = maxDrawableLayerDepth;
        return maxDrawableLayerDepth;
    }

    findFrame(frames: FormatNode[], x: number, left = 0, right = frames.length - 1) {
        if (x < frames[left].x! || x > (frames[right].x! + this.countWidth(frames[right]))) {
            return null;
        }

        while (left <= right) {
            // eslint-disable-next-line no-bitwise
            const mid = (left + right) >>> 1;

            if (frames[mid].x! > x) {
                right = mid - 1;
            } else if (frames[mid].x! + this.countWidth(frames[mid]) <= x) {
                left = mid + 1;
            } else {
                return mid;
            }
        }

        // may be 0-width node, check closest non-zero neighbour
        while (!this.visibleNode(frames[left]) && left < frames.length) {
            ++left;
        }
        if (left >= 0 && left < frames.length && frames[left].x && (frames[left].x! - x) < 0.5) { return left; }
        while (!this.visibleNode(frames[right]) && right > 0) {
            --right;
        }
        if (right >= 0 && right < frames.length && frames[right].x && (x - (frames[right].x! + this.countWidth(frames[right]))) < 0.5) { return right; }

        return null;
    }
    countEventCountWidth(node: FormatNode) {
        return (this.getEvents(node)) - (node.omittedEventCount ?? 0);
    }
    countSampleCountWidth(node: FormatNode) {
        return (this.getSamples(node)) - (node.omittedSampleCount ?? 0);
    }
    getCoordsByPositionWithKnownHeight(h: number, x: number) {

        const row = this.rows[h];

        if (!this.hasFramesWindowFor(h)) {
            return null;
        }

        const leftIndex = this.getFramesWindowLeft(h);
        const rightIndex = this.getFramesWindowRight(h);

        const i = this.findFrame(row, x, leftIndex, rightIndex);


        if (i === null) {
            return null;
        }

        return { h, i };
    }


    getTopOffset(offset: number) {
        return this.reverse ? offset : ((this.maxVerticalRow * this.levelHeight) - offset);
    }

    getCoordsByPosition: (x: number, y: number) => null | { h: number; i: number } = (x, y) => {
        const topOffset = this.getTopOffset(y);
        const h = Math.floor(topOffset / this.levelHeight);

        if (h < 0 || h >= this.rows.length) {
            return null;
        }

        return this.getCoordsByPositionWithKnownHeight(h, x);
    };


    countWidth(node: FormatNode) {
        const omittedFieldName = 'omittedEventCount' as const;
        const evWidth = this.getEvents(node) - (node[omittedFieldName] ?? 0);
        if (evWidth === 0) {
            return 0;
        }
        return Math.min((evWidth) / this.widthRatio!, this.canvasWidth!);
    }

    visibleNode(node?: FormatNode) {
        return (this.getEvents(node)) - (node?.omittedEventCount ?? 0) >= this.minVisibleEv!;
    }


    isBeforeCurrentNode(h: number) {
        return h < this.currentNodeCoords[0];
    }
    private createUpdateWindow = (h: number) => (i: number) => {
        if (this.framesWindow[h * 2] !== undefined) {
            this.framesWindow[h * 2 + 1] = i;
        } else {
            this.framesWindow[h * 2] = i;
            this.framesWindow[h * 2 + 1] = i;
        }
    };

}

export const renderFlamegraph: RenderFlamegraphType = (
    flamegraphContainer,
    profileData,
    fg,
    {
        getState, setState,
        theme,
        isDiff,
        onFinishRendering,
        shortenFrameTexts,
        searchPattern,
        reverse,
        disableHighlightRender,
        scrollParent,
        shouldScroll,
        foundCoords,
    },
) => {
    const shouldSwapDiff = getState('flameBase') === 'diff';

    function findElement(name: string): HTMLElement {
        return flamegraphContainer.querySelector(`.flamegraph__${name}`)!;
    }


    function getCssVariable(variable: string) {
        return getComputedStyle(flamegraphContainer).getPropertyValue(variable);
    }

    const BACKGROUND = getCssVariable('--g-color-base-background');
    const SEARCH_COLOR = theme === 'dark' ? darken('#ee00ee') : '#ee00ee';


    function calculateDiffColor(node: FormatNode, root: FormatNode) {
        const color = diffcolor(node, root, shouldSwapDiff);
        return theme === 'dark' ? darken(color) : color;
    }


    const LEVEL_HEIGHT = parseInt(getCssVariable('--flamegraph-level-height'));
    const BLOCK_SPACING = parseInt(getCssVariable('--flamegraph-block-spacing'));
    const BLOCK_HEIGHT = LEVEL_HEIGHT - BLOCK_SPACING;
    const MAX_TEXT_LABELS = 1500;


    const canvas = findElement('canvas') as HTMLCanvasElement;
    const c = canvas.getContext('2d')!;

    const hl = findElement('highlight');
    const labels = findElement('labels-container');
    const labelTemplate = findElement('label-template') as HTMLTemplateElement;
    const status = findElement('status');
    hl.style.height = String(BLOCK_HEIGHT);

    let canvasWidth: number | undefined;
    let canvasHeight: number | undefined;

    function initCanvasVertical(layersCount: number, shouldPreserveVerticalScroll?: boolean, scrollableElement: HTMLElement = document.documentElement) {
        // need to keep the same vertical scroll when vertically resizing reversed flamegraph
        const prevScroll = scrollableElement.scrollTop;
        const prevScrollHeight = getScrollHeight(scrollableElement);
        const prevBottomOffset = prevScrollHeight - prevScroll;

        canvas.style.height = (layersCount ? layersCount + 1 : profileData.rows.length) * LEVEL_HEIGHT + 'px';
        canvasHeight = canvas.offsetHeight;
        canvas.height = canvasHeight * (devicePixelRatio || 1);
        if (devicePixelRatio) { c.scale(devicePixelRatio, devicePixelRatio); }
        if (shouldPreserveVerticalScroll) {
            const scrollHeight = getScrollHeight(scrollableElement);
            const scroll = scrollableElement.scrollTop;
            const newBottomOffset = scrollHeight - scroll;
            const diff = prevBottomOffset - newBottomOffset;

            scrollableElement.scrollBy(0, -diff);
        }
    }
    function initCanvas() {
        canvasWidth = canvas.offsetWidth;
        canvas.style.width = canvasWidth + 'px';
        canvas.width = canvasWidth * (devicePixelRatio || 1);
    }

    initCanvas();

    c.font = window.getComputedStyle(canvas, null).getPropertyValue('font');
    const textMetrics = c.measureText('O');
    const charWidth = textMetrics.width || 6;
    function readString(id?: number) {
        if (id === undefined) { return ''; }
        return profileData.stringTable[id];
    }

    const shouldShortenTextForOverview = shortenFrameTexts === 'true' || shortenFrameTexts === 'hover';
    const shouldShortenTextForHover = shortenFrameTexts === 'true';

    const getNodeTitle = getNodeTitleFull.bind(null, readString, shorten, shouldShortenTextForOverview);
    const getNodeTitleHl = getNodeTitleFull.bind(null, readString, shorten, shouldShortenTextForHover);

    function drawLabel(text: string, x: number, y: number, w: number, opacity: string, color: string) {
        const dFragment = labelTemplate.content.cloneNode(true) as DocumentFragment;
        const node = dFragment.firstElementChild as HTMLDivElement;
        node.textContent = text;
        node.style.top = y + canvas.offsetTop + 'px';
        node.style.left = x + canvas.offsetLeft + 'px';
        node.style.width = w + 'px';
        node.style.opacity = opacity;
        if (color) {
            node.style.color = color;
        }
        return node;
    }

    function clearLabels() {
        labels.replaceChildren();
    }

    clearLabels();

    const rows = profileData.rows;
    const root = rows[0][0];

    function renderSearch(matched: number, title: string, showReset: boolean) {
        findElement('match-value').textContent = pct(matched, canvasWidth!) + '%';
        findElement('match-value').title = title;
        findElement('match').style.display = showReset ? 'inherit' : 'none';
    }

    // need to calculate cleared percentage

    const renderTitle = renderTitleFull.bind(null, (n) => fg.countEventCountWidth(n), (n) => fg.countSampleCountWidth(n), getNodeTitleHl, isDiff, shouldSwapDiff);

    const eventType = readString(profileData.meta.eventType);

    const getStatusTitle = getStatusTitleFull(eventType, renderTitle);


    function renderImpl(opts?: RenderOpts) {

        clearCanvas();

        const newLabels: HTMLDivElement[] = [];
        let labelCount = 0;


        const marked: Record<number | string, number> = {};
        let markedEventCount = 0;
        let markedSampleCount = 0;
        let markedBaseEventCount = 0;
        let markedBaseSampleCount = 0;


        function mark(f: FormatNode) {
            const width = fg.countWidth(f);
            markedEventCount += ((fg.getEvents(f)) - (f.omittedEventCount ?? 0));
            markedSampleCount += ((fg.getSamples(f)) - (f.omittedSampleCount ?? 0));
            markedBaseEventCount += f.baseEventCount ?? 0;
            markedBaseSampleCount += f.baseSampleCount ?? 0;
            if (!(marked[f.x!] >= width)) {
                marked[f.x!] = width;
            }
        }

        function totalMarked() {
            let keys = Object.keys(marked);
            keys = keys.sort((a, b) => { return Number(a) - Number(b); });
            console.log('keys: ', keys);
            let total = 0;
            let left = 0;
            for (const x of keys) {
                console.log(x, marked[x]);
                const right = Number(x) + marked[x];
                console.log(left, ' |', right, '| ', total);
                if (right > left) {
                    total += right - Math.max(left, Number(x));
                    left = right;
                }
            }
            console.log('total: ', total);
            return total;
        }


        const currentNodeCoords = fg.currentNodeCoords;
        const currentNode = rows[currentNodeCoords[0]][currentNodeCoords[1]];
        const pattern = opts?.pattern;
        const excludePattern = opts?.excludePattern;

        for (let h = 0; h < rows.length; h++) {
            const y = fg.calcTopOffset(h);
            const row = rows[h];
            const alpha = fg.isBeforeCurrentNode(h);


            // eslint-disable-next-line @typescript-eslint/no-loop-func
            const drawFrame = function(i: number) {
                const node = row[i];
                const width = fg.countWidth(node);
                const nodeTitle = getNodeTitle(node);

                const matched = pattern?.test(nodeTitle);
                const matchedExclude = excludePattern?.test(nodeTitle);
                const isMarked = matched && !matchedExclude;
                if (isMarked) {
                    mark(node);
                }

                const color = isMarked ?
                    SEARCH_COLOR :
                    isDiff ? calculateDiffColor(node, currentNode) : node.color!;

                c.fillStyle = color as string;
                c.fillRect(node.x!, y, width, BLOCK_HEIGHT);

                if (width > charWidth * 3 + 6) {
                    labelCount++;
                    if (newLabels.length < MAX_TEXT_LABELS) {

                        const chars = Math.floor((width - 6) / charWidth);
                        const title = nodeTitle.length <= chars ? nodeTitle : nodeTitle.substring(0, chars - 1) + '…';
                        let labelColor: string | undefined;

                        if (alpha && theme === 'dark') {
                            labelColor = WHITE_TEXT_COLOR;
                        }
                        const label = drawLabel(title, node.x!, y, width, alpha ? '0.5' : '1', labelColor!);
                        newLabels.push(label);
                    }
                }


                if (alpha) {
                    c.fillStyle = theme === 'dark' ? '#0000007F' : '#FFFFFF7F';
                    c.fillRect(node.x!, y, width, BLOCK_HEIGHT);
                }
            };

            const renderNode = function(i: number) {
                const node = row[i];

                const isVisible = fg.visibleNode(node);


                if (!isVisible) {
                    return;
                }

                drawFrame(i);

            };

            const leftBorder = fg.getFramesWindowLeft(h);
            const rightBorder = fg.getFramesWindowRight(h);
            for (let i = leftBorder; i <= rightBorder; i++) {
                renderNode(i);
            }

            if (!fg.hasFramesWindowFor(h)) {
                break;
            }
        }

        if (labelCount > MAX_TEXT_LABELS) {
            console.log(`label count limit is ${MAX_TEXT_LABELS}, without it would have shown ${labelCount} labels`);
        }

        labels?.replaceChildren(...newLabels);

        function templateTitle(eventCount: number, sampleCount: number) {
            return `${hugenum(eventCount)} ${eventType} / ${hugenum(sampleCount)} samples`;
        }

        let title = templateTitle(markedEventCount, markedSampleCount);

        if (markedBaseEventCount && markedBaseSampleCount) {
            title = 'Diff: ' + title + '\nBase: ' + templateTitle(markedBaseEventCount, markedBaseSampleCount);
        }

        renderSearch(totalMarked(), title, Boolean(opts?.pattern));

        return { textNodesCount: labelCount };

    }

    let firstRender = true;

    function clearCanvas() {
        c.fillStyle = BACKGROUND;
        c.fillRect(0, 0, canvasWidth!, canvasHeight!);

        clearLabels();
    }

    function render(opts: RenderOpts) {
        const start = performance.now();
        const res = renderImpl(opts);
        const finish = performance.now();
        const delta = finish - start;
        if (firstRender) {
            onFinishRendering?.({ textNodesCount: res.textNodesCount, delta, exceededLimit: res.textNodesCount > MAX_TEXT_LABELS });
            firstRender = false;
        }
        console.log('Rendered flamegraph in', delta, 'ms');
        return res;
    }


    const handleClick = (e: MouseEvent): void => {
        const coords = fg.getCoordsByPosition(e.offsetX, e.offsetY);
        if (!coords) { return; }

        const { i, h } = coords;
        if (!fg.visibleNode(rows[h][i])) {
            canvas.onmouseout?.(e);
            return;
        }
        if (typeof i !== 'number') { return; }

        const omitted = parseStacks(getState('omittedIndexes', '') || '');
        if (e.altKey && !fg.isBeforeCurrentNode(h)) {
            if (!omitted.includes([h, i])) {
                omitted.push([h, i]);
            }
            setState({ omittedIndexes: stringifyStacks(omitted) });
        } else {
            setState({
                frameDepth: h.toString(),
                framePos: i.toString(),
            });
        }

        canvas?.onmouseout?.(e);
    };

    function calcHighlightColor(node: FormatNode) {
        const parsedColor = isDiff ? calculateDiffColor(node, root) : node.color as string;

        let color: string | null = null;
        // currently we calculate diff color on the fly during render
        // highlight is 0.4 darker than default color
        // but for non-diffs the node.color is already darkened by 0.2 so 0.2 is enough
        if (theme === 'dark') {
            color = darken(parsedColor as string, 0.2);
        }
        return color;
    }
    canvas.onclick = handleClick;

    canvas.onmousemove = function (event) {
        const coords = fg.getCoordsByPosition(event.offsetX, event.offsetY);

        if (!coords) {
            canvas.onmouseout?.(event);
            return;
        }
        const { i, h } = coords;
        const node = rows[h][i];
        const currentNodeCoords = fg.currentNodeCoords;
        const currentNode = rows[currentNodeCoords[0]][currentNodeCoords[1]];

        if (!fg.visibleNode(node)) {
            canvas.onmouseout?.(event);
            return;
        }


        renderHighlightRect(h, i);

        const isMainRoot = currentNode && currentNode.textId === root.textId && currentNode.eventCount === root.eventCount;

        status.textContent = 'Function: ' + (isMainRoot ? getStatusTitle(node, null, root) : getStatusTitle(node, currentNode!, root));
        return;


    };


    function renderHighlightRect(h: number, i: number) {
        const node = rows[h][i];
        const width = fg.countWidth(node);
        const left = node.x! + canvas.offsetLeft;
        const top = (fg.calcTopOffset(h) + canvas.offsetTop);
        const title = getNodeTitleHl(node);
        const color = calcHighlightColor(node);
        renderHighlight(title, color, left, top, width);
    }

    function clearHighlight() {
        const currentNodeCoords = fg.currentNodeCoords;
        const currentNode = rows[currentNodeCoords[0]][currentNodeCoords[1]];
        hl.style.display = 'none';
        status.textContent = 'Function: ' + getStatusTitle(currentNode!, null, root);
        canvas.title = '';
        canvas.style.cursor = '';
    }

    canvas.onmouseout = clearHighlight;

    // read query and display h and pos
    const h = parseInt(getState('frameDepth', '0'));
    const pos = parseInt(getState('framePos', '0'));
    const omittedStacks = parseStacks(getState('omittedIndexes', '') || '');

    {
        const layerCount = fg.prerenderOffsets(canvasWidth!, [h, pos], omittedStacks, foundCoords, shouldSwapDiff);
        initCanvasVertical(layerCount, !reverse && shouldScroll, scrollParent);
    }
    render({ pattern: searchPattern });
    if (disableHighlightRender) {
        clearHighlight();
    }
    else {
        renderHighlightRect(h, pos);
    }


    const onResize = () => requestAnimationFrame(() => {
        const initialH = parseInt(getState('frameDepth', '0'));
        const initialI = parseInt(getState('framePos', '0'));
        //@ts-ignore
        canvas.style.width = null;
        initCanvas();
        const layerCount = fg.prerenderOffsets(canvasWidth!, [initialH, initialI], omittedStacks, foundCoords, shouldSwapDiff);
        initCanvasVertical(layerCount, !reverse, scrollParent);
        render({ pattern: searchPattern });
    });
    window.addEventListener('resize', onResize);

    return () => {
        window.removeEventListener('resize', onResize);
    };

    function renderHighlight(title: string, newColor: string | null, left: number, top: number, width: number) {
        hl.firstChild!.textContent = title;
        //@ts-ignore allowing to use null for reset
        hl.style.backgroundColor = newColor;
        hl.style.transform = `translate(${left}px, ${top}px)`;
        hl.style.width = width + 'px';
        hl.style.display = 'block';
        canvas.style.cursor = 'pointer';
    }
};
function getScrollHeight(scrollableElement: HTMLElement) {
    return Math.max(
        scrollableElement.scrollHeight,
        scrollableElement.offsetHeight,
        scrollableElement.clientHeight,
    );
}
