import type React from 'react';


export interface PageProps {
    embed: boolean;
}

export type Page = React.FC<PageProps>;

export type PageComponent = React.ComponentType<PageProps>;
