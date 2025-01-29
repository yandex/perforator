import React from 'react';

import { ArrowShapeTurnUpRight, ChevronDown } from '@gravity-ui/icons';
import { Button, DropdownMenu, Icon } from '@gravity-ui/uikit';

import { LocalStorageKey } from 'src/const/localStorage';
import { uiFactory } from 'src/factory';
import { createSuccessToast } from 'src/utils/toaster';

import type { ShareFormat } from './utils';
import { SHARE_FORMAT_LINK } from './utils';

import './ShareButton.scss';


const SHARE_ICON_SIZE = 16;
const DEFAULT_SHARE_FORMAT = {
    builder: SHARE_FORMAT_LINK,
    name: 'Link',
};

const listShareFormats = (formats: ShareFormat[], selected: ShareFormat) => {
    const selectedIndex = formats.indexOf(selected);
    if (selectedIndex !== -1) {
        formats.splice(selectedIndex, 1);
        formats.unshift(selected);
    }
    return formats;
};

export interface ShareButtonProps {
    getUrl: () => string;
}

export const ShareButton: React.FC<ShareButtonProps> = props => {
    const [shareFormat, setShareFormat] = React.useState<ShareFormat>(
        localStorage.getItem(LocalStorageKey.ShareFormat) || DEFAULT_SHARE_FORMAT.name,
    );

    const shareFormats = React.useMemo(() => uiFactory().shareFormats(), []);
    const formats = React.useMemo(
        () => shareFormats.map(([name, _]) => name),
        [shareFormats],
    );
    const builders = React.useMemo(
        () => Object.fromEntries(shareFormats),
        [shareFormats],
    );

    const copyShareString = (format: ShareFormat) => {
        setShareFormat(format);
        localStorage.setItem(LocalStorageKey.ShareFormat, format);

        const builder = builders[format] || DEFAULT_SHARE_FORMAT.builder;
        const shared = builder(props.getUrl());
        navigator.clipboard.writeText(shared);

        createSuccessToast({
            name: 'share-copy',
            title: 'Copied to clipboard',
        });
    };

    const items = listShareFormats(formats, shareFormat).map(format => ({
        text: format,
        action: () => copyShareString(format),
    }));

    return (
        <span className="share-button">
            <Button
                className="share-button__text"
                pin="round-clear"
                onClick={() => copyShareString(shareFormat)}
            >
                <Icon size={SHARE_ICON_SIZE} data={ArrowShapeTurnUpRight} />
                Share
            </Button>
            <DropdownMenu
                items={items}
                popupProps={{ placement: 'bottom-end' }}
                switcher={
                    <Button
                        className="share-button__chevron"
                        pin="clear-round"
                    >
                        <Icon size={SHARE_ICON_SIZE} data={ChevronDown} />
                    </Button>
                }
            />
        </span>
    );
};
