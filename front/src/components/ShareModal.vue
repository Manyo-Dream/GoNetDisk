<script setup>
import { ref, watch } from 'vue'
import { useI18n } from '../i18n/index.js'
import Modal from './Modal.vue'

const { t } = useI18n()

const props = defineProps({
  visible: Boolean,
  fileName: { type: String, default: '' },
})
const emit = defineEmits(['close', 'create'])

const expirePreset = ref(0)
const code = ref('')
const showCustomCode = ref(false)
const includeInLink = ref(true)

watch(() => props.visible, (v) => {
  if (v) {
    expirePreset.value = 0
    code.value = ''
    showCustomCode.value = false
    includeInLink.value = true
  }
})

function generateCode() {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < 6; i++) {
    result += chars[Math.floor(Math.random() * chars.length)]
  }
  code.value = result
}

function handleSubmit() {
  emit('create', {
    expire_days: expirePreset.value,
    code: code.value || '',
    include_in_link: includeInLink.value,
  })
}
</script>

<template>
  <Modal :visible="visible" :title="t('shares.createLink')" @close="emit('close')">
    <div class="share-form">
      <div class="file-hint">{{ fileName }}</div>

      <div class="field">
        <span>{{ t('shares.expiration') }}</span>
        <div class="expire-options">
          <button
            type="button"
            v-for="opt in [{v:0,l:t('shares.never')},{v:1,l:t('shares.day')},{v:7,l:t('shares.days',{n:7})},{v:30,l:t('shares.days',{n:30})}]"
            :key="opt.v"
            class="btn btn-sm"
            :class="{ 'btn-primary': expirePreset === opt.v }"
            @click="expirePreset = opt.v"
          >{{ opt.l }}</button>
        </div>
      </div>

      <div class="field">
        <span>{{ t('shares.extractionCodeOptional') }} <em>{{ t('shares.optionalHint') }}</em></span>
        <div v-if="!showCustomCode" class="code-toggle">
          <button type="button" class="btn btn-sm" @click="showCustomCode = true">[+] {{ t('shares.setCode') }}</button>
        </div>
        <div v-else class="code-input-row">
          <input
            v-model="code"
            :placeholder="t('shares.enterOrGenerate')"
            maxlength="16"
            spellcheck="false"
            @keyup.enter="handleSubmit"
          />
          <button type="button" class="btn btn-sm" @click="generateCode">[*] {{ t('shares.random') }}</button>
        </div>
        <label class="include-check" v-if="code">
          <input type="checkbox" class="cb" v-model="includeInLink" />
          <span>{{ t('shares.includeCode') }}</span>
        </label>
      </div>

      <div class="modal-actions">
        <button type="button" class="btn btn-sm" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-sm btn-primary" @click="handleSubmit">[<>] {{ t('shares.createLink') }}</button>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.share-form {
  display: flex;
  flex-direction: column;
  gap: var(--gap-4);
}
.file-hint {
  font-size: var(--fs-sm);
  color: var(--text-dim);
  padding-bottom: var(--gap-3);
  border-bottom: 1px solid var(--border-faint);
}
.field {
  display: flex;
  flex-direction: column;
  gap: var(--gap-2);
}
.field span {
  font-size: var(--fs-xs);
  color: var(--text-dim);
  font-weight: 600;
  letter-spacing: 0.04em;
}
.field span em {
  font-weight: 400;
  color: var(--text-muted);
}
.expire-options {
  display: flex;
  gap: var(--gap-2);
}
.code-toggle {
  display: flex;
}
.code-input-row {
  display: flex;
  gap: var(--gap-2);
}
.code-input-row input {
  flex: 1;
  padding: var(--gap-2) var(--gap-3);
}
.include-check {
  display: flex;
  align-items: center;
  gap: var(--gap-2);
  font-size: var(--fs-xs);
  color: var(--text-dim);
  cursor: pointer;
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--gap-2);
  padding-top: var(--gap-2);
}
</style>
