import { useCallback, useEffect, useMemo, useState } from 'react';

import { useQueryClient } from '@tanstack/react-query';

import { Flame } from '@gravity-ui/icons';
import type { Cell, ColumnDef, Header } from '@gravity-ui/table';
import { Table as GravityTable, TreeExpandableCell, useTable } from '@gravity-ui/table';
import type { ExpandedState, OnChangeFn, SortingState } from '@gravity-ui/table/tanstack';
import { getCoreRowModel, getExpandedRowModel, getSortedRowModel } from '@gravity-ui/table/tanstack';
import type { ProgressColorStops } from '@gravity-ui/uikit';
import { Button, ClipboardButton, HelpMark, Icon, Loader, Progress, TextInput, Tooltip } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import type { ClusterTopEntry, ClusterTopGenerationStatus } from 'src/generated/perforator/proto/perforator/perforator';
import { apiClient } from 'src/utils/api';
import { cn } from 'src/utils/cn';
import { useTypedQuery } from 'src/utils/query';

import { ErrorPanel } from '../ErrorPanel/ErrorPanel';

import { clusterTopServicesQueryKeys, getGenerationCachePolicy, useFunctionTopQuery } from './queries';
import type { ClusterTopRow } from './utils';
import { convertToFunctionRow as rawConvertToFunctionRow, convertToServiceRow as rawConvertToServiceRow } from './utils';

import './ClusterTopTable.css';


const LOADING_STRING = 'Loading...';

const b = cn('cluster-top-table');

const nameColumn = b('name');

const selfTimeColorStops: ProgressColorStops[] = [
    { stop: 1, theme: 'success' },
    { stop: 2, theme: 'warning' },
    { stop: 5, theme: 'danger' },
];

const totalTimeColorStops: ProgressColorStops[] = [
    { stop: 10, theme: 'success' },
    { stop: 50, theme: 'warning' },
    { stop: 90, theme: 'danger' },
];

function progressCell(value: number, text: string, colorStops: ProgressColorStops[]) {
    if (value === 0) {
        return '—';
    }

    // everything < 0.5 is visually indistinguishable from 0
    const progressValue = value > 0 ? Math.max(value, 1) : 0;

    return <Progress
        text={<span className={b('progress-text')}>{text}</span>}
        size="m"
        className={b('progress')}
        value={progressValue}
        colorStops={colorStops}
    />;
}

type ClusterTopOrderColumnId = 'Count.Self' | 'Count.Cumulative';

type ClusterTopOrderBy = 'self_time' | 'cumulative_time';

const DEFAULT_SORTING: SortingState = [{ id: 'Count.Self', desc: true }];

const ORDER_BY_BY_COLUMN_ID: Record<ClusterTopOrderColumnId, ClusterTopOrderBy> = {
    'Count.Self': 'self_time',
    'Count.Cumulative': 'cumulative_time',
};

function isClusterTopOrderColumnId(id: string): id is ClusterTopOrderColumnId {
    return id in ORDER_BY_BY_COLUMN_ID;
}

function getOrderBy(sorting: SortingState): ClusterTopOrderBy {
    const columnId = sorting[0]?.id;

    return columnId && isClusterTopOrderColumnId(columnId) ? ORDER_BY_BY_COLUMN_ID[columnId] : ORDER_BY_BY_COLUMN_ID['Count.Self'];
}

function isUnknownFunction(name?: string): boolean {
    return name?.includes?.('UNKNOWN') ?? false;
}

