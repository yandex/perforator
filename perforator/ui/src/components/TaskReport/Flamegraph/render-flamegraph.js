/* eslint-disable prefer-arrow-callback */
/* eslint-disable no-implicit-coercion */
/* eslint-disable no-console */
/* eslint-disable no-return-assign */
/* eslint-disable @typescript-eslint/no-shadow */
/* eslint-disable no-shadow */
/* eslint-disable semi */
/* eslint-disable no-bitwise */
/* eslint-disable no-unused-vars */
/* eslint-disable prefer-const */
/* eslint-disable curly */
/* eslint-disable quotes */
/* eslint-disable no-var */
/* eslint-disable indent */
/* eslint-disable valid-jsdoc */

import {
    convertHslToRgb,
    convertOklabToRgb,
    convertRgbToOklab,
    parseHex,
    serializeHex,
    serializeRgb,
} from 'culori/fn';

import { uiFactory } from 'src/factory'

import { hugenum } from './flame-utils'
import { shorten } from './shorten'


const DARKEN_FACTOR = 0.2
const dw = Math.floor(255 * (1 - DARKEN_FACTOR)).toString(16)
// for dark theme
const WHITE_TEXT_COLOR = `#${dw}${dw}${dw}`

/**
 * @param {string} color hex
 * @returns {string} newColor
 */
function darken(color, factor = DARKEN_FACTOR) {
    const { l, ...parsedColor } = convertRgbToOklab(parseHex(color));

    const resultingColor = {
        ...parsedColor,
        l: Math.max(l - factor, 0),
    };

    return serializeRgb(convertOklabToRgb(resultingColor));
}


/**
 * @param {HTMLDivElement} flamegraphContainer
 * @param {import('../../../models/Profile').ProfileData} profileData
 * @param {object} params
 */
