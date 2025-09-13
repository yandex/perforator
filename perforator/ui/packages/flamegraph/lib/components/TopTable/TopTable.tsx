import * as React from 'react'
import { useCallback, useEffect, useMemo, useRef, useState, useTransition } from 'react';
import { type NavigateFunction } from 'react-router-dom';

import { ArrowUpRightFromSquare, Magnifier } from '@gravity-ui/icons';
import type { ProgressColorStops, TableColumnConfig, TableSettingsData, TableSortState } from '@gravity-ui/uikit';
import { Icon, Link as UIKitLink, Progress, Table, TextInput, withTableSettings, withTableSorting, HelpMark } from '@gravity-ui/uikit';

import { NegativePositiveProgress } from '../NegativePositiveProgress/NegativePositiveProgress';
import type { ProfileData, StringifiedNode } from '../../models/Profile';


import { cn } from '../../utils/cn';
import type { UserSettings, NumTemplatingFormat } from '../../models/UserSettings';

import { GetStateFromQuery, modifyQuery, SetStateFromQuery } from '../../query-utils';
import { useRegexError } from '../../components/RegexpDialog/useRegexError';
import type { QueryKeys } from '../../renderer';
import { shorten } from '../../shorten';
import type { ReadString } from '../../node-title';
import { getNodeTitleFull } from '../../node-title';
import type { TableFunctionTop } from '../../top';
import { isNonDiffKey, isSelfKey, type NonDiffTopKeys, type TopKeys } from '../../top-types';

import { hugenum } from '../../flame-utils';
import { pct } from '../../pct';
import './TopTable.css';
import { GoToDefinitionHref } from '../../models/goto';


const b = cn('top-table');


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

function createNewQueryForSwitch(name: string, { disableAutoTabSwitch }: {disableAutoTabSwitch?: boolean} = {}) {
    const currentQuery = new URLSearchParams(window.location.search);

    const query = modifyQuery<QueryKeys>(currentQuery, {
        flamegraphQuery: encodeURIComponent(name),
        ...(disableAutoTabSwitch ? {} : { tab: 'flame' }),
        exactMatch: 'true',
        keepOnlyFound: 'true',
    });
    return `?${query.toString()}`;
}

const TTable = React.memo(withTableSettings(withTableSorting(Table<TableFunctionTop>)));
function compareFields(field: string) {
    return function (l: any, r: any) {
        return (l[field] ?? 0) - (r[field] ?? 0);
    };
}
interface TopColumnsOpts {
    getNodeTitle: (node: TableFunctionTop) => string;
    eventType?: string;
    totalEventCount: number;
    totalBaseEventCount?: number;
    numTemplating?: NumTemplatingFormat;
    isDiff?: boolean;
    goToDefinitionHref: GoToDefinitionHref;
    navigate: NavigateFunction;
    disableAutoTabSwitch?: boolean;
}

function getNameType(key: TopKeys) {
    if (isSelfKey(key)) {
        return 'Self';
    } else {
        return 'Total';
    }
}

function getHelpContent(key: TopKeys) {
    if (isSelfKey(key)) {
        return 'Time the function takes to run without its children';
    } else {
        return 'Time the function takes to run, with children';
    }
}

function getProgressSteps(key: TopKeys) {
    if (isSelfKey(key)) {
        return selfTimeColorStops;
    } else {
        return totalTimeColorStops;
    }
}


