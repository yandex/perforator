import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { useNavigate, useSearchParams } from 'react-router-dom';

// import type { ActionItem } from '@gravity-ui/components/build/esm/components/ActionsPanel/types';
import type {
    ActionsPanelProps,
    LabelProps,
    SelectOption,
    TableColumnConfig,
    TableSettingsData } from '@gravity-ui/uikit';
import { ActionsPanel,
    Button,
    ClipboardButton,
    Flex,
    Label,
    Loader,
    Select,
    Switch,
    Table,
    Text,
    TextInput,
    withTableCopy,
    withTableSelection,
    withTableSettings,
} from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import type { RenderFormat } from 'src/generated/perforator/proto/perforator/perforator';
import type {
    Task,
} from 'src/generated/perforator/proto/perforator/task_service';
import {
    TaskState } from 'src/generated/perforator/proto/perforator/task_service';
import { formatDate } from 'src/utils/date';
import { getUserLogin } from 'src/utils/login';
import { redirectToTaskPage } from 'src/utils/profileTask';
import { composeDiffQuery } from 'src/utils/selector';

import { ErrorPanel } from '../ErrorPanel/ErrorPanel';
import { Link } from '../Link/Link';
import { type TimeInterval, TimeIntervalInput } from '../TimeIntervalInput/TimeIntervalInput';

import { useTasksQuery } from './queries';

import './Tasks.scss';


type ActionItem = ActionsPanelProps['actions'][number]

const DEFAULT_FROM = 'now-6M';

function statusToLabelTheme(state: TaskState | undefined): LabelProps['theme'] {
    switch (state) {
    case TaskState.Failed:
        return 'danger';
    case TaskState.Finished:
        return 'success';
    default:
        return 'info';
    }
}

const CopyWrapper: React.FC<{ children: string }> = ({ children }) => (
    <div>{children}<ClipboardButton text={children} size={'xs'} /></div>
);

function getSelectorFromSpec(task: Task) {
    const s = task.Spec;
    if (!s) {
        return null;
    }

    if ('MergeProfiles' in s && s.MergeProfiles?.Query?.Selector) {
        return <CopyWrapper>{s.MergeProfiles.Query.Selector}</CopyWrapper>;
    }
    if (
        'DiffProfiles' in s &&
        s.DiffProfiles?.BaselineQuery?.Selector &&
        s.DiffProfiles?.DiffQuery?.Selector
    ) {
        return (
            <React.Fragment>
                <CopyWrapper>
                    {s.DiffProfiles?.BaselineQuery?.Selector}
                </CopyWrapper>
                <CopyWrapper>{s.DiffProfiles?.DiffQuery?.Selector}</CopyWrapper>
            </React.Fragment>
        );
    }

    return null;
}

const TaskTable = React.memo(withTableSelection(
    withTableSettings(withTableCopy(Table<Task>)),
));

const TaskStates: TaskState[] = [
    TaskState.Created,
    TaskState.Failed,
    TaskState.Finished,
    TaskState.Running,
];
const taskStateOptions: SelectOption[] = TaskStates.map((state) => ({
    children: state,
    value: state,
}));

const taskTypeOptions: SelectOption[] = [
    {
        children: 'MergeProfiles',
        value: 'MergeProfiles',
    },
    {
        children: 'DiffProfiles',
        value: 'DiffProfiles',
    },
];

const taskFormatOptions: SelectOption[] = [
    { children: 'HTMLFlamegraph', value: 'HTMLFlamegraph' },
    { children: 'Flamegraph', value: 'Flamegraph' },
    { children: 'RawProfile', value: 'RawProfile' },
];

const typeTemplate = (spec: Task['Spec']) => {
    if (!spec) {
        return null;
    }
    if ('MergeProfiles' in spec) {
        return 'MergeProfiles';
    }

    if ('DiffProfiles' in spec) {
        return 'DiffProfiles';
    }

    return null;
};

const readFormat = (format: RenderFormat | undefined) => {
    if (!format) {
        return null;
    }
    if ('JSONFlamegraph' in (format)) {
        return 'Flamegraph';
    }

    if ('Flamegraph' in format) {
        return 'HTMLFlamegraph';
    }

    if ('RawProfile' in format) {
        return 'RawProfile';
    }

    if ('TextProfile' in format) {
        return 'TextProfile';
    }

    return 'Unknown format';
};

const getFormat = (spec: Task['Spec']) => {
    if (!spec) {
        return null;
    }
    if ('MergeProfiles' in spec) {
        return readFormat(spec.MergeProfiles?.Format);
    }

    if ('DiffProfiles' in spec) {
        return 'Flamegraph';
    }

    return null;
};

