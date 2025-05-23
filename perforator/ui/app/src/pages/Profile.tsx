import React from 'react';

import { useNavigate, useParams, useSearchParams } from 'react-router-dom';

import { Loader } from '@gravity-ui/uikit';

import { ErrorPanel } from 'src/components/ErrorPanel/ErrorPanel';
import type { ProfileTaskQuery } from 'src/models/Task';
import { redirectToTaskPage } from 'src/utils/profileTask';


export interface ProfileProps {}

export const Profile: React.FC<ProfileProps> = () => {
    const { profileId } = useParams();
    const navigate = useNavigate();

    const [searchParams] = useSearchParams();
    const timestamp = Number(searchParams.get('timestamp') ?? 0);
    const eventType = searchParams.get('event_type');
    const serviceName = searchParams.get('service');


    React.useEffect(() => {
        if (!timestamp) {
            return ;
        }
        const query: ProfileTaskQuery = {
            from: new Date(timestamp - 1).toISOString(),
            to: new Date(timestamp + 1).toISOString(),
            selector: `{id = "${profileId!}", event_type = "${eventType}", service="${serviceName}"}`,
            maxProfiles: 1,
        };


        redirectToTaskPage(navigate, query, true);
    });

    if (!timestamp) {
        return <ErrorPanel message="No timestamp was specified" />;
    }

    return <Loader />;
};
