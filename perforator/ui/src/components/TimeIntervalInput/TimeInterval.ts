import { type DateTime, dateTimeParse } from '@gravity-ui/date-utils';


export interface TimeInterval<T = string> {
    start: T;
    end: T;
}

export const parseTime = (value: string): DateTime => dateTimeParse(value)!;

export const parseTimeInterval = (interval: TimeInterval): TimeInterval<DateTime> => ({
    // [now-1d, now] is a fallback for strange values
    start: parseTime(interval.start) || parseTime('now-1d'),
    end: parseTime(interval.end) || parseTime('now'),
});
