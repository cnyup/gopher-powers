// equipmentModel.ts：源 equipment.js equipmentModel 的模型名映射（前端视图）。
// 阵营单位名 → GLB 库内模型名（tank_china / destroyer_china / elite_* …）。
export function equipmentModel(faction: string, type: string, future = false): string {
  if (type === 'ewPlane') return future ? 'space_ewPlane' : 'ewPlane';
  if (future && ['fighter', 'strike', 'bomber', 'airlift', 'freightPlane', 'aegis'].includes(type)) return `space_${type}`;
  if (type === 'navalFighter') return 'fighter';
  if (type === 'navalStrike') return 'strike';
  if (type === 'refinery' && !future) return 'logistics_depot';
  if (type === 'destroyer' && faction === 'china') return 'destroyer_china';
  if (type === 'tank' || type === 'rocket') return `${type}_${faction}`;
  if (type === 'fighter' && faction === 'china') return 'fighter_china';
  return type === 'elite' ? `elite_${faction}` : type;
}

// teamVisual：快照侧 owner 已重排（0=己方）——渲染配色对齐源 team-visuals。
export const FRIENDLY_COLOR = '#68d9f2';
export const HOSTILE_COLOR = '#ff7668';
export function teamColor(owner: number): string {
  return owner === 0 ? FRIENDLY_COLOR : HOSTILE_COLOR;
}
