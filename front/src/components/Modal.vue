<script setup>
import { watch } from 'vue'

const props = defineProps({
  visible: Boolean,
  title: String,
})
const emit = defineEmits(['close'])

function onKeydown(e) {
  if (e.key === 'Escape' && props.visible) emit('close')
}

watch(() => props.visible, (v) => {
  if (v) document.addEventListener('keydown', onKeydown)
  else document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="overlay" role="dialog" aria-modal="true" aria-label="Close dialog" @click.self="emit('close')">
      <div class="modal">
        <div class="modal-header" v-if="title">
          <span>{{ title }}</span>
          <button class="btn btn-icon btn-sm" aria-label="Close dialog" @click="emit('close')">[X]</button>
        </div>
        <div class="modal-body">
          <slot />
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  backdrop-filter: blur(2px);
}
.modal {
  background: var(--bg-raised);
  border: 1px solid var(--border);
  min-width: 360px;
  max-width: 480px;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--gap-3) var(--gap-4);
  border-bottom: 1px solid var(--border);
  font-size: var(--fs-sm);
  font-weight: 600;
  color: var(--accent);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.modal-body {
  padding: var(--gap-4);
}
</style>
