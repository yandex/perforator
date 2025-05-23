import type { ProfileData, StringifiedNode } from './models/Profile';


export function readNodeStrings(data: ProfileData, coords: { h: number; i: number}): StringifiedNode {
    const reader = createStringReader(data);
    return reader(coords);
}

export function createStringReader(data: ProfileData) {
    return (coords: { h: number; i: number}) => {
        const readString = (id?: number) => {
            if (id === undefined) {return '';}
            return data.stringTable[id];
        };


        const node = data.rows[coords.h][coords.i];
        const stringifiedNode: StringifiedNode = {
            ...node,
            file: readString(node.file),
            textId: readString(node.textId),
            kind: readString(node.kind),
            frameOrigin: readString(node.frameOrigin),
        };

        return stringifiedNode;
    };
}
