import React from 'react';

import { useNavigate } from 'react-router-dom';

import { CircleInfo } from '@gravity-ui/icons';
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
        return <div className={b('header')}>
            {header}
            <div className={b('additional-items')}>
                <ShareButton className={b('header-button')} size="m" view={'full'} getUrl={() => location.href} />
                <Button view="flat" className={b('header-button')} onClick={()=> {
                    handleInfoDialog();
                }}>
                    <Icon data={CircleInfo}/>
                Info
                </Button>
            </div>
            <MetadataDialog open={isDialogOpen} onClose={() => setIsDialogOpen(false)} task={task}/>
        </div>;
    }

    return <>
        <EditableTaskQuery task={task} additionalHeaderItems={additionalHeaderItems} header={header} embed={embed}/>
        <MetadataDialog open={isDialogOpen} onClose={() => setIsDialogOpen(false)} task={task}/>
    </>;
};

type AdditionalHeaderItemsProps = {
    task: TaskResult | null;
    onOpenInfoDialog: () => void;
    compact?: boolean;
}

const isRawProfile = (task: TaskResult) => getFormat(task.Spec?.MergeProfiles?.Format) === 'RawProfile';

const AdditionalHeaderItems: React.FC<AdditionalHeaderItemsProps> = ({ task, onOpenInfoDialog, compact }) => {
    const spec = task?.Spec?.MergeProfiles;
    const query = spec?.Query;
    const isDiff = isDiffTaskResult(task);
    const navigate = useNavigate();
    const items = [];

    if (task) {
        if (!isDiff && !isRawProfile(task)) {
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
        if (!isDiff) {
            items.push({
                text: 'Compare with',
                action: () => {
                    navigate(`/diff?selector=${query!.Selector}&maxProfiles=${spec?.MaxSamples}`);
                },
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
        <DropdownMenu items={items}
        />
    </>;
};
