// scene/placement.ts：建筑落位预检（纯函数，无三方依赖）。
//
// 逐条对齐 Go 侧 internal/game/commands.go:116 CanPlace——源项目的放置合法性
// 判断本来在客户端本地跑（render.js:1177 每帧调 game.canPlace 决定 ghost 红/绿），
// 我们拆成两进程后由服务端把规则输入随快照下发，前端复刻同一判据。
//
// 已知边界：快照只含可见建筑（迷雾裁剪），客户端看不全敌方建筑，因此预检在网络
// 视角下可能比服务端宽松。此函数仅作 ghost 着色提示，真正的裁决仍在服务端
// place 指令（PlaceBuilding → CanPlace），不接受本函数的结果作为权威。

export interface Rect { x1: number; y1: number; x2: number; y2: number }
export interface Pt { x: number; y: number }

/** 服务端下发的放置规则（battleView.placeRules）。 */
export interface PlaceRules {
  worldWidth: number;
  worldHeight: number;
  noBuild: Rect[];
  dockZones: Rect[];
  sizes: Record<string, number>;
}

/** 快照里参与间距判定的实体（前端只读这些字段）。 */
export interface PlaceScene {
  ore?: { x: number; y: number; amount?: number }[];
  oil?: { x: number; y: number }[];
  beacons?: { x: number; y: number }[];
  resourceSites?: { x: number; y: number; type?: string; owner?: unknown }[];
  buildings?: { x: number; y: number; size?: number; hp?: number }[];
  /** 己方建筑坐标（310 半径建造范围）。 */
  ownBuildings?: { x: number; y: number }[];
}

const dist = (ax: number, ay: number, bx: number, by: number) => Math.hypot(ax - bx, ay - by);

/** 圆与轴对齐矩形相交（对齐 Go 侧 world.go:436 intersectsRect）。 */
export function intersectsRect(x: number, y: number, radius: number, r: Rect): boolean {
  const cx = Math.min(Math.max(x, r.x1), r.x2);
  const cy = Math.min(Math.max(y, r.y1), r.y2);
  return dist(x, y, cx, cy) <= radius;
}

/**
 * 该落点能否放下 type 建筑（对齐 CanPlace 的完整判据链）。
 * 顺序与原实现一致——dock 走船坞区分支且不做 NoBuild 检查，末段还有前置/扩建豁免。
 */
export function canPlace(
  type: string, x: number, y: number,
  rules: PlaceRules, scene: PlaceScene, hasFactory: boolean,
): boolean {
  const size = rules.sizes[type];
  if (!size) return false;
  const W = rules.worldWidth, H = rules.worldHeight;
  if (x < size || y < size || x > W - size || y > H - size) return false;

  if (type === 'dock') {
    let hit = false;
    for (const r of rules.dockZones ?? []) {
      if (x >= r.x1 && x <= r.x2 && y >= r.y1 && y <= r.y2) {
        // 视角方恒为 side 0：己方只认左半图的船坞区。
        if (x < W / 2) hit = true;
      }
    }
    if (!hit) return false;
  } else {
    for (const r of rules.noBuild ?? []) {
      if (intersectsRect(x, y, size * 0.6, r)) return false;
    }
  }

  for (const o of scene.ore ?? []) {
    if ((o.amount ?? 0) > 0 && dist(o.x, o.y, x, y) < 95) return false;
  }
  for (const o of scene.oil ?? []) {
    if (dist(o.x, o.y, x, y) < 95) return false;
  }
  for (const s of scene.beacons ?? []) {
    if (dist(s.x, s.y, x, y) < 100) return false;
  }
  for (const s of scene.resourceSites ?? []) {
    if (dist(s.x, s.y, x, y) < 80) return false;
  }
  for (const b of scene.buildings ?? []) {
    if ((b.hp ?? 0) > 0 && dist(b.x, b.y, x, y) < ((b.size ?? 0) + size) * 0.63 + 20) return false;
  }

  if (type === 'dock') return hasFactory;

  for (const b of scene.ownBuildings ?? []) {
    if (dist(b.x, b.y, x, y) < 310) return true;
  }
  // 己方已控能源仓的 260 半径扩建豁免。
  for (const s of scene.resourceSites ?? []) {
    if (s.type === 'depot' && s.owner === 0 && dist(s.x, s.y, x, y) < 260) return true;
  }
  return false;
}

/** ghost 配色：合法绿、非法红（对齐源 render.js:1177 的 '#87d7b1' / '#ee7366'）。 */
export const PLACE_OK_COLOR = 0x87d7b1;
export const PLACE_BAD_COLOR = 0xee7366;

export function placeColor(valid: boolean): number {
  return valid ? PLACE_OK_COLOR : PLACE_BAD_COLOR;
}
