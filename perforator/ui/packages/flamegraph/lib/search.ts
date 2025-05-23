import type { ProfileData } from './models/Profile';

import type { Coordinate } from './renderer';

import type { ReadString, StringModifier } from './node-title';
import { getNodeTitleFull } from './node-title';


// TODO replace with RegExp.escape once ES2025 is adopted by typescript and babel
export function escapeRegex(str: string) {
    return str.replace(/[/\-\\^$*+?.()|[\]{}]/g, '\\$&');
}


export function search(readString: ReadString, shorten: StringModifier, shouldShorten: boolean, rows: ProfileData['rows'], query: RegExp | string): Coordinate[] {
    const res: Coordinate[] = [];
    const test = (() => {
        if (typeof query === 'string') {
            const regex = (new RegExp(escapeRegex(query)));
            return (str: string) => regex.test(str);
        }
        return (str: string)=> query.test(str);
    })();

    const getNodeTitle = getNodeTitleFull.bind(null, readString, shorten, shouldShorten);

    for (let h = 0; h < rows.length; h++) {
        for (let i = 0; i < rows[h].length; i++) {
            const node = rows[h][i];
            const name = getNodeTitle(node);
            const matched = test(name);
            if (matched) {
                res.push([h, i]);
            }
        }
    }

    return res;
}
