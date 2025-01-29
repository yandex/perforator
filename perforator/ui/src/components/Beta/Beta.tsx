import React from 'react';

import './Beta.scss';


export interface BetaProps {}

export const Beta: React.FC<BetaProps> = () => {
    return <span className="beta">β</span>;
};
