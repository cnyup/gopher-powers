// FxLayer 回归测试：池化/插值/生命周期/光束几何（node --test，逻辑探针固化）。
// 运行：cd web && node --test src/scene/FxLayer.test.mjs
import { test } from 'node:test';
import assert from 'node:assert/strict';
import * as THREE from 'three';
import { FxLayer } from './FxLayer.ts';

// 最小 DOM mock（income 浮字路径才用 canvas；本套不触达）。
globalThis.document = { createElement: () => ({ width: 0, height: 0, getContext: () => null }) };

function harness() {
  const scene = new THREE.Scene();
  const fx = new FxLayer(scene);
  const root = () => scene.children[0];
  let now = 1000;
  const realNow = performance.now.bind(performance);
  performance.now = () => now;
  return {
    fx, root,
    setNow: (ms) => { now = ms; },
    restore: () => { performance.now = realNow; },
  };
}

test('投射物按 id 差量入池/出池', () => {
  const h = harness();
  try {
    h.fx.sync([
      { id: 1, kind: 'shell', x: 100, y: 200, angle: 0, speed: 300, owner: 0, finished: false },
      { id: 2, kind: 'missile', x: 500, y: 500, angle: Math.PI, speed: 200, owner: 1, finished: false },
    ], []);
    assert.equal(h.root().children.length, 2, '两投射物入池');
    h.fx.sync([{ id: 1, kind: 'shell', x: 130, y: 200, angle: 0, speed: 300, owner: 0, finished: false }], []);
    assert.equal(h.root().children.length, 1, '消失投射物出池');
    h.fx.sync([{ id: 9, kind: 'shell', x: 0, y: 0, angle: 0, speed: 0, owner: 0, finished: true }], []);
    assert.equal(h.root().children.length, 0, 'finished 投射物不入池且旧弹清除');
  } finally { h.restore(); }
});

test('帧间外推：x = snap + v*Δt，且 0.3s 封顶', () => {
  const h = harness();
  try {
    h.fx.sync([{ id: 1, kind: 'shell', x: 130, y: 200, angle: 0, speed: 300, owner: 0, finished: false }], []);
    h.setNow(1100); // +0.1s
    h.fx.tick();
    const m = h.root().children[0];
    assert.ok(Math.abs(m.position.x - 160) < 1e-6, `外推 x=160，实际 ${m.position.x}`);
    assert.ok(Math.abs(m.position.z - 200) < 1e-6, `外推 z 不变，实际 ${m.position.z}`);
    h.setNow(2000); // 距 sync 0.9s > 0.3s 上限
    h.fx.tick();
    assert.ok(m.position.x <= 130 + 300 * 0.3 + 1e-6, `外推封顶 ≤220，实际 ${m.position.x}`);
  } finally { h.restore(); }
});

test('位置跳变 >160 吸附不插值', () => {
  const h = harness();
  try {
    h.fx.sync([{ id: 1, kind: 'shell', x: 0, y: 0, angle: 0, speed: 100, owner: 0, finished: false }], []);
    h.setNow(1200);
    h.fx.sync([{ id: 1, kind: 'shell', x: 500, y: 500, angle: 0, speed: 100, owner: 0, finished: false }], []);
    h.fx.tick();
    assert.ok(Math.abs(h.root().children[0].position.x - 500) < 1e-6, `跳变吸附 x=500，实际 ${h.root().children[0].position.x}`);
  } finally { h.restore(); }
});

test('特效同 key 复用对象，展示时钟到期移除', () => {
  const h = harness();
  try {
    h.fx.sync([], [{ type: 'explosion', x: 800, y: 900, age: 0, duration: 1.0, owner: 0, size: 40 }]);
    const fxObj = h.root().children[0];
    h.fx.sync([], [{ type: 'explosion', x: 800, y: 900, age: 0.2, duration: 1.0, owner: 0, size: 40 }]);
    assert.equal(h.root().children[0], fxObj, '同 key 复用同一 Object3D');
    h.setNow(1600); // dispAge = 0.2 + 0.6 = 0.8 < 1.0
    h.fx.tick();
    assert.equal(h.root().children.length, 1, '0.8s 仍在');
    h.setNow(1801); // dispAge = 0.2 + 0.801 ≥ 1.0
    h.fx.tick();
    assert.equal(h.root().children.length, 0, '到期移除');
  } finally { h.restore(); }
});

test('光束几何：长度/中点/朝向（shot repair）', () => {
  const h = harness();
  try {
    h.fx.sync([], [{ type: 'shot', style: 'repair', x: 0, y: 0, toX: 300, toY: 400, age: 0, duration: 0.6, owner: 0 }]);
    h.setNow(1002);
    h.fx.tick();
    const beam = h.root().children[0];
    assert.ok(Math.abs(beam.scale.x - 500) < 1e-6, `长度=500，实际 ${beam.scale.x}`);
    assert.ok(Math.abs(beam.position.x - 150) < 1e-6 && Math.abs(beam.position.z - 200) < 1e-6, `中点(150,200)，实际 (${beam.position.x},${beam.position.z})`);
    const want = -Math.atan2(400, 300);
    assert.ok(Math.abs(beam.rotation.y - want) < 1e-6, `朝向 ${want}，实际 ${beam.rotation.y}`);
  } finally { h.restore(); }
});

test('脉冲半径：size 覆盖 spec，easeOut 推进', () => {
  const h = harness();
  try {
    h.fx.sync([], [{ type: 'explosion', x: 10, y: 10, age: 0.5, duration: 1.0, owner: 0, size: 40 }]);
    // 不推进本地时钟：dispAge 恰为服务端 age=0.5。
    h.fx.tick();
    const pulse = h.root().children[0];
    // ease(0.5)=0.75 → 40*0.75=30
    assert.ok(Math.abs(pulse.scale.x - 30) < 1e-6, `半径=30，实际 ${pulse.scale.x}`);
  } finally { h.restore(); }
});
