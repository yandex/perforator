import React from 'react';

import { ArrowShapeTurnUpRight, ChevronDown } from '@gravity-ui/icons';
import { Button, DropdownMenu, Icon } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import { cn } from 'src/utils/cn';
import { createErrorToast, createSuccessToast } from 'src/utils/toaster';

import type { ShareFormat } from './utils';
import { SHARE_FORMAT_LINK } from './utils';

import './ShareButton.scss';


const SHARE_ICON_SIZE = 16;
const DEFAULT_SHARE_FORMAT = {
    builder: SHARE_FORMAT_LINK,
    name: 'Link',
};

export interface ShareButtonProps {
    getUrl: () => string;
    view?: 'compact' | 'full';
    size?: 's' | 'm';
    className?: string;
}

const b = cn('share-button');

export const ShareButton: React.FC<ShareButtonProps> = props => {
    const shareFormats = React.useMemo(() => uiFactory().shareFormats(), []);
    const builders = Object.fromEntries(shareFormats);

    const copyShareString = (format: ShareFormat) => {
        const builder = builders[format] || DEFAULT_SHARE_FORMAT.builder;
        const shared = builder(props.getUrl());
        navigator.clipboard.writeText(shared)
            .then(() => createSuccessToast({
                name: 'share-copy',
                title: 'Copied to clipboard',
            }))
            .catch(e => createErrorToast(e, { name: 'share-copy' }));
    };

    const formats = shareFormats.map(([name, _]) => name);
    const items = formats.map(format => ({
        text: format,
        action: () => copyShareString(format),
    }));

    return (
        <span className={b(null, props.className)}>
            <Button
                className={b('text')}
                pin="round-clear"
                onClick={() => copyShareString(DEFAULT_SHARE_FORMAT.name)}
                size={props.size}
            >
                <Icon size={SHARE_ICON_SIZE} data={ArrowShapeTurnUpRight} />
                {props.view === 'compact' ? null : 'Share'}
            </Button>
            <DropdownMenu
                items={items}
                popupProps={{ placement: 'bottom-end' }}
                switcher={
                    <Button
                        className={b('chevron')}
                        pin="clear-round"
                        size={props.size}
                    >
                        <Icon size={SHARE_ICON_SIZE} data={ChevronDown} />
                    </Button>
                }
            />
        </span>
    );
};
