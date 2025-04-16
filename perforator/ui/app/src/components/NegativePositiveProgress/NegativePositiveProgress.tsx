import React from 'react';

import { Progress, type Stack as ProgressStack } from '@gravity-ui/uikit';

import './NegativePositiveProgress.scss';


export type NegativePositiveProgressProps = {
    value: number;
    text: string;
}

const half = 50;


function clamp(n: number, lower: number, upper: number) {
    return Math.max(lower, Math.min(n, upper));
}

export const NegativePositiveProgress: React.FC<NegativePositiveProgressProps> = ({ value: rawValue, text }) => {
    const value = clamp(rawValue, -half, half);
    const stack: ProgressStack[] = [
        { value: value > 0 ? half : half + value, color: 'transparent' },
        { value: Math.abs(value), theme: (value > 0 ? 'success' : 'danger') },
        { value: value > 0 ? half - value : half, color: 'transparent' },
    ];

    return <Progress
        className={'negative-positive'}
        text={text}
        size="m"
        stack={stack}
    />;
};
