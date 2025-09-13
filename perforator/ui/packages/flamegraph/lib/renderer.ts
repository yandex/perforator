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

import type { FormatNode, ProfileData } from './models/Profile';

import { hugenum } from './flame-utils';
import { pct } from './pct';
import type { GetStateFromQuery, SetStateFromQuery } from './query-utils';
import { parseStacks, stringifyStacks } from './query-utils';
import { shorten } from './shorten/shorten';
import { getCanvasTitleFull, getStatusTitleFull, renderTitleFull } from './title';
import { darken, DARKEN_FACTOR, diffcolor } from './colors';
import { getNodeTitleFull } from './node-title';
import { escapeRegex, search as outerSearch } from './search';


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
    | 'omittedIndexes'
    | 'keepOnlyFound'
    | 'exactMatch'
    | 'frameDepth'
    | 'tab'
    | 'topQuery'
    | 'flameBase'
    | 'framePos';
export type RenderFlamegraphOptions = {
    getState: GetStateFromQuery<QueryKeys>;
    setState: SetStateFromQuery<QueryKeys>;
    theme: 'dark' | 'light';
    isDiff: boolean;
    shortenFrameTexts: 'true' | 'false' | 'hover';
    onFinishRendering?: (opts?: {textNodesCount: number, delta: number, exceededLimit: boolean}) => void;
    searchPattern: RegExp | string | null;
    reverse: boolean;
    keepOnlyFound: boolean;
}

function makeByH(coords: Coordinate[]): Record<H, Set<I>> {
    const res: Record<H, Set<I>> = {};
    for (const [h, i] of coords) {
        if (!(res[h] instanceof Set)) {
            res[h] = new Set();
        }
        res[h].add(i);
    }
    return res;
}

interface RenderOpts {
    pattern?: RegExp | string | null;
}


type RenderFlamegraphType = (
    flamegraphContainer: HTMLDivElement,
    profileData: ProfileData,
    fg: FlamegraphOffseter,
    options: RenderFlamegraphOptions,
) => () => void;

/**
 * `Record<H, I[]>`
 * keeps rendering borders for each level
 * uses only pair [left, right]
 * everything up the subtree is filled before rendering
 * everything below the subtree is filled during render
 */
export type FramesWindow = Record<number, Interval>;

export class FlamegraphOffseter {
    currentNodeCoords: Coordinate = [0, 0];
    rows: FormatNode[][];


    private framesWindow: FramesWindow;
    private canvasWidth: number | undefined;
    private widthRatio: number | undefined;
    private minVisibleEv: number | undefined;
    private reverse: boolean;
    private levelHeight: number;
    private shouldReverseDiff: boolean = false;

    constructor(rows: ProfileData['rows'], options: { reverse: boolean; levelHeight: number }) {
        this.rows = rows;
        this.reverse = options.reverse;
        this.levelHeight = options.levelHeight;
    }
    fillFramesWindow([hmax, imax]: Coordinate): FramesWindow {
            const res: Record<number, Interval> = [];
            let nextParentIndex = imax;

            for (let h = Math.min(hmax, this.rows.length - 1); h >= 0; h--) {
                const row = this.rows[h];
                res[h] = [nextParentIndex, nextParentIndex];
                // will be assigned -1 on the last iteration (root)
                // we do not care about it because it will not be assigned anywhere else
                nextParentIndex = row[nextParentIndex].parentIndex;
            }

            return res;
        }
    createOffsetKeeper(h: number) {
        let prevParentIndex: number | null = null;
        let currentOffset = 0;
        const row = this.rows[h];

        return (i: number, bigFrame: boolean) => {
            const node = row[i];

            // can ignore when we know parents
            // node.parentIndex === null means root
            if (node.parentIndex !== prevParentIndex && node.parentIndex !== -1) {
                const parent = this.rows[h - 1][node.parentIndex];
                prevParentIndex = node.parentIndex;
                currentOffset = parent.x!;
            }
            node.x = currentOffset;
            if (bigFrame) {
                const width = this.countWidth(node);
                currentOffset += width;
            }
        };
    }