function topColumns (
    readString: ReadString,
    search: string,
    { getNodeTitle, eventType, totalEventCount, totalBaseEventCount, numTemplating, isDiff, goToDefinitionHref, navigate, disableAutoTabSwitch }: TopColumnsOpts,
): TableColumnConfig<TableFunctionTop>[] {
    const regex = new RegExp(search);

    const calcDiff = (field: NonDiffTopKeys) => (node: TableFunctionTop) => {
        return node[field] / totalEventCount - node[`diff.${field}`] / totalBaseEventCount!;
    };

    const compareCalculatedDiffFields = (field: NonDiffTopKeys) => {
        if (!totalBaseEventCount || !totalEventCount) {
            return () => 0;
        }
        const diffCalc = calcDiff(field);
        return (l: TableFunctionTop, r: TableFunctionTop) => {
            return diffCalc(l) - diffCalc(r);
        };
    };

    function createTableConfigField(key: TopKeys): TableColumnConfig<TableFunctionTop> {
        const nonDiffKey = isNonDiffKey(key);
        const startingName = isDiff ? nonDiffKey ? 'Diff ' : 'Baseline ' : '';
        const name = `${startingName}${getNameType(key)} ${eventType}`;
        const max = nonDiffKey ? totalEventCount : totalBaseEventCount;
        function templateWithPct(count: number): string {
            if (!max) {
                return '0';
            }
            return templateBigNumber(count) + ' (' + pct(count, max) + '%)';
        }
        return {
            id: key,
            name: () => <>
                {name}
                <HelpMark popoverProps={{placement: ['bottom', 'bottom'] }}>{getHelpContent(key)}</HelpMark>
            </>,
            template: (node) => {
                const count = node[key];
                // everything < 0.5 is visually indistinguishable from 0
                const value = max ? Math.max(count * 100 / max, 1) : 0;
                return <Progress
                    text={templateWithPct(count)}
                    size="m"
                    value={value}
                    colorStops={getProgressSteps(key)}
                />;
            },
            meta: {
                defaultSortOrder: 'desc',
                selectedByDefault: !isDiff,
                sort: compareFields(key),
                _originalName: name,
            },
        };
    }


    function diffItemTemplate(key: Extract<NonDiffTopKeys, `${string}.eventCount`>) {
        return (node: TableFunctionTop) => {
            if (!totalEventCount || !totalBaseEventCount) {
                return null;
            }
            const count = (node[key] / totalEventCount - node[`diff.${key}`] / totalBaseEventCount);
            const value = count * 50;
            return <NegativePositiveProgress value={value} text={(value * 2).toPrecision(3) + '%'} />;
        };
    }

    function templateBigNumber(n: number) {
        switch (numTemplating) {
        case 'exponent': {
            return n.toPrecision(4);
        }
        case 'hugenum': {
            return hugenum(n);
        }
        }
        return n.toString();
    }

    const itemTemplate = (item: TableFunctionTop) => {
        const name = getNodeTitle(item);
        const match = name.match(regex);
        const start = match?.index ?? -1;
        if (start === -1) {return name;}
        const end = start + (match?.[0].length ?? 0);
        const goToLink = goToDefinitionHref({ file: readString(item.file), frameOrigin: readString(item.frameOrigin) } as StringifiedNode);
        return (
            <span className="top-table__name-column">
                {name.slice(0, start)}
                <span className="top-table__name_highlight">
                    {name.slice(start, end)}
                </span>
                {name.slice(end)}
                <span className={'top-table__column-icon-link'} onClick={() => navigate(createNewQueryForSwitch(name, { disableAutoTabSwitch }))}>
                    <Icon className={'top-table__column-icon'} data={Magnifier}/>
                </span>

                {goToLink && <UIKitLink className={'top-table__column-icon-link'} target="_blank" href={goToLink}>
                    <Icon className={'top-table__column-icon'} data={ArrowUpRightFromSquare} />
                </UIKitLink>}
            </span>
        );
    };

    return [
        {
            id: 'name',
            meta: { copy: true },
            template: itemTemplate,
        },
        createTableConfigField('self.eventCount'),
        createTableConfigField('all.eventCount'),
        {
            id: 'self.sampleCount',
            name: 'Self Samples',
            meta: { defaultSortOrder: 'desc', selectedByDefault: false, sort: compareFields('self.sampleCount') },
        },
        {
            id: 'all.sampleCount',
            name: 'Total Samples',
            meta: { defaultSortOrder: 'desc', selectedByDefault: false, sort: compareFields('all.sampleCount') },
        },
        {
            id: 'file',
            name: 'File',
            template: ({ file }) => readString(file),
            meta: { selectedByDefault: false, disabled: true },
        },
        ...(isDiff ? [
            createTableConfigField('diff.self.eventCount'),

            createTableConfigField('diff.all.eventCount'),

            {
                id: 'diffcalc.self.eventCount',
                name: `Delta in self ${eventType}`,
                template: diffItemTemplate('self.eventCount'),
                meta: {
                    defaultSortOrder: 'desc',
                    sort:  compareCalculatedDiffFields('self.eventCount'),
                },
            },
            {
                id: 'diffcalc.all.eventCount',
                name: `Delta in total ${eventType}`,
                template: diffItemTemplate('all.eventCount'),
                meta: {
                    defaultSortOrder: 'desc',
                    sort:  compareCalculatedDiffFields('all.eventCount'),
                },
            },

        ] : []),
    ];


}

const DEFAULT_LINE_COUNT = 500;

export interface TopTableProps {
    topData: TableFunctionTop[];
    profileData: ProfileData;
    userSettings: UserSettings;
    goToDefinitionHref: GoToDefinitionHref;
    onFinishRendering?: () => void;
    navigate: NavigateFunction;
    getState: GetStateFromQuery<QueryKeys>;
    setState: SetStateFromQuery<QueryKeys>;
    disableAutoTabSwitch?: boolean;
    className?: string;
    /** @default 500 */
    lines?: number;
}