const getColumns: (args: Pick<ClusterTopTableProps, 'timeIntervalFrom' | 'timeIntervalTo'>) => ColumnDef<ClusterTopRow, string | number>[] = ({ timeIntervalFrom, timeIntervalTo }) => [
    {
        id: 'Name',
        accessorKey: 'Name',
        header: 'Name',
        size: 600,
        enableSorting: false,
        cell: ({ row, getValue }) => {
            if (row.original.type === 'function') {
                return <TreeExpandableCell row={row}>{
                    <>
                        <span className={b('function-name', nameColumn)}>
                            {getValue<string>()}
                            <ClipboardButton text={row.original.Name} size="xs"/>
                        </span>
                    </>
                }</TreeExpandableCell>;
            }

            if (row.original.type === 'service' && row.original.Name !== LOADING_STRING) {
                const query = new URLSearchParams({
                    service: row.original.Name,
                } as Record<string, string>);
                if (row.original.parentFunction && !isUnknownFunction(row.original.parentFunction)) {
                    query.append('flamegraphQuery', row.original.parentFunction);
                    query.append('exactMatch', 'true');
                    query.append('keepOnlyFound', 'true');
                }
                if (timeIntervalFrom && timeIntervalTo) {
                    query.append('from', timeIntervalFrom);
                    query.append('to', timeIntervalTo);
                }
                return <>
                    <span className={b('service-name', nameColumn)}>{getValue<string>()}
                        <ClipboardButton text={row.original.Name} size="xs"/>
                        <Tooltip content="Build a service flamegraph and search for this function">
                            <Button onClick={() => {uiFactory().reachGoal('FLAME_FROM_CLUSTER_TOP');}} target={'_blank'} view={'flat'} size={'xs'} href={`/build?${query.toString()}`}>
                                <Icon height={14} size={14} data={Flame} className={b('icon', { flamegraph: true })}/>
                            </Button>
                        </Tooltip>
                    </span>
                </>;
            }

            return <span className={b('service-name', nameColumn)}>{getValue<string>()}</span>;
        },
    },
    {
        id: 'Count.Self',
        accessorFn: (row) => row.Count.Self,
        header: () => <>Self, Cores <HelpMark>Estimated CPU time spent in function</HelpMark></>,
        size: 100,
        enableSorting: true,
        sortDescFirst: true,
        cell: ({ row, getValue }) => progressCell(
            row.original.Count.SelfPctValue,
            `${getValue<number>()} (${row.original.Count.SelfPct})`,
            selfTimeColorStops,
        ),
    },
    {
        id: 'Count.Cumulative',
        accessorFn: (row) => row.Count.Cumulative,
        header: () => <>Total, Cores <HelpMark>Estimated CPU time spent in function and its children</HelpMark></>,
        size: 100,
        enableSorting: true,
        sortDescFirst: true,
        cell: ({ row, getValue }) => progressCell(
            row.original.Count.CumulativePctValue,
            `${getValue<number>()} (${row.original.Count.CumulativePct})`,
            totalTimeColorStops,
        ),
    },
];

function cellCn(cell?: Cell<ClusterTopRow, unknown>) {
    return b('cell', { count: cell?.column.id.includes('Count') });
}

function headerCn(header: Header<ClusterTopRow, unknown>) {
    return b('header', { count: header.column.id.includes('Count') });
}

const EmptyView = () => {
    return <div className={'cluster-top-table__empty'}><Loader className={b('loader')}/></div>;
};

interface ClusterTopTableProps {
    generation: number;
    timeInterval: number;
    timeIntervalFrom?: string;
    timeIntervalTo?: string;
    generationStatus: ClusterTopGenerationStatus;
}

