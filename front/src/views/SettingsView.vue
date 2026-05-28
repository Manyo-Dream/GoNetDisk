<script setup>
import { ref, onMounted, inject } from 'vue'
import { userApi } from '../api/index.js'
import { formatSize } from '../utils/format.js'

const userInfo = inject('userInfo')
const space = ref(null)
const username = ref('')
const editing = ref(false)
const error = ref('')

async function fetchData() {
  try {
    username.value = userInfo.value?.username || ''
    space.value = await userApi.getSpace()
  } catch (e) {
    console.error(e)
  }
}

async function saveInfo() {
  error.value = ''
  try {
    await userApi.updateInfo({ username: username.value })
    userInfo.value = await userApi.getInfo()
    editing.value = false
  } catch (e) {
    error.value = e.message
  }
}

function spacePct() {
  if (!space.value || space.value.total_space === 0) return 0
  return Math.min(100, (space.value.used_space / space.value.total_space) * 100)
}

onMounted(fetchData)
</script>

<template>
  <div class="settings-view">
    <div class="view-header">
      <h2>[~] SETTINGS</h2>
    </div>

    <div class="settings-grid">
      <div class="card">
        <div class="card-title">ACCOUNT</div>
        <div class="info-row" v-if="userInfo">
          <span>EMAIL</span>
          <span>{{ userInfo.email }}</span>
        </div>
        <div class="info-row" v-if="userInfo">
          <span>USERNAME</span>
          <span v-if="!editing">{{ userInfo.username || '--' }}</span>
        </div>

        <div v-if="editing" class="edit-form">
          <label>
            <span>NEW USERNAME</span>
            <input v-model="username" @keyup.enter="saveInfo" />
          </label>
          <div v-if="error" class="error-msg">! {{ error }}</div>
          <div class="btn-row">
            <button class="btn btn-sm" @click="editing = false">CANCEL</button>
            <button class="btn btn-sm btn-primary" @click="saveInfo">SAVE</button>
          </div>
        </div>

        <button v-if="!editing" class="btn btn-sm" style="margin-top:var(--gap-3)" @click="editing = true">
          [r] EDIT
        </button>
      </div>

      <div class="card">
        <div class="card-title">STORAGE</div>
        <div v-if="space" class="storage-info">
          <div class="storage-bar-wrap">
            <div class="progress-bar" style="height:12px">
              <div class="fill" :style="{ width: spacePct() + '%' }"></div>
            </div>
          </div>
          <div class="storage-numbers">
            <div class="num-block">
              <span class="num-label">USED</span>
              <span class="num-value">{{ formatSize(space.used_space) }}</span>
            </div>
            <div class="num-block">
              <span class="num-label">TOTAL</span>
              <span class="num-value">{{ space.total_space > 0 ? formatSize(space.total_space) : 'UNLIMITED' }}</span>
            </div>
            <div class="num-block">
              <span class="num-label">FREE</span>
              <span class="num-value">{{ space.total_space > 0 ? formatSize(space.total_space - space.used_space) : '--' }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-view {
  display: flex;
  flex-direction: column;
  gap: var(--gap-4);
}
.view-header {
  padding-bottom: var(--gap-3);
  border-bottom: 1px solid var(--border-faint);
}
h2 {
  font-size: var(--fs-lg);
  font-weight: 600;
  color: var(--accent);
}
.settings-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--gap-4);
}
@media (max-width: 640px) {
  .settings-grid {
    grid-template-columns: 1fr;
  }
}
.card {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  padding: var(--gap-4);
}
.card-title {
  font-size: var(--fs-xs);
  color: var(--accent);
  font-weight: 600;
  letter-spacing: 0.06em;
  margin-bottom: var(--gap-4);
  padding-bottom: var(--gap-2);
  border-bottom: 1px solid var(--border-faint);
}
.info-row {
  display: flex;
  justify-content: space-between;
  padding: var(--gap-2) 0;
  font-size: var(--fs-sm);
}
.info-row span:first-child {
  color: var(--text-dim);
}
.edit-form {
  display: flex;
  flex-direction: column;
  gap: var(--gap-3);
  margin-top: var(--gap-3);
}
.edit-form label {
  display: flex;
  flex-direction: column;
  gap: var(--gap-1);
}
.edit-form label span {
  font-size: var(--fs-xs);
  color: var(--text-dim);
  font-weight: 600;
}
.edit-form input {
  padding: var(--gap-2) var(--gap-3);
}
.btn-row {
  display: flex;
  gap: var(--gap-2);
  justify-content: flex-end;
}
.error-msg {
  color: var(--red);
  font-size: var(--fs-xs);
  padding: var(--gap-1) var(--gap-2);
  border: 1px solid var(--red-dim);
}
.storage-info {
  display: flex;
  flex-direction: column;
  gap: var(--gap-4);
}
.storage-numbers {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--gap-2);
}
.num-block {
  text-align: center;
}
.num-label {
  display: block;
  font-size: var(--fs-xs);
  color: var(--text-dim);
  letter-spacing: 0.05em;
  margin-bottom: 2px;
}
.num-value {
  font-size: var(--fs-sm);
  font-weight: 600;
}
</style>
