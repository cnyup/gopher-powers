// fog.ts：迷雾解包与战场迷雾网格（对齐 unpackFog）。
export function unpackFog(fog: { cols: number; rows: number; bits: string }) {
  const bin = atob(fog.bits);
  const n = fog.cols * fog.rows;
  const visible = new Uint8Array(n);
  const explored = new Uint8Array(n);
  for (let i = 0; i < n; i++) {
    const b = bin.charCodeAt(i >> 2);
    visible[i] = b & (2 << ((i % 4) * 2)) ? 1 : 0;
    explored[i] = b & (1 << ((i % 4) * 2)) ? 1 : 0;
  }
  return { cols: fog.cols, rows: fog.rows, visible, explored };
}
