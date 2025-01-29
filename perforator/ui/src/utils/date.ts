import type { DateTimeInput } from '@gravity-ui/date-utils';
import { dateTimeParse } from '@gravity-ui/date-utils';


export const parseDate = (value: DateTimeInput): Date|undefined => (
    dateTimeParse(value)?.toDate()
);

export const formatDate = (value: DateTimeInput, format: string): string|undefined => (
    dateTimeParse(value)?.format(format)
);

export const getIsoDate = (value: DateTimeInput): string|undefined => (
    parseDate(value)?.toISOString()
);
