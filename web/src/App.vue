<script setup lang="ts">
// App：大厅/房间/战局三视图路由（依据 net 状态切换）。
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { NetClient } from './net/client';
import Lobby from './views/Lobby.vue';
import Room from './views/Room.vue';
import Battle from './views/Battle.vue';

const net = new NetClient();
const phase = ref<'lobby' | 'room' | 'battle'>('lobby');
const view = ref<any>(null);
const errorMsg = ref('');

const unsub = net.onMessage(m => {
  if (m.type === 'error') errorMsg.value = m.message;
  if (m.type === 'room') {
    phase.value = m.room && m.room.code ? 'room' : 'lobby';
  }
  if (m.type === 'battle' || m.type === 'snapshot') {
    phase.value = 'battle';
    view.value = m.view;
  }
});
const roomInfo = computed(() => net.room);

onMounted(async () => {
  await net.session();
  net.connect();
});
onUnmounted(() => { unsub(); net.close(); });
</script>

<template>
  <div class="app">
    <Lobby v-if="phase === 'lobby'" :net="net" />
    <Room v-else-if="phase === 'room'" :net="net" :room="roomInfo" />
    <Battle v-else :net="net" :view="view" />
    <div v-if="errorMsg" class="toast" @click="errorMsg = ''">{{ errorMsg }}</div>
  </div>
</template>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
html, body, .app { height: 100%; background: #0b0f14; color: #dfe7ee; font-family: system-ui, sans-serif; }
.toast {
  position: fixed; bottom: 24px; left: 50%; transform: translateX(-50%);
  background: #40222a; color: #ffb4b4; padding: 10px 20px; border-radius: 8px;
  border: 1px solid #7a3644; cursor: pointer; z-index: 99;
}
</style>
