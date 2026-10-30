import { describe, expect, it } from '@jest/globals';

import { defaultPprofOptions, readPprofOptions, writePprofOptions } from './options';


describe('pprof render options', () => {
    it('should use the current UI defaults', () => {
        expect(readPprofOptions(new URLSearchParams())).toEqual(defaultPprofOptions);
        expect(String(writePprofOptions(new URLSearchParams('tab=top&maxDepth=5&lineNumbers=true'), defaultPprofOptions)))
            .toBe('tab=top');
    });

    it.each([
        { maxDepth: 0, minWeight: 0, showFileNames: false, showLineNumbers: true },
        { maxDepth: 0xffffffff, minWeight: 1, showFileNames: true, showLineNumbers: false },
        { maxDepth: 256, minWeight: 1e-10, showFileNames: true, showLineNumbers: true },
    ])('should restore serialized settings %j', options => {
        const query = writePprofOptions(new URLSearchParams('flamegraphQuery=work'), options);
        expect(readPprofOptions(query)).toEqual(options);
        expect(query.get('flamegraphQuery')).toBe('work');
    });

    it.each(['maxDepth=-1', 'maxDepth=1.5', 'maxDepth=4294967296', 'maxDepth=',
        'minWeight=NaN', 'minWeight=Infinity', 'minWeight=-0.1', 'minWeight=1.1',
        'lineNumbers=yes', 'showFileNames=1'])('should reject invalid settings %s', value => {
        expect(() => readPprofOptions(new URLSearchParams(value))).toThrow();
    });
});
