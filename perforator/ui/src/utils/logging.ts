type Fn = (...args: any[]) => any;

export const withMeasureTime: <T extends Fn>(fn: T) => (...args: Parameters<T>) => ReturnType<T> = (fn) => (...args) => {
    const start = performance.now();
    const res = fn(...args);
    // eslint-disable-next-line no-console
    console.log(`${fn.name} took ${performance.now() - start}ms`);
    return res;
};
