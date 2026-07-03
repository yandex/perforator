import * as React from 'react';
import { useMemo, useState } from 'react';

import { ArrowRightArrowLeft, BarsAscendingAlignLeftArrowUp, BarsDescendingAlignLeftArrowDown, Funnel, FunnelXmark, Magnifier, Xmark } from '@gravity-ui/icons';
import { Button, Icon, Switch } from '@gravity-ui/uikit';

import type { DenselyPackedCoordinates } from '../../densely-packed';
import { useSearchPattern } from '../../hooks/use-search-pattern';
import type { GoToDefinitionHref } from '../../models/goto';
import type { ProfileData, StringifiedNode } from '../../models/Profile';
import type { UserSettings } from '../../models/UserSettings';
import { getNodeTitleFull } from '../../node-title';
import type { GetStateFromQuery, SetStateFromQuery } from '../../query-utils';
import { readNodeStrings } from '../../read-string';
import type { Coordinate, QueryKeys, RenderFlamegraphOptions } from '../../renderer';
import { FlamegraphOffseter, renderFlamegraph as newFlame } from '../../renderer';
import { search as searchFn } from '../../search';
import { shorten } from '../../shorten/shorten';
import { getCanvasTitleFull, renderTitleFull } from '../../title';
import { cn } from '../../utils/cn';
import { Hotkey } from '../Hotkey/Hotkey';
import type { SearchUpdate } from '../RegexpDialog/RegexpDialog';
import { RegexpDialog } from '../RegexpDialog/RegexpDialog';

import type { ContextMenuProps, PopupData } from './ContextMenu';
import { ContextMenu } from './ContextMenu';
import type { HoverPopupProps } from './HoverPopup';
import { HoverPopup } from './HoverPopup';
import { useLevelHeight } from './useLevelHeight';

import './Flamegraph.css';


const b = cn('flamegraph');

export type ExternalFlamegraphApi = {
    offsetter: FlamegraphOffseter;
    canvas: HTMLCanvasElement;
};

export interface FlamegraphProps extends Pick<RenderFlamegraphOptions, 'onFinishRendering'> {
    isDiff: boolean;
    theme: 'light' | 'dark';
    userSettings: UserSettings;
    profileData: ProfileData | null;
    goToDefinitionHref: GoToDefinitionHref;
    onSuccess: ContextMenuProps['onSuccess'];
    getState: GetStateFromQuery<QueryKeys>;
    setState: SetStateFromQuery<QueryKeys>;
    className?: string;
    onFrameClick?: (event: React.MouseEvent, frame: StringifiedNode) => void;
    onFrameAltClick?: (event: React.MouseEvent, frame: StringifiedNode) => void;
    onContextClick?: (event: React.MouseEvent, frame: StringifiedNode) => void;
    onContextItemClick?: (frame: StringifiedNode, item: string) => void;
    getHoverText?: (coord: Coordinate) => string;
    isLeftHeavy?: boolean;
    onChangeLeftHeavy?: (leftHeavy: boolean) => void;
    disableHoverPopup?: boolean;
    setOffsetterRef?: (args: ExternalFlamegraphApi | null) => void;
    onResetOmitted?: () => void;
    onSearch?: (search: string) => void;
    onSearchReset?: () => void;
    onKeepOnlyFound?: (value: boolean) => void;
    useSelfAsScrollParent?: boolean;
    showLineNumbers?: boolean;
    setShowLineNumbers?: (showLineNumbers: boolean) => void;
}

const MAX_FIREFOX_DEPTH = 768;