export const ClusterTopTable: React.FC<ClusterTopTableProps> = ({ generation, timeInterval, timeIntervalFrom, timeIntervalTo, generationStatus }) => {
    const queryClient = useQueryClient();
    const [data, setData] = useState<ClusterTopRow[]>([]);
    const [expanded, setExpanded] = useState<ExpandedState>({});

    const [sorting, setSorting] = useState<SortingState>(DEFAULT_SORTING);
    const [getQuery, setQuery] = useTypedQuery<'query'>();
    const currentFilter = getQuery('query');
    const [filterInput, setFilterInput] = useState<string>(currentFilter ?? '');
    const setCurrentFilter = (v: string) => setQuery({ query: v });
    const orderBy = getOrderBy(sorting);

    const { data: functionTopPages, error, isFetchNextPageError, isLoadingError, hasNextPage, isPending: loading, fetchNextPage, isFetchingNextPage } = useFunctionTopQuery({ currentFilter, generation, orderBy, generationStatus });
    const functionTop = useMemo(() => functionTopPages?.pages.flatMap(response => response.Instances), [functionTopPages]);
    const convertToFunctionRow = useCallback((entry: ClusterTopEntry) => rawConvertToFunctionRow(entry, timeInterval), [timeInterval]);
    const convertToServiceRow = useCallback((parentName: string, entry: ClusterTopEntry) => rawConvertToServiceRow(parentName, entry, timeInterval ), [timeInterval]);

    useEffect(() => {
        if (functionTop) {
            setData(functionTop.map(convertToFunctionRow));
        }
    }, [convertToFunctionRow, functionTop]);

    useEffect(() => {
        setExpanded({});
    }, [generation, currentFilter, orderBy]);

    const handleExpandedChange = useCallback<OnChangeFn<ExpandedState>>(
        (updaterOrValue) => {
            const newExpanded = typeof updaterOrValue === 'function' ? updaterOrValue(expanded) : updaterOrValue;
            setExpanded(newExpanded);

            const expandedIds = Object.keys(newExpanded).filter((id) => (newExpanded as Record<string, boolean>)[id] === true);
            const newlyExpandedIds = expandedIds.filter((id) => (expanded as Record<string, boolean>)[id] !== true);

            newlyExpandedIds.forEach((rowId) => {
                const rowIndex = data.findIndex((row) => row.Name === rowId);
                const row = data[rowIndex];

                if (row && row.type === 'function' && !row.services && !row.isLoadingServices) {
                    setData((prevData) => {
                        const newData = [...prevData];
                        newData[rowIndex] = { ...row, isLoadingServices: true };
                        return newData;
                    });

                    queryClient.fetchQuery({
                        queryFn: () => apiClient
                            .getServiceTop({ Generation: generation, OrderBy: orderBy, FunctionPattern: row.Name }),
                        queryKey: clusterTopServicesQueryKeys(generation, orderBy, row.Name),
                        ...getGenerationCachePolicy(generationStatus),
                    })
                        .then((response) => {
                            const services = response.data.Instances?.map(convertToServiceRow.bind(null, row.Name)) ?? [];
                            setData((prevData) => {
                                const newData = [...prevData];
                                newData[rowIndex] = {
                                    ...newData[rowIndex],
                                    services,
                                    isLoadingServices: false,
                                };
                                return newData;
                            });
                        })
                        .catch((err) => {
                            setData((prevData) => {
                                const newData = [...prevData];
                                newData[rowIndex] = {
                                    ...newData[rowIndex],
                                    isLoadingServices: false,
                                    error: err?.response?.data?.message ?? String(err),
                                };
                                return newData;
                            });
                        });
                }
            });
        },
        [expanded, data, generation, orderBy, convertToServiceRow, queryClient, generationStatus],
    );

    const columns = useMemo(() => getColumns({ timeIntervalFrom, timeIntervalTo }), [timeIntervalFrom, timeIntervalTo]);


    const table = useTable({
        columns,
        data,
        enableExpanding: true,
        enableSorting: true,
        enableMultiSort: false,
        enableSortingRemoval: false,
        manualSorting: true,
        getRowId: (row) => row.parentFunction ? row.parentFunction + row.Name : row.Name,
        getRowCanExpand: (row) => row.original.type === 'function',
        getSubRows: (row) => {
            if (row.type === 'function') {
                if (row.isLoadingServices) {
                    return [
                        {
                            Name: LOADING_STRING,
                            parentFunction: row.Name,
                            Count: { Self: 0, Cumulative: 0, CumulativePct: '', SelfPct: '', SelfPctValue: 0, CumulativePctValue: 0 },
                            type: 'service' as const,
                        },
                    ];
                }
                if (row.error) {
                    return [
                        {
                            Name: `Error: ${row.error}`,
                            parentFunction: row.Name,
                            Count: { Self: 0, Cumulative: 0, CumulativePct: '', SelfPct: '', SelfPctValue: 0, CumulativePctValue: 0 },
                            type: 'service' as const,
                        },
                    ];
                }

                return row.services;

            }
            return undefined;
        },
        getCoreRowModel: getCoreRowModel(),
        getExpandedRowModel: getExpandedRowModel(),
        getSortedRowModel: getSortedRowModel(),
        onSortingChange: setSorting,
        onExpandedChange: handleExpandedChange,
        autoResetExpanded: true,
        state: {
            expanded,
            sorting,
        },
    });

    if (error && isLoadingError) {
        return <ErrorPanel message={error?.message} />;
    }

    return (
        <>
            <form className={b('filter__form')} onSubmit={(e) => {e.preventDefault();setCurrentFilter(filterInput);}}>
                <TextInput
                    placeholder="Filter by function pattern..."
                    value={filterInput}
                    onUpdate={setFilterInput}
                    className={b('filter__input')}
                    hasClear
                />
                <Button loading={loading && currentFilter !== ''} className={b('search-button')} view={'action'} disabled={currentFilter === filterInput} onClick={() => setCurrentFilter(filterInput)}>Search</Button>
            </form>
            {isFetchNextPageError && <ErrorPanel message={error?.message} />}
            {loading ? <EmptyView/> :
                <>
                    <GravityTable className={b()} cellClassName={cellCn} headerCellClassName={headerCn} table={table} />
                    {hasNextPage && (
                        <div className={b('load-more')}>
                            <Button onClick={() => fetchNextPage()} loading={isFetchingNextPage} view="outlined">
                        Load more
                            </Button>
                        </div>
                    )}
                </>
            }
        </>
    );
};
