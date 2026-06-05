import React from 'react';

import './ErrorPage.scss';


const PICTURE_SIZE = 300;

export interface ErrorPageProps {
    picture: React.ComponentType<React.SVGProps<SVGSVGElement>>;
    title: string;
}

export const ErrorPage: React.FC<ErrorPageProps> = ({ picture, title }: ErrorPageProps) => {
    return (
        <div className="error-page">
            {React.createElement(picture, { height: PICTURE_SIZE, width: PICTURE_SIZE })}
            <h2 className="error-page__title">{title}</h2>
        </div>
    );
};
