/* eslint-disable @typescript-eslint/member-ordering */
/* eslint-disable @typescript-eslint/no-unused-vars */

import React from 'react';

import type { SubheaderMenuItem } from '@gravity-ui/navigation';

import type { QueryInput } from 'src/components/MergeProfilesForm/QueryInput';
import { QUERY_INPUTS } from 'src/components/MergeProfilesForm/queryInputs';
import { Select, type SelectProps } from 'src/components/Select/Select';
import type { ShareStringBuilder } from 'src/components/ShareButton/utils';
import { SHARE_FORMATS } from 'src/components/ShareButton/utils';
import type { ProfileData } from 'src/models/Profile';
import type { SendError } from 'src/utils/error';
import { fakeRum, type Rum } from 'src/utils/rum';


export class UIFactory {
    configureApp = (): void => {};

    gravityStyles = () => true;

    docsLink = (): Optional<string> => 'https://perforator.tech/docs/';
    supportChatLink = (): Optional<string> => undefined;
    bugReportLink = (): Optional<string> => undefined;
    ciLink = (): Optional<string> => undefined;

    makeTraceUrl = (traceId: string): Optional<string> => undefined;

    authorizationSupported = () => false;
    loginCookie = (): Optional<string> => undefined;
    defaultUser = (): Optional<string> => undefined;
    renderUserLink = (user: Optional<string>): React.ReactNode => user;
    makeUserLink = (user: Optional<string>): Optional<string> => undefined;
    makeUserAvatarLink = (user: Optional<string>): Optional<string> => undefined;

    defaultCluster = () => 'unknown';

    clusterName = () => 'Zone';
    serviceName = () => 'Service';
    podName = () => 'Name';
    nodeName = () => 'Node';

    makeServiceUrl = (cluster: string, service: Optional<string>): Optional<string> => undefined;
    makePodUrl = (cluster: string, pod: Optional<string>): Optional<string> => undefined;
    makeNodeUrl = (cluster: string, node: Optional<string>): Optional<string> => undefined;

    shareFormats = (): [string, ShareStringBuilder][] => SHARE_FORMATS;

    subheaderItemsCount = () => 0;
    useSubheaderItems = (asideRef: React.RefObject<HTMLDivElement>): SubheaderMenuItem[] => [];

    queryInputs = (): QueryInput[] => QUERY_INPUTS;

    rum = (): Rum => fakeRum;
    logError: SendError = (error, additional, level) => console.error(error, additional, level);

    renderSelect = (props: SelectProps): React.ReactNode => (<Select {...props} />);

    parseLegacyFormat: ((data: string) => ProfileData) | undefined = undefined;

    queryLanguageDocsLink = (): string => 'https://perforator.tech/docs/en/reference/querylang';

    footerCopyright = (): string => '';

    sampleSizes = () => [1, 10, 50, 100, 500, 1000];
    defaultSampleSize = () => 100;
}
