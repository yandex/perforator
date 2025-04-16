import React, { useCallback, useEffect, useMemo, useRef, useState, useTransition } from 'react';

import { HelpPopover } from '@gravity-ui/components';
import ArrowUpRightFromSquareIcon from '@gravity-ui/icons/svgs/arrow-up-right-from-square.svg?raw';
import MagnifierIcon from '@gravity-ui/icons/svgs/magnifier.svg?raw';
import type { ProgressColorStops, TableColumnConfig, TableSettingsData, TableSortState } from '@gravity-ui/uikit';
import { Icon, Link as UIKitLink, Progress, Table, TextInput, withTableSettings, withTableSorting } from '@gravity-ui/uikit';

import { Link } from 'src/components/Link/Link';
import { NegativePositiveProgress } from 'src/components/NegativePositiveProgress/NegativePositiveProgress';
import { uiFactory } from 'src/factory';
import type { ProfileData, StringifiedNode } from 'src/models/Profile';
import { useUserSettings } from 'src/providers/UserSettingsProvider';
import type { NumTemplatingFormat } from 'src/providers/UserSettingsProvider/UserSettings';
import { cn } from 'src/utils/cn';

import { hugenum } from '../Flamegraph/flame-utils';
import { pct } from '../Flamegraph/pct';
import { modifyQuery, useTypedQuery } from '../Flamegraph/query-utils';
import { useRegexError } from '../Flamegraph/RegexpDialog/useRegexError';
import type { QueryKeys } from '../Flamegraph/renderer';
import { shorten } from '../Flamegraph/shorten';
import type { ReadString } from '../Flamegraph/utils/node-title';
import { getNodeTitleFull } from '../Flamegraph/utils/node-title';
import type { TableFunctionTop } from '../Flamegraph/utils/top';
import { isNonDiffKey, isSelfKey, type NonDiffTopKeys, type TopKeys } from '../Flamegraph/utils/top-types';

import './TopTable.scss';


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

function createNewQueryForSwitch(name: string) {
    const currentQuery = new URLSearchParams(window.location.search);

    const query = modifyQuery<QueryKeys>(currentQuery, {
        flamegraphQuery: encodeURIComponent(name),
        tab: 'flame',
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
    { getNodeTitle, eventType, totalEventCount, totalBaseEventCount, numTemplating, isDiff }: TopColumnsOpts,
): TableColumnConfig<TableFunctionTop>[] {
    const regex = new RegExp(search);

    const calcDiff = (field: NonDiffTopKeys) => (node: TableFunctionTop) => {
        return node[field] / totalEventCount - node[`diff.${field}`] / totalBaseEventCount!;
    };

    const compareCalculatedDiffFields = (field: NonDiffTopKeys) => {
        if (!totalBaseEventCount || !totalEventCount) {
            return 0;
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
                <HelpPopover placement={['bottom', 'bottom']} content={getHelpContent(key)}/>
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
        const goToLink = uiFactory().goToDefinitionHref({ file: readString(item.file), frameOrigin: readString(item.frameOrigin) } as StringifiedNode);
        return (
            <span className="top-table__name-column">
                {name.slice(0, start)}
                <span className="top-table__name_highlight">
                    {name.slice(start, end)}
                </span>
                {name.slice(end)}
                <Link className={'top-table__column-icon-link'} href={createNewQueryForSwitch(name)}>
                    <Icon className={'top-table__column-icon'} data={MagnifierIcon}/>
                </Link>

                {goToLink && <UIKitLink className={'top-table__column-icon-link'} target="_blank" href={goToLink}>
                    <Icon className={'top-table__column-icon'} data={ArrowUpRightFromSquareIcon} />
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

interface TopTableProps {
    topData: TableFunctionTop[];
    profileData: ProfileData;
}

export const TopTable: React.FC<TopTableProps> = ({
    topData,
    profileData,
}) => {
    const totalBaseEventCount = useMemo(() => profileData.rows[0][0].baseEventCount, [profileData.rows]);
    const isDiff = Boolean(totalBaseEventCount);
    const readString = useCallback((id?: number) => {
        if (id === undefined) {
            return '';
        }
        return profileData.stringTable[id];
    }, [profileData]);

    const eventType = React.useMemo(() => {
        return readString(profileData?.meta.eventType);
    }, [readString, profileData?.meta.eventType]);
    const totalEventCount = React.useMemo(() => profileData.rows[0][0].eventCount, [profileData.rows]);

    const { userSettings } = useUserSettings();
    const maybeShorten = useCallback((str: string) => {
        return userSettings.shortenFrameTexts === 'true' ? shorten(str) : str;
    }, [userSettings.shortenFrameTexts]);
    const getNodeTitle = useCallback((node: TableFunctionTop) => getNodeTitleFull(readString, maybeShorten, node), [maybeShorten, readString]);
    const numTemplating = useMemo(() => userSettings.numTemplating, [userSettings.numTemplating]);
    const [sortState, setSortState] = useState<TableSortState[number]>({ column: (isDiff ? 'diffcalc.self.eventCount' : 'self.eventCount'), order: 'desc' });
    const [getQuery, setQuery] = useTypedQuery<QueryKeys>();
    const searchQuery = getQuery('topQuery');
    const setSearchQuery = useCallback((query: string) => {
        setQuery({ topQuery: query });
    }, [setQuery]);
    const [searchValue, setSearchValue] = useState('');
    const [isSeaching, startTransition] = useTransition();
    const [settings, setSettings] = useState<TableSettingsData>([]);
    const regexError = useRegexError(searchValue);
    const boundTopColumns = useCallback(() => topColumns(readString, regexError ? '' : searchValue, { getNodeTitle, eventType, totalEventCount, numTemplating, isDiff, totalBaseEventCount }),
        [readString, regexError, searchValue, getNodeTitle, eventType, totalEventCount, numTemplating, isDiff, totalBaseEventCount],
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

        return data.slice(0, 500);
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
            uiFactory().rum()?.finishDataRendering?.('top-table');
            hasSentDataRef.current = true;
        }
    }, [topSlice.length]);


    return (
        <div className="top-table">
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
