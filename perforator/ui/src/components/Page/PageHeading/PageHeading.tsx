import React from 'react';

import { useSearchParams } from 'react-router-dom';

import { ArrowUpRightFromSquare } from '@gravity-ui/icons';
import { Icon, Link } from '@gravity-ui/uikit';

import './PageHeading.scss';


const makePerforatorUrl = (searchParams: URLSearchParams) => {
    const ignoredParams = [
        'embed',
        '_theme',
    ];
    ignoredParams.forEach(param => searchParams.delete(param));
    const url = window.location.href.split('?')[0];
    return `${url}?${searchParams}`;
};

export interface PageHeadingProps {
    embed: boolean;
}

export const PageHeading: React.FC<PageHeadingProps> = props => {
    const [searchParams] = useSearchParams();

    if (props.embed) {
        return (
            <div className="page-heading">
                <Link
                    view="normal"
                    href={makePerforatorUrl(searchParams)}
                    target="_blank"
                >
                    View in Perforator
                    <Icon className="page-heading__link-arrow" data={ArrowUpRightFromSquare} />
                </Link>
            </div>
        );
    }
    return (
        <div className="page-heading">
            <h1 className="page-heading__title">Perforator</h1>
        </div>
    );
};
