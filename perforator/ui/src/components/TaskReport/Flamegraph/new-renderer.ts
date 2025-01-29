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
import type { RealTheme } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory/index.ts';
import type { NewFormatNode, NewProfileData } from 'src/models/Profile.ts';
import type { UserSettings } from 'src/providers/UserSettingsProvider/UserSettings.ts';

import { hugenum } from './flame-utils.ts';
import type { GetStateFromQuery } from './query-utils.ts';
import { shorten } from './shorten/shorten.ts';
import { darken, DARKEN_FACTOR, diffcolor } from './utils/colors.ts';


const dw = Math.floor(255 * (1 - DARKEN_FACTOR)).toString(16);
// for dark theme
const WHITE_TEXT_COLOR = `#${dw}${dw}${dw}`;


function pct(a: number, b: number) {
    return a >= b ? '100' : (100 * a / b).toFixed(2);
}

function formatPct(value: any) {
    return value ? `${value}%` : '';
}


export type RenderFlamegraphOptions = {
    getState: GetStateFromQuery;
    setState: (state: Record<string, string | false>) => void;
    theme: RealTheme;
    isDiff: boolean;
    userSettings: UserSettings;
    searchPattern: RegExp | null;
    reverse: boolean;
}

interface RenderOpts {
    subtree?: {initialH?: number; initialI?: number};
    pattern?: RegExp | null;
}


type RenderFlamegraphType = (
    flamegraphContainer: HTMLDivElement,
    profileData: NewProfileData,
    options: RenderFlamegraphOptions,
) => () => void;

