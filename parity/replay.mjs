// parity/replay.mjs —— JS 侧对拍 harness。
// 用法：node parity/replay.mjs <script.json> <out.json>
// script.json: { factionA, factionB, options, commands: [{atTick, action, ...}] }
// 产出：每 20 tick 的规范化快照（时间/双方资产/单位计数/胜负）。
import { readFileSync, writeFileSync } from 'node:fs';
import { Game } from '../src/game.js';

const script = JSON.parse(readFileSync(process.argv[2], 'utf8'));
const out = process.argv[3];

// 种子 LCG（与 Go 侧同实现、同种子）——两侧随机流逐调用对齐。
function makeLCG(seed) {
  let s = seed >>> 0;
  return () => {
    s = (s + 0x6D2B79F5) >>> 0;
    let t = s;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}
const SEED = 20261014;
Math.random = makeLCG(SEED);

const events = { notice: null, voice: null, selection: null, shot: null, end: null };
const game = new Game(script.factionA, script.factionB, events, script.options || {});

const commands = (script.commands || []).slice().sort((a, b) => a.atTick - b.atTick);
let cmdIdx = 0;
const snapshots = [];

function applyCommand(c) {
  // 直接调用 Game 公有指令面（对齐 server/commands.mjs 的白名单路径）。
  switch (c.action) {
    case 'build': return game.startBuild(0, c.unitType);
    case 'place': return game.placeBuilding(0, c.unitType, c.x, c.y);
    case 'train': return game.queueUnit(0, c.unitType);
    case 'move': {
      game.selected = c.selected || [];
      return game.command(c.x, c.y, c.attackMove === true, 0);
    }
    case 'stop': { game.selected = c.selected || []; return game.stopSelected(0); }
    default: throw new Error('未知指令: ' + c.action);
  }
}

function snapshot(tick) {
  const view = {
    tick,
    time: Math.round(game.time * 1000) / 1000,
    running: game.running,
    winner: game.winner,
    credits: game.players.map(p => Math.round(p.credits * 1000) / 1000),
    units: game.players.map((_, side) => game.ownedUnits(side).length),
    buildings: game.players.map((_, side) => game.ownedBuildings(side).length),
    unitList: game.units.filter(u => u.hp > 0).map(u => ({
      id: u.id, type: u.type, owner: u.owner,
      x: Math.round(u.x * 1000) / 1000, y: Math.round(u.y * 1000) / 1000,
      hp: Math.round(u.hp * 1000) / 1000,
    })),
    ore: game.ore.map(o => Math.round(o.amount * 1000) / 1000),
  };
  snapshots.push(view);
}

for (let tick = 0; tick <= script.ticks; tick++) {
  while (cmdIdx < commands.length && commands[cmdIdx].atTick === tick) {
    applyCommand(commands[cmdIdx]);
    cmdIdx++;
  }
  if (tick % 20 === 0) snapshot(tick);
  if (!game.running) break;
  game.update(0.05);
}
// 终态。
snapshot(script.ticks + 1);
writeFileSync(out, JSON.stringify(snapshots, null, 1));
console.log('OK ticks=' + script.ticks + ' snapshots=' + snapshots.length + ' running=' + game.running + ' winner=' + game.winner);
