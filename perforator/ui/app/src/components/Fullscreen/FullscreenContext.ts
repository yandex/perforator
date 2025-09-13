import React from 'react';


interface FullscreenContextProps {
    enabled: boolean;
    setEnabled: (newValue: boolean) => void;
}

export const FullscreenContext = React.createContext<FullscreenContextProps | undefined>(undefined);

export const useFullscreen = () => {
    const value = React.useContext(FullscreenContext);
    if (value === undefined) {
        throw new Error('FullscreenContext is not provided');
    }
    return value;
};
