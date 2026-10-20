// scene/FxLayer.ts：投射物/特效渲染层——快照驱动 + 帧间插值。
// 快照协议（internal/server/view.go battleView）：
//   projectiles[]：{ id, kind, style, x, y, toX, toY, startX, startY, angle, speed,
//                    owner(0=己方/1=敌方), amount, hp, age, finished, ... }
//   effects[]：    { type, x, y, toX?, toY?, age, duration, owner, style, size?, amount? }
// 投射物有稳定 id（差量池，帧间按 angle*speed 外推）；
// 特效无 id（Age<Duration 存活），按 type|style|网格化坐标 匹配复用，
// 展示时钟 = 服务端 age + 本地流逝，两次快照之间平滑推进。
import * as THREE from 'three';

/** 投射物视觉规格：按 kind 分形。 */
type ProjShape = 'shell' | 'missile' | 'torpedo' | 'wing';
const PROJ_SPEC: Record<string, { shape: ProjShape; size: number }> = {
  shell: { shape: 'shell', size: 3.2 },
  missile: { shape: 'missile', size: 4.5 },
  rocket: { shape: 'missile', size: 4.2 },
  loitering: { shape: 'missile', size: 4.0 },
  bomb: { shape: 'shell', size: 3.0 },
  torpedo: { shape: 'torpedo', size: 4.0 },
  wing: { shape: 'wing', size: 5.0 },
};
const PROJ_DEFAULT = { shape: 'shell' as ProjShape, size: 3.5 };

/** 各 kind 的飞行高度（three y 轴，世界单位）。 */
const PROJ_HEIGHT: Record<string, number> = {
  wing: 60, bomb: 50, missile: 26, rocket: 26, loitering: 26, torpedo: 3,
};

/** 特效视觉规格：按 type 分发。 */
type FxKind = 'pulse' | 'beam' | 'flash' | 'text';
interface FxSpec { kind: FxKind; color: number; radius: number }
const FX_SPEC: Record<string, FxSpec> = {
  explosion: { kind: 'pulse', color: 0xffa040, radius: 46 },
  capture: { kind: 'pulse', color: 0xffd24a, radius: 36 },
  sonar: { kind: 'pulse', color: 0x64d8ff, radius: 70 },
  jam: { kind: 'pulse', color: 0xc07ff5, radius: 60 },
  order: { kind: 'pulse', color: 0x6ad48f, radius: 22 },
  build: { kind: 'pulse', color: 0x6ad48f, radius: 30 },
  ability: { kind: 'pulse', color: 0x64d8ff, radius: 40 },
  supplyDrop: { kind: 'pulse', color: 0xffffff, radius: 26 },
  income: { kind: 'text', color: 0x8fe89a, radius: 0 },
  hit: { kind: 'flash', color: 0xffe08a, radius: 16 },
  shot: { kind: 'beam', color: 0xffd24a, radius: 0 },
};

/** shot 特效按 style 分色（激光/防空/维修）。 */
function shotColor(style: string): number {
  if (style === 'laser') return 0x64d8ff;
  if (style === 'aa') return 0xffe08a;
  if (style === 'repair') return 0x6ad48f;
  return 0xffd24a;
}

function ownerColor(owner: number): number {
  return owner === 0 ? 0x8fe8b0 : 0xff9a8a;
}

function effectColor(type: string, style: string): number {
  return type === 'shot' ? shotColor(style) : (FX_SPEC[type]?.color ?? 0xffffff);
}

interface Tracked {
  mesh: THREE.Object3D;
  vx: number; vy: number;      // 速度分量（cos(angle)*speed / sin(angle)*speed）
  snapX: number; snapY: number; // 最近快照位置
  syncAt: number;               // 快照时刻（performance.now ms）
}

interface ActiveFx {
  obj: THREE.Object3D;
  key: string;
  x: number; y: number; toX: number; toY: number;
  age: number; duration: number;
  createdAt: number; // 首次见到该特效的本地时刻
  size?: number; amount?: number;
  color: number;
}

/** 外推上限：超过此时长快照未到，投射物原地等待（防脱缰）。 */
const EXTRAPOLATION_CAP_S = 0.3;
/** 快照间位置跳变阈值：超过视为新弹道段，插值起点重置。 */
const JUMP_RESET_PX = 160;

export class FxLayer {
  private root = new THREE.Group();
  private projectiles = new Map<number, Tracked>();
  private active: ActiveFx[] = [];

  constructor(scene: THREE.Scene) {
    scene.add(this.root);
  }

