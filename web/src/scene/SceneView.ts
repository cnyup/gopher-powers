// scene/SceneView.ts：Three.js 战场最小闭环——快照驱动（地形/单位/建筑/迷雾覆盖/选区/指令）。
// 模型工厂平移（model-craft 等 30+ 模块）后续波次以 adapter 注入替换基本体。
import * as THREE from 'three';
import { prepareVisualAssets, createModel } from './models/visual-assets.js';
import { equipmentModel, teamColor } from './equipmentModel';

export interface EntityLike {
  id: number; type: string; owner: number;
  x: number; y: number; hp?: number; maxHp?: number; size?: number;
}

export class SceneView {
  renderer: THREE.WebGLRenderer;
  scene = new THREE.Scene();
  camera: THREE.PerspectiveCamera;
  raycaster = new THREE.Raycaster();
  private pickables: THREE.Mesh[] = [];
  private unitMeshes = new Map<number, THREE.Mesh>();
  private buildingMeshes = new Map<number, THREE.Mesh>();
  private ground!: THREE.Mesh;
  private marker!: THREE.Mesh;
  selected = new Set<number>();
  onSelectChange: ((ids: number[]) => void) | null = null;
  onGroundClick: ((x: number, y: number) => void) | null = null;
  private world = { width: 3360, height: 2160 };
  private assetsReady = false;
  private assetsLoading: Promise<void> | null = null;
  assetError = '';
  faction = 'china'; // 己方阵营（battle 消息 config 同步）

  constructor(canvas: HTMLCanvasElement) {
    this.renderer = new THREE.WebGLRenderer({ canvas, antialias: true });
    this.renderer.setPixelRatio(Math.min(devicePixelRatio, 2));
    this.scene.background = new THREE.Color(0x0d141c);
    this.scene.fog = new THREE.Fog(0x0d141c, 1800, 4200);
    this.camera = new THREE.PerspectiveCamera(46, 16 / 9, 10, 12000);
    // 战场相机：斜俯视（对齐源 battle-camera 沉浸视角基调）。
    this.camera.position.set(450, 900, 1250);
    this.camera.lookAt(450, 0, 720);
    const sun = new THREE.DirectionalLight(0xfff2dd, 2.2);
    sun.position.set(600, 1400, 400);
    this.scene.add(sun, new THREE.AmbientLight(0x8899bb, 0.55));
    this.createGround();
    this.createFogLayer(this.world.width, this.world.height);
    this.createMarker();
    canvas.addEventListener('pointerdown', this.onPointerDown);
    canvas.addEventListener('pointermove', ev => this.updateGhost(ev));
    canvas.addEventListener('keydown', () => {});
    canvas.tabIndex = 0;
    canvas.addEventListener('contextmenu', e => e.preventDefault());
  }

  private createGround() {
    const geo = new THREE.PlaneGeometry(this.world.width, this.world.height);
    const mat = new THREE.MeshStandardMaterial({ color: 0x27343d, roughness: .95 });
    this.ground = new THREE.Mesh(geo, mat);
    this.ground.rotation.x = -Math.PI / 2;
    this.ground.position.set(this.world.width / 2, 0, this.world.height / 2);
    this.scene.add(this.ground);
    // 网格辅助（战术感）。
    const grid = new THREE.GridHelper(Math.max(this.world.width, this.world.height), 60, 0x1d2a36, 0x16212b);
    grid.position.set(this.world.width / 2, 0.2, this.world.height / 2);
    this.scene.add(grid);
  }

  private fogCanvas = document.createElement('canvas');
  private fogMesh!: THREE.Mesh;
  private fogCell = 80; // 迷雾格边长（world.fog=80）

  private createFogLayer(w: number, h: number) {
    this.fogCanvas.width = Math.ceil(w / this.fogCell);
    this.fogCanvas.height = Math.ceil(h / this.fogCell);
    const tex = new THREE.CanvasTexture(this.fogCanvas);
    tex.magFilter = THREE.NearestFilter;
    const mat = new THREE.MeshBasicMaterial({ map: tex, transparent: true, opacity: .92, depthWrite: false });
    this.fogMesh = new THREE.Mesh(new THREE.PlaneGeometry(w, h), mat);
    this.fogMesh.rotation.x = -Math.PI / 2;
    this.fogMesh.position.set(w / 2, 1.2, h / 2);
    this.scene.add(this.fogMesh);
    this.paintFog(null);
  }

