
export function pct(a: number, b: number) {
    return a >= b ? '100' : (100 * a / b).toFixed(2);
}

export function formatPct(value: any) {
    return value ? `${value}%` : '';
}
