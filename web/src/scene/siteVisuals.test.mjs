// siteVisuals 测试：站点几何/配色/归属着色（纯函数，无需 WebGL）。
import { test } from 'node:test';
import assert from 'node:assert/strict';
import {
  SITE_COLORS, siteColor, ownerTint, siteLift, siteGeometry, createSiteMesh, applyOwnerTint,
} from './siteVisuals.ts';

// three 的色彩管理会做 sRGB↔Linear 转换，setHex/getHex 不保证字面量往返。
// 对比语义用 getHexString()（同一色彩管理管线下的稳定表示）。
const hexStr = (c) => c.getHexString();

test('矿种分色：gold 金黄 / gem 青玉，其余固定', () => {
  assert.equal(siteColor('ore', { kind: 'gold' }), SITE_COLORS.gold);
  assert.equal(siteColor('ore', { kind: 'gem' }), SITE_COLORS.gem);
  assert.equal(siteColor('ore'), SITE_COLORS.gold, '缺 kind 默认金');
  assert.equal(siteColor('oil'), SITE_COLORS.oil);
  assert.equal(siteColor('beacon'), SITE_COLORS.beacon);
  assert.equal(siteColor('resource'), SITE_COLORS.resource);
  // 金与青玉必须不同色。
  assert.notEqual(siteColor('ore', { kind: 'gem' }), siteColor('ore', { kind: 'gold' }));
});

test('归属色：null/undefined 中立，0 己方，1 敌方', () => {
  assert.equal(ownerTint(null), null);
  assert.equal(ownerTint(undefined), null);
  assert.equal(ownerTint(0), SITE_COLORS.mine);
  assert.equal(ownerTint(1), SITE_COLORS.enemy);
  assert.equal(ownerTint(2), SITE_COLORS.enemy, '非 0 数字一律视作敌方');
});

test('离地高度：信标抬高，其余 2.5', () => {
  assert.equal(siteLift('beacon'), 3);
  assert.equal(siteLift('ore'), 2.5);
  assert.equal(siteLift('oil'), 2.5);
  assert.equal(siteLift('resource'), 2.5);
});

test('四类几何互异（含油与信标同族不同参）', () => {
  const kinds = ['ore', 'oil', 'beacon', 'resource'];
  const sigs = kinds.map(k => {
    const g = siteGeometry(k);
    const s = `${g.type}:${g.parameters?.radiusTop ?? 0}:${g.parameters?.radius ?? 0}`;
    g.dispose();
    return s;
  });
  assert.equal(new Set(sigs).size, 4, `四类几何签名应互异，实际 ${sigs.join(' | ')}`);
});

test('createSiteMesh：中立自发光=本体色，己方/敌方染色', () => {
  const neutral = createSiteMesh('ore', { kind: 'gold', owner: null });
  const nm = neutral.material;
  assert.equal(hexStr(nm.emissive), hexStr(nm.color), '中立自发光=本体色');
  assert.equal(nm.emissiveIntensity, 0.18);
  assert.equal(neutral.userData.siteKind, 'ore');
  assert.equal(neutral.userData.baseColor, SITE_COLORS.gold, 'baseColor 记录本源色');

  const mine = createSiteMesh('oil', { owner: 0 });
  assert.equal(mine.material.emissiveIntensity, 0.55, '己方着色强度');
  assert.notEqual(hexStr(mine.material.emissive), hexStr(mine.material.color), '己方自发光不同于本体色');

  const enemy = createSiteMesh('beacon', { owner: 1 });
  assert.equal(enemy.material.emissiveIntensity, 0.55, '敌方着色强度');
});

test('applyOwnerTint 就地更新：gem 矿染色后回中立须还原青玉（不可退化成金）', () => {
  const mesh = createSiteMesh('ore', { kind: 'gem', owner: null });
  const gemHex = hexStr(mesh.material.color);
  assert.equal(hexStr(mesh.material.emissive), gemHex, '初始中立=青玉');

  applyOwnerTint(mesh, 1);
  assert.equal(mesh.material.emissiveIntensity, 0.55, '转敌方');

  applyOwnerTint(mesh, null);
  assert.equal(hexStr(mesh.material.emissive), gemHex, '回中立应还原青玉本源色，不能变金');
  assert.equal(mesh.material.emissiveIntensity, 0.18);
});
