import React from 'react';

import { ClipboardButton } from '@gravity-ui/uikit';

import type { TaskResult } from 'src/models/Task';
import { setPageTitle } from 'src/utils/title';

import type { DefinitionListItem } from '../DefinitionList/DefinitionList';
import { DefinitionList } from '../DefinitionList/DefinitionList';

import './TaskCard.scss';


export interface TaskCardProps {
    taskId: string;
    task: TaskResult | null;
    error?: Error;
}


export const TaskCard: React.FC<TaskCardProps> = ({ task } )=> {
    const spec = task?.Spec?.MergeProfiles;
    const diffSpec = task?.Spec?.DiffProfiles;
    const baselineQuery = diffSpec?.BaselineQuery;
    const diffQuery = diffSpec?.DiffQuery;
    const query = spec?.Query;

    React.useEffect(() => {
        if (query?.Selector) {
            setPageTitle(`Profile: ${query?.Selector}`);
        }
        if (baselineQuery?.Selector && diffQuery?.Selector) {
            setPageTitle(`Diff: ${baselineQuery?.Selector} vs ${diffQuery?.Selector}`);
        }
    }, [query, baselineQuery, diffQuery]);

    const baselineSelector = baselineQuery?.Selector ? (
        <Selector selector={baselineQuery.Selector} />
    ) : null;
    const diffSelector = diffQuery?.Selector ? (
        <Selector selector={diffQuery.Selector} />
    ) : null;


    const properties: DefinitionListItem[] = [
        // ['Selector', querySelector],
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
    ];

    return (
        <div>
            <DefinitionList items={properties} />
        </div>
    );
};

const Selector: React.FC<{ selector: string }> = ({ selector }) => (
    <>
        <code className="task-card__selector">{selector}</code>
        <ClipboardButton className="task-card__button-copy" size="xs" text={selector} />
    </>
);
