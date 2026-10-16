<script setup lang="ts">
// Room：席位/准备/开战。
import { computed } from 'vue';
import type { NetClient, PublicRoom } from '../net/client';

const props = defineProps<{ net: NetClient; room: PublicRoom | null }>();
const factions = ['china', 'russia', 'nato', 'asia', 'middleeast'];
const factionNames: Record<string, string> = { china: '华夏', russia: '俄联', nato: '北约', asia: '东瀛', middleeast: '新月' };
const r = computed(() => props.room);
const isHost = computed(() => !!r.value && props.net.user && r.value.members[r.value.host]?.id === props.net.user.id);
const allReady = computed(() => !!r.value?.members.every(m => m?.ready && m.connected));
</script>

<template>
  <div class="room" v-if="r && r.code">
    <h2>战局 {{ r.code }} <span class="status">{{ r.status }}</span></h2>
    <div class="seats">
      <div v-for="(m, i) in r.members" :key="i" class="seat" :class="{ empty: !m, me: m?.id === net.user?.id }">
        <template v-if="m">
          <div class="name">{{ m.name }} <span v-if="i === r.host" class="host">房主</span></div>
          <div class="fac">{{ factionNames[m.faction] || m.faction }}</div>
          <div class="state">
            <span :class="{ ok: m.connected }">{{ m.connected ? '在线' : '离线' }}</span>
            <span :class="{ ok: m.ready }">{{ m.ready ? '已准备' : '未准备' }}</span>
          </div>
          <div class="ops" v-if="m.id === net.user?.id">
            <select :value="m.faction" @change="net.faction(($event.target as HTMLSelectElement).value)">
              <option v-for="f in factions" :key="f" :value="f">{{ factionNames[f] }}</option>
            </select>
            <button @click="net.ready(!m.ready)">{{ m.ready ? '取消准备' : '准备' }}</button>
          </div>
        </template>
        <template v-else><div class="name">空席</div></template>
      </div>
    </div>
    <div class="foot">
      <button class="primary" :disabled="!isHost || !allReady" @click="net.start()">开战</button>
      <button class="ghost" @click="net.leave()">退出房间</button>
      <span v-if="isHost && !allReady" class="hint">等待双方准备…</span>
    </div>
  </div>
</template>

<style scoped>
.room { max-width: 720px; margin: 0 auto; padding: 40px 20px; }
h2 { margin-bottom: 20px; } .status { font-size: 12px; color: #5f7a94; margin-left: 10px; }
.seats { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.seat { background: #121a24; border: 1px solid #24344a; border-radius: 10px; padding: 16px; }
.seat.me { border-color: #1d5c8f; } .seat.empty { opacity: .4; }
.name { font-size: 16px; } .host { font-size: 11px; color: #d8b65a; margin-left: 6px; }
.fac { color: #7f97ad; font-size: 13px; margin: 6px 0; }
.state { display: flex; gap: 12px; font-size: 12px; color: #a06a6a; } .state .ok { color: #6db584; }
.ops { margin-top: 10px; display: flex; gap: 8px; }
select, button { background: #0b1219; border: 1px solid #2a3b52; color: #dfe7ee; border-radius: 6px; padding: 7px 12px; }
button.primary { background: #1d5c8f; border-color: #1d5c8f; color: #fff; }
button.primary:disabled { opacity: .4; cursor: not-allowed; }
button.ghost { background: transparent; }
.foot { margin-top: 24px; display: flex; gap: 12px; align-items: center; }
.hint { color: #6f8aa3; font-size: 13px; }
</style>
