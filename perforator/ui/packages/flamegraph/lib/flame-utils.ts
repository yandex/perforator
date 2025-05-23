const HUGENUM_SUFFIXES = ['', 'K', 'M', 'G', 'T', 'P', 'E'];

export function hugenum(num: number) {
    let index = 0;
    while (num > 1e3 && index < HUGENUM_SUFFIXES.length) {
        num /= 1e3;
        index += 1;
    }
    if (index == 0) {
        return String(num);
    }
    return num.toFixed(3) + HUGENUM_SUFFIXES[index];
}
