import React from 'react';

import { useNavigate } from 'react-router-dom';

import { Hotkey } from '@perforator/flamegraph';

import { Button, DropdownMenu } from '@gravity-ui/uikit';

import { LocalStorageKey } from 'src/const/localStorage';
import type { ProfileTaskQuery } from 'src/models/Task';
import { cn } from 'src/utils/cn';
import { redirectToTaskPage } from 'src/utils/profileTask';
import { setPageTitle } from 'src/utils/title';
import { createErrorToast } from 'src/utils/toaster';

import { ProfileTable } from '../ProfileTable/ProfileTable';
import { TimeIntervalInput } from '../TimeIntervalInput/TimeIntervalInput';

import { changeQueryToNewInput } from './changeQueryToNewInput';
import type { QueryInput, QueryInputResult } from './QueryInput';
import { QUERY_INPUTS } from './queryInputs';
import { QueryInputSwitcher } from './QueryInputSwitcher/QueryInputSwitcher';
import { SampleSizeInput } from './SampleSizeInput/SampleSizeInput';
import { useProfileStateQuery } from './useProfileStateQuery';

import './MergeProfilesForm.scss';


interface MergeProfilesFormProps {
    onRender?: (query: QueryInputResult) => void;
    removeMergeButton?: boolean;
    className?: string;
    inMemory?: boolean;
    compactTable?: boolean;
    diff?: boolean;
    header?: React.ReactElement;
}

const b = cn('merge-profiles-form');

const defaultInput = (query: QueryInputResult, queryInputs: QueryInput[]): string => {
    return (
        queryInputs.find(input => query[input.queryField as keyof QueryInputResult])?.name
        || localStorage.getItem(LocalStorageKey.QueryInputKind)
        || queryInputs[0]?.name
    );
};

// Support broken selectors like `{key=value, }` and `key=value`.
// Using these selectors, we can create a relatively human-friendly UX for suggestions
const fixSelector = (selector: Optional<string>): Optional<string> => {
    if (selector === undefined) {
        return undefined;
    }
    const fixed = selector
        .replace(/, *}$/, '}')
        .replace(/^{/, '')
        .replace(/}$/, '')
    ;
    return `{${fixed}}`;
};


export const MergeProfilesForm: React.FC<MergeProfilesFormProps> = ({
    onRender,
    removeMergeButton,
    className,
    inMemory,
    compactTable,
    diff,
    header,
}: MergeProfilesFormProps) => {
    const navigate = useNavigate();

    const [query, setQuery] = useProfileStateQuery({ inMemory });

    const queryInputs: QueryInput[] = React.useMemo(() => QUERY_INPUTS, []);
    const [queryInputName, setQueryInputName] = React.useState(defaultInput(query, queryInputs));

    const [tableSelector, setTableSelector] = React.useState<Optional<string>>(query.selector);

    const queryInput = React.useMemo(
        () => queryInputs.find((desc => desc.name === queryInputName)) || queryInputs[0] || {},
        [queryInputs, queryInputName],
    );

    React.useEffect(() => {
        setPageTitle(tableSelector ? `Profiles: ${tableSelector}` : undefined);
    }, [tableSelector]);

    React.useMemo(() => {
        if (!tableSelector) {
            return;
        }

        const newField = changeQueryToNewInput(queryInput, tableSelector);
        if (newField) {
            setQuery({ ...query, ...newField });
        }
        else {
            setTableSelector(undefined);
        }
    }, [queryInput]);

    const renderQueryInputSwitcher = () => (
        <QueryInputSwitcher
            value={queryInput.name}
            inputs={queryInputs}
            onUpdate={name => {
                setQueryInputName(name);
                localStorage.setItem(LocalStorageKey.QueryInputKind, name);
                setQuery({ ...query, [queryInput.queryField]: undefined });
            }}
        />
    );

    const queryWithSelector = ({ raw, text }: {raw?: boolean; text?: boolean} = {}) => ({
        ...query,
        selector: fixSelector(query.selector ?? tableSelector),
        rawProfile: raw ? 'true' : undefined,
        format: text ? 'text' : undefined,
    } as ProfileTaskQuery);

    const submitTask = async ({ raw, text }: {raw?: boolean; text?: boolean} = {}) => {
        const queryToSend = queryWithSelector({ raw, text });
        if (!queryToSend.selector) {
            return;
        }
        try {
            redirectToTaskPage(navigate, queryToSend);
        } catch (error) {
            createErrorToast(
                error,
                { name: 'submit-task-error', title: 'Failed to submit new task' },
            );
        }
    };

    const renderMergeProfilesButton = () => removeMergeButton ? null : (
        <React.Fragment>
            <Button
                onClick={() => submitTask()}
                view="action"
            >
                Merge profiles
                <Hotkey value="cmd+enter" view="dark" />
            </Button>
            <DropdownMenu popupProps={{ placement: 'bottom-end' }} items={[
                { action: () => submitTask({ raw: true }), text: 'Merge into pprof' },
                { action: () => submitTask({ text: true }), text: 'Merge into text format' },

            ]}/>
        </React.Fragment>
    );

    const handleKeyDown = React.useCallback((event: React.KeyboardEvent<HTMLDivElement>) => {
        if ((event.ctrlKey || event.metaKey) && event.code === 'Enter') {
            if (!removeMergeButton) {
                submitTask();
            }
        }
    }, [query, tableSelector]);

    const profileTable = React.useMemo(() => (
        <ProfileTable
            compact={compactTable}
            query={{
                selector: fixSelector(tableSelector),
                from: query.from,
                to: query.to,
            }}
        />
    ), [compactTable, tableSelector, query.from, query.to]);

    if (onRender) {
        onRender(queryWithSelector());
    }

    return (
        <div
            className={`${className ? className : ''}`}
            tabIndex={-1}
            onKeyDown={handleKeyDown}
        >
            <div className={b(null)}>
                <TimeIntervalInput
                    className={b('time-interval-input')}
                    headerControls={!diff}
                    initInterval={{
                        start: query.from,
                        end: query.to,
                    }}
                    onUpdate={interval => {
                        setQuery({
                            ...query,
                            from: interval.start,
                            to: interval.end,
                        });
                    }}
                    header={header}
                />
                <div className="merge-profiles-form__header">
                    {renderQueryInputSwitcher()}
                    <SampleSizeInput
                        value={query.maxProfiles}
                        onUpdate={value => setQuery({
                            ...query,
                            maxProfiles: value,
                        })}
                    />
                </div>
                <div className="merge-profiles-form__inputs">
                    {queryInput.render ? queryInput.render(query, setQuery, setTableSelector) : null}
                </div>
                <div className="merge-profiles-form__buttons">
                    {renderMergeProfilesButton()}
                </div>
            </div>

            <div className="merge-profiles-form__table">
                <h3 className="merge-profiles-form__table-heading">
                    Preview of profiles matching selector
                </h3>
                <div>
                    {profileTable}
                </div>
            </div>
        </div>
    );
};
