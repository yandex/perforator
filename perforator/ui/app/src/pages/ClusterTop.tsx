import { useCallback, useEffect, useMemo } from 'react';

import { Alert, Label, Loader, Select, type SelectOption } from '@gravity-ui/uikit';

import { ClusterTopTable } from 'src/components/ClusterTopTable/ClusterTopTable';
import { countHoursInterval } from 'src/components/ClusterTopTable/utils';
import { ErrorPanel } from 'src/components/ErrorPanel/ErrorPanel';
import { useAsyncResult } from 'src/components/TaskReport/TaskFlamegraph/useFetchResult';
import { ClusterTopGenerationStatus } from 'src/generated/perforator/proto/perforator/perforator';
import { apiClient } from 'src/utils/api';
import { useTypedQuery } from 'src/utils/query';

import type { Page } from './Page';


function generationStatusLabel(status: ClusterTopGenerationStatus) {
    switch (status) {
    case ClusterTopGenerationStatus.IN_PROGRESS:
        return <Label theme="warning" size="xs">Building</Label>;
    case ClusterTopGenerationStatus.COMPLETED:
        return <Label theme="success" size="xs">Completed</Label>;
    default:
        return <Label theme="unknown" size="xs">Unknown</Label>;
    }
}

export const ClusterTop: Page = (props) => {
    const [getQuery, setQuery] = useTypedQuery<'generation'>();
    const currentGeneration = getQuery('generation', '') ?? '';
    const setGeneration = (value: string) => setQuery({ generation: value });
    const getData = useCallback(() => {
        return apiClient
            .getGenerations(null, {}).then(value => value.data);
    }, []);
    const { error, data: generations, loading } = useAsyncResult({ getData: getData });
    const options = useMemo<SelectOption[]>(() => {
        return (
            generations?.Generations.map((gen) => ({
                content: <>{gen.ID}: {gen.From} - {gen.To} {generationStatusLabel(gen.GenerationStatus)}</>,
                value: String(gen.ID),
            } as SelectOption)) ?? []
        );
    }, [generations]);
    const currentGenerationObject = useMemo(
        () => generations?.Generations.find(({ ID }) => ID === Number(currentGeneration)),
        [currentGeneration, generations?.Generations],
    );

    useEffect(() => {
        if (currentGeneration === '' && !loading && !error && (generations?.Generations?.length ?? 0) > 0) {
            setGeneration(String(generations?.Generations[0].ID));
        }
    }, [currentGeneration, error, generations?.Generations, loading, setGeneration]);

    const timeInterval = currentGenerationObject ? countHoursInterval(currentGenerationObject) : null;

    if (error) {
        return <ErrorPanel message={error?.message}/>;
    }

    if (loading) {
        return <Loader/>;
    }

    return (<>
        {props.header}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <div>Cluster Top</div>
            <Select
                options={options}
                value={[currentGeneration]}
                onUpdate={([val]) => setGeneration(val)}
                placeholder={'generation'}
                renderSelectedOption={(option) => <>{option.content}</>}
                width="auto"
            />
            {currentGenerationObject?.GenerationStatus === ClusterTopGenerationStatus.IN_PROGRESS && (
                <Alert
                    theme="warning"
                    message="Cluster top for this generation is still being built. Data may be incomplete and can change. Switch to a completed generation for accurate results."
                />
            )}
            {currentGeneration && currentGenerationObject && timeInterval && <ClusterTopTable generation={Number(currentGeneration)} timeInterval={timeInterval}/>}
        </div>
    </>
    );
};
