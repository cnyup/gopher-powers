<script setup lang="ts">
// Battle：战局 HUD 最小闭环——快照渲染（资源/单位/建筑）+ 指令面板。
// Three.js 场景层 M3 后续波次接入（src/scene/）。
import { computed, watch } from 'vue';
import type { NetClient } from '../net/client';

const props = defineProps<{ net: NetClient; view: any }>();
const v = computed(() => props.view);
const me = computed(() => v.value?.players?.[0] ?? null);
const units = computed(() => (v.value?.units ?? []) as any[]);
const buildings = computed(() => (v.value?.buildings ?? []) as any[]);

function train(type: string) { props.net.command('train', { unitType: type }); }
function build(type: string) { props.net.command('build', { unitType: type }); }
// place 指令在战场视图（点击落位）接入后启用。
// function place(type: string, x: number, y: number) { props.net.command('place', { unitType: type, x, y }); }

watch(() => v.value?.running, running => {
  if (running === false) console.log('战局结束，winner =', v.value?.winner);
});
</script>

<template>
  <div class="battle" v-if="v">
    <header>
      <span class="res">资金 {{ Math.floor(me?.credits ?? 0) }}</span>
      <span class="res">电力 {{ me?.powerIn ?? 0 }}/{{ me?.powerOut ?? 0 }}</span>
      <span class="res">时间 {{ v.time.toFixed(0) }}s</span>
      <span class="res" :class="{ paused: v.paused }">{{ v.paused ? '暂停中' : '作战中' }}</span>
      <button class="ghost" @click="net.leave()">认输退出</button>
    </header>
    <main>
      <section class="field">
        <div class="entities">
          <div v-for="b in buildings" :key="b.id" class="ent b" :title="b.type">
            {{ b.type[0].toUpperCase() }}
          </div>
          <div v-for="u in units" :key="u.id" class="ent u" :title="u.type">
            {{ u.type[0].toUpperCase() }}
          </div>
        </div>
        <p class="note">单位 {{ units.length }} · 建筑 {{ buildings.length }}（Three.js 战场视图在后续波次接入）</p>
      </section>
      <aside>
        <div class="panel">
          <h3>建造</h3>
          <button @click="build('power')">发电站</button>
          <button @click="build('refinery')">精炼厂</button>
          <button @click="build('barracks')">兵营</button>
          <button @click="build('factory')">战车厂</button>
        </div>
        <div class="panel">
          <h3>生产</h3>
          <button @click="train('rifle')">步兵</button>
          <button @click="train('tank')">主战坦克</button>
        </div>
      </aside>
    </main>
  </div>
</template>

<style scoped>
.battle { height: 100%; display: flex; flex-direction: column; }
header { display: flex; gap: 18px; align-items: center; padding: 10px 16px; background: #0e1620; border-bottom: 1px solid #1d2c40; }
.res { font-size: 14px; color: #cfe4f5; } .res.paused { color: #d8b65a; }
button { background: #1d3048; color: #cfe4f5; border: none; border-radius: 6px; padding: 7px 12px; cursor: pointer; font-size: 13px; }
button.ghost { background: transparent; border: 1px solid #2a3b52; margin-left: auto; }
main { flex: 1; display: flex; }
.field { flex: 1; padding: 16px; }
.entities { display: flex; flex-wrap: wrap; gap: 6px; }
.ent { width: 34px; height: 34px; border-radius: 6px; display: grid; place-items: center; font-weight: 700; font-size: 14px; }
.ent.b { background: #233950; color: #9fc3e8; } .ent.u { background: #24402f; color: #a5e8b8; }
.note { margin-top: 14px; color: #6f8aa3; font-size: 12px; }
aside { width: 200px; padding: 12px; border-left: 1px solid #1d2c40; display: flex; flex-direction: column; gap: 12px; }
.panel { background: #121a24; border: 1px solid #24344a; border-radius: 8px; padding: 10px; }
.panel h3 { font-size: 12px; color: #7f97ad; margin-bottom: 8px; }
.panel { display: flex; flex-direction: column; gap: 6px; }
</style>
