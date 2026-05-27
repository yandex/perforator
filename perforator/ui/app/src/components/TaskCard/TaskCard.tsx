import React from 'react';

import { Card, ClipboardButton, DefinitionList, DefinitionListItem, Disclosure } from '@gravity-ui/uikit';

import type { TaskResult } from 'src/models/Task';
import { setPageTitle } from 'src/utils/title';

import './TaskCard.scss';


export interface TaskCardProps {
    taskId: string;
    task: TaskResult | null;
    error?: Error;
}


export const TaskCard: React.FC<TaskCardProps> = ({ task, taskId } )=> {
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


    const properties = [
        ['Baseline Selector', baselineSelector],
        ['Diff Selector', diffSelector],
    ].filter(([_, value]) => Boolean(value));

    return properties.length > 0 ? (
        <Card className="task-card">
            <Disclosure defaultExpanded summary={<h2 className="task-card__title"> Task {taskId}</h2>}>
                <DefinitionList>
                    {properties.map(item => <DefinitionListItem name={item[0]}>{item[1]}</DefinitionListItem>)}
                </DefinitionList>
            </Disclosure>
        </Card>
    ) : null;
};

const Selector: React.FC<{ selector: string }> = ({ selector }) => (
    <>
        <code className="task-card__selector">{selector}</code>
        <ClipboardButton className="task-card__button-copy" size="xs" text={selector} />
    </>
);