export const renderFlamegraph: RenderFlamegraphType = (
    flamegraphContainer,
    profileData,
    { getState, setState, theme, isDiff, userSettings, searchPattern, reverse },
) => {

    function findElement(name: string): HTMLElement {
        return flamegraphContainer.querySelector(`.flamegraph__${name}`)!;
    }


    function getCssVariable(variable: string) {
        return getComputedStyle(flamegraphContainer).getPropertyValue(variable);
    }


    function maybeShorten(str: string) {
        return userSettings.shortenFrameTexts === 'true' || userSettings.shortenFrameTexts === 'hover' ? shorten(str) : str;
    }
    function shortenTitle(title: string) {
        return userSettings.shortenFrameTexts === 'true' ? shorten(title) : title;
    }


    const BACKGROUND = getCssVariable('--g-color-base-background');
    const SEARCH_COLOR = theme === 'dark' ? darken('#ee00ee') : '#ee00ee';


    function calculateDiffColor(node: NewFormatNode, root: NewFormatNode) {
        const color = diffcolor(node, root);
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
    const labelTemplate = findElement('label-template');
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
        if (id === undefined) {return '';}
        return profileData.stringTable[id];
    }

    function getNodeTitle(node: NewFormatNode): string {
        const kind = readString(node.kind);
        let nodeTitle = maybeShorten(readString(node.textId)) + ' ' + readString(node.file);
        if (kind !== '') {
            nodeTitle += ` (${kind})`;
        }
        if (node.inlined) {
            nodeTitle += ' (inlined)';
        }
        return nodeTitle;
    }

    function drawLabel(text: string, x: number, y: number, w: number, opacity: string, color: string) {
        const node = labelTemplate.firstChild!.cloneNode(true) as HTMLDivElement;
        node.firstChild!.textContent = text;
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

    const minVisibleWidth = 1e-2;
    const rows = profileData.rows;
    const root = rows[0][0];
    const widthEv = root.eventCount;
    let widthRatio = widthEv / canvasWidth!;
    let minVisibleEv = minVisibleWidth * widthRatio;

    function countWidth(node: NewFormatNode) {
        return Math.min(node.eventCount / widthRatio, canvasWidth!);
    }
    function visible(eventCount: number) {
        return eventCount >= minVisibleEv;
    }


    function findFrame(frames: NewFormatNode[], x: number, left = 0, right = frames.length - 1) {
        if (x < frames[left].x! || x > (frames[right].x! + countWidth(frames[right]))) {
            return null;
        }

        while (left <= right) {
            // eslint-disable-next-line no-bitwise
            const mid = (left + right) >>> 1;

            if (frames[mid].x! > x) {
                right = mid - 1;
            } else if (frames[mid].x! + frames[mid].eventCount / widthRatio <= x) {
                left = mid + 1;
            } else {
                return mid;
            }
        }

        if (left >= 0 && left < frames.length && frames[left].x && (frames[left].x! - x) < 0.5) { return left; }
        if (right >= 0 && right < frames.length && frames[right].x && (x - (frames[right].x! + frames[right].eventCount / widthRatio)) < 0.5) { return right; }

        return null;
    }


    function renderSearch(matched: number, showReset: boolean) {
        findElement('match-value').textContent = pct(matched, canvasWidth!) + '%';
        findElement('match').style.display = showReset ? 'inherit' : 'none';
    }


    function fillFramesWindow([hmax, imax]: [number, number]): Record<number, [number, number]> {
        const res: Record<number, [number, number]> = [];
        let nextParentIndex = imax;

        for (let h = Math.min(hmax, rows.length - 1); h >= 0; h--) {
            const row = rows[h];
            res[h] = [nextParentIndex, nextParentIndex];
            // will be assigned -1 on the last iteration (root)
            // we do not care about it because it will not be assigned anywhere else
            nextParentIndex = row[nextParentIndex].parentIndex;
        }

        return res;
    }

    let currentNode: NewFormatNode | null = null;

    type TitleArgs = {
        getPct: (arg: {
            rootPct: string | undefined;
            selectedPct: string | undefined;
        }) => string;
        getNumbers: (arg: { sampleCount: string; eventCount: string; percent: string }) => string;
        wrapNumbers: (numbers: string) => string;
        getDelta: (delta: string) => string;
    };

    function renderTitle({
        getPct,
        getNumbers,
        getDelta,
        wrapNumbers = (nubmers: string) => nubmers,
    }: TitleArgs) {
        // eslint-disable-next-line @typescript-eslint/no-shadow
        return function (f: NewFormatNode, selectedFrame: NewFormatNode | null, root?: NewFormatNode): string {
            const calcPercent = (baseFrame?: NewFormatNode | null) => baseFrame ? pct(f.eventCount, baseFrame.eventCount) : undefined;
            const percent = getPct({
                rootPct: calcPercent(root),
                selectedPct: calcPercent(selectedFrame),
            });
            const shortenedTitle = getNodeTitle(f);
            const numbers = getNumbers({
                sampleCount: hugenum(f.sampleCount),
                eventCount: hugenum(f.eventCount),
                percent,
            });

            let diffString = '';


            if (isDiff) {
                let delta = 0;
                const anyFrame = (selectedFrame || root) as NewFormatNode;
                if (anyFrame.baseEventCount && f.baseEventCount && anyFrame.baseEventCount > 1e-3) {
                    delta =
                        f.eventCount / anyFrame.eventCount -
                        f.baseEventCount / anyFrame.baseEventCount;
                } else {
                    delta = f.eventCount / anyFrame.eventCount;
                }
                const deltaString = (delta >= 0.0 ? '+' : '') + (delta * 100).toFixed(2) + '%';
                diffString += getDelta(deltaString);
            }

            return shortenedTitle + wrapNumbers(numbers + diffString);
        };
    }

    const getStatusTitle = renderTitle({
        getPct: ({ rootPct, selectedPct }) => (
            [rootPct, selectedPct].filter(Boolean).map(formatPct).join('/')
        ),
        getNumbers: ({ sampleCount, eventCount, percent }) => `${eventCount} cycles, ${sampleCount} samples, ${percent}`,
        wrapNumbers: numbers => ` (${numbers})`,
        getDelta: delta => `, ${delta}`,

    });

    const getCanvasTitle = renderTitle({
        getPct: ({ rootPct, selectedPct }) => (
            (rootPct ? `Percentage of root frame: ${formatPct(rootPct)}\n` : '')
            + (selectedPct ? `Percentage of selected frame: ${formatPct(selectedPct)}\n` : '')
        ),
        getNumbers: ({ sampleCount, eventCount, percent }) => `\nSamples: ${sampleCount}\nCycles: ${eventCount}\n${percent}`,
        wrapNumbers: numbers => numbers.trimEnd(),
        getDelta: delta => `Diff: ${delta}\n`,
    });


    /**
     * `Record<H, I[]>`
     * keeps rendering borders for each level
     * uses only pair [left, right]
     * everything up the subtree is filled before rendering
     * everything below the subtree is filled during render
     */
    let framesWindow: Record<number, [number, number]> = fillFramesWindow([0, 0]);
    function renderImpl(opts?: RenderOpts) {
        if (opts?.subtree) {
            c.fillStyle = BACKGROUND;
            c.fillRect(0, 0, canvasWidth!, canvasHeight!);
        }
        clearLabels();

        if (reverse) {
            annotations.after(content);
        } else {
            annotations.before(content);
        }

        const newLabels: HTMLDivElement[] = [];

        const { initialH = 0, initialI = 0 } = opts?.subtree ?? {};
        const maxEventCount = rows[initialH][initialI].eventCount;
        widthRatio = maxEventCount / canvasWidth!;
        minVisibleEv = minVisibleWidth * widthRatio;
        currentNode = rows[initialH][initialI];

        const marked: Record<number | string, number> = {};

        function mark(f: NewFormatNode) {
            const width = countWidth(f);
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


        framesWindow = fillFramesWindow([initialH, initialI]);

        function createOffsetKeeper(h: number) {
            let prevParentIndex: number | null = null;
            let currentOffset = 0;
            const row = rows[h];

            return function (i: number, bigFrame: boolean) {
                const node = row[i];

                // can ignore when we know parents
                // node.parentIndex === null means root
                if (node.parentIndex !== prevParentIndex && node.parentIndex !== -1) {
                    const parent = rows[h - 1][node.parentIndex];
                    prevParentIndex = node.parentIndex;
                    currentOffset = parent.x!;
                }
                node.x = currentOffset;
                if (bigFrame) {
                    const width = countWidth(node);
                    currentOffset += width;
                }
            };
        }

        function createShouldDrawFrame(h: number) {
            const currentLevelFramesWindow = framesWindow[h];
            const parentFramesWindow = framesWindow[h - 1];

            return function (i: number) {
                const node = rows[h][i];

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

        const createUpdateWindow = (h: number) => (i: number) => {
            if (Array.isArray(framesWindow?.[h])) {
                framesWindow[h][1] = i;
            } else {
                framesWindow[h] = [i, i];
            }
        };

        for (let h = 0; h < rows.length; h++) {
            const y = reverse ? h * LEVEL_HEIGHT : canvasHeight! - (h + 1) * LEVEL_HEIGHT;
            const row = rows[h];
            const alpha = h < (initialH ?? 0);


            const drawFrame = function (i: number) {
                const node = row[i];
                const width = countWidth(node);
                const nodeTitle = getNodeTitle(node);

                const isMarked = opts?.pattern?.test(nodeTitle);
                if (isMarked) {
                    mark(node);
                }
                const color = isMarked ?
                    SEARCH_COLOR :
                    isDiff ? calculateDiffColor(node, root) : node.color!;

                c.fillStyle = color as string;
                c.fillRect(node.x!, y, width, BLOCK_HEIGHT);

                if (width > charWidth * 3 + 6 && newLabels.length < MAX_TEXT_LABELS) {

                    const chars = Math.floor((width - 6) / charWidth);
                    const title = nodeTitle.length <= chars ? nodeTitle : nodeTitle.substring(0, chars - 1) + '…';
                    let labelColor: string | undefined;

                    if (alpha && theme === 'dark') {
                        labelColor = WHITE_TEXT_COLOR;
                    }
                    const label = drawLabel(title, node.x!, y, width, alpha ? '0.5' : '1', labelColor!);
                    newLabels.push(label);
                }


                if (alpha) {
                    c.fillStyle = theme === 'dark' ? '#0000007F' : '#FFFFFF7F';
                    c.fillRect(node.x!, y, width, BLOCK_HEIGHT);
                }
            };

            const shouldDrawFrame = createShouldDrawFrame(h);
            const updateOffsets = createOffsetKeeper(h);
            const updateFrameWindows = createUpdateWindow(h);

            const renderNode = function (i: number) {
                const node = row[i];
                if (!shouldDrawFrame(i)) {
                    return;
                }
                updateFrameWindows(i);
                const isVisible = visible(node.eventCount);
                updateOffsets(i, isVisible);

                if (!isVisible) {
                    return;
                }

                drawFrame(i);

            };

            for (let i = 0; i < row.length; i++) {
                renderNode(i);
            }

            if (!framesWindow?.[h]) {
                break;
            }
        }
        labels?.replaceChildren(...newLabels);

        renderSearch(totalMarked(), Boolean(opts?.pattern));


    }

    let firstRender = true;
    function render(opts: RenderOpts) {
        const start = performance.now();
        const res = renderImpl(opts);
        if (firstRender) {
            uiFactory().rum()?.finishDataRendering?.('task-flamegraph');
            firstRender = false;
        }
        const finish = performance.now();
        console.log('Rendered flamegraph in', finish - start, 'ms');
        return res;
    }


    function getTopOffset(offset: number) {
        return reverse ? offset : (canvasHeight! - offset);
    }

    function getCoordsByPosition(x: number, y: number): null | {h: number; i: number} {
        const topOffset = getTopOffset(y);
        const h = Math.floor(topOffset / LEVEL_HEIGHT);
        if (h < 0 || h >= rows.length) {
            return null;
        }
        const row = rows[h];

        if (!framesWindow[h]) {
            return null;
        }
        const [leftIndex, rightIndex] = framesWindow[h];

        const i = findFrame(row, x, leftIndex, rightIndex);
        if (i === null) {
            return null;
        }

        return { h, i };
    }


    const handleClick = (e: MouseEvent): void => {
        const coords = getCoordsByPosition(e.offsetX, e.offsetY);
        if (!coords) { return; }

        const { i, h } = coords;
        if (!visible(rows[h][i].eventCount)) {
            canvas.onmouseout?.(e);
            return;
        }
        if (typeof i !== 'number') { return; }
        setState({
            frameDepth: h.toString(),
            framePos: i.toString(),
        });
        render({ subtree: { initialH: h, initialI: i }, pattern: searchPattern });
        canvas?.onmousemove?.(e);
    };

    canvas.onmousemove = function (event) {
        const coords = getCoordsByPosition(event.offsetX, event.offsetY);

        if (!coords) {
            canvas.onmouseout?.(event);
            return;
        }
        const { i, h } = coords;
        const row = rows[h];
        const node = row[i];


        if (!visible(node.eventCount)) {
            canvas.onmouseout?.(event);
            return;
        }
        const width = countWidth(node);

        const left = node.x! + canvas.offsetLeft;
        const top = ((reverse ? h * LEVEL_HEIGHT : canvasHeight! - (h + 1) * LEVEL_HEIGHT) + canvas.offsetTop);
        const title = getNodeTitle(node);
        const isMainRoot = currentNode && currentNode.textId === root.textId && currentNode.eventCount === root.eventCount;
        const highlightTitle = isMainRoot ? getCanvasTitle(node, null, root) : getCanvasTitle(node, currentNode!, root);
        const parsedColor = isDiff ? diffcolor(node, root) : node.color!;

        let newColor: string | null = null;
        if (theme === 'dark') {
            newColor = darken(parsedColor as string, 0.2);
        }
        // currently we calculate diff color on the fly during render
        // highlight is 0.4 darker than default color
        // but for non-diffs the node.color is already darkened by 0.2 so 0.2 is enough
        if (theme === 'dark' && isDiff) {
            newColor = darken(newColor as string, 0.2);
        }
        renderHighlight(title, newColor, left, top, width, highlightTitle);

        canvas.onclick = handleClick;
        status.textContent = 'Function: ' + (isMainRoot ? getStatusTitle(node, null, root) : getStatusTitle(node, currentNode!, root));
        return;


    };


    function clearHighlight() {
        hl.style.display = 'none';
        status.textContent = 'Function: ' + getStatusTitle(currentNode!, null, root);
        canvas.title = '';
        canvas.style.cursor = '';
    }

    canvas.onmouseout = clearHighlight;

    // read query and display h and pos
    const h = Number(getState('frameDepth', '0'));
    const pos = Number(getState('framePos', '0'));

    render({ pattern: searchPattern, subtree: { initialH: h, initialI: pos } });

    const onResize = () => requestAnimationFrame(() => {
        //@ts-ignore
        canvas.style.width = null;
        initCanvas();
        render({ pattern: searchPattern });
    });
    window.addEventListener('resize', onResize);

    return () => {
        window.removeEventListener('resize', onResize);
    };

    function renderHighlight(title: string, newColor: string | null, left: number, top: number, width: number, highlightTitle: string) {
        hl.firstChild!.textContent = shortenTitle(title);
        //@ts-ignore allowing to use null for reset
        hl.style.backgroundColor = newColor;
        hl.style.left = left + 'px';
        hl.style.top = top + 'px';
        hl.style.width = width + 'px';
        hl.style.display = 'block';
        canvas.title = highlightTitle;
        canvas.style.cursor = 'pointer';
    }
};
