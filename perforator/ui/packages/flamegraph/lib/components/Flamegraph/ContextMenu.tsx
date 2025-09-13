import * as React from 'react'

import { CopyCheck } from '@gravity-ui/icons';
import type { MenuItemProps, PopupProps, ToastProps } from '@gravity-ui/uikit';
import { CopyToClipboard, Icon, Menu, Popup } from '@gravity-ui/uikit';

import { Hotkey } from '../Hotkey/Hotkey';
import type { StringifiedNode } from '../../models/Profile';

import { getAtLessPath } from '../../file-path';
import type { GetStateFromQuery, SetStateFromQuery } from '../../query-utils';
import { parseStacks, stringifyStacks } from '../../query-utils';
import type { QueryKeys } from '../../renderer';
import { GoToDefinitionHref } from '../../models/goto';


export type PopupData = { offset: [number, number]; node: StringifiedNode; coords: [number, number] };



export type ContextMenuProps = {
    popupData: PopupData;
    anchorRef: PopupProps['anchorRef'];
    onClosePopup: () => void;
    setQuery: SetStateFromQuery<QueryKeys>;
    getQuery: GetStateFromQuery<QueryKeys>;
    goToDefinitionHref: GoToDefinitionHref;
    onSuccess: (options: Pick<ToastProps, 'renderIcon' | 'name' | 'content'>) => void
};
export const ContextMenu: React.FC<ContextMenuProps> = ({ popupData, anchorRef, onClosePopup, setQuery, getQuery, goToDefinitionHref, onSuccess }) => {
    const href = goToDefinitionHref(popupData.node);
    const hasFile = Boolean(popupData.node.file);
    const shouldShowGoTo = (
        hasFile &&
        popupData.node.frameOrigin !== 'kernel' &&
        href
    );
    const commonButtonProps: Partial<MenuItemProps> = {
        onClick: onClosePopup,
    } as const;

    return <Popup
        open={Boolean(popupData)}
        anchorRef={anchorRef}
        offset={{crossAxis: popupData.offset[0], mainAxis: popupData.offset[1]}}
        floatingClassName={'flamegraph__popup'}
        placement={['top-start']}
        onEscapeKeyDown={onClosePopup}
    >
        <Menu>
            <CopyToClipboard text={popupData.node.textId} >
                {() => <Menu.Item
                    {...commonButtonProps}
                    onClick={() => {
                        onSuccess({ renderIcon: () => <Icon data={CopyCheck}/>, name: 'copy', content: 'Name copied to clipboard' });
                        onClosePopup();
                    }}
                >

                Copy name
                </Menu.Item>}
            </CopyToClipboard>
            {shouldShowGoTo ? (
                <Menu.Item
                    {...commonButtonProps}
                    href={href}
                    target="_blank"
                >
                Go to source
                </Menu.Item>

            ) : null}
            {hasFile ? (
                <CopyToClipboard text={getAtLessPath(popupData.node)} >
                    {() => <Menu.Item
                        {...commonButtonProps}
                        onClick={() => {
                            onSuccess({ renderIcon: () => <Icon data={CopyCheck}/>, name: 'copy', content: 'File path copied to clipboard' });
                            onClosePopup();
                        }}
                    >
                    Copy file path
                    </Menu.Item>}
                </CopyToClipboard>
            ) : null}
            <Menu.Item 
            {...commonButtonProps} 
            onClick={() => {
                setQuery({
                    exactMatch: 'true',
                    flamegraphQuery: popupData.node.textId,
                });
                onClosePopup();
            }}>
                Find similar nodes
            </Menu.Item>
            <Menu.Group>
                <Menu.Item
                    {...commonButtonProps}
                    onClick={() => {
                        const omitted = parseStacks(getQuery('omittedIndexes', '') || '');
                        omitted.push(popupData.coords);
                        setQuery({ omittedIndexes: stringifyStacks(omitted) });
                        onClosePopup();
                    }}
                >
            Omit stack
                    <Hotkey value="alt+click" />
                </Menu.Item>
            </Menu.Group>
        </Menu>
    </Popup>;
};
