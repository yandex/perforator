
export type Field = 'eventCount' | 'sampleCount';

export type CountType = 'self' | 'all';
type Diff = '' | 'diff.';

export type NonDiffTopKeys = `${CountType}.${Field}`;
export type DiffTopKeys = `diff.${NonDiffTopKeys}`;
export type TopKeys = `${Diff}${NonDiffTopKeys}`;
export type SelfTopKeys = `${Diff}self.${Field}`;

export function isNonDiffKey(key: TopKeys): key is NonDiffTopKeys {
    return !key.startsWith('diff.');
}

export function isSelfKey(key: TopKeys): key is SelfTopKeys {
    return key.includes('self');
}