    calcTopOffset(h: number) {
        return this.reverse ? h * this.levelHeight : (this.rows.length * this.levelHeight) - (h + 1) * this.levelHeight;
    }

    createShouldDrawFrame(h: number) {
        const currentLevelFramesWindow = this.framesWindow[h];
        const parentFramesWindow = this.framesWindow[h - 1];

        return (i: number) => {
            const node = this.rows[h][i];

            if (currentLevelFramesWindow && !(currentLevelFramesWindow[0] <= (i) && currentLevelFramesWindow[1] >= i)) {
                return false;
            }

            // parentFramesWindow always undefined for root so null checks can be ignored
            if (
                parentFramesWindow && node.parentIndex !== -1 &&
                !(parentFramesWindow[0] <= (node.parentIndex!) && parentFramesWindow[1] >= node.parentIndex!)
            ) {
                return false;
            }
            return true;
        };
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
    backpropagateKeepOnlyFound(keepCoordinates: Coordinate[]) {
        for (let h = 0; h < this.rows.length; h++) {
            for (let i = 0; i < this.rows[h].length; i++) {
                const node = this.rows[h][i];
                node.omittedEventCount = (this.getEvents(node));
                node.omittedSampleCount = (this.getSamples(node));
            }
        }

        let maxH = 0;
        for(let i = 0; i < keepCoordinates.length; i++){
            const h = keepCoordinates[i][0];
            if(h > maxH){
                maxH = h;
            }
        }

        const keptCoordinatesByHs = makeByH(keepCoordinates);

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
        for (let h = 0; h < this.rows.length; h++) {
            const row = this.rows[h];
            for (let i = 0; i < row.length; i++) {
                const node = row[i];
                if(node.omittedEventCount) {
                    node.omittedEventCount = undefined;
                }
                if(node.omittedSampleCount) {
                    node.omittedSampleCount = undefined;
                }
                if(node.omittedNode) {
                    node.omittedNode = false;
                }
            }
        }
    }

    prerenderOffsets(
        canvasWidth: number,
        initialCoordinates: Coordinate,
        omittedOffsetCoordinates: Coordinate[] = [],
        keepFoundCoordinates: Coordinate[] | null = null,
        shouldReverseDiff: boolean = false,
        visitors: Array<{run: (node: FormatNode) => void}> = []
    ) {
        this.clearOmittedEventCount();
        this.shouldReverseDiff = shouldReverseDiff;
        this.canvasWidth = canvasWidth;
        this.currentNodeCoords = initialCoordinates;
        const [initialH, initialI] = initialCoordinates;
        this.framesWindow = this.fillFramesWindow(initialCoordinates);
        if (keepFoundCoordinates) {
            this.backpropagateKeepOnlyFound(keepFoundCoordinates);
        }
        this.backpropagateOmittedEventCount(omittedOffsetCoordinates);
        const root = this.rows[initialH][initialI];
        this.widthRatio = (this.getEvents(root) - (root.omittedEventCount ?? 0)) / canvasWidth!;
        this.minVisibleEv = minVisibleWidth * this.widthRatio;

        for (let h = 0; h < this.rows.length; h++) {
            const shouldDrawFrame = this.createShouldDrawFrame(h);
            const updateOffsets = this.createOffsetKeeper(h);
            const updateFrameWindows = this.createUpdateWindow(h);
            for (let i = 0; i < this.rows[h].length; i++) {
                if (!shouldDrawFrame(i)) {
                    continue;
                }
                updateFrameWindows(i);
                for (let vi = 0; vi < visitors.length; vi++) {
                    const visitor = visitors[vi];
                    visitor.run(this.rows[h][i]);
                }
                const isVisible = this.visibleNode(this.rows[h][i]);
                updateOffsets(i, isVisible);

            }

            if (!this.framesWindow?.[h]) {
                break;
            }
        }
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
        while(!this.visibleNode(frames[left]) && left < frames.length){
            ++left
        }
        if (left >= 0 && left < frames.length && frames[left].x && (frames[left].x! - x) < 0.5) { return left; }
        while(!this.visibleNode(frames[right]) && right > 0){
            --right
        }
        if (right >= 0 && right < frames.length && frames[right].x && (x - (frames[right].x! + this.countWidth(frames[right]))) < 0.5) { return right; }

        return null;
    }
    countEventCountWidth(node: FormatNode) {
        return (this.getEvents(node))- (node.omittedEventCount ?? 0);
    }
    countSampleCountWidth(node: FormatNode) {
        return (this.getSamples(node)) - (node.omittedSampleCount ?? 0);
    }
    getCoordsByPositionWithKnownHeight(h: number, x: number) {

        const row = this.rows[h];

        if (!this.framesWindow[h]) {
            return null;
        }
        const [leftIndex, rightIndex] = this.framesWindow[h];

        const i = this.findFrame(row, x, leftIndex, rightIndex);


        if (i === null) {
            return null;
        }

        return { h, i };
    }


    getTopOffset(offset: number) {
        return this.reverse ? offset : ((this.rows.length * this.levelHeight) - offset);
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

    getEvents(node?: FormatNode) {
        const fieldName = this.shouldReverseDiff ? 'baseEventCount' : 'eventCount' as const;
        return node?.[fieldName] ?? 0;
    }

    getSamples(node: FormatNode) {
        const fieldName = this.shouldReverseDiff ? 'baseSampleCount' : 'sampleCount' as const;
        return node?.[fieldName] ?? 0;
    }
    visibleNode(node?: FormatNode) {
        return (this.getEvents(node)) - (node?.omittedEventCount ?? 0) >= this.minVisibleEv!;
    }

    keepRendering(h: number) {
        if (Array.isArray(this.framesWindow?.[h])) {
            return true;
        }
        return false;
    }

    isBeforeCurrentNode(h: number) {
        return h < this.currentNodeCoords[0];
    }
    private createUpdateWindow = (h: number) => (i: number) => {
        if (Array.isArray(this.framesWindow?.[h])) {
            this.framesWindow[h][1] = i;
        } else {
            this.framesWindow[h] = [i, i];
        }
    };

}

export const renderFlamegraph: RenderFlamegraphType = (
    flamegraphContainer,
    profileData,
    fg,
    { getState, setState, theme, isDiff, onFinishRendering, shortenFrameTexts, searchPattern, reverse, keepOnlyFound },
) => {
    const shouldSwapDiff = getState('flameBase') === 'diff'

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
    const MAX_TEXT_LABELS = 500;


    const canvas = findElement('canvas') as HTMLCanvasElement;
    const c = canvas.getContext('2d')!;

    const hl = findElement('highlight');
    const labels = findElement('labels-container');
    const labelTemplate = findElement('label-template') as HTMLTemplateElement;
    const status = findElement('status');
    const annotations = findElement('annotations');
    const content = findElement('content');
    hl.style.height = String(BLOCK_HEIGHT);

    let canvasWidth: number | undefined;
    let canvasHeight: number | undefined;

    function initCanvas() {
        canvas.style.height = profileData.rows.length * LEVEL_HEIGHT + 'px';
        canvasWidth = canvas.offsetWidth;
        canvasHeight = canvas.offsetHeight;
        canvas.style.width = canvasWidth + 'px';
        canvas.width = canvasWidth * (devicePixelRatio || 1);
        canvas.height = canvasHeight * (devicePixelRatio || 1);
        if (devicePixelRatio) { c.scale(devicePixelRatio, devicePixelRatio); }
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
    const search = outerSearch.bind(null, readString, shorten, shouldShortenTextForOverview, profileData.rows);

    const maybeSearch = (query: RegExp | string | null): Coordinate[] | null => {
        let foundCoords: Coordinate[] | null;
        if (keepOnlyFound && query) {
            foundCoords = search(query);
        } else {
            foundCoords = null;
        }
        return foundCoords;
    };

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


        if (reverse) {
            annotations.after(content);
        } else {
            annotations.before(content);
        }

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

        for (let h = 0; h < rows.length; h++) {
            const y = fg.calcTopOffset(h);
            const row = rows[h];
            const alpha = fg.isBeforeCurrentNode(h);


            const drawFrame = function(i: number) {
                const node = row[i];
                const width = fg.countWidth(node);
                const nodeTitle = getNodeTitle(node);

                const pattern = opts?.pattern;
                const isMarked = typeof pattern === 'string' ? (new RegExp(escapeRegex(pattern))).test(nodeTitle) : pattern?.test(nodeTitle);
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
                    if(newLabels.length < MAX_TEXT_LABELS){

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

            const shouldDrawFrame = fg.createShouldDrawFrame(h);

            const renderNode = function(i: number) {
                const node = row[i];
                const should = shouldDrawFrame(i);
                if (!should) {
                    return;
                }

                const isVisible = fg.visibleNode(node);


                if (!isVisible) {
                    return;
                }

                drawFrame(i);

            };

            for (let i = 0; i < row.length; i++) {
                renderNode(i);
            }

            if (!fg.keepRendering(h)) {
                break;
            }
        }

        if(labelCount > MAX_TEXT_LABELS) {
            console.log(`label count limit is ${MAX_TEXT_LABELS}, without it would have shown ${labelCount} labels`)
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

        return {textNodesCount: labelCount}

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
            onFinishRendering?.({textNodesCount: res.textNodesCount, delta, exceededLimit: res.textNodesCount > MAX_TEXT_LABELS})
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
        let initialCoord: Coordinate | undefined;
        if (e.altKey && !fg.isBeforeCurrentNode(h)) {
            if (!omitted.includes([h, i])) {
                omitted.push([h, i]);
            }
            setState({ omittedIndexes: stringifyStacks(omitted) });
            initialCoord = fg.currentNodeCoords;
        } else {
            setState({
                frameDepth: h.toString(),
                framePos: i.toString(),
            });
            initialCoord = [h, i];
        }

        const foundCoords = maybeSearch(searchPattern);
        fg.prerenderOffsets(canvasWidth!, initialCoord, omitted, foundCoords, shouldSwapDiff);
        render({ pattern: searchPattern });
        canvas?.onmousemove?.(e);
    };

    function calcHighlightColor(node: FormatNode) {
        const parsedColor = isDiff ? calculateDiffColor(node, root) : node.color as string;

        let color: string | null = null
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
        const row = rows[h];
        const node = row[i];
        const currentNodeCoords = fg.currentNodeCoords;
        const currentNode = rows[currentNodeCoords[0]][currentNodeCoords[1]];

        if (!fg.visibleNode(node)) {
            canvas.onmouseout?.(event);
            return;
        }

        const width = fg.countWidth(node);

        const left = node.x! + canvas.offsetLeft;
        const top = (fg.calcTopOffset(h) + canvas.offsetTop);
        const title = getNodeTitleHl(node);
        const isMainRoot = currentNode && currentNode.textId === root.textId && currentNode.eventCount === root.eventCount;
        const color = calcHighlightColor(node);
        renderHighlight(title, color, left, top, width);

        status.textContent = 'Function: ' + (isMainRoot ? getStatusTitle(node, null, root) : getStatusTitle(node, currentNode!, root));
        return;


    };


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

    const foundCoords = maybeSearch(searchPattern);

    fg.prerenderOffsets(canvasWidth!, [h, pos], omittedStacks, foundCoords, shouldSwapDiff);
    render({ pattern: searchPattern });

    const onResize = () => requestAnimationFrame(() => {

        const initialH = parseInt(getState('frameDepth', '0'));
        const initialI = parseInt(getState('framePos', '0'));
        //@ts-ignore
        canvas.style.width = null;
        initCanvas();
        fg.prerenderOffsets(canvasWidth!, [initialH, initialI], omittedStacks, foundCoords, shouldSwapDiff);
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
        hl.style.transform = `translate(${left}px, ${top}px)`
        hl.style.width = width + 'px';
        hl.style.display = 'block';
        canvas.style.cursor = 'pointer';
    }
};