  /** 快照同步：投射物按 id 差量；特效按 key 匹配复用。 */
  sync(projectiles: any[], effects: any[]) {
    const now = performance.now();

    // —— 投射物差量池 ——
    const seen = new Set<number>();
    for (const p of projectiles) {
      if (!p || p.finished || typeof p.id !== 'number') continue;
      seen.add(p.id);
      const vx = Math.cos(p.angle ?? 0) * (p.speed ?? 0);
      const vy = Math.sin(p.angle ?? 0) * (p.speed ?? 0);
      let t = this.projectiles.get(p.id);
      if (!t) {
        const spec = PROJ_SPEC[p.kind] ?? PROJ_DEFAULT;
        t = {
          mesh: this.makeProjectileMesh(spec.shape, spec.size, p.owner, p.kind),
          vx, vy, snapX: p.x, snapY: p.y, syncAt: now,
        };
        this.root.add(t.mesh);
        this.projectiles.set(p.id, t);
      } else if (Math.hypot(t.snapX - p.x, t.snapY - p.y) > JUMP_RESET_PX) {
        // 位置大幅跳变（重定向/瞬移）：不插值，直接吸附。
        t.snapX = p.x; t.snapY = p.y; t.syncAt = now;
        t.mesh.position.set(p.x, PROJ_HEIGHT[p.kind] ?? 14, p.y);
      }
      t.vx = vx; t.vy = vy;
      t.snapX = p.x; t.snapY = p.y; t.syncAt = now;
      t.mesh.rotation.y = -(p.angle ?? 0); // 模拟角 → three 绕 y 旋转
    }
    for (const [id, t] of this.projectiles) {
      if (!seen.has(id)) {
        this.root.remove(t.mesh);
        this.disposeObj(t.mesh);
        this.projectiles.delete(id);
      }
    }

    // —— 特效匹配复用 ——
    const incoming = new Map<string, any>();
    for (const e of effects) {
      if (!e || typeof e.duration !== 'number' || e.duration <= 0) continue;
      const key = `${e.type}|${e.style ?? ''}|${Math.round(e.x)}|${Math.round(e.y)}`;
      if (!incoming.has(key)) incoming.set(key, e);
    }
    // 移除不再出现的。
    for (let i = this.active.length - 1; i >= 0; i--) {
      const f = this.active[i];
      if (!incoming.has(f.key)) {
        this.root.remove(f.obj);
        this.disposeObj(f.obj);
        this.active.splice(i, 1);
      }
    }
    // 更新已有的 / 创建新到的。
    const activeByKey = new Map(this.active.map(f => [f.key, f]));
    for (const [key, e] of incoming) {
      const exists = activeByKey.get(key);
      if (exists) {
        exists.age = e.age;
        exists.duration = e.duration;
        exists.toX = e.toX ?? e.x;
        exists.toY = e.toY ?? e.y;
        continue;
      }
      const obj = this.makeEffectObj(e);
      if (!obj) continue;
      this.active.push({
        obj, key,
        x: e.x, y: e.y, toX: e.toX ?? e.x, toY: e.toY ?? e.y,
        age: e.age, duration: e.duration,
        createdAt: now, size: e.size, amount: e.amount,
        color: effectColor(e.type, e.style ?? ''),
      });
      this.root.add(obj);
    }
  }

  /** 帧推进：投射物外推插值 + 特效按展示时钟动画。 */
  tick() {
    const now = performance.now();
    for (const [, t] of this.projectiles) {
      const el = Math.min((now - t.syncAt) / 1000, EXTRAPOLATION_CAP_S);
      t.mesh.position.x = t.snapX + t.vx * el;
      t.mesh.position.z = t.snapY + t.vy * el;
    }
    for (let i = this.active.length - 1; i >= 0; i--) {
      const f = this.active[i];
      const dispAge = f.age + (now - f.createdAt) / 1000;
      if (dispAge >= f.duration) {
        this.root.remove(f.obj);
        this.disposeObj(f.obj);
        this.active.splice(i, 1);
        continue;
      }
      this.applyFx(f, dispAge / f.duration);
    }
  }

  private applyFx(f: ActiveFx, pr: number) {
    const spec = FX_SPEC[f.key.split('|')[0]] ?? FX_SPEC.order;
    const ease = 1 - (1 - pr) * (1 - pr); // easeOutQuad
    switch (spec.kind) {
      case 'pulse': {
        const base = f.size && f.size > 0 ? f.size : spec.radius;
        f.obj.scale.setScalar(Math.max(base * ease, 0.01));
        this.setOpacity(f.obj, pr < 0.75 ? 0.85 : 0.85 * (1 - (pr - 0.75) / 0.25));
        break;
      }
      case 'beam': {
        const dx = f.toX - f.x, dz = f.toY - f.y;
        const len = Math.hypot(dx, dz) || 1;
        f.obj.scale.set(len, 1, 1);
        f.obj.position.set(f.x + dx / 2, 8, f.y + dz / 2);
        f.obj.rotation.y = -Math.atan2(dz, dx);
        this.setOpacity(f.obj, 0.85 * (1 - pr));
        break;
      }
      case 'flash': {
        const base = f.size && f.size > 0 ? f.size : spec.radius;
        f.obj.scale.setScalar(base * (0.4 + 0.8 * pr));
        this.setOpacity(f.obj, pr < 0.25 ? pr / 0.25 : 1 - (pr - 0.25) / 0.75);
        break;
      }
      case 'text': {
        f.obj.position.set(f.x, 14 + 34 * pr, f.y);
        this.setOpacity(f.obj, 1 - pr);
        break;
      }
    }
  }

