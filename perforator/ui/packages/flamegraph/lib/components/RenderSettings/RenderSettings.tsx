import React from 'react';

import { Gear } from '@gravity-ui/icons';
import { Button, Checkbox, Disclosure, Icon, Popup, TextInput } from '@gravity-ui/uikit';

import type { PprofRenderOptions } from '../../pprof/options';
import { defaultPprofOptions, validatePprofOptions } from '../../pprof/options';

import './RenderSettings.css';


interface Props {
    options: PprofRenderOptions;
    busy: boolean;
    onApply: (options: PprofRenderOptions) => void;
}

interface FieldErrors {
    depth?: string;
    weight?: string;
}

export function RenderSettings({ options, busy, onApply }: Props) {
    const [anchor, setAnchor] = React.useState<HTMLButtonElement | null>(null);
    const [open, setOpen] = React.useState(false);
    const [advanced, setAdvanced] = React.useState(false);
    const [depth, setDepth] = React.useState(String(options.maxDepth));
    const [weight, setWeight] = React.useState(String(options.minWeight * 100));
    const [files, setFiles] = React.useState(options.showFileNames);
    const [errors, setErrors] = React.useState<FieldErrors>({});
    const [focusInvalidField, setFocusInvalidField] = React.useState<keyof FieldErrors>();
    const depthRef = React.useRef<HTMLInputElement>(null);
    const weightRef = React.useRef<HTMLInputElement>(null);
    const id = React.useId();
    const depthId = `${id}-depth`;
    const weightId = `${id}-weight`;

    React.useEffect(() => {
        if (focusInvalidField) {
            (focusInvalidField === 'depth' ? depthRef : weightRef).current?.focus();
            setFocusInvalidField(undefined);
        }
    }, [focusInvalidField]);

    const changeOpen = (value: boolean) => {
        if (value) {
            setDepth(String(options.maxDepth));
            setWeight(String(options.minWeight * 100));
            setFiles(options.showFileNames);
            setErrors({});
            setAdvanced(false);
        }
        setOpen(value);
    };

    const reset = () => {
        setDepth(String(defaultPprofOptions.maxDepth));
        setWeight(String(defaultPprofOptions.minWeight * 100));
        setFiles(defaultPprofOptions.showFileNames);
        setErrors({});
    };

    const apply = (event: React.FormEvent) => {
        event.preventDefault();
        const next = {
            maxDepth: Number(depth),
            // Preserve the exact fraction on unchanged Apply, including tiny defaults.
            minWeight: Number(weight) === options.minWeight * 100 ? options.minWeight : Number(weight) / 100,
            showFileNames: files,
            showLineNumbers: options.showLineNumbers,
        };
        const nextErrors: FieldErrors = {};

        if (!depth.trim()) {
            nextErrors.depth = 'Enter a maximum depth.';
        } else if (!Number.isFinite(next.maxDepth)) {
            nextErrors.depth = 'Enter a whole number for maximum depth.';
        } else if (!Number.isInteger(next.maxDepth) || next.maxDepth < 0 || next.maxDepth > 0xffffffff) {
            nextErrors.depth = 'Maximum depth must be an integer between 0 and 4294967295.';
        }

        if (!weight.trim()) {
            nextErrors.weight = 'Enter a minimum weight.';
        } else if (!Number.isFinite(Number(weight))) {
            nextErrors.weight = 'Enter a number for minimum weight.';
        } else if (Number(weight) < 0 || Number(weight) > 100) {
            nextErrors.weight = 'Minimum weight must be a percentage between 0 and 100.';
        }

        setErrors(nextErrors);
        if (Object.keys(nextErrors).length > 0) {
            if (nextErrors.depth) {
                setAdvanced(true);
            }
            setFocusInvalidField(nextErrors.depth ? 'depth' : 'weight');
            return;
        }

        setFocusInvalidField(undefined);
        validatePprofOptions(next);
        onApply(next);
        setOpen(false);
    };

    return <>
        <Button ref={setAnchor} onClick={() => changeOpen(!open)}>
            <Icon data={Gear} />
            Render settings
        </Button>
        <Popup
            open={open}
            anchorElement={anchor}
            onOpenChange={changeOpen}
            placement="bottom-start"
            role="dialog"
            initialFocus={weightRef}
            returnFocus
            modal
        >
            <form className="pprof-flamegraph__settings" onSubmit={apply} noValidate>
                <h2 className="pprof-flamegraph__settings-title">Local render settings</h2>
                <label htmlFor={weightId}>Minimum weight (% of the full profile, 0–100)</label>
                <TextInput
                    id={weightId}
                    controlRef={weightRef}
                    value={weight}
                    onUpdate={value => {
                        setWeight(value);
                        setErrors(current => ({ ...current, weight: undefined }));
                    }}
                    type="text"
                    controlProps={{ inputMode: 'decimal' }}
                    error={Boolean(errors.weight)}
                />
                {errors.weight && <div role="alert" className="pprof-flamegraph__validation">{errors.weight}</div>}

                <Checkbox checked={files} onUpdate={setFiles}>File names</Checkbox>
                <Disclosure summary="Advanced" expanded={advanced} onUpdate={setAdvanced}>
                    <div className="pprof-flamegraph__advanced">
                        <label htmlFor={depthId}>Maximum depth (0 means unlimited)</label>
                        <TextInput
                            id={depthId}
                            controlRef={depthRef}
                            value={depth}
                            onUpdate={value => {
                                setDepth(value);
                                setErrors(current => ({ ...current, depth: undefined }));
                            }}
                            type="text"
                            controlProps={{ inputMode: 'numeric' }}
                            error={Boolean(errors.depth)}
                        />
                        {errors.depth && <div role="alert" className="pprof-flamegraph__validation">{errors.depth}</div>}
                        <div>Includes service label levels (container, process and thread). 0 means unlimited.</div>
                    </div>
                </Disclosure>
                <div className="pprof-flamegraph__settings-actions">
                    <Button type="button" view="flat" onClick={reset}>Reset</Button>
                    <Button type="button" view="flat" onClick={() => changeOpen(false)}>Cancel</Button>
                    <Button type="submit" view="action" disabled={busy}>Apply</Button>
                </div>
            </form>
        </Popup>
    </>;
}
