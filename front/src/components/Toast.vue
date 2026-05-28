<script setup>
import { ref } from 'vue'

const toasts = ref([])
let _id = 0

function show(message, duration = 2500) {
  const id = ++_id
  toasts.value.push({ id, message })
  setTimeout(() => {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }, duration)
}

defineExpose({ show })
</script>

<template>
  <Teleport to="body">
    <div class="toast-container" v-if="toasts.length > 0">
      <TransitionGroup name="toast">
        <div
          v-for="t in toasts"
          :key="t.id"
          class="toast-item"
        >{{ t.message }}</div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-container {
  position: fixed;
  top: 52px;
  right: var(--gap-6);
  z-index: 100;
  display: flex;
  flex-direction: column;
  gap: var(--gap-2);
  pointer-events: none;
}
.toast-item {
  background: var(--bg-surface);
  border: 1px solid var(--accent);
  padding: var(--gap-2) var(--gap-4);
  font-size: var(--fs-xs);
  color: var(--text);
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
  white-space: pre-line;
  line-height: 1.6;
  max-width: 360px;
  pointer-events: auto;
}

.toast-enter-active {
  transition: all 0.2s ease-out;
}
.toast-leave-active {
  transition: all 0.4s ease-in;
}
.toast-enter-from {
  opacity: 0;
  transform: translateX(20px);
}
.toast-leave-to {
  opacity: 0;
  transform: translateX(20px);
}
</style>
