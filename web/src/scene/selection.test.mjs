// selection 测试：框选矩形命中判定 + 小地图坐标映射（纯函数，无需 WebGL）。
import { test } from 'node:test';
import assert from 'node:assert/strict';
import {
  normalizeRect, rectContains, selectInRect,
  worldToMinimap, minimapToWorld, downsampleFog,
} from './selection.ts';

test('normalizeRect：反向拖拽也归一成正矩形', () => {
  const a = normalizeRect({ x1: 100, y1: 200, x2: 20, y2: 50 });
  assert.deepEqual(a, { x: 20, y: 50, w: 80, h: 150 });

  const b = normalizeRect({ x1: 20, y1: 50, x2: 100, y2: 200 });
  assert.deepEqual(b, { x: 20, y: 50, w: 80, h: 150 }, '正拖与反拖结果一致');

  const zero = normalizeRect({ x1: 5, y1: 5, x2: 5, y2: 5 });
  assert.deepEqual(zero, { x: 5, y: 5, w: 0, h: 0 }, '零面积合法');
});

test('rectContains：边界含内，外部排除', () => {
  const r = { x: 10, y: 10, w: 100, h: 100 };
  assert.equal(rectContains(r, 10, 10), true, '左上角在内');
  assert.equal(rectContains(r, 110, 110), true, '右下角在内');
  assert.equal(rectContains(r, 60, 60), true, '中心在内');
  assert.equal(rectContains(r, 9.9, 60), false, '左侧外部');
  assert.equal(rectContains(r, 110.1, 60), false, '右侧外部');
  assert.equal(rectContains(r, 60, 9.9), false, '上方外部');
  assert.equal(rectContains(r, 60, 110.1), false, '下方外部');
});

test('selectInRect：只选己方单位，敌方/建筑不入选', () => {
  const units = [
    { id: 1, owner: 0, x: 50, y: 50 },
    { id: 2, owner: 0, x: 80, y: 80 },
    { id: 3, owner: 1, x: 60, y: 60 },  // 敌方，在框内也不选
    { id: 4, owner: 0, x: 500, y: 500 }, // 己方，但在框外
    { id: 5, owner: 0, x: 60, y: 60, kind: 'building' }, // 建筑不入选
  ];
  const rect = { x: 10, y: 10, w: 100, h: 100 };
  const ids = selectInRect(units, rect);
  assert.deepEqual(ids, [1, 2], `应只选己方单位 1,2，实际 ${JSON.stringify(ids)}`);
});

test('selectInRect：空框不选中任何单位', () => {
  const units = [{ id: 1, owner: 0, x: 50, y: 50 }];
  const rect = { x: 500, y: 500, w: 10, h: 10 };
  assert.deepEqual(selectInRect(units, rect), []);
});

test('worldToMinimap / minimapToWorld 互逆', () => {
  const world = { width: 3360, height: 2160 };
  const mini = { width: 200, height: 128 };
  const p = worldToMinimap(1680, 1080, world, mini);
  assert.ok(Math.abs(p.x - 100) < 1e-6, `世界中心应映射小地图中心 x，实际 ${p.x}`);
  assert.ok(Math.abs(p.y - 64) < 1e-6, `世界中心应映射小地图中心 y，实际 ${p.y}`);

  const back = minimapToWorld(100, 64, world, mini);
  assert.ok(Math.abs(back.x - 1680) < 1e-6, `逆映射应回到世界中心 x，实际 ${back.x}`);
  assert.ok(Math.abs(back.y - 1080) < 1e-6, `逆映射应回到世界中心 y，实际 ${back.y}`);
});

test('worldToMinimap：四角映射到小地图四角', () => {
  const world = { width: 1000, height: 500 };
  const mini = { width: 100, height: 50 };
  assert.deepEqual(worldToMinimap(0, 0, world, mini), { x: 0, y: 0 });
  assert.deepEqual(worldToMinimap(1000, 500, world, mini), { x: 100, y: 50 });
});

test('downsampleFog：把大格迷雾降到小地图分辨率（任一可见即亮）', () => {
  // 4x2 的迷雾，降到 2x1。
  // 左块(cols0-1)：有可见 → visible；右块(cols2-3)：无可见但有已探索 → explored。
  const visible = [1, 0, 0, 0,
                   0, 0, 0, 0];
  const explored = [1, 1, 1, 0,
                    1, 0, 0, 0];
  const out = downsampleFog({ cols: 4, rows: 2, visible, explored }, { width: 2, height: 1 });
  assert.equal(out.length, 2, '输出 2x1 = 2 格');
  assert.equal(out[0], 'visible', '左块有可见格 → visible');
  assert.equal(out[1], 'explored', '右块无可见但有已探索 → explored');
});

test('downsampleFog：全未知区域返回 dark', () => {
  const visible = new Array(4).fill(0);
  const explored = new Array(4).fill(0);
  const out = downsampleFog({ cols: 2, rows: 2, visible, explored }, { width: 1, height: 1 });
  assert.equal(out[0], 'dark', '全未知 → dark');
});
