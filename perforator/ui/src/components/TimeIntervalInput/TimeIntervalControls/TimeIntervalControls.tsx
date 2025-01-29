import React from 'react';

import { Button } from '@gravity-ui/uikit';

import { cn } from 'src/utils/cn';

import { CustomRangeInput } from '../CustomRangeInput/CustomRangeInput';
import { DateRangePicker } from '../DateRangePicker/DateRangePicker';
import { parseTime, type TimeInterval } from '../TimeInterval';

import './TimeIntervalControls.scss';


const RANGE_PRESETS = [
    '30m',
    '1h',
    '6h',
    '1d',
    '1w',
];

const b = cn('time-interval-selector');

const compareTimes = (lhs: string, rhs: string) => (
    parseTime(lhs)?.valueOf() === parseTime(rhs)?.valueOf()
);

const compareTimeIntervals = (lhs: TimeInterval, rhs: TimeInterval) => (
    compareTimes(lhs.start, rhs.start)
    && compareTimes(lhs.end, rhs.end)
);

const lastInterval = (value: string): TimeInterval => ({
    start: `now-${value}`,
    end: 'now',
});

interface TimeIntervalControlsProps {
    interval: TimeInterval;
    onUpdate?: (value: TimeInterval) => void;
    header?: boolean;
}

export const TimeIntervalControls: React.FC<TimeIntervalControlsProps> = props => {
    const { interval } = props;

    const setInterval = React.useCallback((newInterval: TimeInterval) => {
        props.onUpdate?.(newInterval);
    }, [props.onUpdate]);

    const renderDatePicker = React.useCallback(() => (
        <DateRangePicker
            interval={interval}
            onUpdate={setInterval}
        />
    ), [interval, setInterval]);

    const renderPresetButton = React.useCallback((value: string) => {
        const newInterval = lastInterval(value);
        const isSelected = compareTimeIntervals(interval, newInterval);
        return (
            <Button
                key={`preset-${value}`}
                size="s"
                view="flat"
                onClick={() => setInterval(newInterval)}
                selected={isSelected}
            >
                {value}
            </Button>
        );
    }, [interval, setInterval]);

    const renderPresetButtons = React.useCallback(() => (
        RANGE_PRESETS.map(renderPresetButton)
    ), [renderPresetButton]);

    const [customInterval, setCustomInterval] = React.useState<Optional<TimeInterval>>();

    const renderCustomRangeInput = React.useCallback(() => {
        return (
            <CustomRangeInput
                selected={customInterval && compareTimeIntervals(interval, customInterval)}
                onUpdate={value => {
                    const newInterval = value ? lastInterval(value) : undefined;
                    setCustomInterval(newInterval);
                    if (newInterval) {
                        setInterval(newInterval);
                    }
                }}
            />
        );
    }, [customInterval, interval, setCustomInterval, setInterval]);

    return (
        <div className={b('controls', { header: props.header })}>
            {renderDatePicker()}
            <div className={b('controls-buttons')}>
                {renderPresetButtons()}
                {renderCustomRangeInput()}
            </div>
        </div>
    );
};
