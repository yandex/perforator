import React from 'react';

import { useNavigate } from 'react-router-dom';

import { Button, Card, ClipboardButton, Link } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import type { TaskResult } from 'src/models/Task';
import { TaskState } from 'src/models/Task';
import { setPageTitle } from 'src/utils/title';

import type { DefinitionListItem } from '../DefinitionList/DefinitionList';
import { DefinitionList } from '../DefinitionList/DefinitionList';
import { ShareButton } from '../ShareButton/ShareButton';

import { TaskProgress } from './TaskProgress/TaskProgress';

import './TaskCard.scss';


export interface TaskCardProps {
    taskId: string;
    task: TaskResult | null;
    error?: Error;
}

export const TaskCard: React.FC<TaskCardProps> = props => {
    const { task } = props;
    const navigate = useNavigate();

    const state = task?.Status?.State || TaskState.Unknown;
    const spec = task?.Spec?.MergeProfiles;
    const diffSpec = task?.Spec?.DiffProfiles;
    const baselineQuery = diffSpec?.BaselineQuery;
    const diffQuery = diffSpec?.DiffQuery;
    const query = spec?.Query;
    const traceId = task?.Spec?.TraceBaggage?.Baggage?.traceparent?.match(/^[^-]{2}-([^-]*)-.*/)?.[1];

    React.useMemo(() => {
        if (query?.Selector) {
            setPageTitle(`Profile: ${query?.Selector}`);
        }
        if (baselineQuery?.Selector && diffQuery?.Selector) {
            setPageTitle(`Diff: ${baselineQuery?.Selector} vs ${diffQuery?.Selector}`);
        }
    }, [query, baselineQuery, diffQuery]);

    const querySelector = query?.Selector ? (
        <Selector selector={query.Selector}/>
    ) : null;
    const baselineSelector = baselineQuery?.Selector ? (
        <Selector selector={baselineQuery.Selector}/>
    ) : null;
    const diffSelector = diffQuery?.Selector ? (
        <Selector selector={diffQuery.Selector}/>
    ) : null;

    const renderTraceLink = () => {
        if (!traceId) {
            return null;
        }
        const traceUrl = uiFactory().makeTraceUrl(traceId);
        const link = traceUrl
            ? (
                <Link href={traceUrl} target="_blank">
                    {traceId}
                </Link>
            ) : traceId;
        return (
            <>
                {link}
                <ClipboardButton className="task-card__button-copy" size="xs" text={traceId} />
            </>
        );
    };

    const properties: DefinitionListItem[] = [
        ['Selector', querySelector],
        ['Baseline Selector', baselineSelector],
        ['Diff Selector', diffSelector],
        ['Service', query?.Service],
        [
            'Time interval',
            (
                query?.TimeInterval?.From && query?.TimeInterval?.To
                    ? `from ${query?.TimeInterval?.From ?? '-inf'} to ${query?.TimeInterval?.To ?? 'inf'}`
                    : null
            ),
        ],
        ['Sample count', spec?.MaxSamples],
        ['Baseline sample count', diffSpec?.BaselineQuery?.MaxSamples],
        ['Diff sample count', diffSpec?.DiffQuery?.MaxSamples],
        ['Trace', renderTraceLink()],
        ['Flamegraph format', 'JSONFlamegraph' in (spec?.Format || {}) ? undefined : 'HTML'],
    ];

    return (
        <Card className="task-card">
            <div className="task-card__buttons">
                <ShareButton getUrl={() => window.location.href} />
                {query?.Selector ? (
                    <Button
                        className="task-card__button-compare"
                        href={`/diff?selector=${query.Selector}&maxProfiles=${spec?.MaxSamples}`}
                        onClick={(e) => {
                            if (!e.metaKey && !e.altKey && e.button === 0) {
                                e.preventDefault();
                                navigate(`/diff?selector=${query.Selector}&maxProfiles=${spec?.MaxSamples}`);
                            }
                        }}
                    >
                        {'Compare with\u2026'}
                    </Button>
                ) : null}
            </div>
            <h2 className="task-card__title">Task {props.taskId}</h2>
            <DefinitionList items={properties} />
            <TaskProgress
                state={state}
                error={task?.Status?.Error || props.error?.toString()}
            />
        </Card>
    );
};

const Selector: React.FC<{selector: string}> = ({ selector }) => (
    <>
        <code className="task-card__selector">{selector}</code>
        <ClipboardButton className="task-card__button-copy" size="xs" text={selector} />
    </>
);
