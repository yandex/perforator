import React from 'react';

import { useNavigate } from 'react-router-dom';

import { Button, DropdownMenu } from '@gravity-ui/uikit';

import { Hotkey } from 'src/components/Hotkey/Hotkey';
import { LocalStorageKey } from 'src/const/localStorage';
import { uiFactory } from 'src/factory';
import type { ProfileTaskQuery } from 'src/models/Task';
import { cn } from 'src/utils/cn';
import { redirectToTaskPage } from 'src/utils/profileTask';
import { setPageTitle } from 'src/utils/title';
import { createErrorToast } from 'src/utils/toaster';

import { ProfileTable } from '../ProfileTable/ProfileTable';
import { TimeIntervalInput } from '../TimeIntervalInput/TimeIntervalInput';

import type { QueryInput, QueryInputResult } from './QueryInput';
import { QueryInputSwitcher } from './QueryInputSwitcher/QueryInputSwitcher';
import { SampleSizeInput } from './SampleSizeInput/SampleSizeInput';
import { useProfileStateQuery } from './utils';

import './MergeProfilesForm.scss';


interface MergeProfilesFormProps {
    onRender?: (query: QueryInputResult) => void;
    removeMergeButton?: boolean;
    className?: string;
    inMemory?: boolean;
    compactTable?: boolean;
    diff?: boolean;
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

export const MergeProfilesForm: React.FC<MergeProfilesFormProps> = props => {
    const navigate = useNavigate();

    const [query, setQuery] = useProfileStateQuery({ inMemory: props.inMemory });

    const queryInputs: QueryInput[] = React.useMemo(() => uiFactory().queryInputs(), []);
    const [queryInputName, setQueryInputName] = React.useState(defaultInput(query, queryInputs));

    const [tableSelector, setTableSelector] = React.useState<Optional<string>>(query.selector);

    const queryInput = React.useMemo(
        () => queryInputs.find((desc => desc.name === queryInputName)) || queryInputs[0] || {},
        [queryInputs, queryInputName],
    );

    React.useMemo(() => {
        setPageTitle(tableSelector ? `Profiles: ${tableSelector}` : undefined);
    }, [tableSelector]);

    React.useMemo(() => {
        if (!tableSelector) {
            return;
        }
        if (queryInput.queryField === 'selector') {
            // fill selector after switching from another input mode
            setQuery({
                ...query,
                selector: tableSelector,
            });
        } else {
            // do not display an outdated profiles table from the previous input mode
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

    const queryWithSelector = ({ raw }: {raw?: boolean} = {}) => ({
        ...query,
        selector: fixSelector(query.selector ?? tableSelector),
        rawProfile: raw ? 'true' : undefined,
    } as ProfileTaskQuery);

    const submitTask = async ({ raw }: {raw?: boolean} = {}) => {
        const queryToSend = queryWithSelector({ raw });
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

    const renderMergeProfilesButton = () => props.removeMergeButton ? null : (
        <React.Fragment>
            <Button
                onClick={() => submitTask()}
                view="action"
            >
                Merge profiles
                <Hotkey value="cmd+enter" />
            </Button>
            <DropdownMenu popupProps={{ placement: 'bottom-end' }} items={[
                { action: () => submitTask({ raw: true }), text: 'Merge into pprof' },
            ]}/>
        </React.Fragment>
    );

    const handleKeyDown = React.useCallback((event: React.KeyboardEvent<HTMLDivElement>) => {
        if ((event.ctrlKey || event.metaKey) && event.code === 'Enter') {
            if (!props.removeMergeButton) {
                submitTask();
            }
        }
    }, [query, tableSelector]);

    const profileTable = React.useMemo(() => (
        <ProfileTable
            compact={props.compactTable}
            query={{
                selector: fixSelector(tableSelector),
                from: query.from,
                to: query.to,
            }}
        />
    ), [props.compactTable, tableSelector, query.from, query.to]);

    if (props.onRender) {
        props.onRender(queryWithSelector());
    }

    const { diff } = props;

    return (
        <div
            className={`${props.className ? props.className : ''}`}
            tabIndex={-1}
            onKeyDown={handleKeyDown}
        >
            <div className={b(null)}>
                <TimeIntervalInput
                    className={b('time-interval-input', { diff })}
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
