// placement.test.mjs：落位预检纯函数——逐条锁定 Go 侧 CanPlace 的判据分支。
import test from 'node:test';
import assert from 'node:assert/strict';
import { canPlace, intersectsRect, placeColor, PLACE_OK_COLOR, PLACE_BAD_COLOR } from './placement.ts';

const rules = {
  worldWidth: 3520, worldHeight: 2240,
  noBuild: [{ x1: 1000, y1: 0, x2: 2520, y2: 2240 }],
  dockZones: [{ x1: 900, y1: 970, x2: 1000, y2: 1270 }],
  sizes: { hq: 92, power: 66, barracks: 66, factory: 88, dock: 92 },
};

// 己方基地在左半图 (600,1200) 附近。
const scene = () => ({
  ore: [], oil: [], beacons: [], resourceSites: [], buildings: [],
  ownBuildings: [{ x: 600, y: 1200 }],
});

test('intersectsRect：圆心在矩形内必相交', () => {
  assert.equal(intersectsRect(1500, 500, 10, rules.noBuild[0]), true);
});

test('intersectsRect：半径贴边算相交，差一点不算', () => {
  const r = { x1: 100, y1: 100, x2: 200, y2: 200 };
  // (96,150) 在矩形左侧外 4 单位：半径 4 恰好贴边（<=），半径 3 差一点
  assert.equal(intersectsRect(96, 150, 4, r), true);
  assert.equal(intersectsRect(96, 150, 3, r), false);
});

test('canPlace：未知建筑类型直接拒绝（尺寸表缺失）', () => {
  assert.equal(canPlace('nope', 600, 1250, rules, scene(), true), false);
});

test('canPlace：贴世界边界不足 size 时拒绝', () => {
  // power size=66，x=60 < 66
  assert.equal(canPlace('power', 60, 1250, rules, scene(), true), false);
  assert.equal(canPlace('power', 3520 - 60, 1250, rules, scene(), true), false);
});

test('canPlace：落在禁建区（NoBuild 半径 size*0.6）拒绝', () => {
  // barracks size=66 → 半径 39.6；落在矩形内部必相交
  assert.equal(canPlace('barracks', 1500, 600, rules, scene(), true), false);
});

test('canPlace：NoBuild 边界外的合法点可通过建造范围', () => {
  // 左半图、禁建区外、距己方基地 <310
  assert.equal(canPlace('barracks', 850, 1200, rules, scene(), true), true);
});

test('canPlace：距己方建筑 >310 时无建造范围，拒绝', () => {
  assert.equal(canPlace('barracks', 300, 300, rules, scene(), true), false);
});

test('canPlace：NoBuild 对 dock 不生效（走船坞区分支）', () => {
  // (950,1100) 在船坞区内（900-1000 × 970-1270），且在 NoBuild 覆盖范围内
  assert.equal(canPlace('dock', 950, 1100, rules, scene(), true), true);
});

test('canPlace：dock 必须有 factory 前置', () => {
  assert.equal(canPlace('dock', 950, 1100, rules, scene(), false), false);
});

test('canPlace：dock 在船坞区外拒绝', () => {
  assert.equal(canPlace('dock', 400, 1100, rules, scene(), true), false);
});

test('canPlace：资源点间距——矿(有量 95)/油(95)/信标(100)/资源站(80)', () => {
  // 锚点 (700,1800)，己方建筑同点 → 310 建造范围内
  const px = 700, py = 1800;
  const okAll = { ...scene(), noBuild: [], ownBuildings: [{ x: px, y: py }] };
  const noBuildNone = { ...rules, noBuild: [] };
  assert.equal(canPlace('barracks', px, py, noBuildNone, okAll, true), true);
  // 略小于阈值 → 拒绝
  assert.equal(canPlace('barracks', px, py, noBuildNone, { ...okAll, ore: [{ x: px + 90, y: py, amount: 100 }] }, true), false);
  assert.equal(canPlace('barracks', px, py, noBuildNone, { ...okAll, oil: [{ x: px + 90, y: py }] }, true), false);
  assert.equal(canPlace('barracks', px, py, noBuildNone, { ...okAll, beacons: [{ x: px + 95, y: py }] }, true), false);
  assert.equal(canPlace('barracks', px, py, noBuildNone, { ...okAll, resourceSites: [{ x: px + 75, y: py }] }, true), false);
  // 略大于阈值 → 允许（确认不是一刀切拒绝）
  assert.equal(canPlace('barracks', px, py, noBuildNone, { ...okAll, ore: [{ x: px + 100, y: py, amount: 100 }] }, true), true);
  assert.equal(canPlace('barracks', px, py, noBuildNone, { ...okAll, beacons: [{ x: px + 105, y: py }] }, true), true);
});

test('canPlace：矿 remaining=0 不参与间距判定', () => {
  const px = 700, py = 1800;
  const base = { ...scene(), noBuild: [], ownBuildings: [{ x: px, y: py }] };
  const withEmptyOre = { ...base, ore: [{ x: px + 90, y: py, amount: 0 }] };
  assert.equal(canPlace('barracks', px, py, { ...rules, noBuild: [] }, withEmptyOre, true), true);
});

test('canPlace：现有建筑间距 (b.size+size)*0.63+20', () => {
  const px = 700, py = 1800;
  const base = { ...scene(), noBuild: [], ownBuildings: [{ x: px, y: py }] };
  // 敌建筑 size=88，barracks size=66 → 阈值 (88+66)*0.63+20 = 117.02
  assert.equal(canPlace('barracks', px, py, { ...rules, noBuild: [] },
    { ...base, buildings: [{ x: px + 100, y: py, size: 88, hp: 100 }] }, true), false);
  // 120 > 117.02 → 可以
  assert.equal(canPlace('barracks', px, py, { ...rules, noBuild: [] },
    { ...base, buildings: [{ x: px + 120, y: py, size: 88, hp: 100 }] }, true), true);
});

test('canPlace：hp=0 的建筑不参与间距判定', () => {
  const px = 700, py = 1800;
  const base = { ...scene(), noBuild: [], ownBuildings: [{ x: px, y: py }] };
  const dead = { ...base, buildings: [{ x: px + 10, y: py, size: 88, hp: 0 }] };
  assert.equal(canPlace('barracks', px, py, { ...rules, noBuild: [] }, dead, true), true);
});

test('canPlace：己控能源仓 260 半径可在外围扩建', () => {
  const base = { noBuild: [], ore: [], oil: [], beacons: [], buildings: [], ownBuildings: [] };
  const depot = { ...base, resourceSites: [{ x: 700, y: 1800, type: 'depot', owner: 0 }] };
  // 距 depot 250 < 260 → 允许（虽然无己方建筑 310 覆盖）
  assert.equal(canPlace('barracks', 950, 1800, { ...rules, noBuild: [] }, depot, true), true);
  // 中立能源仓不提供豁免
  const neutral = { ...base, resourceSites: [{ x: 700, y: 1800, type: 'depot', owner: null }] };
  assert.equal(canPlace('barracks', 950, 1800, { ...rules, noBuild: [] }, neutral, true), false);
});

test('placeColor：合法绿、非法红，与源 render.js 配色一致', () => {
  assert.equal(placeColor(true), PLACE_OK_COLOR);
  assert.equal(placeColor(false), PLACE_BAD_COLOR);
  assert.equal(PLACE_OK_COLOR, 0x87d7b1);
  assert.equal(PLACE_BAD_COLOR, 0xee7366);
});
