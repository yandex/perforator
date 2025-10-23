import type { FormatNode, ProfileData } from './models/Profile';


export const createCleanupFn = (cleanupKey: keyof FormatNode) => (rows: ProfileData['rows']): ProfileData['rows'] => {
    for (let h = 0; h < rows.length; h++) {
        for (let i = 0; i < rows[h].length; i++) {
            delete rows[h][i][cleanupKey];
        }
    }

    return rows;
};