const getColumnsConfig = (): TableColumnConfig<Task>[] => [
    {
        id: 'Status',
        template: (data) => (
            <Label theme={statusToLabelTheme(data.Status?.State)}>
                {data.Status?.State}
            </Label>
        ),
    },
    {
        id: 'ID',
        template: (profileData) => (
            <Link href={`/task/${profileData?.Meta?.ID}`}>
                {profileData?.Meta?.ID}
            </Link>
        ),
    },
    {
        id: 'Type',
        template: (profileData) => typeTemplate(profileData.Spec),
    },
    {
        id: 'CreationTime',
        template: (profileData) =>
            formatDate(
                Number(profileData?.Meta?.CreationTime) / 1000,
                'YYYY-MM-DD HH:mm:ss',
            ),
    },
    ...(uiFactory().authorizationSupported() ? [{
        id: 'Author',
        template: (profileData: Task) => uiFactory().renderUserLink(profileData.Meta?.Author),
    }] : []),
    {
        id: 'Query',
        template: (profileData) => {
            const text = getSelectorFromSpec(profileData);
            return <Text variant="code-1">{text}</Text>;
        },
    },
    {
        id: 'Format',
        template: (data) => getFormat(data.Spec),
    },
    {
        id: 'ProfileCount',
        template: (data) => data.Result?.MergeProfiles?.ProfileMeta?.length,
    },
    {
        id: 'ProfilesTimes',
        template: (data) => {
            const times = data.Result?.MergeProfiles?.ProfileMeta.map(
                (profile) => profile.Timestamp,
            ).sort();

            const first = times?.[0];
            const last = times?.[times?.length - 1];

            if (!first && !last) {
                return null;
            }

            return `${first} – ${last}`;
        },
    },
];

type TasksQuery = { from?: string; to?: string; user?: string }

const USER_FILTER_DEBOUNCE_MS = 300;

const taskRowDescriptor = (task: Task) => ({ id: task?.Meta?.ID });

interface TasksProps {
    header: React.ReactElement;
}

