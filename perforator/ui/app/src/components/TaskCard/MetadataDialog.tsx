import { ArrowUpRightFromSquare } from '@gravity-ui/icons';
import type { DialogProps } from '@gravity-ui/uikit';
import { Button, ClipboardButton, Dialog, HelpMark, Icon, Link } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import type { TaskStatus } from 'src/generated/perforator/proto/perforator/task_service';
import { type TaskResult } from 'src/models/Task';
import { getFormat, isDiffTaskResult } from 'src/utils/renderingFormat';

import type { DefinitionListItem } from '../DefinitionList/DefinitionList';
import { DefinitionList } from '../DefinitionList/DefinitionList';

import i18n from './i18n';


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
        [i18n('selector'), querySelector],
        [i18n('baselineSelector'), baselineSelector],
        [i18n('diffSelector'), diffSelector],
        [i18n('service'), query?.Service],
        [
            i18n('timeInterval'),
            (
                query?.TimeInterval?.From && query?.TimeInterval?.To
                    ? `${i18n('intervalFromPrefix')}${query?.TimeInterval?.From ?? '-inf'}${i18n('intervalToSeparator')}${query?.TimeInterval?.To ?? 'inf'}`
                    : null
            ),
        ],
        [i18n('profileCount'), spec?.MaxSamples],
        [i18n('baselineProfileCount'), diffSpec?.BaselineQuery?.MaxSamples],
        [i18n('diffProfileCount'), diffSpec?.DiffQuery?.MaxSamples],
        [i18n('trace'), renderTraceLink(traceId)],
        [i18n('flamegraphFormat'), format === 'Flamegraph' ? 'HTML' : undefined],
        [i18n('executor'), getExecutor({ attempts: status?.Attempts })],
    ];
    return <Dialog size="l" open={props.open} onClose={props.onClose}>
        <Dialog.Header caption={i18n('metadataTitle')}/>
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
