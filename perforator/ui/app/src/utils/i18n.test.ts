import { readdirSync, readFileSync } from 'node:fs';
import { basename, dirname, join, relative, resolve, sep } from 'node:path';

import { expect, it, jest } from '@jest/globals';

import type { I18N } from '@gravity-ui/i18n';


function getCatalogues(directory: string): string[] {
    return readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
        const path = join(directory, entry.name);
        if (entry.isDirectory()) {
            return getCatalogues(path);
        }
        return entry.name === 'en.json' && basename(dirname(path)) === 'i18n' ? [path] : [];
    });
}

it('registers every app catalogue once under its owning feature namespace', () => {
    const sourceRoot = resolve(__dirname, '..');

    jest.isolateModules(() => {
        const { i18n } = require(join(sourceRoot, 'utils/i18n')) as {i18n: I18N};
        const registerKeyset = jest.spyOn(i18n, 'registerKeyset');
        const keyset = jest.spyOn(i18n, 'keyset');
        const catalogues = getCatalogues(sourceRoot).sort();
        const namespaces: string[] = [];
        expect(catalogues.length).toBeGreaterThan(0);

        for (const [index, catalogue] of catalogues.entries()) {
            const namespace = relative(sourceRoot, dirname(dirname(catalogue))).split(sep).join('/');
            const en = JSON.parse(readFileSync(catalogue, 'utf8')) as Record<string, string>;
            // Derive the expected namespace from the catalogue's path, independently of its adapter.
            namespaces.push(namespace);
            require(join(dirname(catalogue), 'index'));
            expect(registerKeyset.mock.calls[index]).toEqual(['en', namespace, en]);
            expect(keyset.mock.calls[index]).toEqual([namespace]);
            for (const key of Object.keys(en)) {
                expect({ namespace, key, registered: i18n.has(namespace, key, 'en') }).toEqual({ namespace, key, registered: true });
            }
        }

        const registeredNamespaces = registerKeyset.mock.calls.map(([, namespace]) => namespace);
        expect(registeredNamespaces).toEqual(namespaces);
        expect(keyset.mock.calls.map(([namespace]) => namespace)).toEqual(namespaces);
        expect(new Set(registeredNamespaces).size).toBe(catalogues.length);
    });
});
