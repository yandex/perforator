import React from 'react';

import { FooterItem } from '@gravity-ui/navigation';
import { Icon } from '@gravity-ui/uikit';

import { openLink } from './utils';


const ITEM_ICON_SIZE = 18;

export interface NavigationFooterLinkProps {
    text: string;
    compact: boolean;
    icon?: (props: React.SVGProps<SVGSVGElement>) => React.JSX.Element;
    renderIcon?: () => React.JSX.Element;
    url?: string;
    onClick?: () => void;
}

export const NavigationFooterLink: React.FC<NavigationFooterLinkProps> = ({ text, compact, icon, renderIcon, url, onClick }: NavigationFooterLinkProps) => {
    const handleClick = onClick ?? (() => openLink(url));
    const iconElement = renderIcon
        ? renderIcon()
        : (icon ? (<Icon size={ITEM_ICON_SIZE} data={icon} />) : null);
    return (
        <FooterItem
            compact={compact}
            item={{
                id: text,
                title: text,
                onItemClick: handleClick,
                itemWrapper: (params, makeItem) => makeItem({ ...params, icon: iconElement }),
            }}
        />
    );
};
