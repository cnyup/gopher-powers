<script setup lang="ts">
// Battle：战局 HUD + Three.js 战场（快照驱动 + 选区 + 右键移动）。
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import type { NetClient } from '../net/client';
import { SceneView } from '../scene/SceneView';

const props = defineProps<{ net: NetClient; view: any }>();
const canvas = ref<HTMLCanvasElement | null>(null);
const selection = ref<number[]>([]);
let scene: SceneView | null = null;
let raf = 0;

const v = computed(() => props.view);
const me = computed(() => v.value?.players?.[0] ?? null);
const units = computed(() => (v.value?.units ?? []) as any[]);

onMounted(() => {
  if (!canvas.value) return;
  scene = new SceneView(canvas.value);
  scene.onSelectChange = ids => { selection.value = ids; };
  scene.onGroundClick = (x, y) => {
    if (selection.value.length) {
      props.net.command('move', { x, y, selected: selection.value });
    }
  };
  const loop = () => {
    scene?.render();
    raf = requestAnimationFrame(loop);
  };
  raf = requestAnimationFrame(loop);
  const ro = new ResizeObserver(() => {
    const el = canvas.value!.parentElement!;
    scene?.resize(el.clientWidth, el.clientHeight);
  });
  ro.observe(canvas.value.parentElement!);
  onUnmounted(() => ro.disconnect());
});
onUnmounted(() => { cancelAnimationFrame(raf); scene = null; });

watch(() => props.view, view => scene?.syncView(view));

function train(type: string) { props.net.command('train', { unitType: type }); }
function build(type: string) { props.net.command('build', { unitType: type }); }
function stop() { if (selection.value.length) props.net.command('stop', { selected: selection.value }); }
</script>

<template>
  <div class="battle" v-if="v">
    <header>
      <span class="res">资金 {{ Math.floor(me?.credits ?? 0) }}</span>
      <span class="res">电力 {{ me?.powerIn ?? 0 }}/{{ me?.powerOut ?? 0 }}</span>
      <span class="res">时间 {{ v.time?.toFixed(0) }}s</span>
      <span class="res" :class="{ paused: v.paused }">{{ v.paused ? '暂停中' : '作战中' }}</span>
      <span class="sel" v-if="selection.length">已选 {{ selection.length }} 单位</span>
      <button class="ghost" @click="net.leave()">认输退出</button>
    </header>
    <main>
      <section class="field">
        <canvas ref="canvas"></canvas>
      </section>
      <aside>
        <div class="panel">
          <h3>选中</h3>
          <button :disabled="!selection.length" @click="stop()">停止</button>
        </div>
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
        <div class="panel roster">
          <h3>部队 {{ units.length }}</h3>
          <div v-for="u in units.slice(0, 12)" :key="u.id" class="ro">
            #{{ u.id }} {{ u.type }} hp{{ Math.ceil(u.hp ?? 0) }}
          </div>
        </div>
      </aside>
    </main>
  </div>
</template>

<style scoped>
.battle { height: 100%; display: flex; flex-direction: column; }
header { display: flex; gap: 18px; align-items: center; padding: 10px 16px; background: #0e1620; border-bottom: 1px solid #1d2c40; }
.res { font-size: 14px; color: #cfe4f5; } .res.paused { color: #d8b65a; }
.sel { font-size: 13px; color: #6ad48f; }
button { background: #1d3048; color: #cfe4f5; border: none; border-radius: 6px; padding: 7px 12px; cursor: pointer; font-size: 13px; }
button:disabled { opacity: .4; }
button.ghost { background: transparent; border: 1px solid #2a3b52; margin-left: auto; }
main { flex: 1; display: flex; min-height: 0; }
.field { flex: 1; position: relative; min-width: 0; }
.field canvas { position: absolute; inset: 0; width: 100%; height: 100%; display: block; }
aside { width: 210px; padding: 12px; border-left: 1px solid #1d2c40; display: flex; flex-direction: column; gap: 10px; overflow-y: auto; }
.panel { background: #121a24; border: 1px solid #24344a; border-radius: 8px; padding: 10px; display: flex; flex-direction: column; gap: 6px; }
.panel h3 { font-size: 12px; color: #7f97ad; margin-bottom: 4px; font-weight: 600; }
.roster { flex: 1; min-height: 0; }
.ro { font-size: 12px; color: #9db4c8; }
</style>
