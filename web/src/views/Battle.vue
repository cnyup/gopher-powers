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
const pending = computed<string | null>(() => me.value?.readyBuilding || null);
const placing = ref<string | null>(null);
const units = computed(() => (v.value?.units ?? []) as any[]);

onMounted(() => {
  if (!canvas.value) return;
  scene = new SceneView(canvas.value);
  scene.loadAssets(); // 45MB GLB 模型库异步载入，就绪后新单位/建筑自动切换真模型
  scene.onSelectChange = ids => { selection.value = ids; };
  scene.onPlace = (type, x, y) => place(type, x, y);
  watch(placing, p => { if (p) scene?.beginPlace(p); else scene?.cancelPlace(); });
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

watch(() => props.view, view => {
  if (view?.players?.[0]?.faction && scene) scene.faction = view.players[0].faction;
  if (scene?.assetError) console.warn('模型库载入失败：', scene.assetError);
  scene?.syncView(view);
});

function train(type: string) { props.net.command('train', { unitType: type }); }
function build(type: string) {
  // 已有待放建筑时先切换目标；完成建造（readyBuilding 非空）时进入放置模式。
  props.net.command('build', { unitType: type });
}
function place(type: string, x: number, y: number) { props.net.command('place', { unitType: type, x, y }); placing.value = null; }
function stop() { if (selection.value.length) props.net.command('stop', { selected: selection.value }); }
// readyBuilding 就绪 → 点击「放置」按钮进入落位模式（或直接自动进入）。
watch(pending, p => { if (p && !placing.value) placing.value = p; });
const esc = (e: KeyboardEvent) => { if (e.key === 'Escape') { placing.value = null; scene?.cancel(); } };
onMounted(() => document.addEventListener('keydown', esc));
onUnmounted(() => document.removeEventListener('keydown', esc));
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
          <div class="placing-hint" v-if="placing">放置中：{{ placing }}（ESC 取消）</div>
          <div class="placing-hint dim" v-else-if="me?.buildQueue">建造中：{{ me.buildQueue.type }} {{ Math.round((me.buildQueue.progress ?? 0) * 100) }}%</div>
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
.placing-hint { font-size: 12px; color: #6ad48f; } .placing-hint.dim { color: #d8b65a; }
.ro { font-size: 12px; color: #9db4c8; }
</style>
