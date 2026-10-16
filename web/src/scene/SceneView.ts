// scene/SceneView.ts：Three.js 战场最小闭环——快照驱动（地形/单位/建筑/迷雾覆盖/选区/指令）。
// 模型工厂平移（model-craft 等 30+ 模块）后续波次以 adapter 注入替换基本体。
import * as THREE from 'three';

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
    this.createMarker();
    canvas.addEventListener('pointerdown', this.onPointerDown);
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

  private createMarker() {
    this.marker = new THREE.Mesh(
      new THREE.RingGeometry(14, 20, 32),
      new THREE.MeshBasicMaterial({ color: 0x6ad48f, transparent: true, opacity: .85 }),
    );
    this.marker.rotation.x = -Math.PI / 2;
    this.marker.visible = false;
    this.scene.add(this.marker);
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
          m = new THREE.Mesh(this.shapeFor({ ...u } as EntityLike & { kind: string }), this.matFor(u.owner));
          m.userData = { id: u.id, kind: 'unit' };
          this.scene.add(m);
          this.pickables.push(m);
          this.unitMeshes.set(u.id, m);
        }
        m.position.set(x, 5, z);
        m.lookAt(x + 1, 5, z);
        (m.material as THREE.MeshStandardMaterial).color.set(u.owner === 0 ? 0x3f8f5f : 0xa04848);
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
          m = new THREE.Mesh(this.shapeFor({ ...b } as EntityLike & { kind: string }), this.matFor(b.owner));
          m.userData = { id: b.id, kind: 'building' };
          this.scene.add(m);
          this.buildingMeshes.set(b.id, m);
        }
        m.position.set(x, 8, z);
        (m.material as THREE.MeshStandardMaterial).color.set(b.owner === 0 ? 0x4f7f9f : 0xa04848);
      }
      for (const [id, m] of this.buildingMeshes) {
        if (!seenB.has(id)) {
          this.scene.remove(m);
          this.buildingMeshes.delete(id);
        }
      }
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

  flashMarker(x: number, z: number) {
    this.marker.position.set(x, .6, z);
    this.marker.visible = true;
    window.setTimeout(() => { this.marker.visible = false; }, 900);
  }

  resize(w: number, h: number) {
    this.renderer.setSize(w, h, false);
    this.camera.aspect = w / h;
    this.camera.updateProjectionMatrix();
  }

  render() {
    this.renderer.render(this.scene, this.camera);
  }
}
