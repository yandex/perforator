import { describe, expect, it } from '@jest/globals';

import { type ClusterTopGeneration, ClusterTopGenerationStatus } from 'src/generated/perforator/proto/perforator/perforator';

import { classifyGenerations, dayKeyOf, formatGenLabel } from './validation';


function makeGen(id: number, from: string, to: string): ClusterTopGeneration {
    return {
        ID: id,
        From: from,
        To: to,
        GenerationStatus: ClusterTopGenerationStatus.COMPLETED,
    };
}

describe('classifyGenerations', () => {
    it('treats all-24h gens as daily, anchored to start day', () => {
        const gens = [
            makeGen(1, '2026-03-01T09:00:00Z', '2026-03-02T09:00:00Z'),
            makeGen(2, '2026-03-02T09:00:00Z', '2026-03-03T09:00:00Z'),
        ];
        const c = classifyGenerations(gens);
        expect(c.mode).toBe('daily');
        const expectedKeys = gens.map(dayKeyOf);
        expect(Array.from(c.availableDays).sort()).toEqual(Array.from(new Set(expectedKeys)).sort());
        for (const key of expectedKeys) {
            expect(c.byDay.get(key)).toBeDefined();
        }
    });

    it('treats sub-24h gens as subday and groups them by day sorted by From', () => {
        const gens: ClusterTopGeneration[] = [];
        // 12 x 2h gens on a single local day
        for (let h = 0; h < 24; h += 2) {
            const from = `2026-03-01T${String(h).padStart(2, '0')}:00:00`;
            const toHour = h + 2;
            const to = toHour === 24 ? '2026-03-02T00:00:00' : `2026-03-01T${String(toHour).padStart(2, '0')}:00:00`;
            gens.push(makeGen(h, from, to));
        }
        const shuffled = [gens[5], gens[0], gens[3], gens[1], gens[4], gens[2], ...gens.slice(6)];
        const c = classifyGenerations(shuffled);
        expect(c.mode).toBe('subday');
        const day = dayKeyOf(gens[0]);
        const list = c.byDay.get(day)!;
        expect(list).toHaveLength(12);
        for (let i = 1; i < list.length; i++) {
            expect(new Date(list[i - 1].From!).getTime()).toBeLessThan(new Date(list[i].From!).getTime());
        }
    });

    it('returns subday when mixing 24h and 2h gens', () => {
        const gens = [
            makeGen(1, '2026-03-01T09:00:00Z', '2026-03-02T09:00:00Z'),
            makeGen(2, '2026-03-03T00:00:00Z', '2026-03-03T02:00:00Z'),
        ];
        expect(classifyGenerations(gens).mode).toBe('subday');
    });

    it('does not include gap days in availableDays', () => {
        const mon = makeGen(1, '2026-03-02T09:00:00Z', '2026-03-03T09:00:00Z');
        const wed = makeGen(2, '2026-03-04T09:00:00Z', '2026-03-05T09:00:00Z');
        const c = classifyGenerations([mon, wed]);
        expect(c.availableDays.has(dayKeyOf(mon))).toBe(true);
        expect(c.availableDays.has(dayKeyOf(wed))).toBe(true);
        expect(c.availableDays.has('2026-03-03')).toBe(false);
    });

    it('handles empty input', () => {
        const c = classifyGenerations([]);
        expect(c.mode).toBe('daily');
        expect(c.byDay.size).toBe(0);
        expect(c.availableDays.size).toBe(0);
    });
});

describe('formatGenLabel', () => {
    it('formats same-day ranges as HH:mm – HH:mm', () => {
        const gen = makeGen(1, '2026-03-01T00:00:00', '2026-03-01T02:00:00');
        expect(formatGenLabel(gen)).toBe('00:00 – 02:00');
    });

    it('appends (+1d) when To is on the next calendar day', () => {
        const gen = makeGen(1, '2026-03-01T09:00:00', '2026-03-02T09:00:00');
        expect(formatGenLabel(gen)).toBe('09:00 – 09:00 (+1d)');
    });
});

describe('dayKeyOf', () => {
    it('produces YYYY-MM-DD for backend ISO inputs', () => {
        const gen = makeGen(1, '2026-03-01T09:00:00', '2026-03-02T09:00:00');
        expect(dayKeyOf(gen)).toBe('2026-03-01');
    });
});
