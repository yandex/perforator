import { afterEach } from '@jest/globals';

import '@testing-library/jest-dom/jest-globals';


class IntersectionObserverMock implements IntersectionObserver {
    readonly root = null;
    readonly rootMargin = '';
    readonly scrollMargin = '';
    readonly thresholds = [];

    disconnect() {}
    observe() {}
    takeRecords(): IntersectionObserverEntry[] { return []; }
    unobserve() {}
}

globalThis.IntersectionObserver = IntersectionObserverMock;

afterEach(() => {
    localStorage.clear();
});
