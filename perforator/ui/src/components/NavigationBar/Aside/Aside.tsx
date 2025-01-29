import React from 'react';

import BarsDescendingAlignLeftIcon from '@gravity-ui/icons/svgs/bars-descending-align-left.svg?raw';
import ClockArrowRotateLeftIcon from '@gravity-ui/icons/svgs/clock-arrow-rotate-left.svg?raw';
import ScalesUnbalancedIcon from '@gravity-ui/icons/svgs/scales-unbalanced.svg?raw';
import type { MenuItem } from '@gravity-ui/navigation';
import { PageLayoutAside } from '@gravity-ui/navigation';

import PerforatorLogo from 'src/assets/perforator.svg?raw';
import { Link } from 'src/components/Link/Link';
import { uiFactory } from 'src/factory';

import { NavigationFooter } from '../NavigationFooter/NavigationFooter';
import { SettingsPanel } from '../SettingsPanel/SettingsPanel';

import type { AsideProps } from './AsideProps';


interface MenuLink {
    title: string;
    icon: string;
    link: string;
}

const menuLinks: MenuLink[] = [
    {
        title: 'Profiles',
        icon: BarsDescendingAlignLeftIcon,
        link: '/',
    },
    {
        title: 'History',
        icon: ClockArrowRotateLeftIcon,
        link: '/tasks',
    },
    {
        title: 'Diff',
        icon: ScalesUnbalancedIcon,
        link: '/diff',
    },
];

const makeMenuItem = (link: MenuLink): MenuItem => ({
    id: link.title,
    title: link.title,
    icon: link.icon,
    current: window.location.pathname === link.link,
    itemWrapper: (props, makeItem) => (
        <Link
            className="gn-composite-bar-item__link"
            href={link.link || '#'}
        >
            {makeItem(props)}
        </Link>
    ),
});

export const Aside: React.FC<AsideProps> = (props) => {
    const asideRef = React.useRef<HTMLDivElement>(null);

    const [showSettings, setShowSettings] = React.useState<boolean>(false);

    const panelItems = React.useMemo(() => [
        {
            id: 'settings',
            content: <SettingsPanel />,
            visible: showSettings,
        },
    ], [showSettings]);

    return (
        <PageLayoutAside
            ref={asideRef}
            logo={{
                icon: PerforatorLogo,
                text: 'Perforator',
                iconSize: 32,
                href: '/',
            }}
            headerDecoration
            onChangeCompact={props.setCompact}
            subheaderItems={uiFactory().useSubheaderItems(asideRef)}
            menuItems={menuLinks.map(makeMenuItem)}
            panelItems={panelItems}
            onClosePanel={() => setShowSettings(false)}
            renderFooter={({ compact }) => (
                <NavigationFooter
                    compact={compact}
                    toggleSettings={() => setShowSettings(settings => !settings)}
                />
            )}
        />
    );
};