export const Tasks: React.FC<TasksProps> = ({ header }) => {
    const [selected, setSelected] = useState<string[]>([]);

    const [stateParams, setState] = useSearchParams();
    const state: TasksQuery = {
        from: DEFAULT_FROM,
        ...Object.fromEntries(stateParams),
    };

    const navigate = useNavigate();
    const login = useMemo(() => getUserLogin(), []);

    const [settings, setSettings] = useState<TableSettingsData>([]);
    const [statusFilter, setStatusFilter] = useState<string[]>([]);
    const [typeFilter, setTypeFilter] = useState<string[]>([]);
    const [formatFilter, setFormatFilter] = useState<string[]>([]);
    const [mine, setMine] = useState<boolean>(
        state.user ? login === state.user : true,
    );
    const [userFilter, setUserFilter] = useState<string>(
        state.user || login || '',
    );

    const debouncedTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const [debouncedUserFilter, setDebouncedUserFilter] = useState(userFilter);
    useEffect(() => {
        if (debouncedTimerRef.current) {
            clearTimeout(debouncedTimerRef.current);
            debouncedTimerRef.current = null;
        }
        debouncedTimerRef.current = setTimeout(
            () => setDebouncedUserFilter(userFilter),
            USER_FILTER_DEBOUNCE_MS,
        );
        return () => {
            if (debouncedTimerRef.current) {
                clearTimeout(debouncedTimerRef.current);
                debouncedTimerRef.current = null;
            }
        };}, [userFilter]);

    const {
        data,
        error,
        isPending,
        hasNextPage,
        fetchNextPage,
        isFetchingNextPage,
    } = useTasksQuery({
        from: state.from,
        to: state.to,
        user: debouncedUserFilter,
    });

    const tasks = useMemo(
        () => (data ? data.pages.flatMap((page) => page.Tasks ?? []) : null),
        [data],
    );

    const [selectedProfile, selectedDiffProfile] = useMemo(() => {
        const [diffTaskId, baselineTaskId] = selected;
        return [
            tasks?.find((task) => task.Meta?.ID === diffTaskId),
            tasks?.find((task) => task.Meta?.ID === baselineTaskId),
        ];
    }, [selected, tasks]);

    const handleDiff = useCallback(async () => {
        const baseSpec = selectedProfile?.Spec;
        const diffSpec = selectedDiffProfile?.Spec;
        if (!diffSpec || !baseSpec) {
            return;
        }
        if (!baseSpec.MergeProfiles?.Query) {
            return;
        }
        if (!('MergeProfiles' in diffSpec) || !diffSpec.MergeProfiles?.Query) {
            return;
        }

        const baseQuery = baseSpec.MergeProfiles.Query;
        const diffQuery = diffSpec.MergeProfiles.Query;

        const query = composeDiffQuery(
            //@ts-ignore
            {
                selector: baseQuery.Selector,
                maxProfiles: baseQuery.MaxSamples,
            },
            {
                selector: diffQuery.Selector,
                maxProfiles: diffQuery.MaxSamples,
            },
        );

        redirectToTaskPage(navigate, query);
    }, [selectedProfile?.Spec, selectedDiffProfile?.Spec, navigate]);

    const updateTimeInterval = (interval: TimeInterval) => {
        setState({
            ...state,
            from: interval.start,
            to: interval.end,
            ...(login ? { user: login } : {}),
        } as Record<string, string>);
    };

    const handleSetUserFilter = (value: string) => {
        setMine(value === login);

        setUserFilter(value);
        setState({ ...state, user: value }, { replace: true });
    };
    const handleMine = (value: boolean) => {
        if (value) {
            setMine(true);
            setUserFilter(login || '');
            setState({ ...state, user: login || '' });
        } else {
            setMine(false);
            setUserFilter('');
            setState({ ...state, user: '' });
        }
    };
    const filteredTasks = useMemo(() => {
        return tasks?.filter(
            (task) =>
                (!statusFilter.length ||
                    statusFilter.includes(task.Status?.State as string)) &&
                (!typeFilter.length ||
                    typeFilter.includes(typeTemplate(task.Spec) as string)) &&
                (!formatFilter.length ||
                    formatFilter.includes(getFormat(task.Spec) as string)),
        );
    }, [tasks, statusFilter, typeFilter, formatFilter]);

    const errorMessage = useMemo(() => {
        const validators = [
            () =>
                selected.length === 2
                    ? undefined
                    : 'Diff can be calculated only for two profiles',
            () =>
                'MergeProfiles' in (selectedProfile?.Spec || {}) &&
                    'MergeProfiles' in (selectedDiffProfile?.Spec || {})
                    ? undefined
                    : 'Diff can be calculated only for MergeProfiles',
            () =>
                selectedProfile?.Status?.State === TaskState.Finished &&
                    selectedDiffProfile?.Status?.State === TaskState.Finished
                    ? undefined
                    : 'Diff can be calculated only for finished tasks',
        ];


        for (const validator of validators) {
            const err = validator();
            if (err) {
                return err;
            }
        }

        return null;
    }, [selected.length, selectedDiffProfile, selectedProfile]);

    const actions: ActionItem[] = useMemo(() => {
        return [
            {
                id: 'diff-profiles',
                button: {
                    props: {
                        children: 'Diff profiles',
                        onClick: handleDiff,
                        disabled: Boolean(errorMessage),
                    },
                },
                dropdown: {
                    item: {
                        action: handleDiff,
                        text: 'Diff profiles',
                        disabled: Boolean(errorMessage),
                    },
                },
            },
        ];
    }, [errorMessage, handleDiff]);

    const renderUserInput = () => !uiFactory().authorizationSupported() ? null : (
        <>
            <TextInput
                placeholder={'login'}
                onUpdate={handleSetUserFilter}
                value={userFilter}
                size={'m'}
                className={'tasks__user-input'}
            />
            <Switch
                checked={mine}
                onUpdate={handleMine}
                content={'Show only mine'}
            />
        </>
    );

    const columnsConfig = React.useMemo(() => getColumnsConfig(), []);

    return <React.Fragment>
        <TimeIntervalInput
            headerControls
            initInterval={{
                start: state.from || DEFAULT_FROM,
                end: state.to || 'now',
            }}
            onUpdate={updateTimeInterval}
            header={header}
        />
        <Flex gap={4} alignItems={'center'}>
            <Select
                qa="task-state-filter"
                placeholder={'Task State'}
                label={'Task State'}
                value={statusFilter}
                multiple
                hasClear
                options={taskStateOptions}
                onUpdate={setStatusFilter}
                disabled={false}
            />
            <Select
                qa="task-type-filter"
                placeholder={'Task Type'}
                label="Task Type"
                multiple
                options={taskTypeOptions}
                value={typeFilter}
                onUpdate={setTypeFilter}
                hasClear
            />
            <Select
                qa="task-format-filter"
                placeholder={'Task Format'}
                label="Task Format"
                multiple
                options={taskFormatOptions}
                value={formatFilter}
                onUpdate={setFormatFilter}
            />
            {renderUserInput()}
        </Flex>
        {/* eslint-disable-next-line no-nested-ternary */}
        {tasks ? (
            <React.Fragment>
                {selected.length !== 0 ? (
                    <ActionsPanel
                        className={'tasks-table__actions-panel'}
                        actions={actions}
                        renderNote={errorMessage ? () => errorMessage : undefined}
                    />
                ) : null}
                <TaskTable
                    data={filteredTasks!}
                    selectedIds={selected}
                    onSelectionChange={setSelected}
                    settings={settings}
                    updateSettings={setSettings}
                    getRowDescriptor={taskRowDescriptor}
                    className="tasks-table"
                    columns={columnsConfig}
                />
                {hasNextPage
                    ? <><Button loading={isFetchingNextPage} className='tasks__pagination' onClick={() => fetchNextPage()}>Load more</Button>
                        {error ? <Text className="tasks__pagination-error" color="danger">Error: {error?.message}</Text> : null}
                    </>
                    : null}
            </React.Fragment>
        ) : error ? (
            <ErrorPanel message={error?.message} />
        ) : isPending ? (
            <Loader />
        ) : null
        }
    </React.Fragment>;
};