  /** 迷雾绘制：visible=透明 / explored=半暗 / 未知=全黑。 */
  paintFog(fog: { cols: number; rows: number; visible: Uint8Array; explored: Uint8Array } | null) {
    const ctx = this.fogCanvas.getContext('2d')!;
    const w = this.fogCanvas.width, h = this.fogCanvas.height;
    ctx.clearRect(0, 0, w, h);
    ctx.fillStyle = '#060a0f';
    ctx.fillRect(0, 0, w, h);
    if (fog && fog.cols === w && fog.rows === h) {
      for (let y = 0; y < h; y++) for (let x = 0; x < w; x++) {
        const i = y * w + x;
        if (fog.explored[i]) {
          ctx.fillStyle = fog.visible[i] ? 'rgba(6,10,15,0)' : 'rgba(6,10,15,0.55)';
          ctx.fillRect(x, y, 1, 1);
        }
      }
    }
    (this.fogMesh.material as THREE.MeshBasicMaterial).map!.needsUpdate = true;
  }

  private createMarker() {
    this.marker = new THREE.Mesh(
      new THREE.RingGeometry(14, 20, 32),
      new THREE.MeshBasicMaterial({ color: 0x6ad48f, transparent: true, opacity: .85 }),
    );
    this.marker.rotation.x = -Math.PI / 2;
    this.marker.visible = false;
    this.scene.add(this.marker);
  }

  /** 载入 GLB 模型库（45MB，异步；就绪前基本体兜底渲染）。 */
  loadAssets(): Promise<void> {
    if (!this.assetsLoading) {
      this.assetsLoading = prepareVisualAssets().then(() => {
        this.assetsReady = true;
        // 就绪前已用基本体兜底的单位/建筑，清缓存待下帧重建为真模型。
        for (const [, m] of this.unitMeshes) this.scene.remove(m);
        for (const [, m] of this.buildingMeshes) this.scene.remove(m);
        this.unitMeshes.clear();
        this.buildingMeshes.clear();
        this.pickables = this.pickables.filter(o => !o.userData?.id);
      }).catch((e) => { this.assetError = String(e); });
    }
    return this.assetsLoading!;
  }

  /** 世界坐标（模拟 y → three z）。 */
  private toScene(x: number, y: number): [number, number] {
    return [x, y];
  }

  private matFor(owner: number): THREE.MeshStandardMaterial {
    // owner 已按快照重排：0=己方绿、1=敌方红。
    return new THREE.MeshStandardMaterial({ color: owner === 0 ? 0x3f8f5f : 0xa04848, roughness: .7 });
  }

  private shapeFor(e: EntityLike): THREE.BufferGeometry {
    const isBuilding = ((e as any).kind === 'building') || ((e as any).size ?? 0) > 0;
    if (isBuilding) {
      const s = e.size ? e.size / 46 : 1;
      return new THREE.BoxGeometry(24 * s, 16 * s, 24 * s);
    }
    if (['rifle', 'engineer', 'scout'].includes(e.type)) {
      return new THREE.CapsuleGeometry(4.5, 9, 4, 8);
    }
    return new THREE.BoxGeometry(11, 7, 15); // 载具
  }

  /** 快照同步：差量更新单位/建筑网格。 */
  syncView(view: any) {
    if (view?.running !== undefined && view.units && view.buildings) {
      const seenU = new Set<number>();
      for (const u of view.units as EntityLike[]) {
        seenU.add(u.id);
        let m = this.unitMeshes.get(u.id);
        const [x, z] = this.toScene(u.x, u.y);
        if (!m) {
          const holder = new THREE.Group();
          holder.userData = { id: u.id, kind: 'unit' };
          if (this.assetsReady) {
            try {
              const model = createModel(equipmentModel(this.faction, u.type), teamColor(u.owner));
              model.scale.setScalar(0.3);
              holder.add(model);
            } catch { /* 库缺该型号时基本体兜底 */ }
          }
          if (holder.children.length === 0) {
            const fallback = new THREE.Mesh(this.shapeFor({ ...u } as EntityLike & { kind: string }), this.matFor(u.owner));
            holder.add(fallback);
          }
          // 拾取代理：不可见盒包住模型。
          const proxy = new THREE.Mesh(this.shapeFor({ ...u } as EntityLike & { kind: string }), new THREE.MeshBasicMaterial({ visible: false }));
          proxy.userData = { id: u.id, kind: 'unit' };
          holder.add(proxy);
          this.pickables.push(proxy);
          m = holder as unknown as THREE.Mesh;
          this.scene.add(m);
          this.unitMeshes.set(u.id, m);
        }
        m.position.set(x, 5, z);
      }
      for (const [id, m] of this.unitMeshes) {
        if (!seenU.has(id)) {
          this.scene.remove(m);
          this.pickables = this.pickables.filter(p => p !== m);
          this.unitMeshes.delete(id);
          this.selected.delete(id);
        }
      }
      const seenB = new Set<number>();
      for (const b of view.buildings as EntityLike[]) {
        seenB.add(b.id);
        let m = this.buildingMeshes.get(b.id);
        const [x, z] = this.toScene(b.x, b.y);
        if (!m) {
          const holder = new THREE.Group();
          holder.userData = { id: b.id, kind: 'building' };
          if (this.assetsReady) {
            try {
              const model = createModel(equipmentModel(this.faction, b.type), teamColor(b.owner));
              holder.add(model);
            } catch { /* 基本体兜底 */ }
          }
          if (holder.children.length === 0) {
            const fallback = new THREE.Mesh(this.shapeFor({ ...b } as EntityLike & { kind: string }), this.matFor(b.owner));
            holder.add(fallback);
          }
          const proxy = new THREE.Mesh(this.shapeFor({ ...b } as EntityLike & { kind: string }), new THREE.MeshBasicMaterial({ visible: false }));
          proxy.userData = { id: b.id, kind: 'building' };
          holder.add(proxy);
          this.pickables.push(proxy);
          m = holder as unknown as THREE.Mesh;
          this.scene.add(m);
          this.buildingMeshes.set(b.id, m);
        }
        m.position.set(x, 8, z);
      }
      for (const [id, m] of this.buildingMeshes) {
        if (!seenB.has(id)) {
          this.scene.remove(m);
          this.buildingMeshes.delete(id);
        }
      }
    }
    if (view?.fog) {
      // 解包 base64 迷雾位并绘制。
      const bin = atob(view.fog.bits);
      const cols = view.fog.cols, rows = view.fog.rows;
      const visible = new Uint8Array(cols * rows), explored = new Uint8Array(cols * rows);
      for (let i = 0; i < cols * rows; i++) {
        const b = bin.charCodeAt(i >> 2);
        visible[i] = b & (2 << ((i % 4) * 2)) ? 1 : 0;
        explored[i] = b & (1 << ((i % 4) * 2)) ? 1 : 0;
      }
      this.paintFog({ cols, rows, visible, explored });
    }
  }

