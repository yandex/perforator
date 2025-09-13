import * as React from 'react';
import { useState } from 'react';

import { ArrowRightArrowLeft, BarsAscendingAlignLeftArrowUp, BarsDescendingAlignLeftArrowDown, Funnel, FunnelXmark, Magnifier, Xmark } from '@gravity-ui/icons';
import { Button, Icon, Switch } from '@gravity-ui/uikit';

import type { GoToDefinitionHref } from '../../models/goto';
import type { ProfileData, StringifiedNode } from '../../models/Profile';
import type { UserSettings } from '../../models/UserSettings';
import { getNodeTitleFull } from '../../node-title';
import type { GetStateFromQuery, SetStateFromQuery } from '../../query-utils';
import { readNodeStrings } from '../../read-string';
import type { Coordinate, QueryKeys, RenderFlamegraphOptions } from '../../renderer';
import { FlamegraphOffseter, renderFlamegraph as newFlame } from '../../renderer';
import { getCanvasTitleFull, renderTitleFull } from '../../title';
import { cn } from '../../utils/cn';
import { Hotkey } from '../Hotkey/Hotkey';
import { RegexpDialog } from '../RegexpDialog/RegexpDialog';

import type { ContextMenuProps, PopupData } from './ContextMenu';
import { ContextMenu } from './ContextMenu';
import type { HoverPopupProps } from './HoverPopup';
import { HoverPopup } from './HoverPopup';
import { useLevelHeight } from './useLevelHeight';

import './Flamegraph.css';


const b = cn('flamegraph');

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
    getHoverText?: (coord: Coordinate) => string;
}


export const Flamegraph: React.FC<FlamegraphProps> = ({
    isDiff,
    theme,
    userSettings,
    profileData,
    goToDefinitionHref,
    onFinishRendering,
    onSuccess,
    getState: getQuery,
    setState: setQuery,
    className,
    onFrameClick,
    getHoverText
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
    const reverse = (getQuery('flamegraphReverse') ?? String(userSettings.reverseFlameByDefault)) === 'true';
    const isDiffSwitchingSupported = profileData.meta.version > 1;

    React.useEffect(() => {
        if (profileData) {
            flamegraphOffsets.current = new FlamegraphOffseter(profileData.rows, { reverse, levelHeight });
        }
    }, [profileData, reverse, levelHeight]);

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
    }, [setQuery]);
    const haveOmittedNodes = Boolean(getQuery('omittedIndexes'));
    const keepOnlyFound = getQuery('keepOnlyFound') === 'true';
    const switchKeepOnlyFound = React.useCallback(() => {
        setQuery({ keepOnlyFound: !keepOnlyFound ? 'true' : false });
    }, [keepOnlyFound, setQuery]);
    const handleOmittedNodesReset = React.useCallback(() => {
        setQuery({ omittedIndexes: false });
    }, [setQuery]);

    const handleSearchUpdate = (text: string, exactMatch?: boolean) => {
        setQuery({ 'flamegraphQuery': encodeURIComponent(text), exactMatch: exactMatch ? 'true' : undefined });
        setShowDialog(false);
    };
    const exactMatch = getQuery('exactMatch');


    React.useEffect(() => {
        if (flamegraphContainer.current && profileData && flamegraphOffsets.current) {
            flamegraphContainer.current.style.setProperty('--flamegraph-font', userSettings.monospace === 'system' ? 'monospace' : 'var(--g-font-family-monospace)');

            const renderOptions = {
                setState: setQuery,
                getState: getQuery,
                theme,
                shortenFrameTexts: userSettings.shortenFrameTexts,
                isDiff,
                searchPattern: search ? exactMatch === 'true' ? decodeURIComponent(search) : RegExp(decodeURIComponent(search)) : null,
                reverse,
                keepOnlyFound,
                onFinishRendering,
            };

            return newFlame(flamegraphContainer.current, profileData, flamegraphOffsets.current, renderOptions);
        }
        return () => { };
    }, [exactMatch, getQuery, isDiff, keepOnlyFound, profileData, reverse, search, setQuery, theme, userSettings, levelHeight, onFinishRendering]);

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
        }
    }, [profileData, onFrameClick, popupData]);

    const handleMouseMove = React.useCallback((e: MouseEvent) => {
        const offsetX = e.offsetX;
        const offsetY = e.offsetY;
        const fg = flamegraphOffsets.current!;
        const coordsClient = fg.getCoordsByPosition(offsetX, offsetY);

        if (!coordsClient) {
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

    return (
        <>
            <div ref={flamegraphContainer} className={b(null, className)}>
                {showDialog && <RegexpDialog
                    showDialog={showDialog}
                    onCloseDialog={() => setShowDialog(false)}
                    onSearchUpdate={handleSearchUpdate}
                    initialSearch={search}
                    initialExact={getQuery('exactMatch') === 'true'}
                />}
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
                            <Button onClick={switchKeepOnlyFound}>
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
                    </div>
                    <div className="flamegraph__frames-count">Showing {framesCount} frames</div>
                </div>

                <div className="flamegraph__annotations">
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

                <div id="profile" className="flamegraph__content">
                    <div ref={canvasRef} onClickCapture={handleOnClick} onContextMenu={handleContextMenu}>
                        <canvas ref={flamegraphCanvas} className="flamegraph__canvas" />
                    </div>
                    <template className="flamegraph__label-template" dangerouslySetInnerHTML={{
                        __html: '<div class="flamegraph__label"></div>' }} />
                    <div className="flamegraph__labels-container" />
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
                />
            )}
            {
                hoverData && <HoverPopup hoverData={hoverData} anchorRef={canvasRef} getText={getHoverText || getHoverTitle}/>
            }
        </>
    );
};
