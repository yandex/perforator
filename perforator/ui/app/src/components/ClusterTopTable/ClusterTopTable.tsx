import { useCallback, useEffect, useState } from 'react';

import { Flame } from '@gravity-ui/icons';
import { Table as GravityTable, TreeExpandableCell, useTable } from '@gravity-ui/table';
import type { Cell, ColumnDef, ExpandedState, Header } from '@gravity-ui/table/tanstack';
import { getCoreRowModel, getExpandedRowModel } from '@gravity-ui/table/tanstack';
import { Button, ClipboardButton, Icon, Loader, Text, TextInput } from '@gravity-ui/uikit';

import type { ClusterTopEntry } from 'src/generated/perforator/proto/perforator/perforator';
import { apiClient } from 'src/utils/api';
import { cn } from 'src/utils/cn';
import { useTypedQuery } from 'src/utils/query';
import { createErrorToast } from 'src/utils/toaster';

import { ErrorPanel } from '../ErrorPanel/ErrorPanel';
import { useAsyncResult } from '../TaskReport/TaskFlamegraph/useFetchResult';

import type { ClusterTopRow } from './utils';
import { convertToFunctionRow as rawConvertToFunctionRow, convertToServiceRow as rawConvertToServiceRow } from './utils';

import './ClusterTopTable.css';


const PAGE_LIMIT = '100';

const LOADING_STRING = 'Loading...';

const columns: ColumnDef<ClusterTopRow, string | number>[] = [
    {
        id: 'Name',
        accessorKey: 'Name',
        header: 'Name',
        size: 400,
        cell: ({ row, getValue }) => {
            if (row.original.type === 'function') {
                return <TreeExpandableCell row={row}>{
                    <>
                        <Text variant={'code-1'}>
                            {getValue<string>()}
                        </Text>
                        <ClipboardButton text={row.original.Name} size="xs"/>
                    </>
                }</TreeExpandableCell>;
            }

            if (row.original.type === 'service' && row.original.Name !== LOADING_STRING) {
                return <>
                    <Text variant={'code-1'} className={b('service-name')}>{getValue<string>()}</Text><ClipboardButton text={row.original.Name} size="xs"/>
                    <Button target={'_blank'} view={'flat'} size={'xs'} href={`/build?service=${row.original.Name}&flamegraphQuery=${row.original.parentFunction}&exactMatch=true`}>
                        <Icon height={14} size={14} data={Flame}/>
                    </Button>
                </>;
            }

            return <Text variant={'code-1'} className={b('service-name')}>{getValue<string>()}</Text>;
        },
    },
    {
        id: 'Count.Self',
        accessorFn: (row) => row.Count.Self,
        header: 'Self, Cores',
        size: 150,
    },
    {
        id: 'Count.Cumulative',
        accessorFn: (row) => row.Count.Cumulative,
        header: () => 'Cumulative, Cores',
        size: 150,
    },
    {
        id: 'Count.SelfPct',
        accessorFn: (row) => row.Count.SelfPct,
        header: 'Self, %',
        size: 150,
    },
    {
        id: 'Count.CumulativePct',
        accessorFn: (row) => row.Count.CumulativePct,
        header: 'Cumulative, %',
        size: 150,
    },
];

const b = cn('cluster-top-table');

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
}

export const ClusterTopTable: React.FC<ClusterTopTableProps> = ({ generation, timeInterval }) => {
    const [data, setData] = useState<ClusterTopRow[]>([]);
    const [expanded, setExpanded] = useState<ExpandedState>({});
    const [offset, setOffset] = useState<string>('0');
    const [hasMore, setHasMore] = useState<boolean>(false);
    const [isLoadingMore, setIsLoadingMore] = useState<boolean>(false);
    const [getQuery, setQuery] = useTypedQuery<'query'>();
    const currentFilter = getQuery('query');
    const [filterInput, setFilterInput] = useState<string>(currentFilter ?? '');
    const setCurrentFilter = (v: string) => setQuery({ query: v });

    const getData = useCallback(
        () => apiClient.getFunctionTop({
            Generation: generation,
            Pagination: { Offset: '0', Limit: PAGE_LIMIT },
            FunctionPattern: currentFilter || undefined,
        }).then((value) => value.data),
        [generation, currentFilter],
    );
    const { error, data: functionTop, loading } = useAsyncResult({ getData, clearPrevResult: true });

    const convertToFunctionRow = useCallback((entry: ClusterTopEntry) => rawConvertToFunctionRow(entry, timeInterval), [timeInterval]);
    const convertToServiceRow = useCallback((parentName: string, entry: ClusterTopEntry) => rawConvertToServiceRow(parentName, entry, timeInterval), [timeInterval]);

    useEffect(() => {
        if (functionTop?.Instances) {
            setData(functionTop.Instances.map(convertToFunctionRow));
            setHasMore(functionTop.HasMore);
            setOffset(PAGE_LIMIT);
        }
    }, [convertToFunctionRow, functionTop]);

    useEffect(() => {
        setExpanded({});
        setOffset('0');
        setHasMore(false);
    }, [generation, currentFilter]);

    const handleLoadMore = useCallback(() => {
        if (isLoadingMore) {return;}

        setIsLoadingMore(true);
        apiClient
            .getFunctionTop({
                Generation: generation,
                Pagination: { Offset: offset, Limit: PAGE_LIMIT },
                FunctionPattern: currentFilter || undefined,
            })
            .then((response) => {
                const newData = response.data.Instances?.map(convertToFunctionRow) ?? [];
                setData((prevData) => [...prevData, ...newData]);
                setHasMore(response.data.HasMore);
                setOffset((prevOffset) => String(Number(prevOffset) + Number(PAGE_LIMIT)));
            })
            .catch((err) => {
                console.error('Failed to load more data:', err);
                createErrorToast(err, { name: 'cluster-top-load-more', title: 'Cluster top load more errored' });
            })
            .finally(() => {
                setIsLoadingMore(false);
            });
    }, [generation, offset, isLoadingMore, currentFilter, convertToFunctionRow]);

    const handleExpandedChange = useCallback(
        (updaterOrValue: ExpandedState | ((old: ExpandedState) => ExpandedState)) => {
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

                    apiClient
                        .getServiceTop({ Generation: generation, FunctionPattern: row.Name })
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
        [expanded, data, generation, convertToServiceRow],
    );


    const table = useTable({
        columns,
        data,
        enableExpanding: true,
        getRowId: (row) => row.parentFunction ? row.parentFunction + row.Name : row.Name,
        getRowCanExpand: (row) => row.original.type === 'function',
        getSubRows: (row) => {
            if (row.type === 'function') {
                if (row.isLoadingServices) {
                    return [
                        {
                            Name: LOADING_STRING,
                            Count: { Self: 0, Cumulative: 0, CumulativePct: '', SelfPct: '' },
                            type: 'service' as const,
                        },
                    ];
                }
                if (row.error) {
                    return [
                        {
                            Name: `Error: ${row.error}`,
                            Count: { Self: 0, Cumulative: 0, CumulativePct: '', SelfPct: '' },
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
        onExpandedChange: handleExpandedChange,
        state: {
            expanded,
        },
    });

    if (error) {
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
            {loading ? <EmptyView/> :
                <>
                    <GravityTable cellClassName={cellCn} size={'s'} headerCellClassName={headerCn} table={table} />
                    {hasMore && (
                        <div className={b('load-more')}>
                            <Button onClick={handleLoadMore} loading={isLoadingMore} view="outlined">
                        Load more
                            </Button>
                        </div>
                    )}
                </>
            }
        </>
    );
};
