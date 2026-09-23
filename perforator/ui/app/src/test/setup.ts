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

// Gravity UI's modal and menu hooks use matchMedia, which jsdom does not implement.
window.matchMedia ??= (media: string): MediaQueryList => ({
    media,
    matches: false,
    onchange: null,
    addListener() {},
    removeListener() {},
    addEventListener() {},
    removeEventListener() {},
    dispatchEvent: () => false,
});

afterEach(() => {
    localStorage.clear();
});
