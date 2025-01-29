import React from 'react';

import { RangeDateSelection } from '@gravity-ui/date-components';

import { uiFactory } from 'src/factory';
import { cn } from 'src/utils/cn';

import { parseTimeInterval, type TimeInterval } from './TimeInterval';
import { TimeIntervalControls } from './TimeIntervalControls/TimeIntervalControls';

import './TimeIntervalInput.scss';


export type { TimeInterval } from './TimeInterval';


const MIN_SELECTION_PRECISION = 1;  // 1 millisecond
const MIN_SELECTION_DURATION = 5 * 1000;  // 5 seconds
const MAX_SELECTION_DURATION = 365 * 24 * 60 * 60 * 100;  // 1 year

export interface TimeIntervalInputProps {
    initInterval: TimeInterval;
    onUpdate: (range: TimeInterval) => void;
    className?: string;
    headerControls?: boolean;
}

const b = cn('time-interval-selector');

export const TimeIntervalInput: React.FC<TimeIntervalInputProps> = props => {
    const [interval, setInterval] = React.useState(props.initInterval);

    const handleUpdate = React.useCallback((newInterval: TimeInterval) => {
        props.onUpdate(newInterval);
        setInterval(newInterval);
    }, [props.onUpdate, setInterval]);

    const className = b(
        { gravity: uiFactory().gravityStyles() },
        props.className,
    );

    return (
        <div className={className}>
            <TimeIntervalControls
                interval={interval}
                onUpdate={handleUpdate}
                header={props.headerControls}
            />
            <RangeDateSelection
                className="time-interval-selector__ruler"
                displayNow
                hasScaleButtons
                minDuration={MIN_SELECTION_DURATION}
                maxDuration={MAX_SELECTION_DURATION}
                align={MIN_SELECTION_PRECISION}
                scaleButtonsPosition="end"
                value={parseTimeInterval(interval)}
                onUpdate={range => handleUpdate({
                    start: range.start.toISOString(),
                    end: range.end.toISOString(),
                })}
            />
        </div>
    );
};
