type Fn = (...args: any[]) => any;

export const withMeasureTime: <T extends Fn>(fn: T, displayName?: string, sendFinish?: (ms: number) => void) => (...args: Parameters<T>) => ReturnType<T> = (fn, displayName, sendFinish) => (...args) => {
    const start = performance.now();
    const res = fn(...args);
    const finish = performance.now();
    // eslint-disable-next-line no-console
    console.log(`${displayName || fn.name} took ${finish - start}ms`);
    sendFinish?.(finish - start);
    return res;
};