export function renderFlamegraph(flamegraphContainer, profileData, { getState, setState, theme, isDiff, userSettings, searchPattern, reverse }) {
            /**
             * @param {string} name
             */
            function findElement(name) {
                return flamegraphContainer.querySelector(`.flamegraph__${name}`);
            }

            function maybeShorten(str) {
                return userSettings.shortenFrameTexts === 'true' || userSettings.shortenFrameTexts === 'hover' ? shorten(str) : str;
            }
            function shortenTitle(title) {
                return userSettings.shortenFrameTexts === 'true' ? shorten(title) : title;
            }

            function getCssVariable(variable) {
                return getComputedStyle(flamegraphContainer).getPropertyValue(variable);
            }

            const BACKGROUND = getCssVariable('--g-color-base-background');

            const LEVEL_HEIGHT = parseInt(getCssVariable('--flamegraph-level-height'));
            const BLOCK_SPACING = parseInt(getCssVariable('--flamegraph-block-spacing'));
            const BLOCK_HEIGHT = LEVEL_HEIGHT - BLOCK_SPACING;
            const MAX_TEXT_LABELS = 500;

            const levels = profileData.levels;
            const strtab = profileData.stringTable;
            const frameLevels = profileData.levels.length;

            let firstRender = true;

            var px;
            var mainRoot = { level: 0, index: 0 };
            var root = mainRoot;
            var diff = isDiff;

            /** @type HTMLCanvasElement */
            const canvas = findElement('canvas');
            const c = canvas.getContext('2d');
            /** @type HTMLDivElement */
            const hl = findElement('highlight');
            const labels = findElement('labels-container');
            const labelTemplate = findElement('label-template');
            const status = findElement('status');
            const annotations = findElement('annotations');
            const content = findElement('content');
            hl.style.height = BLOCK_HEIGHT;

            let canvasWidth;
            let canvasHeight;

            function initCanvas() {
                canvas.style.height = frameLevels * LEVEL_HEIGHT + 'px';
                canvasWidth = canvas.offsetWidth;
                canvasHeight = canvas.offsetHeight;
                canvas.style.width = canvasWidth + 'px';
                canvas.width = canvasWidth * (devicePixelRatio || 1);
                canvas.height = canvasHeight * (devicePixelRatio || 1);
                if (devicePixelRatio) c.scale(devicePixelRatio, devicePixelRatio);
            }

            initCanvas();

            c.font = window.getComputedStyle(canvas, null).getPropertyValue('font');
            var textMetrics = c.measureText('O');
            var charWidth = textMetrics.width || 6;
            var charHeight = (textMetrics.actualBoundingBoxAscent + textMetrics.actualBoundingBoxDescent) || 12;
            var diffmult = 1.0;

            function drawLabel(text, x, y, w, opacity, color) {
                let node = labelTemplate.firstChild.cloneNode(true);
                node.firstChild.textContent = text;
                node.style.top = y + canvas.offsetTop + "px";
                node.style.left = x + canvas.offsetLeft + "px";
                node.style.width = w + "px";
                node.style.opacity = opacity;
                if (color) {
                    node.style.color = color;
                }
                return node;
            }

            function clearLabels() {
                labels.replaceChildren();
            }

            function getColor(p) {
                const v = Math.random();
                return '#' + (p[0] + ((p[1] * v) << 16 | (p[2] * v) << 8 | (p[3] * v))).toString(16);
            }

            function pct(a, b) {
                return a >= b ? '100' : (100 * a / b).toFixed(2);
            }

            function formatPct(value) {
                return value ? `${value}%` : '';
            }

            function renderTitle({
                getPct,
                getNumbers,
                getDelta,
                wrapNumbers = nubmers => nubmers,
            }) {
                return function (f, selectedFrame, root) {
                    const calcPercent = baseFrame => baseFrame ? pct(f.eventCount, baseFrame.eventCount) : undefined;
                    const percent = getPct({
                        rootPct: calcPercent(root),
                        selectedPct: calcPercent(selectedFrame),
                    });
                    const shortenedTitle = shortenTitle(f.title);
                    let numbers = getNumbers({
                        sampleCount: hugenum(f.sampleCount),
                        eventCount: hugenum(f.eventCount),
                        percent,
                    });

                    let diffString = '';
                    if (diff) {
                        let delta = 0;
                        const anyFrame = selectedFrame || root;
                        if (anyFrame.baseEventCount > 1e-3) {
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
                    formatPct(rootPct)
                    + (root && selectedPct ? '/' : '')
                    + formatPct(selectedPct)
                ),
                getDelta: delta => `, ${delta}`,
                getNumbers: ({ sampleCount, eventCount, percent }) => `${eventCount} cycles, ${sampleCount} samples, ${percent}`,
                wrapNumbers: numbers => ` (${numbers})`,
            })

            const getCanvasTitle = renderTitle({
                getPct: ({ rootPct, selectedPct }) => (
                    (rootPct ? `Percentage of root frame: ${formatPct(rootPct)}\n` : '')
                    + (selectedPct ? `Percentage of selected frame: ${formatPct(selectedPct)}\n` : '')
                ),
                getDelta: delta => `Diff: ${delta}\n`,
                getNumbers: ({ sampleCount, eventCount, percent }) => `\nSamples: ${sampleCount}\nCycles: ${eventCount}\n${percent}`,
                wrapNumbers: numbers => numbers.trimEnd(),
            })

            /**
             * @param {{level: number, index: number}} pos
             */
            function frame(pos) {
                if (pos.level >= levels.length || pos.index >= levels[pos.level][0].length) {
                    return null;
                }

                let f = {
                    left: levels[pos.level][0][pos.index],
                    width: levels[pos.level][1][pos.index],
                    title: strtab[levels[pos.level][2][pos.index]],
                    eventCount: levels[pos.level][3][pos.index],
                    sampleCount: levels[pos.level][4][pos.index],
                    color: strtab[levels[pos.level][5][pos.index]],
                }

                if (levels[pos.level].length >= 7) {
                    f.baseEventCount = levels[pos.level][6][pos.index];
                    f.baseSampleCount = levels[pos.level][7][pos.index];
                }

                return f;
            }

            function visible(f) {
                return f.width * px > 1e-2;
            }

            function findFrame(frames, x) {
                let left = 0;
                let right = frames[0].length - 1;

                while (left <= right) {
                    const mid = (left + right) >>> 1;

                    if (frames[0][mid] > x) {
                        right = mid - 1;
                    } else if (frames[0][mid] + frames[1][mid] <= x) {
                        left = mid + 1;
                    } else {
                        return mid;
                    }
                }

                if (frames[0][left] && (frames[0][left] - x) * px < 0.5) return left;
                if (frames[0][right] && (x - (frames[0][right] + frames[1][right])) * px < 0.5) return right;

                return null;
            }


            function renderSearch(matched, r) {
                findElement("match-value").textContent =
                    pct(matched, frame(root).width) + "%";
                findElement("match").style.display = r ? "inherit" : "none";
            }

            function hsv2hsl(h, s, v) {
                const l = v - v * s / 2;
                const m = Math.min(l, 1 - l);
                return [h, m ? (v - l) / m : 0, l];
            }

            function diffcolor(node, root) {
                const lhs = node.eventCount;
                const rhs = root.baseEventCount > 1e-5
                     ? node.baseEventCount * root.eventCount / root.baseEventCount
                     : 0;

                const diff = rhs > 1e-5 ? (lhs - rhs) / rhs : 1.0;
                const d = Math.min(Math.abs(diff), 1.0);

                if (d < 1e-3) {
                    const value = Math.round(180 + 60 * (1 - d * 1e3));
                    const hex = value.toString(16);
                    return `#${hex}${hex}${hex}`;
                }

                let hoff = 0.16;
                let hpow = 4.0;
                let hcoef = -0.14;
                if (diff <= 0) {
                    hoff = 0.58;
                    hpow = 2.0;
                    hcoef = 0.10;
                }

                let soff = 0.0;
                let spow = 4.5;
                let scoef = 0.75;

                const h = hoff + Math.pow(d, 1.0 / hpow) * hcoef;
                const s = soff + Math.pow(d, 1.0 / spow) * scoef;
                const hsl = hsv2hsl(h, s, 1.0);

                // return `hsl(${hsl[0] * 360} ${hsl[1] * 100} ${hsl[2] * 100})`;
                return serializeHex(convertHslToRgb({ mode: "hsl", h: hsl[0] * 360, s: hsl[1], l: hsl[2] }))
            }

            function rectCanvasPosition(rect, root) {
                const rx0 = Math.max(rect.left - root.left, 0);
                const rx1 = Math.min(rect.left - root.left + rect.width, root.width);
                const left = rx0 * px;
                const width = (rx1 - rx0) * px;

                return [left, width];
            }

            function renderImpl(newRoot, params) {
                const pattern = params?.pattern;
                if (root) {
                    c.fillStyle = BACKGROUND;
                    c.fillRect(0, 0, canvasWidth, canvasHeight);
                }
                clearLabels();

                if (reverse) {
                    annotations.after(content);
                } else {
                    annotations.before(content);
                }

                root = newRoot || mainRoot;
                px = canvasWidth / frame(root).width;

                const rootFrame = frame(root);
                const mainRootFrame = frame(mainRoot);
                const x0 = frame(root).left;
                const x1 = x0 + frame(root).width;
                const diff = frame(mainRoot).baseEventCount !== undefined;
                if (diff) {
                    diffmult = mainRootFrame.eventCount / mainRootFrame.baseEventCount;
                }

                const marked = {};

                function mark(f) {
                    return marked[f.left] >= f.width || (marked[f.left] = f.width);
                }

                function totalMarked() {
                    let keys = Object.keys(marked)
                    keys = keys.sort(function(a, b) { return +a - +b; });
                    console.log('keys: ', keys);
                    let total = 0;
                    let left = 0;
                    for (let x of keys) {
                        console.log(x, marked[x]);
                        let right = +x + marked[x];
                        console.log(left, ' |', right, '| ', total);
                        if (right > left) {
                            total += right - Math.max(left, x);
                            left = right;
                        }
                    }
                    console.log('total: ', total);
                    return total;
                }

                const newLabels = [];

                function drawFrame(level, index, y, alpha) {
                    const f = frame({ level: level, index: index })

                    if (!visible(f)) {
                        return
                    }

                    if (f.left < x1 && f.left + f.width > x0) {
                        const [rx, rw] = rectCanvasPosition(f, frame(root));

                        const frameTitle = maybeShorten(f.title);
                        const SEARCH_COLOR = '#ee00ee';
                        const newColor =
                        pattern && frameTitle.match(pattern) && mark(f)
                        ? SEARCH_COLOR
                        : (diff ? diffcolor(f, frame(root)) : f.color);
                        const color =
                            theme === 'dark' ? darken(newColor) : newColor;
                        c.fillStyle = color;
                        c.fillRect(rx, y, rw, BLOCK_HEIGHT);

                        if (rw >= charWidth * 3 + 6 && newLabels.length < MAX_TEXT_LABELS) {
                            const chars = Math.floor((f.width * px - 6) / charWidth);
                            const title = frameTitle.length <= chars ? frameTitle : frameTitle.substring(0, chars - 1) + '…';
                            // c.fillStyle = '#000000';
                            // c.fillText(title, Math.round(rx + 3), Math.round(y + (charHeight + BLOCK_HEIGHT) / 2));
                            let color;
                            if (alpha && theme === 'dark') {
                                color = WHITE_TEXT_COLOR;
                            }
                            const label = drawLabel(title, rx, y, rw, alpha ? 0.5 : 1, color);
                            newLabels.push(label);
                        }

                        if (alpha) {
                            c.fillStyle = theme === 'dark' ? '#0000007F' : '#FFFFFF7F';
                            c.fillRect(rx, y, rw, BLOCK_HEIGHT);
                        }
                    }
                }


                for (let h = 0; h < levels.length; h++) {
                    const y = reverse ? h * LEVEL_HEIGHT : canvasHeight - (h + 1) * LEVEL_HEIGHT;
                    for (let i = 0; i < levels[h][0].length; i++) {
                        drawFrame(h, i, y, h < root.level);
                    }
                }

                labels?.replaceChildren(...newLabels)

                console.log('Text labels: ', newLabels.length);

                const total = totalMarked();

                renderSearch(total, pattern);


                return total;
            }

            function render(newRoot, params) {
                let start = performance.now();
                let res = renderImpl(newRoot, params)
                if (firstRender) {
                    uiFactory().rum()?.finishDataRendering?.('task-flamegraph')
                    firstRender = false;
                }
                let finish = performance.now();
                console.log('Rendered flamegraph in', finish - start, 'ms')
                return res
            }

            canvas.onmousemove = function(event) {
                const h = Math.floor((reverse ? event.offsetY : (canvasHeight - event.offsetY)) / LEVEL_HEIGHT);
                if (h >= 0 && h < levels.length) {
                    const pos = findFrame(levels[h], event.offsetX / px + frame(root).left);
                    if (pos !== null) {
                        let f = frame({ level: h, index: pos });
                        if (visible(f)) {
                            const [left, width] = rectCanvasPosition(f, frame(root));
                            hl.style.left = (left + canvas.offsetLeft) + 'px';
                            hl.style.width = width + 'px';
                            if (theme === 'dark') {
                                hl.style.backgroundColor = darken(f.color, 0.4)
                            }

                            hl.style.top = ((reverse ? h * LEVEL_HEIGHT : canvasHeight - (h + 1) * LEVEL_HEIGHT) + canvas.offsetTop) + 'px';
                            hl.firstChild.textContent = shortenTitle(f.title);
                            hl.style.display = 'block';
                            const isMainRoot = root.index === mainRoot.index && root.level === mainRoot.level;
                            // .EventType currently is only equal to cycles
                            canvas.title = isMainRoot ? getCanvasTitle(f, null, frame(root)) : getCanvasTitle(f, frame(root), frame(mainRoot));

                            canvas.style.cursor = 'pointer';
                            canvas.onclick = function(e) {
                                if (f != root) {
                                    // rendering deep frame
                                    setState({
                                        frameDepth: h.toString(),
                                        framePos: pos.toString(),
                                    });

                                    render({ level: h, index: pos }, { pattern: searchPattern });
                                    canvas.onmousemove(e);
                                }
                            };
                            status.textContent = 'Function: ' + (isMainRoot ? getStatusTitle(f, null, frame(root)) : getStatusTitle(f, frame(root), frame(mainRoot)));
                            return;
                        }
                    }
                }
                canvas.onmouseout();
            }

            canvas.onmouseout = function() {
                hl.style.display = 'none';
                status.textContent = 'Function: ' + getStatusTitle(frame(root), null, frame(mainRoot));
                canvas.title = '';
                canvas.style.cursor = '';
                canvas.onclick = '';
            }

            // read query and display h and pos
            const h = Number(getState("frameDepth", '0'));
            const pos = Number(getState("framePos", '0'));


            render( { level: h, index: pos }, { pattern: searchPattern });


            const onResize = () => requestAnimationFrame(() => {
                canvas.style.width = null;
                initCanvas();
                render(root, { pattern: searchPattern });
            });
            window.addEventListener('resize', onResize)

            return () => {
                window.removeEventListener('resize', onResize)
            }
}
