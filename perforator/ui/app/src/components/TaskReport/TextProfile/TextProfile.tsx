import React from 'react';

import { ErrorPanel } from 'src/components/ErrorPanel/ErrorPanel';

import { useFetchResult } from '../TaskFlamegraph/useFetchResult';

import './TextProfile.css';


export type TextProfileProps = {
    url: string;
}

export const TextProfile: React.FC<TextProfileProps> = ({ url }) => {
    const { data: text = '', error } = useFetchResult({ url, extractData: res => res.text() });

    if (error) {
        return <ErrorPanel message={error.message}/>;
    }

    return <code className="text-profile" >{text}</code>;
};