const FIREFOX_CANVAS_SIZE_ERROR = 'CanvasRenderingContext2D.scale: Canvas exceeds max size.';
export const Flamegraph: React.FC<FlamegraphProps> = ({
    isDiff,
    theme,
    userSettings,
    profileData,
    goToDefinitionHref,
    onFinishRendering,
    onSuccess,
    getState: getQuery,
    setState: outerSetQuery,
    className,
    onFrameClick,
    getHoverText,
    isLeftHeavy,
    onChangeLeftHeavy,
    disableHoverPopup: disableHoverPopup,
    onContextClick,
    onFrameAltClick,
    onContextItemClick,
    setOffsetterRef,
    onResetOmitted,
    onSearch,
    onKeepOnlyFound,
    onSearchReset,
    showLineNumbers,
    setShowLineNumbers,
    useSelfAsScrollParent,
}) => {
    const flamegraphContainer = React.useRef<HTMLDivElement | null>(null);
    const flamegraphCanvas = React.useRef<HTMLCanvasElement | null>(null);
    const levelHeight = useLevelHeight(flamegraphContainer);
    const canvasRef = React.useRef<HTMLDivElement | null>(null);
    const [popupData, setPopupData] = useState<null | PopupData>(null);
    const [hoverData, setHoverData] = useState<null | HoverPopupProps['hoverData']>(null);
    const [showDialog, setShowDialog] = useState(false);
    const flamegraphOffsets = React.useRef<FlamegraphOffseter | null>(null);
    const search = getQuery('flamegraphQuery');
    const excludeSearch = getQuery('flamegraphExclude');
    const reverse = (getQuery('flamegraphReverse') ?? String(userSettings.reverseFlameByDefault)) === 'true';
    const isDiffSwitchingSupported = profileData.meta.version > 1;
    const [shouldTrim, setShouldTrim] = React.useState(false);
    const shouldOmitHighlight = React.useRef(true);

    const setQuery = React.useCallback<SetStateFromQuery<QueryKeys>>((q) => {
        outerSetQuery(q);
        shouldOmitHighlight.current = true;
    }, [outerSetQuery]);

    React.useLayoutEffect(() => {
        if (profileData) {
            const offsetter = new FlamegraphOffseter(shouldTrim ? profileData.rows.slice(0, MAX_FIREFOX_DEPTH) : profileData.rows, { reverse, levelHeight });
            flamegraphOffsets.current = offsetter;
            setOffsetterRef?.({ offsetter, canvas: flamegraphCanvas.current });
        }
    }, [profileData, reverse, levelHeight, shouldTrim]);

    const handleSearch = React.useCallback(() => {
        setShowDialog(true);
    }, []);

    const shouldSwapDiff = getQuery('flameBase') === 'diff';
    const setShouldSwapdiff = (value: boolean) => {
        if (value) {
            setQuery({ 'flameBase': 'diff' });
        } else {
            setQuery({ 'flameBase': 'base' });
        }
    };


    const handleReverse = React.useCallback(() => {
        setQuery({ 'flamegraphReverse': String(!reverse) });
    }, [reverse, setQuery]);

    const handleSearchReset = React.useCallback(() => {
        setQuery({ flamegraphQuery: false });
        onSearchReset?.();
    }, [setQuery]);
    const haveOmittedNodes = Boolean(getQuery('omittedIndexes'));
    const keepOnlyFound = getQuery('keepOnlyFound') === 'true';
    const switchKeepOnlyFound = React.useCallback(() => {
        onKeepOnlyFound?.(!keepOnlyFound);
        const newValue = !keepOnlyFound ? 'true' : false;
        setQuery({ keepOnlyFound: newValue });
    }, [keepOnlyFound, setQuery]);
    const handleOmittedNodesReset = React.useCallback(() => {
        setQuery({ omittedIndexes: false });
        onResetOmitted?.();
    }, [setQuery]);

    const handleSearchUpdate = (update: SearchUpdate) => {
        setQuery({
            'flamegraphQuery': encodeURIComponent(update.text),
            exactMatch: update.exactMatch ? 'true' : undefined,
            'flamegraphExclude': update.excludeText ? encodeURIComponent(update.excludeText) : false,
            caseInsensitive: update.caseInsensitive ? 'true' : undefined,
        });
        onSearch?.(update.text);
        setShowDialog(false);
    };
    const exactMatch = getQuery('exactMatch') === 'true';
    const caseInsensitive = getQuery('caseInsensitive') === 'true';

    const { searchPattern, excludeSearchPattern } = useSearchPattern(search, excludeSearch, exactMatch, caseInsensitive);

    const foundCoords: DenselyPackedCoordinates | null = useMemo(() => {
        if (!profileData || !search || !keepOnlyFound) {
            return null;
        }
        const readString = (id?: number) => {
            if (id === undefined) {
                return '';
            }
            return profileData.stringTable[id];
        };
        const shouldShortenTextForOverview = userSettings.shortenFrameTexts === 'true' || userSettings.shortenFrameTexts === 'hover';

        return searchFn(readString, shorten, shouldShortenTextForOverview, profileData.rows, searchPattern, excludeSearchPattern);
    }, [profileData, search, keepOnlyFound, userSettings.shortenFrameTexts, searchPattern, excludeSearchPattern]);

    React.useEffect(() => {
        if (flamegraphContainer.current) {
            flamegraphContainer.current.style.setProperty('--flamegraph-font', userSettings.monospace === 'system' ? 'monospace' : 'var(--g-font-family-monospace)');
        }
    }, [userSettings.monospace]);

    React.useEffect(() => {
        if (flamegraphContainer.current && profileData && flamegraphOffsets.current) {
            setHoverData(null);

            const renderOptions: RenderFlamegraphOptions = {
                setState: setQuery,
                getState: getQuery,
                theme,
                shortenFrameTexts: userSettings.shortenFrameTexts,
                isDiff,
                searchPattern: searchPattern,
                excludeSearchPattern: excludeSearchPattern,
                reverse,
                onFinishRendering,
                foundCoords,
                // by default, we show highlight, e.g. after clicks
                // and don't show it only on first render
                disableHighlightRender: shouldOmitHighlight.current,
                shouldScroll: true,
                scrollParent: useSelfAsScrollParent ? flamegraphContainer.current : document.documentElement,
            };

            try {
                const destructor = newFlame(flamegraphContainer.current, { ...profileData, rows: shouldTrim ? profileData.rows.slice(0, MAX_FIREFOX_DEPTH) : profileData.rows }, flamegraphOffsets.current, renderOptions);

                if (shouldOmitHighlight.current) {
                    shouldOmitHighlight.current = false;
                }

                return () => {
                    destructor();
                    shouldOmitHighlight.current = true;
                };
            } catch (e) {
                console.error(e);
                // see https://github.com/mozilla-firefox/firefox/blob/89b2affdc5d2a1588763e5cb4ac046093c4136a9/dom/canvas/CanvasRenderingContext2D.cpp#L1748
                if (e?.message === FIREFOX_CANVAS_SIZE_ERROR) {
                    setShouldTrim(true);
                } else {
                    throw e;
                }
            }
        }
        return () => { };
    }, [getQuery, isDiff, profileData, reverse, setQuery, theme, userSettings, levelHeight, onFinishRendering, isLeftHeavy, shouldTrim, foundCoords, searchPattern, excludeSearchPattern, useSelfAsScrollParent]);

    React.useEffect(() => {
        if (!profileData) {
            return;
        }
        const scrollEl = useSelfAsScrollParent ? flamegraphContainer.current : document.documentElement;
        scrollEl?.scrollTo({ top: !reverse ? scrollEl.scrollHeight : 0 });
    }, [reverse, profileData, useSelfAsScrollParent]);

    const handleContextMenu = React.useCallback((event: React.MouseEvent) => {
        if (!flamegraphContainer.current || !profileData || !flamegraphOffsets.current) {
            return;
        }
        event.preventDefault();

        const offsetX = event.nativeEvent.offsetX;
        const offsetY = event.nativeEvent.offsetY;
        const coordsClient = flamegraphOffsets.current!.getCoordsByPosition(offsetX, offsetY);

        if (!coordsClient) {
            return;
        }

        const stringifiedNode = readNodeStrings(profileData, coordsClient);
        setPopupData({ offset: [offsetX, -offsetY], node: stringifiedNode, coords: [coordsClient.h, coordsClient.i] });

        if (!onContextClick) {
            return;
        }

        onContextClick(event, stringifiedNode);
    }, [profileData]);

    const handleOnClick: React.MouseEventHandler = React.useCallback((e) => {
        if (popupData) {
            setPopupData(null);
            e.preventDefault();
            e.stopPropagation();
        }


        if (
            !onFrameClick ||
            !flamegraphContainer.current ||
            !profileData ||
            !flamegraphOffsets.current
        ) {
            return;
        }
        e.preventDefault();

        const offsetX = e.nativeEvent.offsetX;
        const offsetY = e.nativeEvent.offsetY;
        const coordsClient = flamegraphOffsets.current!.getCoordsByPosition(offsetX, offsetY);

        if (!coordsClient) {
            return;
        }

        const stringifiedNode = readNodeStrings(profileData, coordsClient);
        if (stringifiedNode) {
            onFrameClick(e, stringifiedNode);
            if (e.altKey) {
                onFrameAltClick?.(e, stringifiedNode);
            }
        }
    }, [popupData, onFrameClick, profileData, onFrameAltClick]);

    const handleMouseMove = React.useCallback((e: MouseEvent) => {
        const offsetX = e.offsetX;
        const offsetY = e.offsetY;
        const fg = flamegraphOffsets.current!;
        const coordsClient = fg.getCoordsByPosition(offsetX, offsetY);

        if (!coordsClient || disableHoverPopup) {
            setHoverData(null);
            return;
        }

        const nodeX = profileData.rows[coordsClient.h][coordsClient.i].x!;
        const nodeY = (Math.floor(offsetY / levelHeight) + 1) * levelHeight;
        const canvasRect = canvasRef.current?.getBoundingClientRect?.();

        setHoverData({ offset: [nodeX + canvasRect.left, canvasRect.top + nodeY], coords: [coordsClient.h, coordsClient.i] });
    }, [levelHeight, profileData.rows]);

    const getHoverTitle = React.useCallback((coords: Coordinate) => {
        const fg = flamegraphOffsets.current!;

        function readString(id?: number) {
            if (id === undefined) {
                return '';
            }
            return profileData.stringTable[id];
        }

        const rows = profileData.rows;

        const getNodeTitleHl = getNodeTitleFull.bind(null, readString, s => s, false);

        const renderTitle = renderTitleFull.bind(null, (n) => (fg.countEventCountWidth(n)), (n) => fg.countSampleCountWidth(n), getNodeTitleHl, isDiff, shouldSwapDiff);

        const eventType = readString(profileData.meta.eventType);

        const getCanvasTitle = getCanvasTitleFull(eventType, renderTitle);

        const canvasTitle = getCanvasTitle(rows[coords[0]][coords[1]], rows[fg.currentNodeCoords[0]][fg.currentNodeCoords[1]], rows[0][0]);

        return canvasTitle;
    }, [isDiff, profileData.meta.eventType, profileData.rows, profileData.stringTable, shouldSwapDiff]);

    const clearHover = React.useCallback(() => setHoverData(null), []);
    React.useEffect(() => {
        canvasRef.current?.addEventListener('mousemove', handleMouseMove);
        canvasRef.current?.addEventListener('mouseleave', clearHover);

        return () => {
            canvasRef.current?.removeEventListener('mousemove', handleMouseMove);
            canvasRef.current?.removeEventListener('mouseleave', clearHover);
        };
    }, [clearHover, handleMouseMove, setHoverData]);

    const handleKeyDown = React.useCallback((event: KeyboardEvent) => {
        if ((event.ctrlKey || event.metaKey) && event.code === 'KeyF') {
            event.preventDefault();
            handleSearch();
        } else if (event.altKey && event.code === 'KeyF') {
            switchKeepOnlyFound();
        } else if (event.key === 'Escape') {
            handleSearchReset();
        }
    }, [handleSearch, handleSearchReset, switchKeepOnlyFound]);

    React.useEffect(() => {
        window.addEventListener('keydown', handleKeyDown);

        return () => {
            window.removeEventListener('keydown', handleKeyDown);
        };
    }, [handleKeyDown]);

    const framesCount = profileData?.rows?.reduce((acc, row) => acc + row.length, 0);

    const header = (
        <div className={b('header-wrapper')}>
            <div className="flamegraph__header">
                <div className="flamegraph__buttons">
                    <Button className="flamegraph__button flamegraph__button_reverse" onClick={handleReverse}>
                        <Icon data={reverse ? BarsDescendingAlignLeftArrowDown : BarsAscendingAlignLeftArrowUp} /> Reverse
                    </Button>
                    <Button className="flamegraph__button flamegraph__button_search" onClick={handleSearch}>
                        <Icon className="regexp-dialog__header-icon" data={Magnifier} />
                                Search
                        <Hotkey value="cmd+F" />
                    </Button>
                    {search ?
                        <Button className="flamegraph__button flamegraph__button_keep-only-found" onClick={switchKeepOnlyFound}>
                            <Icon data={keepOnlyFound ? FunnelXmark : Funnel} />
                            {keepOnlyFound ? 'Show all stacks' : 'Show matched stacks'}
                            <Hotkey value="alt+F" />
                        </Button>
                        : null}
                    {isDiff && isDiffSwitchingSupported ?
                        <Switch className="flamegraph__switch" checked={shouldSwapDiff} onUpdate={setShouldSwapdiff}>
                            <Icon data={ArrowRightArrowLeft} />
                                    Swap Base and Diff Profiles
                        </Switch>
                        : null}
                    {
                        isLeftHeavy !== undefined && onChangeLeftHeavy !== undefined ?
                            <Switch qa={'flamegraph-left-heavy'} className="flamegraph__switch flamegraph__switch_left-heavy" checked={isLeftHeavy} onUpdate={onChangeLeftHeavy}>
                                <Icon data={ArrowRightArrowLeft} />
                                    Left-heavy
                            </Switch>
                            : null
                    }
                    {
                        showLineNumbers !== undefined && setShowLineNumbers !== undefined ?
                            <Switch className="flamegraph__switch flamegraph__switch_left-heavy" checked={showLineNumbers} onUpdate={setShowLineNumbers}>
                                Line numbers
                            </Switch>
                            : null
                    }
                </div>
                <div className="flamegraph__frames-count">Showing {framesCount} frames</div>
            </div>
            <div className={b('annotations')}>
                <div className="flamegraph__status" />
                <div className="flamegraph__deletion" style={{ display: haveOmittedNodes ? 'inherit' : 'none' }}>
                    <Button
                        className="flamegraph__clear-deletion"
                        view="flat-danger"
                        title="Clear Deletions"
                        onClick={handleOmittedNodesReset}
                    >
                        <Icon data={Xmark} size={20} /> Reset omitted stacks
                    </Button>
                </div>
                <div className="flamegraph__match">
                        Matched: <span className="flamegraph__match-value" />
                    <Button
                        className="flamegraph__clear"
                        view="flat-danger"
                        title="Clear"
                        onClick={handleSearchReset}
                    >
                        <Icon data={Xmark} size={20} />
                    </Button>
                </div>
            </div>
        </div>
    );

    return (
        <>
            <div ref={flamegraphContainer} className={b(null, className)}>
                {showDialog && <RegexpDialog
                    showDialog={showDialog}
                    onCloseDialog={() => setShowDialog(false)}
                    onSearchUpdate={handleSearchUpdate}
                    initialSearch={search}
                    initialExact={getQuery('exactMatch') === 'true'}
                    initialExcludeText={excludeSearch}
                    initialCaseInsensitive={getQuery('caseInsensitive') === 'true'}
                />}
                {header}
                <div id="profile" className="flamegraph__content">
                    <div ref={canvasRef} onClickCapture={handleOnClick} onContextMenu={handleContextMenu}>
                        <canvas ref={flamegraphCanvas} className="flamegraph__canvas" />
                    </div>
                    <template className="flamegraph__label-template" dangerouslySetInnerHTML={{
                        __html: '<div class="flamegraph__label"></div>' }} />
                    <div className="flamegraph__labels-container" translate="no"/>
                    <div className='flamegraph__highlight'>
                        <span />
                    </div>
                </div>
            </div>
            {popupData && (
                <ContextMenu
                    onSuccess={onSuccess}
                    onClosePopup={() => { setPopupData(null); }}
                    popupData={popupData}
                    anchorRef={canvasRef}
                    getQuery={getQuery}
                    setQuery={setQuery}
                    goToDefinitionHref={goToDefinitionHref}
                    onContextItemClick={onContextItemClick}
                />
            )}
            {
                hoverData && !disableHoverPopup && <HoverPopup hoverData={hoverData} anchorRef={canvasRef} getText={getHoverText || getHoverTitle}/>
            }

        </>
    );
};
