// scene/selection.ts：框选命中判定与小地图坐标换算（纯函数，无三方依赖）。
// 抽成纯函数便于在 node 下直接测试，SceneView 只负责把指针事件转成世界坐标。

export interface DragRect { x1: number; y1: number; x2: number; y2: number }
export interface Rect { x: number; y: number; w: number; h: number }
export interface Size { width: number; height: number }
export interface FogLike { cols: number; rows: number; visible: Uint8Array | number[]; explored: Uint8Array | number[] }

/** 把任意方向的拖拽归一成左上角 + 宽高。 */
export function normalizeRect(r: DragRect): Rect {
  const x = Math.min(r.x1, r.x2);
  const y = Math.min(r.y1, r.y2);
  return { x, y, w: Math.abs(r.x2 - r.x1), h: Math.abs(r.y2 - r.y1) };
}

/** 点是否落在矩形内（含边界）。 */
export function rectContains(r: Rect, x: number, y: number): boolean {
  return x >= r.x && x <= r.x + r.w && y >= r.y && y <= r.y + r.h;
}

interface SelectableEntity {
  id: number;
  owner?: number;
  x: number;
  y: number;
  kind?: string;
}

/**
 * 框选：只选己方（owner===0，快照已按席位重排）的非建筑实体。
 * 敌方单位即使落在框内也不选——避免误发指令。
 */
export function selectInRect(entities: SelectableEntity[], r: Rect): number[] {
  const out: number[] = [];
  for (const e of entities) {
    if (e.owner !== 0) continue;
    if (e.kind === 'building') continue;
    if (rectContains(r, e.x, e.y)) out.push(e.id);
  }
  return out;
}

/** 世界坐标 → 小地图像素坐标。 */
export function worldToMinimap(x: number, y: number, world: Size, mini: Size): { x: number; y: number } {
  return {
    x: (x / world.width) * mini.width,
    y: (y / world.height) * mini.height,
  };
}

/** 小地图像素坐标 → 世界坐标（点击小地图跳转视角用）。 */
export function minimapToWorld(mx: number, my: number, world: Size, mini: Size): { x: number; y: number } {
  return {
    x: (mx / mini.width) * world.width,
    y: (my / mini.height) * world.height,
  };
}

export type FogCell = 'visible' | 'explored' | 'dark';

/**
 * 迷雾降采样：把 fog 网格压到小地图分辨率。
 * 每格取对应区域的最大可见度（任一可见即 visible，否则任一已探索即 explored，否则 dark）。
 */
export function downsampleFog(fog: FogLike, mini: Size): FogCell[] {
  const out: FogCell[] = new Array(mini.width * mini.height);
  for (let my = 0; my < mini.height; my++) {
    const f0 = Math.floor((my / mini.height) * fog.rows);
    const f1 = Math.max(f0 + 1, Math.floor(((my + 1) / mini.height) * fog.rows));
    for (let mx = 0; mx < mini.width; mx++) {
      const c0 = Math.floor((mx / mini.width) * fog.cols);
      const c1 = Math.max(c0 + 1, Math.floor(((mx + 1) / mini.width) * fog.cols));
      let visible = false;
      let explored = false;
      for (let fy = f0; fy < Math.min(f1, fog.rows) && !visible; fy++) {
        for (let fx = c0; fx < Math.min(c1, fog.cols); fx++) {
          const i = fy * fog.cols + fx;
          if (fog.visible[i]) { visible = true; break; }
          if (fog.explored[i]) explored = true;
        }
      }
      out[my * mini.width + mx] = visible ? 'visible' : explored ? 'explored' : 'dark';
    }
  }
  return out;
}
