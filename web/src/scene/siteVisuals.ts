// scene/siteVisuals.ts：资源点视觉——几何/配色/归属着色（纯函数，无场景容器依赖）。
// 四类站点同构：矿 ore（gold 金 / gem 青玉）、油 oil、信标 beacon、资源站 resource。
import * as THREE from 'three';

export type SiteKind = 'ore' | 'oil' | 'beacon' | 'resource';

export const SITE_COLORS = {
  gold: 0xf0c14a,
  gem: 0x74d7e8,
  oil: 0x3c4652,
  beacon: 0x8f7fe8,
  resource: 0x5f7f8f,
  neutral: 0x8a8f96,
  mine: 0x3f8f5f,
  enemy: 0xa04848,
} as const;

/** 站点离地高度（避免与地面 z-fighting）。 */
export function siteLift(kind: SiteKind): number {
  return kind === 'beacon' ? 3 : 2.5;
}

/** 站点基元几何（每次新建，调用方负责 dispose）。 */
export function siteGeometry(kind: SiteKind): THREE.BufferGeometry {
  switch (kind) {
    case 'ore': return new THREE.OctahedronGeometry(16, 0);
    case 'oil': return new THREE.CylinderGeometry(14, 16, 10, 12);
    case 'beacon': return new THREE.CylinderGeometry(3, 5, 46, 8);
    default: return new THREE.BoxGeometry(24, 8, 24);
  }
}

/** 站点本体色：矿按 kind 分金/青玉，其余固定。 */
export function siteColor(kind: SiteKind, site?: { kind?: string }): number {
  if (kind === 'ore') return site?.kind === 'gem' ? SITE_COLORS.gem : SITE_COLORS.gold;
  if (kind === 'oil') return SITE_COLORS.oil;
  if (kind === 'beacon') return SITE_COLORS.beacon;
  return SITE_COLORS.resource;
}

/** 归属色：null/undefined = 中立（返回 null，保留本色）；0 = 己方；1 = 敌方。 */
export function ownerTint(owner: unknown): number | null {
  if (typeof owner !== 'number') return null;
  return owner === 0 ? SITE_COLORS.mine : SITE_COLORS.enemy;
}

/** 造站点网格：几何 + 材质 + 归属自发光。 */
export function createSiteMesh(kind: SiteKind, site: any): THREE.Mesh {
  const color = siteColor(kind, site);
  const mat = new THREE.MeshStandardMaterial({
    color, roughness: .55, emissive: color, emissiveIntensity: .18,
  });
  const mesh = new THREE.Mesh(siteGeometry(kind), mat);
  // baseColor 记录本源色——回中立时要还原它，不能按 kind 重算（gem 会退化成 gold）。
  mesh.userData = { siteKind: kind, baseColor: color };
  applyOwnerTint(mesh, site?.owner);
  return mesh;
}

/** 就地更新归属自发光（每帧调用，不新建材质）。 */
export function applyOwnerTint(mesh: THREE.Mesh, owner: unknown) {
  const mat = mesh.material as THREE.MeshStandardMaterial;
  const tint = ownerTint(owner);
  if (tint === null) {
    // 中立：回落到本体色，去掉归属光。
    mat.emissive.setHex(mesh.userData.baseColor ?? SITE_COLORS.neutral);
    mat.emissiveIntensity = 0.18;
    return;
  }
  mat.emissive.setHex(tint);
  mat.emissiveIntensity = 0.55;
}
