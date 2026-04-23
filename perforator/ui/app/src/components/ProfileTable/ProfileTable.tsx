import { useEffect, useMemo, useRef, useState } from 'react';

import { AxiosError } from 'axios';

import { ArrowUpRightFromSquare } from '@gravity-ui/icons';
import type { PaginationProps, TableSortState } from '@gravity-ui/uikit';
import {
    Icon,
    Link,
    Loader,
    Pagination,
    Table,
    withTableCopy,
    withTableSorting,
} from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import type { ListProfilesRequest, ProfileMeta } from 'src/generated/perforator/proto/perforator/perforator';
import { SortDirection } from 'src/models/Sort';
import { apiClient, getPagination } from 'src/utils/api';
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

const CompactLink: React.FC<{ href: string; routerLink?: boolean }> = ({ href, routerLink }) => {
    const LocalLink = routerLink ? RouterLink : Link;
    return (
        <LocalLink href={href}>
            <Icon data={ArrowUpRightFromSquare} />
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
                const href = `/profile/${profile.ProfileID}?timestamp=${parseDate(profile.Timestamp ?? '')!.valueOf()}&event_type=${profile.EventType}&service=${profile.Service}`;
                return renderLink(href, profile.ProfileID, true);
            },
        },
    ]);
};

const rowDescriptor = (profile: Profile) => ({ id: profile.ProfileID });

const initialPaginationState = { page: 1, pageSize: 100 };
export function ProfileTable({ query, compact }: ProfileTableProps) {
    const prevQueryRef = useRef<ProfileTableQuery | null>(null);
    const [data, setData] = useState<Profile[]>([]);
    const [hasMore, setHasMore] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);
    const [sortState, setSortState] = useState<TableSortState>([{ column: 'HumanReadableTimestamp', order: 'desc' }]);

    const [paginationState, setPaginationState] = useState(initialPaginationState);

    useEffect(() => {
        const newQuery = prevQueryRef.current !== null && prevQueryRef.current !== query;
        if (newQuery) {
            prevQueryRef.current = query;
            setPaginationState(initialPaginationState);
        }
    }, [query]);
    const handleUpdate: PaginationProps['onUpdate'] = (page, pageSize) =>
        setPaginationState((prevState) => ({ ...prevState, page, pageSize }));

    const columns = useMemo(() => (prepareProfileColumns({ compact })), [compact]);

    useEffect(() => {
        const controller = new AbortController();

        const fetchData = async () => {
            setData([]);
            setError(null);

            let selector;
            if (query.selector) {
                selector = query.selector;
            } else {
                selector = '{}';
            }

            const params: ListProfilesRequest = {
                Query: {
                    Selector: selector,
                    TimeInterval: {
                        From: getIsoDate(query.from ?? ''),
                        To: getIsoDate(query.to ?? ''),
                    },
                    // this is useless, is only here to make typescript happy
                    MaxSamples: 0,
                },
                Paginated: getPagination(paginationState),
                OrderBy: {
                    Direction: SortDirection.Descending,
                    // @ts-ignore
                    Columns: 'timestamp',
                },
            };

            if (sortState.length) {
                const column = columns.find((x) => x.id === sortState[0].column)?.meta?.id;
                if (params.OrderBy && column) {
                    params.OrderBy.Direction = sortState[0].order === 'asc' ? SortDirection.Ascending : SortDirection.Descending;
                    // @ts-ignore
                    params.OrderBy.Columns = column;
                }
            }

            try {
                setLoading(true);
                const response = await apiClient.getProfiles(params, { signal: controller.signal });

                const profiles = response?.data?.Profiles?.map((profile) => {
                    const timestamp = formatDate(profile.Timestamp ?? '', 'YYYY-MM-DD HH:mm:ss');
                    return { ...profile, HumanReadableTimestamp: timestamp };
                });

                setData(profiles);
                const more = Boolean(response?.data?.HasMore);
                setHasMore(more);
            } catch (e: unknown) {
                if (e instanceof AxiosError && e.code === 'ERR_CANCELED') {
                    return;
                }
                setError(e instanceof AxiosError ? e?.response?.data?.message : String(e));
            } finally {
                setLoading(false);
            }

        };

        fetchData();

        return () => controller.abort();
    }, [query, sortState, paginationState, columns]);

    const SortedTable = useMemo(() => compact ? withTableSorting(Table<Profile>) : withTableCopy(withTableSorting(Table<Profile>)), [compact]);

    if (loading) {
        return <Loader />;
    }

    if (error) {
        return <ErrorPanel message={error} />;
    }

    return (
        <>
            <SortedTable
                columns={columns}
                data={data}
                getRowDescriptor={rowDescriptor}
                className="profiles-table"
                sortState={sortState}
                onSortStateChange={setSortState}
            />
            <Pagination
                className="profiles-table_pagination"
                page={paginationState.page}
                pageSize={paginationState.pageSize}
                total={(paginationState.page + (hasMore ? 1 : 0)) * paginationState.pageSize}
                showPages={false}
                size="s"
                onUpdate={handleUpdate}
            />
        </>
    );
}
