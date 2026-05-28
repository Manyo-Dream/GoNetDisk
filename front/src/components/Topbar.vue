<script setup>
import { ref, onMounted } from 'vue'
import { userApi } from '../api/index.js'
import { formatSize } from '../utils/format.js'

const props = defineProps({
  userInfo: Object,
})
const emit = defineEmits(['logout'])

const space = ref(null)

onMounted(async () => {
  try {
    space.value = await userApi.getSpace()
  } catch (e) { /* ignore */ }
})

function spacePct() {
  if (!space.value) return 0
  if (space.value.total_space === 0) return 0
  return Math.min(100, (space.value.used_space / space.value.total_space) * 100)
}
</script>

<template>
  <header class="topbar">
    <div class="topbar-left">
      <span class="path-indicator" v-if="space">
        STORAGE  {{ formatSize(space.used_space) }} / {{ formatSize(space.total_space) }}
        <span class="pct">{{ spacePct().toFixed(0) }}%</span>
      </span>
    </div>
    <div class="topbar-right">
      <span class="user-tag" v-if="userInfo">
        [{{ userInfo.username || userInfo.email }}]
      </span>
      <button class="btn btn-sm" @click="emit('logout')">EXIT</button>
    </div>
    <div class="space-bar" v-if="space">
      <div class="fill" :style="{ width: spacePct() + '%' }"></div>
    </div>
  </header>
</template>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--gap-4);
  height: 40px;
  min-height: 40px;
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border);
  position: relative;
  gap: var(--gap-4);
}
.topbar-left, .topbar-right {
  display: flex;
  align-items: center;
  gap: var(--gap-3);
  z-index: 1;
}
.path-indicator {
  font-size: var(--fs-xs);
  color: var(--text-dim);
  letter-spacing: 0.05em;
}
.pct {
  color: var(--accent);
  font-weight: 600;
}
.user-tag {
  font-size: var(--fs-xs);
  color: var(--text-dim);
}
.space-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--bg-root);
}
.space-bar .fill {
  height: 100%;
  background: var(--accent);
  transition: width 0.3s linear;
}
</style>
