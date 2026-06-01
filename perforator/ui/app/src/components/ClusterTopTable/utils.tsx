import dayjs from '@gravity-ui/date-utils/build/dayjs';

import type { ClusterTopCount, ClusterTopEntry, ClusterTopGeneration } from 'src/generated/perforator/proto/perforator/perforator';


interface ClusterTopRowCount {
    Self: number;
    Cumulative: number;
    SelfPct: string;
    CumulativePct: string;
    SelfPctValue: number;
    CumulativePctValue: number;
}

export interface ClusterTopRow {
    Name: string;
    Count: ClusterTopRowCount;
    type: 'function' | 'service';
    services?: ClusterTopRow[];
    isLoadingServices?: boolean;
    error?: string;
    // present only for services
    parentFunction?: string;
}

type WireFormatClusterTopCount = ClusterTopCount & {
  /** Cpu-hours */
  Self: number | 'NaN';
  /** Cpu-hours */
  Cumulative: number | 'NaN';
  SelfPct: number | 'NaN';
  CumulativePct: number | 'NaN';
}

function takeNumber(value: unknown): number | undefined {
    if (typeof value === 'number') {
        return value;
    }
    return undefined;
}


function formatPct(value: number): string {
    const formatted = value.toFixed(2);
    const [integer, decimal] = formatted.split('.');
    const paddedInteger = integer.padStart(2, '0');
    return `${paddedInteger}.${decimal}%`;
}

function mapCount(count: WireFormatClusterTopCount | undefined, timeInterval: number): ClusterTopRowCount {
    if (!count) {
        return {
            Self: 0,
            Cumulative: 0,
            SelfPct: '0%',
            CumulativePct: '0%',
            SelfPctValue: 0,
            CumulativePctValue: 0,
        };
    }

    const self = takeNumber(count?.Self) ?? 0;
    const cumulative = takeNumber(count?.Cumulative) ?? 0;
    const selfPct = takeNumber(count?.SelfPct) ?? 0;

    const cumulativePct = takeNumber(count?.CumulativePct) ?? 0;

    return {
        Self: Math.round((self) / timeInterval),
        Cumulative: Math.round((cumulative) / timeInterval),
        SelfPct: formatPct(selfPct),
        CumulativePct: formatPct(cumulativePct),
        SelfPctValue: selfPct,
        CumulativePctValue: cumulativePct,
    };
}

export function convertToFunctionRow(entry: ClusterTopEntry, timeInterval: number): ClusterTopRow {
    return {
        Name: entry.Name,
        Count: mapCount(entry.Count, timeInterval),
        type: 'function',
        services: undefined,
        isLoadingServices: false,
    };
}

export function convertToServiceRow(parentName: string, entry: ClusterTopEntry, timeInterval: number): ClusterTopRow {
    return {
        Name: entry.Name,
        Count: mapCount(entry.Count, timeInterval),
        type: 'service',
        parentFunction: parentName,
    };
}


export function countHoursInterval(generation: ClusterTopGeneration) {
    const timeInterval = dayjs(generation.To).diff(dayjs(generation.From), 'hours');
    return timeInterval;
}
