import { createCleanupFn } from './cleanup';
import type { FormatNode, ProfileData } from './models/Profile';


export function populateWithChildrenArrays(rows: ProfileData['rows']) {
    for (let h = rows.length - 1; h > 0; h--) {
        const row = rows[h];
        for (let i = 0; i < row.length; i++) {
            const parentNode = rows[h - 1][row[i].parentIndex];
            if (!parentNode.children) {
                parentNode.children = [];
            }
            parentNode.children.push(row[i]);
        }
    }
}


const clearChildren = createCleanupFn('children');

const clearLevels = createCleanupFn('level');

const clearIndices = createCleanupFn('index');


function populateWithLevels(rows: ProfileData['rows']): ProfileData['rows'] {
    for (let h = 0; h < rows.length; h++) {
        for (let i = 0; i < rows[h].length; i++) {
            rows[h][i].level = h;
        }
    }

    return rows;
}


type NodeVisitor = (
    h: number,
    oldIndex: number,
     newIndex: number,
    ) => void;

function populateWithNewParentIndices(rows: ProfileData['rows'], coordsVisitor: NodeVisitor = () => {}): ProfileData['rows'] {
    for (let h = 1; h < rows.length; h++) {
        for (let i = 0; i < rows[h].length; i++) {
            for (let j = 0; j < (rows[h][i]?.children?.length ?? 0); j++) {
                coordsVisitor(h, rows[h][i].index, i);
                rows[h][i].children[j].parentIndex = i;
            }
        }
    }

    return rows;
}

function populateWithIndices(rows: ProfileData['rows']): ProfileData['rows'] {
    for (let h = 0; h < rows.length; h++) {
        for (let i = 0; i < rows[h].length; i++) {
            rows[h][i].index = i;
        }
    }

    return rows;
}


type SortableFields = keyof Pick<FormatNode, 'baseEventCount' | 'baseSampleCount' | 'eventCount' | 'sampleCount' | 'textId'>;

const createSortFormatNodes = (fieldName: SortableFields) => (a: FormatNode, b: FormatNode): number => b[fieldName] - a[fieldName];

export function createLeftHeavy(rows: ProfileData['rows'], fieldName: SortableFields = 'eventCount', coordsVisitor?: NodeVisitor): ProfileData['rows'] {
    const sortFormatNodes = createSortFormatNodes(fieldName);
    if (validateIsLeftHeavy(rows, sortFormatNodes)) {
        return rows;
    }
    return createDirectedReorder(sortFormatNodes)(rows, coordsVisitor);
}

const sortByStrings = (stringTable: string[]) => (a: FormatNode, b: FormatNode) => {
    if (a.textId === b.textId && a.file === b.file) {
        return 0;
    }
    if (stringTable[a.textId] + (stringTable?.[a.file] ?? '') > stringTable[b.textId] + (stringTable?.[b.file] ?? '')) {
        return 1;
    } else if (stringTable[a.textId] < stringTable[b.textId]) {
        return -1;
    }

    return 0;
};


export function validateIsLeftHeavy(rows: ProfileData['rows'], compareFn: (a: FormatNode, b: FormatNode) => number): boolean {
    for (let h = 0; h < rows.length; h++) {
        for (let i = 1; i < rows[h].length; i++) {
            const node = rows[h][i];
            const prevNode = rows[h][i - 1];
            if (node.parentIndex === prevNode.parentIndex && compareFn(node, prevNode) < 0) {
                return false;
            }
        }
    }

    return true;
}

export function inverseLeftHeavy(rows: ProfileData['rows'], stringTable: ProfileData['stringTable'], coordsVisitor?: NodeVisitor): ProfileData['rows'] {
    const sortFn = sortByStrings(stringTable);
    return createDirectedReorder(sortFn)(rows, coordsVisitor);
}

type SortFn = (a: FormatNode, b: FormatNode) => number

function createDirectedReorder(sortFn: SortFn) {
    return (rows: ProfileData['rows'], coordsVisitor?: NodeVisitor) => {
        if (validateIsLeftHeavy(rows, sortFn)) {
            return rows;
        }
        populateWithChildrenArrays(rows);
        populateWithLevels(rows);
        populateWithIndices(rows);


        const res = [[]];
        const queue = [rows[0][0]];
        let prevNode = rows[0][0];

        while (queue.length) {
            const node = queue.shift();
            if (prevNode.level !== node.level) {
                res.push([]);
            }
            res[res.length - 1].push(node);

            if (node.children) {
                const children = node.children.sort(sortFn);
                queue.push(...children);
            }
            prevNode = node;
        }

        populateWithNewParentIndices(res, coordsVisitor);


        clearChildren(res);
        clearLevels(res);
        clearIndices(res);

        return res;
    };
}
