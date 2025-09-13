import type { FormatNode } from './/models/Profile';

import { hugenum } from './flame-utils';
import { formatPct, pct } from './pct';


type TitleArgs = {
    getPct: (arg: {
        rootPct: string | undefined;
        selectedPct: string | undefined;
    }) => string;
    getNumbers: (arg: { sampleCount: string; eventCount: string; percent: string }) => string;
    wrapNumbers: (numbers: string) => string;
    getDelta: (delta: string) => string;
};

type Last<T extends any[]> = T extends [...infer _, infer L] ? L : never;

type NodeGetter<T> = (n: FormatNode) => T;
type RenderTitleResult = (args: Last<Parameters<typeof renderTitleFull>>) => ReturnType<typeof renderTitleFull>;

export function renderTitleFull(countEventCountWidth: NodeGetter<number>, countSampleCountWidth: NodeGetter<number>, getNodeTitle: NodeGetter<string>, isDiff: boolean, isReversed: boolean,
    {
        getPct, getNumbers, getDelta, wrapNumbers = (numbers: string) => numbers,
    }: TitleArgs) {
    // eslint-disable-next-line @typescript-eslint/no-shadow
    return function (f: FormatNode, selectedFrame: FormatNode | null, root?: FormatNode): string {
        const calcPercent = (baseFrame?: FormatNode | null) => baseFrame ? pct(countEventCountWidth(f), countEventCountWidth(baseFrame)) : undefined;
        const percent = getPct({
            rootPct: calcPercent(root),
            selectedPct: calcPercent(selectedFrame),
        });
        const shortenedTitle = getNodeTitle(f);
        const numbers = getNumbers({
            sampleCount: hugenum(countSampleCountWidth(f)),
            eventCount: hugenum(countEventCountWidth(f)),
            percent,
        });

        let diffString = '';


        if (isDiff) {
            let delta = 0;
            const anyFrame = (selectedFrame || root) as FormatNode;
            if (anyFrame.baseEventCount && f.baseEventCount && anyFrame.baseEventCount > 1e-3) {
                delta =
                    f.eventCount / anyFrame.eventCount -
                    f.baseEventCount / anyFrame.baseEventCount;
                if(isReversed) {
                    delta *= -1;
                }
            } else {
                delta = f.eventCount / anyFrame.eventCount;
            }
            const deltaString = (delta >= 0.0 ? '+' : '') + (delta * 100).toFixed(2) + '%';
            diffString += getDelta(deltaString);
        }

        return shortenedTitle + wrapNumbers(numbers + diffString);
    };
}

function capitalize(s: string) {
    return s.charAt(0).toUpperCase() + s.slice(1);
}

export const getStatusTitleFull = (eventName: string, renderTitle: RenderTitleResult) => renderTitle({
    getPct: ({ rootPct, selectedPct }) => (
        [rootPct, selectedPct].filter(Boolean).map(formatPct).join('/')
    ),
    getNumbers: ({ sampleCount, eventCount, percent }) => `${eventCount} ${eventName}, ${sampleCount} samples, ${percent}`,
    wrapNumbers: numbers => ` (${numbers})`,
    getDelta: delta => `, ${delta}`,
});

export const getCanvasTitleFull = (eventName: string, renderTitle: RenderTitleResult) => renderTitle({
    getPct: ({ rootPct, selectedPct }) => (
        (rootPct ? `Percentage of root frame: ${formatPct(rootPct)}\n` : '')
        + (selectedPct ? `Percentage of selected frame: ${formatPct(selectedPct)}\n` : '')
    ),
    getNumbers: ({ sampleCount, eventCount, percent }) => `\nSamples: ${sampleCount}\n${capitalize(eventName)}: ${eventCount}\n${percent}`,
    wrapNumbers: numbers => numbers.trimEnd(),
    getDelta: delta => `Diff: ${delta}\n`,
});
