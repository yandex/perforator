import { describe, expect, it } from '@jest/globals';

import { changeQueryToNewInput } from './changeQueryToNewInput';
import type { QueryInput } from './QueryInput';


const makeQueryInput = (queryField: string): QueryInput => ({
    name: queryField.charAt(0).toUpperCase() + queryField.slice(1),
    queryField,
    render: () => null,
});

const selectorInput = makeQueryInput('selector');
const tokensInput = makeQueryInput('tokens');
const serviceInput = makeQueryInput('service');

const serviceOnlySelector = '{service="my-service"}';
const complexSelector = '{service="my-service",timestamp>="2024-01-01T00:00:00Z"}';


describe('changeQueryToNewInput', () => {
    describe('switching to selector mode', () => {
        it('should return the tableSelector as-is', () => {
            expect(changeQueryToNewInput(selectorInput, serviceOnlySelector)).toEqual({
                selector: serviceOnlySelector,
            });
        });

        it('should return the tableSelector even when it is empty', () => {
            expect(changeQueryToNewInput(selectorInput, '')).toEqual({
                selector: '',
            });
        });
    });

    describe('switching to tokens mode', () => {
        it('should convert a service-only selector to tokens', () => {
            const result = changeQueryToNewInput(tokensInput, serviceOnlySelector);
            expect(result).toEqual({
                tokens: JSON.stringify([['service', '=', 'my-service']]),
            });
        });

        it('should convert a service-only selector with trailing comma to tokens', () => {
            const result = changeQueryToNewInput(tokensInput, '{service="my-service", }');
            expect(result).toEqual({
                tokens: JSON.stringify([['service', '=', 'my-service']]),
            });
        });

        it('should return undefined for a complex selector', () => {
            expect(changeQueryToNewInput(tokensInput, complexSelector)).toBeUndefined();
        });

        it('should return undefined for an empty tableSelector', () => {
            expect(changeQueryToNewInput(tokensInput, '')).toBeUndefined();
        });
    });

    describe('switching to service mode', () => {
        it('should extract service name from a service-only selector', () => {
            const result = changeQueryToNewInput(serviceInput, serviceOnlySelector);
            expect(result).toEqual({
                service: 'my-service',
            });
        });

        it('should extract service from a selector with trailing comma', () => {
            const result = changeQueryToNewInput(serviceInput, '{service="my-service" , }');
            expect(result).toEqual({
                service: 'my-service',
            });
        });

        it('should return undefined for a complex selector', () => {
            expect(changeQueryToNewInput(serviceInput, complexSelector)).toBeUndefined();
        });

        it('should return undefined for an empty tableSelector', () => {
            expect(changeQueryToNewInput(serviceInput, '')).toBeUndefined();
        });
    });

    describe('unknown query field', () => {
        it('should return undefined for an unrecognized queryField', () => {
            const unknownInput = makeQueryInput('unknown');
            expect(changeQueryToNewInput(unknownInput, serviceOnlySelector)).toBeUndefined();
        });
    });
});
