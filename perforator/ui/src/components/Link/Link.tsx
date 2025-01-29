import React from 'react';

import type { LinkProps as LinkNavigationProps } from 'react-router-dom';
import { useLinkClickHandler } from 'react-router-dom';

import type { LinkProps as LinkPropsBase } from '@gravity-ui/uikit';
import { Link as LinkBase } from '@gravity-ui/uikit';


interface LinkProps
    extends LinkPropsBase,
        Pick<LinkNavigationProps, 'replace' | 'state'> {}

export const Link = React.forwardRef<HTMLAnchorElement, LinkProps>(
    ({ onClick, replace = false, state, target, href, ...rest }, ref) => {

        const handleClick = useLinkClickHandler(href, {
            replace,
            state,
            target,
        });

        return (
            <LinkBase
                {...rest}
                href={href}
                onClick={(event) => {
                    onClick?.(event);
                    if (!event.defaultPrevented) {
                        handleClick(event as React.MouseEvent<HTMLAnchorElement>);
                    }
                }}
                ref={ref}
                target={target}
            />
        );
    },
);

Link.displayName = 'Link';
