import React from 'react';

import './DefinitionList.scss';


export type DefinitionListItem = [string, React.ReactNode];

export interface DefinitionListProps {
    items: DefinitionListItem[];
}

export const DefinitionList: React.FC<DefinitionListProps> = props => {
    const elements = props.items
        .filter(([_, value]) => Boolean(value))
        .map(([key, value]) => (
            <div className="definition-list__row" key={key}>
                <dt>{key}</dt>
                <dd>{value}</dd>
            </div>
        ));
    return (
        <dl className="definition-list">
            {elements}
        </dl>
    );
};
