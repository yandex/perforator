import React from 'react';

import './ErrorPage.scss';


const PICTURE_SIZE = 300;

export interface ErrorPageProps {
    picture: React.ComponentType<React.SVGProps<SVGSVGElement>>;
    title: string;
}

export const ErrorPage: React.FC<ErrorPageProps> = props => {
    return (
        <div className="error-page">
            {React.createElement(props.picture, { height: PICTURE_SIZE, width: PICTURE_SIZE })}
            <h2 className="error-page__title">{props.title}</h2>
        </div>
    );
};
