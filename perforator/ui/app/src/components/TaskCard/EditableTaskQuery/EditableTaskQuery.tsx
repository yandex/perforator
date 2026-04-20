import React from 'react';

import { useNavigate } from 'react-router-dom';

import { dateTimeParse } from '@gravity-ui/date-utils';
import { Check, Xmark } from '@gravity-ui/icons';
import { Button, Icon } from '@gravity-ui/uikit';

import { SwitchableSelectorInput } from 'src/components/MergeProfilesForm/TokensInput/SwitchableSelectorInput';
import { uiFactory } from 'src/factory';
import type { TaskResult } from 'src/models/Task';
import { boolToString } from 'src/utils/bool';
import { redirectToTaskPage } from 'src/utils/profileTask';
import { cutIdFromSelector, cutSpaceFromSelector, cutTimeFromSelector, parseTimestampFromSelector, removeOptionalTailingComma } from 'src/utils/selector';

import { areIntervalsEqual, type TimeInterval } from '../../TimeIntervalInput/TimeInterval';
import { TimeIntervalInput as TimeIntervalInputRaw } from '../../TimeIntervalInput/TimeIntervalInput';

import './EditableTaskQuery.scss';


interface EditableTaskQueryProps {
    task: TaskResult | null;
    additionalHeaderItems?: React.ReactElement;
    header: React.ReactElement;
    embed?: boolean;
}

const TimeIntervalInput = React.memo(TimeIntervalInputRaw);

export const EditableTaskQuery: React.FC<EditableTaskQueryProps> = ({ task, additionalHeaderItems, header, embed }) => {
    const spec = task?.Spec?.MergeProfiles;
    const query = spec?.Query;
    const selector = query?.Selector;
    const maxSamples = query?.MaxSamples || spec?.MaxSamples as number;
    const format = spec?.Format?.JSONFlamegraph;
    const areLineNumbersEnabled = format?.ShowLineNumbers;

    const navigate = useNavigate();
    const isSingleProfile = maxSamples === 1;

    const time = React.useMemo<TimeInterval | null>(() => {
        const baseTime = selector ? parseTimestampFromSelector(selector!) : null;

        if (!baseTime?.from || !baseTime?.to) {
            return null;
        }

        if (isSingleProfile) {
            return {
                start: dateTimeParse(baseTime?.from)?.subtract(5, 'm').toISOString(),
                end: dateTimeParse(baseTime?.to)?.add(5, 'm').toISOString(),
            } as TimeInterval;
        }
        else {
            return {
                start: baseTime.from,
                end: baseTime.to,
            } as TimeInterval;
        }
    }, [isSingleProfile, selector]);

    const initialSelector = selector ? cutSpaceFromSelector(cutTimeFromSelector(selector)) : undefined;
    const [currentSelector, setCurrentSelector] = React.useState<string | undefined>(initialSelector);
    const [timelineValue, setTimelineValue] = React.useState<TimeInterval | undefined>(undefined);

    React.useEffect(() => {
        if (initialSelector) {
            setCurrentSelector(initialSelector);
        }
    }, [initialSelector]);

    const hasSelectorChanged = currentSelector !== initialSelector;
    const hasSelectionChanged = timelineValue && time ? !areIntervalsEqual(timelineValue, time) : false;
    const hasChanges = hasSelectorChanged || hasSelectionChanged;

    const handleSave = React.useCallback(() => {
        if (!selector) {
            return;
        }

        const maxProfiles = maxSamples;
        const selectorToSave = currentSelector ? removeOptionalTailingComma(currentSelector) : cutTimeFromSelector(selector);

        let newSelector = selectorToSave;
        if (isSingleProfile) {
            newSelector = cutIdFromSelector(newSelector);
        }

        if (timelineValue || time) {
            const timeToSave = timelineValue || time;
            if (timeToSave) {

                uiFactory().reachGoal('EDIT_TASK_HEADER');

                redirectToTaskPage(navigate, {
                    selector: newSelector,
                    maxProfiles: maxProfiles,
                    from: timeToSave.start,
                    to: timeToSave.end,
                    lineNumbers: boolToString(areLineNumbersEnabled),
                });
            }
        }
    }, [areLineNumbersEnabled, currentSelector, isSingleProfile, maxSamples, navigate, selector, time, timelineValue]);

    const handleCancel = React.useCallback(() => {
        setCurrentSelector(initialSelector);
        if (time) {
            setTimelineValue(time);
        }
    }, [initialSelector, time]);

    const handleKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
        if ((event.ctrlKey || event.metaKey) && event.code === 'Enter') {
            handleSave();
        }
    };

    const controls = React.useMemo(() => (
        <div className="editable-task-query__controls">
            <Button
                className="editable-task-query__button"
                onClick={handleCancel}
                view="flat"
                disabled={!hasChanges}
            >
                <Icon size={14} data={Xmark} />
            </Button>
            <Button
                className="editable-task-query__button"
                disabled={!hasChanges}
                onClick={handleSave}
                view="action"
            >
                <Icon size={14} data={Check} />
            </Button>
        </div>
    ), [handleCancel, handleSave, hasChanges]);

    if (!task || !time) {
        return header;
    }

    return (
        <div className="editable-task-query" onKeyDown={handleKeyDown}>
            <div className="editable-task-query__header">
                <TimeIntervalInput
                    initInterval={time}
                    interval={timelineValue}
                    onUpdate={setTimelineValue}
                    header={header}
                    additionalHeaderItems={additionalHeaderItems}
                    numberOfIntervals={isSingleProfile ? 10 : undefined}
                    headerControls
                    additionalItems={embed ? controls : undefined}
                />
                {!embed && (
                    <div className="editable-task-query__input">
                        <SwitchableSelectorInput
                            className="editable-task-query__query-language-editor"
                            wrapperClassName="editable-task-query__editor-wrapper"
                            selector={currentSelector}
                            onUpdate={setCurrentSelector}
                            height="18px"
                        />
                        {controls}
                    </div>
                )}
            </div>
        </div>
    );
};
