import React from 'react';

import type { RangeDateSelectionProps, RangeValue } from '@gravity-ui/date-components';
import { RangeDateSelection } from '@gravity-ui/date-components';
import type { DateTime } from '@gravity-ui/date-utils';

import { uiFactory } from 'src/factory';
import { cn } from 'src/utils/cn';

import { parseTimeInterval, type TimeInterval } from './TimeInterval';
import { TimeIntervalControls } from './TimeIntervalControls/TimeIntervalControls';

import './TimeIntervalInput.scss';


export type { TimeInterval } from './TimeInterval';


const MIN_SELECTION_PRECISION = 1;  // 1 millisecond
const MIN_SELECTION_DURATION = 5 * 1000;  // 5 seconds
const MAX_SELECTION_DURATION = 365 * 24 * 60 * 60 * 100;  // 1 year

export interface TimeIntervalInputProps extends Pick<RangeDateSelectionProps, 'numberOfIntervals'> {
    initInterval: TimeInterval;
    interval?: TimeInterval;
    onUpdate: (range: TimeInterval) => void;
    className?: string;
    headerControls?: boolean;
    additionalHeaderItems?: React.ReactElement;
    header?: React.ReactElement;
    additionalItems?: React.ReactElement;
}

const b = cn('time-interval-selector');


export const TimeIntervalInput: React.FC<TimeIntervalInputProps> = ({
    initInterval,
    onUpdate,
    additionalHeaderItems,
    className,
    headerControls,
    interval,
    numberOfIntervals,
    header,
    additionalItems,
}) => {
    const [intervalInternal, setIntervalInternal] = React.useState(initInterval);

    const handleUpdate = React.useCallback((newInterval: TimeInterval) => {
        onUpdate(newInterval);
        setIntervalInternal(newInterval);
    }, [onUpdate, setIntervalInternal]);

    const handleRangeUpdate = React.useCallback((range: RangeValue<DateTime>) => {
        handleUpdate({
            start: range.start.toISOString(),
            end: range.end.toISOString(),
        });
    }, [handleUpdate]);

    const value = React.useMemo(() => parseTimeInterval(interval ?? intervalInternal), [interval, intervalInternal]);

    const classNameWithGravity = b(
        {
            gravity: uiFactory().gravityStyles(),
        },
        className,
    );


    return (
        <div className={classNameWithGravity}>
            <div className={b('header')}>
                {header}
                <TimeIntervalControls
                    interval={interval ?? intervalInternal}
                    onUpdate={handleUpdate}
                    header={headerControls}
                    additionalHeaderItems={additionalHeaderItems}
                />
            </div>
            <div className={b('wrapper')}>
                <RangeDateSelection
                    className="time-interval-selector__ruler"
                    displayNow
                    hasScaleButtons
                    minDuration={MIN_SELECTION_DURATION}
                    maxDuration={MAX_SELECTION_DURATION}
                    align={MIN_SELECTION_PRECISION}
                    scaleButtonsPosition="end"
                    value={value}
                    onUpdate={handleRangeUpdate}
                    numberOfIntervals={numberOfIntervals}
                />
                {additionalItems}
            </div>
        </div>
    );
};
