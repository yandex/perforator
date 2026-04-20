import React from 'react';

import { FullscreenContext } from './FullscreenContext';


export const FullscreenProvider: React.FC<{children: React.ReactNode; initialEnalbed: boolean}> = ({ children, initialEnalbed }) => {
    const [enabled, setEnabled] = React.useState(initialEnalbed);

    return <FullscreenContext.Provider value={{ enabled, setEnabled }}>{children}</FullscreenContext.Provider>;
};
