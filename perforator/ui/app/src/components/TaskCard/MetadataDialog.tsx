import { ArrowUpRightFromSquare } from '@gravity-ui/icons';
import type { DialogProps } from '@gravity-ui/uikit';
import { Button, ClipboardButton, Dialog, HelpMark, Icon, Link } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import type { TaskStatus } from 'src/generated/perforator/proto/perforator/task_service';
import { type TaskResult } from 'src/models/Task';
import { getFormat, isDiffTaskResult } from 'src/utils/renderingFormat';

import type { DefinitionListItem } from '../DefinitionList/DefinitionList';
import { DefinitionList } from '../DefinitionList/DefinitionList';


export interface MetadataDialogProps extends Pick<DialogProps, 'open' | 'onClose'> {
    task: TaskResult | null;
}

export const MetadataDialog: React.FC<MetadataDialogProps> = ({ task, ...props }) => {
    const status = task?.Status;
    const spec = task?.Spec?.MergeProfiles;
    const diffSpec = task?.Spec?.DiffProfiles;
    const baselineQuery = diffSpec?.BaselineQuery;
    const diffQuery = diffSpec?.DiffQuery;
    const query = spec?.Query;
    const traceId = task?.Spec?.TraceBaggage?.Baggage?.traceparent?.match(/^[^-]{2}-([^-]*)-.*/)?.[1];

    const isDiff = isDiffTaskResult(task);
    const isLegacyFormat = isDiff && 'FlamegraphOptions' in (task?.Spec?.DiffProfiles || {});
    const format = getFormat(spec?.Format) ?? getFormat(diffSpec?.RenderFormat) ?? (isLegacyFormat ? 'Flamegraph' : undefined);

    const querySelector = query?.Selector ? (
        <Selector selector={query.Selector} />
    ) : null;
    const baselineSelector = baselineQuery?.Selector ? (
        <Selector selector={baselineQuery.Selector} />
    ) : null;
    const diffSelector = diffQuery?.Selector ? (
        <Selector selector={diffQuery.Selector} />
    ) : null;


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
        ['Profile count', spec?.MaxSamples],
        ['Baseline profile count', diffSpec?.BaselineQuery?.MaxSamples],
        ['Diff profile count', diffSpec?.DiffQuery?.MaxSamples],
        ['Trace', renderTraceLink(traceId)],
        ['Flamegraph format', format === 'Flamegraph' ? 'HTML' : undefined],
        ['Executor', getExecutor({ attempts: status?.Attempts })],
    ];
    return <Dialog size="l" open={props.open} onClose={props.onClose}>
        <Dialog.Header caption={'Task Metadata'}/>
        <Dialog.Body>
            <DefinitionList items={properties} />
        </Dialog.Body>
    </Dialog>;
};


const getExecutor = ({ attempts }: { attempts?: TaskStatus['Attempts'] }) => {
    const executor = attempts?.[attempts?.length - 1]?.Executor;
    if (!executor) {
        return undefined;
    }

    return <>
        <Executor executor={executor} />
        {attempts.length > 1 ?
            (
                <HelpMark popoverProps={{ className: 'task-card__popover-content' }} >{attempts.map(attempt => {
                    return (
                        <div><Executor executor={attempt.Executor} /></div>
                    );
                })}</HelpMark>
            )
            : null}
    </>;
};

const Selector: React.FC<{ selector: string }> = ({ selector }) => (
    <>
        <code className="task-card__selector">{selector}</code>
        <ClipboardButton className="task-card__button-copy" size="xs" text={selector} />
    </>
);


const Executor: React.FC<{ executor: string }> = ({ executor }) => {
    const href = uiFactory().makeExecutorLink(executor);

    return <>
        <code className="task-card__selector">{executor}</code>
        <ClipboardButton className="task-card__button-copy" size="xs" text={executor} />
        {href ? <Button size="xs" view={'flat'} href={href}>
            <Icon size={12} data={ArrowUpRightFromSquare} />
        </Button> : null}
    </>;
};


const renderTraceLink = (traceId?: string) => {
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
