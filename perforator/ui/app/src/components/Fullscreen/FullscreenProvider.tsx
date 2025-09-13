import React from 'react';

import { FullscreenContext } from './FullscreenContext';


export const FullscreenProvider: React.FC<{children: React.ReactNode}> = ({ children }) => {
    const [enabled, setEnabled] = React.useState(false);

    return <FullscreenContext.Provider value={{ enabled, setEnabled }}>{children}</FullscreenContext.Provider>;
};
