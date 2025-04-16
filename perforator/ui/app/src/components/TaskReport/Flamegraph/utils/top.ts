import type { FormatNode, ProfileData, StringifiableFields } from 'src/models/Profile';

import type { H, I } from '../renderer';

import type { TopKeys } from './top-types';


export type TableFunctionTop = FunctionTop

export function calculateTopForTable(data: ProfileData['rows'], stringTableLength: number) {
    const topData = calculateTop(data, stringTableLength);

    const res: (TableFunctionTop)[] = [];
    for (const value of topData.values()) {
        delete value.shortestPath;
        res.push(value);
    }

    return res;
}


function populateWithSelfEventCount(rows: ProfileData['rows']) {
    for (let h = 0; h < rows.length; h++) {
        const row = rows[h];
        for (let i = 0; i < rows[h].length; i++) {
            row[i].selfEventCount = row[i].eventCount;
            row[i].selfSampleCount = row[i].sampleCount;
            row[i].baseSelfEventCount = (row[i].baseEventCount ?? 0);
            row[i].baseSelfSampleCount = (row[i].baseSampleCount ?? 0);
            if (row[i].parentIndex !== -1) {
                const parentIndex = row[i].parentIndex;
                const parentNode = rows[h - 1][parentIndex];
                parentNode.selfEventCount = parentNode.selfEventCount ? parentNode.selfEventCount - row[i].eventCount : 0;
                parentNode.selfSampleCount = parentNode.selfSampleCount ? parentNode.selfSampleCount - row[i].sampleCount : 0;
                parentNode.baseSelfEventCount = parentNode.baseSelfEventCount ? parentNode.baseSelfEventCount - (row[i].baseEventCount ?? 0) : 0;
                parentNode.baseSelfSampleCount = parentNode.baseSelfSampleCount ? parentNode.baseSelfSampleCount - (row[i].baseSampleCount ?? 0) : 0;
            }
        }
    }
}

function populateWithChildren(rows: ProfileData['rows']) {
    for (let h = rows.length - 1; h > 0; h--) {
        const row = rows[h];
        for (let i = 0; i < row.length; i++) {
            const parentNode = rows[h - 1][row[i].parentIndex];
            if (!parentNode.childrenIndices) {
                parentNode.childrenIndices = new Set();
            }
            parentNode.childrenIndices.add(i);
        }
    }
}

type FunctionTop = Record<TopKeys, number> & Pick<FormatNode, StringifiableFields | 'inlined'> &
{ shortestPath?: I[] }

function getNodeKeyFull(len: number, n: FormatNode) {
    return len ** 2 * (n.kind ?? 0) + (n.file ?? 0) * len + n.textId + (n.inlined ? 1 : 0);
}

export function calculateTop(rows: ProfileData['rows'], stringTableLength: number) {

    const res: Map<number, FunctionTop> = new Map();


    populateWithSelfEventCount(rows);


    const getNodeKey = getNodeKeyFull.bind(null, stringTableLength);
    for (let h = 0; h < rows.length; h++) {
        const row = rows[h];
        for (let i = 0; i < row.length; i++) {
            const node = row[i];
            const funcKey = getNodeKey(node);
            if (!res.has(funcKey)) {
                res.set(funcKey, {
                    'all.eventCount': 0,
                    'all.sampleCount': 0,
                    'self.eventCount': 0,
                    'self.sampleCount': 0,
                    'diff.all.eventCount': 0,
                    'diff.all.sampleCount': 0,
                    'diff.self.eventCount': 0,
                    'diff.self.sampleCount': 0,
                    textId: node.textId,
                    file: node.file,
                    frameOrigin: node.frameOrigin,
                    inlined: node.inlined,
                    kind: node.kind,
                });
            }
            const funcTopData = res.get(funcKey)!;
            funcTopData['self.eventCount'] += node.selfEventCount!;
            funcTopData['self.sampleCount'] += node.selfSampleCount!;
            funcTopData['diff.self.eventCount'] += (node.baseSelfEventCount ?? 0);
            funcTopData['diff.self.sampleCount'] += (node.baseSelfSampleCount ?? 0);
        }
    }

    populateWithChildren(rows);

    calcTotalTime(res, rows, getNodeKey);

    return res;
}

function isSubpath(path: I[], subpath: I[]) {
    if (subpath.length >= path.length) {
        return false;
    }
    for (let i = 0; i < subpath.length; i++) {
        if (path[i] !== subpath[i]) {
            return false;
        }
    }
    return true;
}

// we can't just sum all the `all.eventCount` over all nodes with the same name (because recursion)
// so instead we do DFS and keep the shortest path for every function
function calcTotalTime<K>(res: Map<K, FunctionTop>, rows: FormatNode[][], getNodeTitle: (node: FormatNode) => K) {
    function walker(h: H, i: I, indexesPath: I[]) {
        const node = rows[h][i];
        const key = getNodeTitle(node);
        const funcTopData = res.get(key)!;


        if (!funcTopData.shortestPath || !isSubpath(indexesPath, funcTopData.shortestPath)) {
            funcTopData['all.eventCount'] += node.eventCount;
            funcTopData['all.sampleCount'] += node.sampleCount;
            funcTopData['diff.all.eventCount'] += (node.baseEventCount ?? 0);
            funcTopData['diff.all.sampleCount'] += (node.baseSampleCount ?? 0);
            funcTopData.shortestPath = [...indexesPath];
        }

        for (const childIndex of node.childrenIndices ?? []) {
            walker(h + 1, childIndex, indexesPath.concat(childIndex));
        }
    }

    walker(0, 0, [0]);
}
