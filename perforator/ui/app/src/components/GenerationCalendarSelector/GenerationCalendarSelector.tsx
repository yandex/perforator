import { useEffect, useMemo, useRef, useState } from 'react';

import { Calendar } from '@gravity-ui/date-components';
import { type DateTime, dateTimeParse } from '@gravity-ui/date-utils';
import { Calendar as CalendarIcon } from '@gravity-ui/icons';
import { Button, Icon, Popup, Select, type SelectOption } from '@gravity-ui/uikit';

import type { ClusterTopGeneration } from 'src/generated/perforator/proto/perforator/perforator';
import { cn } from 'src/utils/cn';

import { generationStatusLabel } from './statusLabel';
import { classifyGenerations, dayKeyOf, formatGenLabel } from './validation';

import './GenerationCalendarSelector.css';


interface Props {
    generations: ClusterTopGeneration[];
    value: string;
    onUpdate: (id: string) => void;
}

function dateToKey(date: DateTime): string {
    return date.format('YYYY-MM-DD');
}

const b = cn('generation-calendar');

export function GenerationCalendarSelector({ generations, value, onUpdate }: Props) {
    const classified = useMemo(() => classifyGenerations(generations), [generations]);

    const currentGen = useMemo(
        () => generations.find((g) => String(g.ID) === value),
        [generations, value],
    );

    const currentDayKey = currentGen ? dayKeyOf(currentGen) : undefined;

    const [selectedDay, setSelectedDay] = useState<string | undefined>(currentDayKey);
    const [calendarOpen, setCalendarOpen] = useState(false);
    const buttonRef = useRef<HTMLButtonElement>(null);

    useEffect(() => {
        if (currentDayKey && currentDayKey !== selectedDay) {
            setSelectedDay(currentDayKey);
        }
    }, [currentDayKey, selectedDay]);

    const calendarValue = useMemo<DateTime | null>(() => {
        const key = selectedDay ?? currentDayKey;
        return key ? dateTimeParse(key) ?? null : null;
    }, [selectedDay, currentDayKey]);

    const handleCalendarUpdate = (date: DateTime) => {
        const key = dateToKey(date);
        const list = classified.byDay.get(key);
        if (!list || list.length === 0) {
            return;
        }
        setSelectedDay(key);
        if (classified.mode === 'daily') {
            onUpdate(String(list[0].ID));
        } else if (!list.some((g) => String(g.ID) === value)) {
            onUpdate(String(list[0].ID));
        }
        setCalendarOpen(false);
    };

    const subdayOptions = useMemo<SelectOption[]>(() => {
        if (classified.mode !== 'subday' || !selectedDay) {
            return [];
        }
        return (classified.byDay.get(selectedDay) ?? []).map((g) => ({
            value: String(g.ID),
            content: <>{formatGenLabel(g)} {generationStatusLabel(g.GenerationStatus)}</>,
        } as SelectOption));
    }, [classified, selectedDay]);

    const buttonLabel = selectedDay ?? currentDayKey ?? 'Select date';

    return (
        <div className={b()}>
            <Button ref={buttonRef} view="outlined" onClick={() => setCalendarOpen((o) => !o)}>
                <Icon data={CalendarIcon}/>
                {buttonLabel}
            </Button>
            <Popup
                open={calendarOpen}
                anchorElement={buttonRef.current}
                onOpenChange={(open) => setCalendarOpen(open)}
                placement="bottom-start"
            >
                <div className={b('calendar')}>
                    <Calendar
                        value={calendarValue}
                        onUpdate={handleCalendarUpdate}
                        isDateUnavailable={(d) => !classified.availableDays.has(dateToKey(d))}
                    />
                </div>
            </Popup>
            {classified.mode === 'daily' && currentGen && (
                <>
                    <div>{formatGenLabel(currentGen)}</div>
                    {generationStatusLabel(currentGen.GenerationStatus)}
                </>
            )}
            {classified.mode === 'subday' && (
                <Select
                    options={subdayOptions}
                    value={value ? [value] : []}
                    onUpdate={([val]) => onUpdate(val)}
                    placeholder="generation"
                    renderSelectedOption={(option) => <>{option.content}</>}
                    width="auto"
                />
            )}
        </div>
    );
}
