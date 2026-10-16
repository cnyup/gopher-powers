// unitData.ts：源 data.js 的 UNITS 前端视图（tags/capacity/sonar 等模型层字段）。
// 完整表由 Go 侧 data 包生成快照对齐——此处先平移模型层依赖的最小集。
const raw: Record<string, any> = {
  fighter: { tags: ['air', 'jet'] }, strike: { tags: ['air', 'jet'] },
  bomber: { tags: ['air', 'jet'] }, airlift: { tags: ['air', 'jet'] },
  freightPlane: { tags: ['air', 'jet'] }, ewPlane: { tags: ['air', 'jet'] },
  drone: { tags: ['drone', 'air'] }, ghost: { tags: ['drone', 'air', 'stealth'] },
  submarine: { tags: ['ship', 'submerged'], sonar: 0 },
  destroyer: { tags: ['ship'], sonar: 290 }, frigate: { tags: ['ship'] },
  patrol: { tags: ['ship'] }, carrier: { tags: ['ship'], capacity: 6, wing: 3 },
  landing: { tags: ['ship'], capacity: 8 }, containerShip: { tags: ['ship'] },
  apc: { tags: ['vehicle'], capacity: 5 },
  rifle: { tags: ['infantry'] }, engineer: { tags: ['infantry'] },
  scout: { tags: ['infantry', 'stealth'] }, elite: { tags: ['infantry'] },
  tank: { tags: ['armor'] }, aa: { tags: ['vehicle'] },
  harvester: { tags: ['vehicle'] }, supply: { tags: ['vehicle', 'support'] },
  rocket: { tags: ['vehicle', 'artillery'] }, loiterer: { tags: ['vehicle', 'artillery'] },
  laser: { tags: ['vehicle'] },
};
export const UNITS = new Proxy(raw, {
  get(t, k: string) { return t[k]; },
});
