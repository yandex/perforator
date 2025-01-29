import type { Rgb } from 'culori/fn';
import { convertHslToRgb, convertOklabToRgb, convertRgbToOklab, parseHex, serializeHex } from 'culori/fn';

import type { FormatNode, ProfileData } from 'src/models/Profile';


function namehash(name: string, reversed = false): number {
    let vector = 0;
    let weight = 1;
    let max = 1;
    let mod = 10;
    const start = reversed ? name.length - 1 : 0;
    const check = reversed ? (j: number) => j > 0 : (j: number) => j < name.length;
    const increment = reversed ? -1 : 1;

    for (let j = start; check(j); j += increment) {
        const i = name.charCodeAt(j) % mod;

        vector += (i / (mod - 1)) * weight;
        mod += 1;
        max += Number(weight);
        weight *= 0.7;

        if (mod > 13) {
            break;
        }
    }
    return 1 - vector / max;
}

function hex(n: number) {
    return Math.round(Math.min(n, 255)).toString(16);
}
export function hashcolor(name: string, module?: string): string {
    const v1 = namehash(name);
    const v2 = namehash(name, true);
    const v3 = v2;

    let R: number, G: number, B: number;

    if (module === 'kernel' || name.includes('[kernel]')) {
        R = 96 + 55 * v2;
        G = 96 + (255 - 96) * v1;
        B = 205 + 50 * v3;
    } else if (module === 'python' || name.includes('[python]') || name.endsWith('.py')) {
        R = 103 + 50 * v2;
        G = 178 + 77 * v1;
        B = 120 + 50 * v3;
    } else {
        R = 205 + 50 * v3;
        G = 0 + 230 * v1;
        B = 0 + 55 * v2;
    }

    return `#${hex(R)}${hex(G)}${hex(B)}`;
}

export const DARKEN_FACTOR = 0.2;


export function darken(color: string, factor = DARKEN_FACTOR): string {
    const { l, ...parsedColor } = convertRgbToOklab(parseHex(color) as Omit<Rgb, 'mode'>);

    const resultingColor = {
        ...parsedColor,
        l: Math.max(l - factor, 0),
    };

    return serializeHex(convertOklabToRgb(resultingColor)) as string;
}
export function prerenderColors(data: ProfileData, opts?: { theme?: 'light' | 'dark' }): ProfileData {

    function readString(id?: number) {
        if (!id) {return '';}
        return data.stringTable[id];
    }


    for (let h = 0; h < data.rows.length; h++) {
        for (let i = 0; i < data.rows[h].length; i++) {
            const color = hashcolor(readString(data.rows[h][i].textId), readString(data.rows[h][i].frameOrigin));
            // if (typeof data.rows[h][i].color === 'number') {
            //     const color = readString(data.rows[h][i].color as number);
            //     data.rows[h][i].color = opts?.theme === 'dark' ? darken(color) : color;
            // } else {
            data.rows[h][i].color = opts?.theme === 'dark' ? darken(color) : color;
            // }
        }
    }

    return data;
}


export function hsv2hsl(h: number, s: number, v: number) {
    const l = v - v * s / 2;
    const m = Math.min(l, 1 - l);
    return [h, m ? (v - l) / m : 0, l];
}


export function diffcolor(node: FormatNode, root: FormatNode) {
    const lhs = node.eventCount;
    const rhs = root.baseEventCount && root.baseEventCount > 1e-5
        ? (node.baseEventCount ?? 0) * root.eventCount / root.baseEventCount
        : 0;

    const diff = rhs > 1e-5 ? (lhs - rhs) / rhs : 1.0;
    const d = Math.min(Math.abs(diff), 1.0);

    if (d < 1e-3) {
        const value = Math.round(180 + 60 * (1 - d * 1e3));
        // eslint-disable-next-line @typescript-eslint/no-shadow
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

    const soff = 0.0;
    const spow = 4.5;
    const scoef = 0.75;

    const h = hoff + Math.pow(d, 1.0 / hpow) * hcoef;
    const s = soff + Math.pow(d, 1.0 / spow) * scoef;
    const hsl = hsv2hsl(h, s, 1.0);

    // return `hsl(${hsl[0] * 360} ${hsl[1] * 100} ${hsl[2] * 100})`;
    return serializeHex(convertHslToRgb({ h: hsl[0] * 360, s: hsl[1], l: hsl[2] }));
}