  private setOpacity(obj: THREE.Object3D, opacity: number) {
    const mat = (obj as THREE.Mesh | THREE.Sprite).material as THREE.Material & { opacity: number };
    mat.opacity = Math.max(0, Math.min(1, opacity));
  }

  // —— 构造 ——

  private makeProjectileMesh(shape: ProjShape, size: number, owner: number, kind: string): THREE.Object3D {
    const color = ownerColor(owner);
    const mat = new THREE.MeshBasicMaterial({ color, transparent: true, opacity: 0.95 });
    let geo: THREE.BufferGeometry;
    switch (shape) {
      case 'missile': {
        geo = new THREE.ConeGeometry(size * 0.5, size * 3.4, 6);
        geo.rotateZ(-Math.PI / 2); // 锥尖指向 +x
        break;
      }
      case 'torpedo': {
        geo = new THREE.CapsuleGeometry(size * 0.45, size * 2.8, 4, 6);
        geo.rotateZ(-Math.PI / 2);
        break;
      }
      case 'wing':
        geo = new THREE.BoxGeometry(size * 2.4, size * 0.4, size * 1.7);
        break;
      default:
        geo = new THREE.SphereGeometry(size, 8, 6);
    }
    const mesh = new THREE.Mesh(geo, mat);
    mesh.position.y = PROJ_HEIGHT[kind] ?? 14;
    return mesh;
  }

  private makeEffectObj(e: any): THREE.Object3D | null {
    const type: string = e.type;
    const spec = FX_SPEC[type];
    if (!spec) return null;
    const color = effectColor(type, e.style ?? '');
    switch (spec.kind) {
      case 'pulse': {
        const geo = new THREE.RingGeometry(0.55, 1, 48);
        const mat = new THREE.MeshBasicMaterial({ color, transparent: true, opacity: 0.85, side: THREE.DoubleSide, depthWrite: false });
        const mesh = new THREE.Mesh(geo, mat);
        mesh.rotation.x = -Math.PI / 2; // 平铺地面
        mesh.position.set(e.x, 2.5, e.y);
        return mesh;
      }
      case 'beam': {
        const geo = new THREE.BoxGeometry(1, 3, 3); // 沿 x 单位长，帧内缩放到实际长度
        const mat = new THREE.MeshBasicMaterial({ color, transparent: true, opacity: 0.85, depthWrite: false });
        return new THREE.Mesh(geo, mat);
      }
      case 'flash': {
        const geo = new THREE.SphereGeometry(1, 10, 8);
        const mat = new THREE.MeshBasicMaterial({ color, transparent: true, opacity: 1, depthWrite: false });
        const mesh = new THREE.Mesh(geo, mat);
        mesh.position.set(e.x, 10, e.y);
        return mesh;
      }
      case 'text': {
        const amt = typeof e.amount === 'number' ? Math.round(e.amount) : 0;
        const spriteMat = new THREE.SpriteMaterial({
          map: makeTextTexture(`+$${amt}`, color), transparent: true, depthWrite: false,
        });
        const sprite = new THREE.Sprite(spriteMat);
        sprite.scale.set(46, 23, 1);
        sprite.position.set(e.x, 14, e.y);
        return sprite;
      }
    }
  }

  private disposeObj(obj: THREE.Object3D) {
    obj.traverse(o => {
      const mesh = o as THREE.Mesh;
      if (mesh.geometry) mesh.geometry.dispose();
      const mat = (mesh as THREE.Sprite | THREE.Mesh).material as THREE.Material & { map?: THREE.Texture } | undefined;
      if (mat) {
        mat.map?.dispose();
        mat.dispose();
      }
    });
  }

  dispose() {
    for (const [, t] of this.projectiles) { this.root.remove(t.mesh); this.disposeObj(t.mesh); }
    this.projectiles.clear();
    for (const f of this.active) { this.root.remove(f.obj); this.disposeObj(f.obj); }
    this.active = [];
  }
}

/** 收入浮字纹理（income 特效，+$/数值）。 */
function makeTextTexture(text: string, color: number): THREE.CanvasTexture {
  const canvas = document.createElement('canvas');
  canvas.width = 128;
  canvas.height = 64;
  const ctx = canvas.getContext('2d')!;
  const hex = '#' + color.toString(16).padStart(6, '0');
  ctx.font = 'bold 34px system-ui, sans-serif';
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  ctx.fillStyle = hex;
  ctx.fillText(text, 64, 32);
  return new THREE.CanvasTexture(canvas);
}
