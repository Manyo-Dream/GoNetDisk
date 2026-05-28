<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  items: { type: Array, default: () => [] },
})
const emit = defineEmits(['action'])

const visible = ref(false)
const x = ref(0)
const y = ref(0)
const el = ref(null)

function show(e) {
  e.preventDefault()
  visible.value = true
  x.value = e.clientX
  y.value = e.clientY
}

function hide() {
  visible.value = false
}

function doAction(action) {
  emit('action', action)
  hide()
}

function onClickOutside(e) {
  if (el.value && !el.value.contains(e.target)) {
    hide()
  }
}

onMounted(() => document.addEventListener('click', onClickOutside))
onUnmounted(() => document.removeEventListener('click', onClickOutside))

defineExpose({ show })
</script>

<template>
  <div
    v-if="visible"
    ref="el"
    class="ctx-menu"
    :style="{ left: x + 'px', top: y + 'px' }"
  >
    <button
      v-for="item in items"
      :key="item.action"
      :class="{ danger: item.danger }"
      @click="doAction(item.action)"
    >
      {{ item.label }}
    </button>
  </div>
</template>

<style scoped>
.ctx-menu {
  position: fixed;
  z-index: 200;
  background: var(--bg-raised);
  border: 1px solid var(--border);
  min-width: 160px;
  z-index: 200;
}
.ctx-menu button {
  display: block;
  width: 100%;
  text-align: left;
  padding: var(--gap-2) var(--gap-4);
  font-size: var(--fs-sm);
  cursor: pointer;
}
.ctx-menu button:hover {
  background: var(--bg-hover);
  color: var(--accent);
}
.ctx-menu button.danger:hover {
  color: var(--red);
}
</style>
