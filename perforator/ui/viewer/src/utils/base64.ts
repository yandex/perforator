/** polyfill instead of UInt8Array.fromBase64
 * @deprecated
 */
export function base64toUint8Array(base64: string): Uint8Array<ArrayBuffer> {
    const str = atob(base64);
    const len = str.length;

    const array = new Uint8Array(len);

    for(let i = 0; i < len; i++){
        array[i] = str.charCodeAt(i);
    }

    return array;
}
