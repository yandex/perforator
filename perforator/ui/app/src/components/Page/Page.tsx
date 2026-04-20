import type React from 'react';


export interface PageInternalProps {
    header: React.ReactElement;
}

export interface PagePublicProps {
    embed: boolean;
}

export interface PageProps extends PageInternalProps, PagePublicProps {}

export type Page = React.FC<PageProps>;

export type PageComponent = React.ComponentType<PageProps>
