import React from 'react';

import { useNavigate } from 'react-router-dom';

import { CircleInfo } from '@gravity-ui/icons';
import type { DropdownMenuItem } from '@gravity-ui/uikit';
import { Button, DropdownMenu, Icon } from '@gravity-ui/uikit';

import type { ProfileTaskQuery, TaskResult } from 'src/models/Task';
import { cn } from 'src/utils/cn';
import { redirectToTaskPage } from 'src/utils/profileTask';
import { getFormat, isDiffTaskResult } from 'src/utils/renderingFormat';

import { ShareButton } from '../ShareButton/ShareButton';

import { EditableTaskQuery } from './EditableTaskQuery/EditableTaskQuery';
import { MetadataDialog } from './MetadataDialog';

import './TaskHeader.scss';


const b = cn('task-header');


export interface TaskHeaderProps {
        task: TaskResult | null;
        header: React.ReactElement;
        embed?: boolean;
}

export const TaskHeader: React.FC<TaskHeaderProps> = ({ task, embed, header }) => {
    const isDiff = isDiffTaskResult(task);
    const [isDialogOpen, setIsDialogOpen] = React.useState(false);
    const handleInfoDialog = React.useCallback(() => setIsDialogOpen(true), [setIsDialogOpen]);
    const additionalHeaderItems = React.useMemo(
        () =>
            embed ? undefined : (
                <AdditionalHeaderItems
                    task={task}
                    onOpenInfoDialog={handleInfoDialog}
                />
            ),
        [handleInfoDialog, task, embed],
    );
    if (isDiff) {
        return <>
            <MinimalHeader header={header} additionalHeaderItems={additionalHeaderItems}/>
            <MetadataDialog open={isDialogOpen} onClose={() => setIsDialogOpen(false)} task={task}/>
        </>;
    }

    return <>
        <EditableTaskQuery task={task} additionalHeaderItems={additionalHeaderItems} header={header} embed={embed}/>
        <MetadataDialog open={isDialogOpen} onClose={() => setIsDialogOpen(false)} task={task}/>
    </>;
};

interface MinimalHeaderProps {
    header: React.ReactElement;
    additionalHeaderItems?: React.ReactElement;
}

export const MinimalHeader: React.FC<MinimalHeaderProps> = ({ additionalHeaderItems, header }) => (
    <div className={b('header')}>
        {header}
        <div className={b('additional-items')}>
            {additionalHeaderItems}
        </div>
    </div>
);

type AdditionalHeaderItemsProps = {
    task: TaskResult | null;
    onOpenInfoDialog: () => void;
    compact?: boolean;
}

const isRawProfile = (task: TaskResult) => getFormat(task.Spec?.MergeProfiles?.Format) === 'RawProfile';

const AdditionalHeaderItems: React.FC<AdditionalHeaderItemsProps> = ({ task, onOpenInfoDialog, compact }) => {
    const isMergeTask = 'MergeProfiles' in (task?.Spec ?? {});
    const spec = task?.Spec?.MergeProfiles;
    const query = spec?.Query;
    const navigate = useNavigate();
    const items: DropdownMenuItem[] = [];

    if (task) {
        if (isMergeTask && !isRawProfile(task)) {
            items.push(
                {
                    text: 'Get pprof',
                    action: () => redirectToTaskPage(navigate, {
                        selector: query?.Selector,
                        maxProfiles: spec?.MaxSamples,
                        rawProfile: 'true',
                    } as ProfileTaskQuery),
                },
            );
        }
        if (isMergeTask) {
            items.push({
                text: 'Compare with',
                action: () => {
                    navigate(`/diff?selector=${query!.Selector}&maxProfiles=${spec?.MaxSamples}`);
                },
            });
        }

        const taskUrl = task.Result?.MergeProfiles?.ProfileURL;
        if (isMergeTask && getFormat(task.Spec?.MergeProfiles?.Format) === 'HTMLVisualisation' && taskUrl) {
            items.push({
                text: 'Download html report',
                href: taskUrl,
            });
        }
    }

    return <>
        <ShareButton className={b('header-button')} size="m" view={compact ? 'compact' : 'full'} getUrl={() => location.href} />
        <Button view="flat" className={b('header-button')} onClick={()=> {
            onOpenInfoDialog();
        }} >
            <Icon data={CircleInfo}/>
                Info
        </Button>
        {items.length > 0 ? <DropdownMenu items={items}/> : null}
    </>;
};
