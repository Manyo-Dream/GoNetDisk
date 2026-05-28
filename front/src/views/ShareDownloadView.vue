<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { shareApi } from '../api/index.js'
import { formatSize, formatDate } from '../utils/format.js'

const route = useRoute()
const info = ref(null)
const loading = ref(true)
const error = ref('')
const pwd = ref('')
const pwdError = ref('')
const needsCode = ref(false)
const codeInUrl = ref(false)

const shareCode = computed(() => route.query.code || '')

onMounted(async () => {
  if (!shareCode.value) {
    error.value = 'Missing share code'
    loading.value = false
    return
  }
  try {
    const urlPwd = route.query.pwd
    info.value = await shareApi.getInfo(shareCode.value, urlPwd)
    needsCode.value = info.value.has_code
    if (urlPwd && needsCode.value) {
      pwd.value = urlPwd
      codeInUrl.value = true
      handleDownload()
    }
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
})

function handleDownload() {
  if (needsCode.value && !pwd.value) {
    pwdError.value = 'Extraction code required'
    return
  }
  pwdError.value = ''
  const url = shareApi.download(shareCode.value, pwd.value)
  window.open(url, '_blank')
}
</script>

<template>
  <div class="share-page">
    <div v-if="loading" class="share-card">
      <span class="cursor-blink">_</span> LOADING…
    </div>

    <div v-else-if="error" class="share-card">
      <span class="err-icon">!</span>
      <p class="err-msg">{{ error }}</p>
    </div>

    <div v-else-if="info" class="share-card">
      <div class="card-header">
        <span class="logo">[<>]</span>
        <h1>SHARED FILE</h1>
      </div>

      <div class="file-info">
        <div class="info-row">
          <span>NAME</span>
          <span class="val">{{ info.file_name }}</span>
        </div>
        <div class="info-row">
          <span>SIZE</span>
          <span class="val">{{ formatSize(info.file_size) }}</span>
        </div>
        <div class="info-row">
          <span>SHARED</span>
          <span class="val">{{ formatDate(info.created_at) }}</span>
        </div>
      </div>

      <div v-if="needsCode && !codeInUrl" class="pwd-section">
        <label>
          <span>EXTRACTION CODE</span>
          <input
            v-model="pwd"
            type="text"
            placeholder="input code…"
            @keyup.enter="handleDownload"
            autocomplete="off"
          />
        </label>
        <div v-if="pwdError" class="error-msg">! {{ pwdError }}</div>
      </div>

      <div class="card-actions">
        <button class="btn btn-primary" @click="handleDownload">
          [v] DOWNLOAD
        </button>
      </div>

      <div class="card-footer">
        <router-link to="/">[&lt;] BACK TO GONETDISK</router-link>
      </div>
    </div>
  </div>
</template>

<style scoped>
.share-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100%;
  padding: var(--gap-6);
}
.share-card {
  width: 400px;
  background: var(--bg-surface);
  border: 1px solid var(--border);
  padding: var(--gap-6);
}
.card-header {
  text-align: center;
  margin-bottom: var(--gap-6);
}
.logo {
  display: block;
  font-size: var(--fs-xl);
  color: var(--accent);
  font-weight: 700;
  margin-bottom: var(--gap-1);
}
h1 {
  font-size: var(--fs-lg);
  font-weight: 600;
}
.file-info {
  display: flex;
  flex-direction: column;
  gap: var(--gap-2);
  margin-bottom: var(--gap-6);
  padding-bottom: var(--gap-4);
  border-bottom: 1px solid var(--border-faint);
}
.info-row {
  display: flex;
  justify-content: space-between;
  font-size: var(--fs-sm);
}
.info-row span:first-child {
  color: var(--text-dim);
  font-weight: 600;
  letter-spacing: 0.04em;
}
.val {
  font-weight: 500;
}
.pwd-section {
  margin-bottom: var(--gap-4);
}
.pwd-section label {
  display: flex;
  flex-direction: column;
  gap: var(--gap-1);
}
.pwd-section label span {
  font-size: var(--fs-xs);
  color: var(--text-dim);
  font-weight: 600;
  letter-spacing: 0.06em;
}
.pwd-section input {
  padding: var(--gap-2) var(--gap-3);
}
.card-actions {
  display: flex;
  justify-content: center;
  margin-bottom: var(--gap-4);
}
.card-footer {
  text-align: center;
  padding-top: var(--gap-4);
  border-top: 1px solid var(--border-faint);
  font-size: var(--fs-xs);
}
.error-msg {
  color: var(--red);
  font-size: var(--fs-xs);
  margin-top: var(--gap-2);
  padding: var(--gap-1) var(--gap-2);
  border: 1px solid var(--red-dim);
}
.err-icon {
  display: block;
  text-align: center;
  font-size: var(--fs-2xl);
  color: var(--red);
  margin-bottom: var(--gap-4);
}
.err-msg {
  text-align: center;
  color: var(--text-dim);
  font-size: var(--fs-sm);
}
</style>