  private groundPoint(ev: PointerEvent): THREE.Vector3 | null {
    const rect = this.renderer.domElement.getBoundingClientRect();
    const ndc = new THREE.Vector2(
      ((ev.clientX - rect.left) / rect.width) * 2 - 1,
      -((ev.clientY - rect.top) / rect.height) * 2 + 1,
    );
    this.raycaster.setFromCamera(ndc, this.camera);
    const hit = this.raycaster.intersectObject(this.ground);
    return hit.length ? hit[0].point : null;
  }

  private onPointerDown = (ev: PointerEvent) => {
    if (ev.button === 2) {
      // 右键：移动指令目标点。
      const p = this.groundPoint(ev);
      if (p) {
        this.flashMarker(p.x, p.z);
        this.onGroundClick?.(p.x, p.z);
      }
      return;
    }
    if (ev.button === 0 && this.placingType) {
      const p = this.groundPoint(ev);
      if (p) {
        const t = this.placingType;
        this.cancelPlace();
        this.onPlace?.(t, p.x, p.z);
      }
      return;
    }
    if (ev.button === 0) {
      const rect = this.renderer.domElement.getBoundingClientRect();
      const ndc = new THREE.Vector2(
        ((ev.clientX - rect.left) / rect.width) * 2 - 1,
        -((ev.clientY - rect.top) / rect.height) * 2 + 1,
      );
      this.raycaster.setFromCamera(ndc, this.camera);
      const hits = this.raycaster.intersectObjects(this.pickables);
      this.selected.clear();
      if (hits.length) {
        this.selected.add(hits[0].object.userData.id as number);
      }
      this.onSelectChange?.([...this.selected]);
    }
  };

  private ghost: THREE.Mesh | null = null;
  placingType: string | null = null;
  onPlace: ((type: string, x: number, y: number) => void) | null = null;

  private updateGhost(ev: PointerEvent) {
    if (!this.placingType) { this.clearGhost(); return; }
    const p = this.groundPoint(ev);
    if (!p) return;
    if (!this.ghost) {
      const size = { power: 66, refinery: 84, barracks: 66, factory: 88 }[this.placingType] ?? 66;
      this.ghost = new THREE.Mesh(
        new THREE.BoxGeometry(size, 14, size),
        new THREE.MeshBasicMaterial({ color: 0x6ad48f, transparent: true, opacity: .45 }),
      );
      this.scene.add(this.ghost);
    }
    this.ghost.position.set(p.x, 7, p.z);
  }

  clearGhost() {
    if (this.ghost) { this.scene.remove(this.ghost); this.ghost = null; }
  }

  beginPlace(type: string) { this.placingType = type; }
  cancelPlace() { this.placingType = null; this.clearGhost(); }

  flashMarker(x: number, z: number) {
    this.marker.position.set(x, .6, z);
    this.marker.visible = true;
    window.setTimeout(() => { this.marker.visible = false; }, 900);
  }

  /** ESC 取消放置（由 Battle 绑定 document）。 */
  cancel() { this.cancelPlace(); }

  resize(w: number, h: number) {
    this.renderer.setSize(w, h, false);
    this.camera.aspect = w / h;
    this.camera.updateProjectionMatrix();
  }

  render() {
    this.renderer.render(this.scene, this.camera);
  }
}
