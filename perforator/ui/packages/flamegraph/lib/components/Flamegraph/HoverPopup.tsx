import * as React from 'react';
import { useState } from 'react';

import { useClientPoint, useFloating } from '@floating-ui/react';
import { shift, size } from '@floating-ui/react-dom';

import { Popup } from '@gravity-ui/uikit';

import { colorize } from './bracketColorizer';
import type { PopupData } from './ContextMenu';


export type HoverPopupProps = {
    hoverData: Omit<PopupData, 'node'>;
    anchorRef: React.RefObject<HTMLElement>;
    getText: (arg: PopupData['coords']) => string;
};
const TIMEOUT_MS = 500;
const MAX_WIDTH = 600;

export const HoverPopup: React.FC<HoverPopupProps> = ({ hoverData, anchorRef, getText }) => {
    const [isOpen, setIsOpen] = useState(false);
    const timeoutRef = React.useRef(null);
    const floatingMiddlewares = [
        shift({
            boundary: anchorRef?.current,
            crossAxis: true,
        }),
        size({
            apply: ({availableWidth, elements}) => {
                const value = `${Math.max(0, Math.min(MAX_WIDTH ,availableWidth))}px`;
                elements.floating.style.maxWidth = value;
            },
            boundary: anchorRef?.current
         }),
    ];
    const { refs, floatingStyles, context } = useFloating({
        open: isOpen,
        placement: 'bottom-start',
        middleware: floatingMiddlewares,
        onOpenChange: setIsOpen,
    });
    const point = useClientPoint(context,
        { x: hoverData?.offset?.[0], y: hoverData?.offset?.[1] },
    );

    const hoverX = hoverData?.offset?.[0];
    const hoverY = hoverData?.offset?.[1];

    React.useEffect(() => {
        if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
            setIsOpen(false);
        }
        timeoutRef.current = setTimeout(() => {
            setIsOpen(true);
        }, TIMEOUT_MS);

        return () => {
            clearTimeout(timeoutRef.current);
        };
    }, [hoverX, hoverY]);

    React.useEffect(() => {
        const element = anchorRef?.current;
        if (element !== undefined && element !== refs.reference.current) {
            refs.setReference(element);
        }
    }, [anchorRef, refs]);


    return (
        <Popup
            open={isOpen}
            floatingInteractions={[point]}
            floatingContext={context}
            floatingStyles={floatingStyles}
            floatingMiddlewares={floatingMiddlewares}
            floatingClassName={'flamegraph__func-popup'}
            anchorRef={anchorRef}
            floatingRef={refs.setFloating}
            disableEscapeKeyDown={true}
            disableTransition
        >
            <div>
                {isOpen && colorize(getText(hoverData.coords))}
            </div>
        </Popup>
    );
};
