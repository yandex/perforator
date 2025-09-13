import * as React from 'react'
import { TopTable, type TopTableProps } from '../TopTable/TopTable'
import { Divider } from '@gravity-ui/uikit'
import { Flamegraph, type FlamegraphProps } from '../Flamegraph/Flamegraph'
import { cn } from '../../utils/cn'
import { calculateTopForTable } from '../../top'
import { parseStacks } from '../../query-utils'
import './SideBySide.css'


export type SideBySideProps = FlamegraphProps & Pick<TopTableProps, 'navigate'>

const b = cn('visualisation_sbs')

export function SideBySide(props: SideBySideProps) {
    const {profileData, getState} = props;
    const frameDepth = parseInt(getState('frameDepth', '0'));
    const framePos = parseInt(getState('framePos', '0'));
    const omitted = getState('omittedIndexes', '');



    const topData = React.useMemo(() => {
        return profileData ? calculateTopForTable(profileData.rows, profileData.stringTable.length, {rootCoords: [frameDepth, framePos] as [number, number], omitted: parseStacks(omitted)}) : null;
    }, [profileData, frameDepth, framePos, omitted]);
    return <div className={b()}>
                <TopTable
                    lines={100}
                    disableAutoTabSwitch
                    className={b('top-table')}
                    {...props}
                    topData={topData!}
                />
                <Divider orientation={'vertical'} />
                <Flamegraph
                    className={b('flamegraph')}
                    {...props}
                />
            </div>
}
