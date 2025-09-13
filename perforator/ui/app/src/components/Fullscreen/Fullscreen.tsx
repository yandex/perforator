import React from 'react';

import { useFullscreen } from './FullscreenContext';

import './Fullscreen.css';


export const Fullscreen: React.FC<{children: React.ReactNode}> = ({ children }) => {
    const { enabled } = useFullscreen();
    return <div className={enabled ? 'fullscreen' : undefined}>{children}</div>;
};