export const TopTable: React.FC<TopTableProps> = ({
    topData,
    profileData,
    userSettings,
    goToDefinitionHref,
    onFinishRendering,
    navigate,
    getState: getQuery,
    setState: setQuery,
    className,
    disableAutoTabSwitch,
    lines = DEFAULT_LINE_COUNT,
}) => {
    const totalBaseEventCount = useMemo(() => profileData.rows[0][0].baseEventCount, [profileData.rows]);
    const isDiff = Boolean(totalBaseEventCount);
    const readString = useCallback((id?: number) => {
        if (id === undefined) {
            return '';
        }
        return profileData.stringTable[id];
    }, [profileData]);

    const frameDepth = Number(getQuery('frameDepth', '0'));
    const framePos = Number(getQuery('framePos', '0'));

    const eventType = React.useMemo(() => {
        return readString(profileData?.meta.eventType);
    }, [readString, profileData?.meta.eventType]);
    const totalEventCount = React.useMemo(() => profileData.rows[frameDepth][framePos].eventCount, [profileData.rows, frameDepth, framePos]);

    const getNodeTitle = useCallback(
        (node: TableFunctionTop) => getNodeTitleFull(readString, shorten, userSettings.shortenFrameTexts === 'true', node),
        [readString, userSettings.shortenFrameTexts],
    );
    const numTemplating = useMemo(() => userSettings.numTemplating, [userSettings.numTemplating]);
    const [sortState, setSortState] = useState<TableSortState[number]>({ column: (isDiff ? 'diffcalc.self.eventCount' : 'self.eventCount'), order: 'desc' });
    const searchQuery = getQuery('topQuery');
    const setSearchQuery = useCallback((query: string) => {
        setQuery({ topQuery: query });
    }, [setQuery]);
    const [searchValue, setSearchValue] = useState('');
    const [isSeaching, startTransition] = useTransition();
    const [settings, setSettings] = useState<TableSettingsData>([]);
    const regexError = useRegexError(searchValue);
    const boundTopColumns = useCallback(() => topColumns(readString, regexError ? '' : searchValue, { getNodeTitle, eventType, totalEventCount, numTemplating, isDiff, totalBaseEventCount, goToDefinitionHref, navigate, disableAutoTabSwitch }),
        [readString, regexError, searchValue, getNodeTitle, eventType, totalEventCount, numTemplating, isDiff, totalBaseEventCount, disableAutoTabSwitch],
    );


    const topSlice = useMemo( () => {
        let data = [];
        if (searchValue && !regexError) {
            const regex = new RegExp(searchValue);
            // not a .filter() for optimization purposes
            for (let i = 0; i < topData.length; i++) {
                const item = topData[i];
                if (regex.test(getNodeTitle(item))) {
                    data.push(item);
                }
            }
        }
        else {
            data = topData;
        }
        if (sortState) {
            const baseSortFn = boundTopColumns().find(
                (col) => col.id === sortState.column,
            )?.meta?.sort;
            const sortFn: (a: TableFunctionTop, b: TableFunctionTop) => number =
                sortState.order === 'asc'
                    ? baseSortFn
                    : (...args) => -1 * baseSortFn(...args);

            data = data.sort(sortFn);
        }

        return data.slice(0, lines);
    }, [boundTopColumns, getNodeTitle, regexError, searchValue, sortState, topData]);
    const handleSortChange = useCallback(([newSortState]: TableSortState) => {
        setSortState(newSortState);
    }, []);
    const sort = useMemo(() => sortState ? [sortState] : [], [sortState]);
    const handleUpdate = useCallback((value: string): void => {
        setSearchQuery(value);
        startTransition(() => setSearchValue(value));
    }, [setSearchQuery]);
    const columns = useMemo(() => boundTopColumns(), [boundTopColumns]);


    const hasSentDataRef = useRef(false);

    useEffect(() => {
        if (topSlice.length > 0 && !hasSentDataRef.current) {
            onFinishRendering?.();
            hasSentDataRef.current = true;
        }
    }, [topSlice.length]);


    return (
        <div className={b(null, className)}>
            <TextInput
                value={searchQuery}
                placeholder="Search"
                autoFocus
                className="top-table__search"
                onUpdate={handleUpdate}
                hasClear
                error={regexError ?? false}
            />
            <div className={b('table-wrapper', { search: isSeaching })}>
                <TTable
                    settings={settings}
                    updateSettings={setSettings}
                    className={b('table')}
                    sortState={sort}
                    onSortStateChange={handleSortChange}
                    columns={columns}
                    data={topSlice}
                />
            </div>
        </div>
    );
};
