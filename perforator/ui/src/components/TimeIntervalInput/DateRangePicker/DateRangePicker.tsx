import React from 'react';

import { RelativeRangeDatePicker, type RelativeRangeDatePickerValue } from '@gravity-ui/date-components';
import { isLikeRelative } from '@gravity-ui/date-utils';

import { parseTime, type TimeInterval } from '../TimeInterval';

import './DateRangePicker.scss';


type DatePickerValue = RelativeRangeDatePickerValue['start'];

const makeDatePickerTime = (value: string): DatePickerValue => (
    isLikeRelative(value)
        ? {
            type: 'relative',
            value,
        } : {
            type: 'absolute',
            value: parseTime(value),
        }
);

const makeDatePickerInterval = (interval: TimeInterval) => ({
    start: makeDatePickerTime(interval.start),
    end: makeDatePickerTime(interval.end),
});

const datePickerValueToString = (time: DatePickerValue): string | null => {
    if (!time) {
        return null;
    }
    return (
        time.type === 'relative'
            ? time.value
            : time.value.toISOString()
    );
};

interface DateRangePickerProps {
    interval: TimeInterval;
    onUpdate?: (value: TimeInterval) => void;
}

export const DateRangePicker: React.FC<DateRangePickerProps> = props => (
    <RelativeRangeDatePicker
        className="date-range-picker"
        value={makeDatePickerInterval(props.interval)}
        onUpdate={range => {
            if (range) {
                const start = datePickerValueToString(range.start);
                const end = datePickerValueToString(range.end);
                if (start && end) {
                    props.onUpdate?.({ start, end });
                }
            }
        }}
        format="DD.MM.YYYY HH:mm"
        withApplyButton
        withPresets
    />
);
