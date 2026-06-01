import dayjs from '@gravity-ui/date-utils/build/dayjs';

import type { ClusterTopGeneration } from 'src/generated/perforator/proto/perforator/perforator';


export type GenerationMode = 'daily' | 'subday';

export interface ClassifiedGenerations {
    mode: GenerationMode;
    byDay: Map<string, ClusterTopGeneration[]>;
    availableDays: Set<string>;
}

export function isDailyDuration(gen: ClusterTopGeneration): boolean {
    if (!gen.From || !gen.To) {
        return false;
    }
    return dayjs(gen.To).diff(dayjs(gen.From), 'hour') === 24;
}

export function dayKeyOf(gen: ClusterTopGeneration): string {
    return dayjs(gen.From).format('YYYY-MM-DD');
}

export function formatGenLabel(gen: ClusterTopGeneration): string {
    const from = dayjs(gen.From);
    const to = dayjs(gen.To);
    const dayDiff = to.startOf('day').diff(from.startOf('day'), 'day');
    const base = `${from.format('HH:mm')} – ${to.format('HH:mm')}`;
    return dayDiff > 0 ? `${base} (+${dayDiff}d)` : base;
}

export function classifyGenerations(gens: ClusterTopGeneration[]): ClassifiedGenerations {
    const valid = gens.filter((g) => g.From && g.To);
    const mode: GenerationMode = valid.length === 0 || valid.every(isDailyDuration) ? 'daily' : 'subday';

    const byDay = new Map<string, ClusterTopGeneration[]>();
    for (const gen of valid) {
        const key = dayKeyOf(gen);
        const list = byDay.get(key);
        if (list) {
            list.push(gen);
        } else {
            byDay.set(key, [gen]);
        }
    }
    for (const list of byDay.values()) {
        list.sort((a, b) => dayjs(a.From).valueOf() - dayjs(b.From).valueOf());
    }

    return {
        mode,
        byDay,
        availableDays: new Set(byDay.keys()),
    };
}
