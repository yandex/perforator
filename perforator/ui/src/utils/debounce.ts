import React from 'react';


const DEBOUNCE_TIMEOUT = 300;

export const useDebounce = () => {
    const timer = React.useRef<ReturnType<typeof setTimeout>>();
    return <ReturnType>(
        callback: () => ReturnType,
        timeout = DEBOUNCE_TIMEOUT,
    ): Promise<ReturnType> => {
        if (timer.current) {
            clearTimeout(timer.current);
        }
        return new Promise(resolve => {
            timer.current = setTimeout(() => resolve(callback()), timeout);
        });
    };
};
