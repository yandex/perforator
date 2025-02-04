import { useEffect, useMemo, useState } from 'react';

import { AxiosError } from 'axios';

import ArrowUpRightFromSquareIcon from '@gravity-ui/icons/svgs/arrow-up-right-from-square.svg?raw';
import type { TableSortState } from '@gravity-ui/uikit';
import {
    Icon,
    Link,
    Loader,
    Table,
    withTableCopy,
    withTableSorting,
} from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import type { ProfileMeta } from 'src/generated/perforator/proto/perforator/perforator';
import { apiClient } from 'src/utils/api';
import { formatDate, getIsoDate, parseDate } from 'src/utils/date';

import { ErrorPanel } from '../ErrorPanel/ErrorPanel';
import { Link as RouterLink } from '../Link/Link';

import './ProfileTable.scss';


export interface ProfileTableQuery {
    selector?: string;
    from?: string;
    to?: string;
}

export interface ProfileTableProps {
    query: ProfileTableQuery;
    compact?: boolean;
}

interface Profile extends ProfileMeta {
    HumanReadableTimestamp?: string;
}

const CompactLink: React.FC<{href: string; routerLink?: boolean}> = ({ href, routerLink }) =>{
    const LocalLink = routerLink ? RouterLink : Link;
    return (
        <LocalLink href={href}>
            <Icon data={ArrowUpRightFromSquareIcon} />
        </LocalLink>
    );
};


const prepareProfileColumns = ({ compact }: { compact?: boolean } = {}) => {
    const renderLink = (href: Optional<string>, text: Optional<string>, localLink?: boolean) => {
        if (!text) {
            return null;
        }
        if (!href) {
            return text;
        }
        const LocalLink = localLink ? RouterLink : Link;
        return (
            compact ? (
                <CompactLink href={href} routerLink={localLink} />
            ) : (
                <LocalLink view="normal" href={href}>
                    {text}
                </LocalLink>
            )
        );
    };

    return ([
        {
            id: 'HumanReadableTimestamp',
            name: 'Time',
            meta: { sort: true, defaultSortOrder: 'desc', id: 'timestamp' },
        },
        {
            id: 'DC',
            name: uiFactory().clusterName(),
            template: (profile: Profile) => {
                const cluster = profile.Cluster ?? uiFactory().defaultCluster();
                return <span>{cluster}</span>;
            },
        },
        {
            id: 'Service',
            name: uiFactory().serviceName(),
            template: (profile: Profile) => {
                const cluster = profile.Cluster ?? uiFactory().defaultCluster();
                const service = profile.Service;
                const href = uiFactory().makeServiceUrl(cluster, service);
                return renderLink(href, service);
            },
            meta: {
                copy: compact ? false : (profile: Profile) => profile.Service,
            },
        },
        {
            id: 'PodID',
            name: uiFactory().podName(),
            template: (profile: Profile) => {
                const cluster = profile.Cluster ?? uiFactory().defaultCluster();
                const pod = profile.PodID ?? profile.Attributes.pod;
                const href = uiFactory().makePodUrl(cluster, pod);
                return renderLink(href, pod);
            },
            meta: {
                copy: compact ? false : (profile: Profile) => profile.PodID ?? profile.Attributes.pod,
            },
        },
        {
            id: 'NodeID',
            name: uiFactory().nodeName(),
            template: (profile: Profile) => {
                const cluster = profile.Cluster ?? uiFactory().defaultCluster();
                const node = profile.NodeID ?? profile.Attributes.host;
                const href = uiFactory().makeNodeUrl(cluster, node);
                return renderLink(href, node);
            },
            meta: {
                copy: compact ? false : (profile: Profile) =>
                    profile.NodeID ?? profile.Attributes.host,
            },
        },
        {
            id: 'ProfileID',
            name: 'Profile ID',
            template: (profile: Profile) => {
                const href = `/profile/${profile.ProfileID}?timestamp=${parseDate(profile.Timestamp ?? '')!.valueOf()}`;
                return renderLink(href, profile.ProfileID, true);
            },
        },
    ]);
};

const rowDescriptor = (profile: Profile) => ({ id: profile.ProfileID });

export function ProfileTable({ query, compact }: ProfileTableProps) {
    const [data, setData] = useState<Profile[]>([]);
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);
    const [sortState, setSortState] = useState<TableSortState>([{ column: 'HumanReadableTimestamp', order: 'desc' }]);

    const offset = 0;
    const limit = 100;

    const columns = useMemo(() =>(prepareProfileColumns({ compact })), [compact]);

    useEffect(() => {
        const fetchData = async () => {
            setData([]);
            setError(null);

            if (!query.selector) {
                setData([]);
                return;
            }

            const params: any = {
                'Query.Selector': query.selector,
                'Query.TimeInterval.From': getIsoDate(query.from ?? ''),
                'Query.TimeInterval.To': getIsoDate(query.to ?? ''),
                'Paginated.Offset': offset,
                'Paginated.Limit': limit,
            };

            if (sortState.length) {
                const column = columns.find((x) => x.id === sortState[0].column)?.meta?.id;
                params['OrderBy.Direction'] = sortState[0].order === 'asc' ? 'Ascending' : 'Descending';
                params['OrderBy.Columns'] = column;
            }

            try {
                setLoading(true);
                const response = await apiClient.getProfiles(params);

                const profiles = response?.data?.Profiles?.map((profile) => {
                    const timestamp = formatDate(profile.Timestamp ?? '', 'YYYY-MM-DD HH:mm:ss');
                    return { ...profile, HumanReadableTimestamp: timestamp };
                });

                setData(profiles);
            } catch (e: unknown) {
                setError(e instanceof AxiosError ? e?.response?.data?.message : String(e));
            } finally {
                setLoading(false);
            }

        };

        fetchData();
    }, [query, sortState]);

    const SortedTable = useMemo(() => compact ? withTableSorting(Table<Profile>) : withTableCopy(withTableSorting(Table<Profile>)), [compact]);

    if (loading) {
        return <Loader />;
    }

    if (error) {
        return <ErrorPanel message={error} />;
    }

    return (
        <SortedTable
            columns={columns}
            data={data}
            getRowDescriptor={rowDescriptor}
            className="profiles-table"
            sortState={sortState}
            onSortStateChange={setSortState}
        />
    );
}
