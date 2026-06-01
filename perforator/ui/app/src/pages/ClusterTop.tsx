import { useCallback, useEffect, useMemo } from 'react';

import { useQuery } from '@tanstack/react-query';

import { Alert, Loader } from '@gravity-ui/uikit';

import { Beta } from 'src/components/Beta/Beta';
import { ClusterTopTable } from 'src/components/ClusterTopTable/ClusterTopTable';
import { clusterTopGenerationsQueryKeys, getGenerationCachePolicy } from 'src/components/ClusterTopTable/queries';
import { countHoursInterval } from 'src/components/ClusterTopTable/utils';
import { ErrorPanel } from 'src/components/ErrorPanel/ErrorPanel';
import { GenerationCalendarSelector } from 'src/components/GenerationCalendarSelector';
import { ClusterTopGenerationStatus } from 'src/generated/perforator/proto/perforator/perforator';
import { apiClient } from 'src/utils/api';
import { useTypedQuery } from 'src/utils/query';

import type { Page } from './Page';


export const ClusterTop: Page = (props) => {
    const [getQuery, setQuery] = useTypedQuery<'generation'>();
    const currentGeneration = getQuery('generation', '') ?? '';
    const setGeneration = useCallback((value: string) => setQuery({ generation: value }), [setQuery]);
    const {
        error,
        data: generations,
        isPending,
    } = useQuery({
        queryKey: clusterTopGenerationsQueryKeys(),
        queryFn: () => apiClient.getGenerations(null, {}).then(value => value.data),
        ...getGenerationCachePolicy(ClusterTopGenerationStatus.COMPLETED),
    });
    const currentGenerationObject = useMemo(
        () => generations?.Generations.find(({ ID }) => ID === Number(currentGeneration)),
        [currentGeneration, generations?.Generations],
    );

    useEffect(() => {
        if (currentGeneration === '' && !isPending && !error && (generations?.Generations?.length ?? 0) > 0) {
            setGeneration(String(generations?.Generations[0].ID));
        }
    }, [currentGeneration, error, generations?.Generations, isPending, setGeneration]);

    const timeInterval = currentGenerationObject ? countHoursInterval(currentGenerationObject) : null;

    if (error) {
        return <ErrorPanel message={error?.message}/>;
    }

    if (isPending) {
        return <Loader/>;
    }

    return (<>
        {props.header}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <div>Cluster Top <Beta/></div>
            <GenerationCalendarSelector
                generations={generations?.Generations ?? []}
                value={currentGeneration}
                onUpdate={setGeneration}
            />
            {currentGenerationObject?.GenerationStatus === ClusterTopGenerationStatus.IN_PROGRESS && (
                <Alert
                    theme="warning"
                    message="Cluster top for this generation is still being built. Data may be incomplete and can change. Switch to a completed generation for accurate results."
                />
            )}
            {currentGeneration && currentGenerationObject && timeInterval &&
                    <ClusterTopTable
                        generation={Number(currentGeneration)}
                        generationStatus={currentGenerationObject.GenerationStatus}
                        timeInterval={timeInterval}
                        timeIntervalFrom={currentGenerationObject.From}
                        timeIntervalTo={currentGenerationObject.To}
                    />
            }
        </div>
    </>
    );
};
