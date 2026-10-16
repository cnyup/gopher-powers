<script setup lang="ts">
// Lobby：昵称 + 建房/加入。
import { ref } from 'vue';
import type { NetClient } from '../net/client';

const props = defineProps<{ net: NetClient }>();
const name = ref('');
const joinCode = ref('');
const faction = ref('china');
const factions = [
  { id: 'china', name: '华夏' }, { id: 'russia', name: '俄联' },
  { id: 'nato', name: '北约' }, { id: 'asia', name: '东瀛' }, { id: 'middleeast', name: '新月' },
];
const maps = [
  { id: 'valley', name: '山谷对峙' }, { id: 'island', name: '跨岛争夺' }, { id: 'plains', name: '平原遭遇' },
];

async function saveName() {
  if (name.value.trim()) await props.net.rename(name.value.trim());
}
</script>

<template>
  <div class="lobby">
    <h1>大国崛起 <span class="ver">Go 版</span></h1>
    <div class="card">
      <label>指挥官昵称</label>
      <div class="row">
        <input v-model="name" placeholder="输入昵称" maxlength="16" @change="saveName" />
        <span class="whoami">{{ net.user?.name || '未命名' }}</span>
      </div>
    </div>
    <div class="card">
      <label>创建战局</label>
      <div class="row">
        <select v-model="faction">
          <option v-for="f in factions" :key="f.id" :value="f.id">{{ f.name }}</option>
        </select>
        <select v-model="maps[0].id" disabled>
          <option>山谷对峙（默认）</option>
        </select>
        <button class="primary" @click="net.create('valley', faction, 'annihilation')">创建房间</button>
      </div>
    </div>
    <div class="card">
      <label>加入战局</label>
      <div class="row">
        <select v-model="faction">
          <option v-for="f in factions" :key="f.id" :value="f.id">{{ f.name }}</option>
        </select>
        <input v-model="joinCode" placeholder="六位房间码" maxlength="6" inputmode="numeric" />
        <button class="primary" @click="net.join(joinCode, faction)">加入</button>
      </div>
    </div>
    <div class="record" v-if="net.user">
      战绩 {{ net.user.wins }} 胜 / {{ net.user.losses }} 负 / {{ net.user.draws }} 平
    </div>
  </div>
</template>

<style scoped>
.lobby { max-width: 560px; margin: 0 auto; padding: 48px 20px; display: flex; flex-direction: column; gap: 16px; }
h1 { font-size: 28px; } .ver { font-size: 12px; color: #5f7a94; margin-left: 8px; }
.card { background: #121a24; border: 1px solid #24344a; border-radius: 10px; padding: 16px; }
label { display: block; font-size: 12px; color: #7f97ad; margin-bottom: 8px; }
.row { display: flex; gap: 8px; align-items: center; }
input, select {
  background: #0b1219; border: 1px solid #2a3b52; color: #dfe7ee;
  border-radius: 6px; padding: 8px 10px; font-size: 14px; flex: 1;
}
button {
  border: none; border-radius: 6px; padding: 9px 18px; font-size: 14px; cursor: pointer;
  background: #1d3048; color: #cfe4f5;
}
button.primary { background: #1d5c8f; color: #fff; }
.whoami { color: #6f8aa3; font-size: 13px; }
.record { text-align: center; color: #6f8aa3; font-size: 13px; }
</style>
