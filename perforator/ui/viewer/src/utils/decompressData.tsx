export async function decompressData(compressed: Uint8Array<ArrayBuffer>): Promise<string> {
    const stream = new Blob([compressed]).stream();
    const decompressedStream = stream.pipeThrough(new DecompressionStream('gzip'));
    const reader = decompressedStream.getReader();
    const chunks: Uint8Array[] = [];

    // eslint-disable-next-line no-constant-condition
    while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        chunks.push(value);
    }

    const totalLength = chunks.reduce((acc, chunk) => acc + chunk.length, 0);
    const result = new Uint8Array(totalLength);
    let offset = 0;
    for (const chunk of chunks) {
        result.set(chunk, offset);
        offset += chunk.length;
    }

    return new TextDecoder().decode(result);
}
