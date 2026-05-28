<script setup>
import Modal from './Modal.vue'

const props = defineProps({
  visible: Boolean,
  title: String,
  message: String,
  danger: Boolean,
  confirmText: { type: String, default: 'CONFIRM' },
})

const emit = defineEmits(['confirm', 'close'])
</script>

<template>
  <Modal :visible="visible" :title="title" @close="emit('close')">
    <div class="confirm-body">
      <p class="confirm-message">{{ message }}</p>
      <div class="confirm-actions">
        <button class="btn btn-sm" @click="emit('close')">CANCEL</button>
        <button
          class="btn btn-sm"
          :class="danger ? 'btn-danger' : 'btn-primary'"
          @click="emit('confirm')"
        >{{ confirmText }}</button>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.confirm-body {
  display: flex;
  flex-direction: column;
  gap: var(--gap-4);
}
.confirm-message {
  font-size: var(--fs-sm);
  color: var(--text);
  white-space: pre-line;
  line-height: 1.7;
}
.confirm-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--gap-2);
}
</style>
