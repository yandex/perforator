import React from 'react';

import type { FooterMenuItem } from '@gravity-ui/navigation';
import { Footer } from '@gravity-ui/navigation';

import { uiFactory } from 'src/factory';

import './PageFooter.scss';


export const PageFooter: React.FC = () => {
    const items: FooterMenuItem[] = [];
    if (uiFactory().docsLink()) {
        items.push({
            text: 'Docs',
            href: uiFactory().docsLink(),
            target: '_blank',
            className: 'page-footer__menu-item',
        });
    }
    const version = import.meta.env?.VITE_RELEASE_VERSION ?? import.meta.env?.VITE_REVISION;
    if (version) {
        items.push({
            text: `Version: ${import.meta.env?.VITE_RELEASE_VERSION ?? import.meta.env?.VITE_REVISION}`,
            href: uiFactory().ciLink(),
            target: '_blank',
            className: 'page-footer__menu-item',
        });
    }
    return (
        <Footer
            className="page-footer"
            copyright={uiFactory().footerCopyright()}
            menuItems={items}
        />
    );
};
