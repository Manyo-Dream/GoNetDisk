<script setup>
import { ref, onMounted, inject } from 'vue'
import { shareApi } from '../api/index.js'
import { formatDate } from '../utils/format.js'

import ConfirmDialog from '../components/ConfirmDialog.vue'

const toast = inject('toast')

const shares = ref([])
const loading = ref(false)

const confirmData = ref({
  visible: false,
  title: '',
  message: '',
  danger: false,
  confirmText: 'CONFIRM',
  onConfirm: null,
})

function showConfirm(title, message, danger, confirmText, cb) {
  confirmData.value = {
    visible: true,
    title,
    message,
    danger,
    confirmText: confirmText || 'CONFIRM',
    onConfirm: cb,
  }
}

function onConfirm() {
  const cb = confirmData.value.onConfirm
  confirmData.value.visible = false
  if (cb) cb()
}

async function fetchShares() {
  loading.value = true
  try {
    const res = await shareApi.list({ page: 1, page_size: 50 })
    shares.value = res.list || []
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

async function copyLink(item, withCode = false) {
  let link = `${window.location.origin}/#/share?code=${item.share_code}`
  if (withCode && item.code) {
    link += `&pwd=${encodeURIComponent(item.code)}`
  }
  try {
    await navigator.clipboard.writeText(link)
  } catch (e) {
    /* clipboard blocked — link still shown */
  }
  toast(`[<] Link copied: …/${item.share_code}${withCode && item.code ? ' [+pwd]' : ''}`)
}

async function revokeShare(item) {
  showConfirm(
    `[x] REVOKE SHARE`,
    `Revoke share link for "${item.file_name || item.share_code}"?\nThe link will stop working immediately.`,
    true,
    'REVOKE',
    async () => {
      try {
        await shareApi.revoke(item.share_code)
        fetchShares()
      } catch (e) {
        toast('! ' + e.message)
      }
    }
  )
}

function statusBadge(item) {
  if (item.expire_at) {
    const exp = new Date(item.expire_at)
    if (exp < new Date()) return { text: 'EXPIRED', cls: 'badge-red' }
  }
  return { text: 'ACTIVE', cls: 'badge-green' }
}

onMounted(fetchShares)
</script>

<template>
  <div class="shares-view">
    <div class="view-header">
      <h2>&lt;&gt; MY SHARES</h2>
      <span class="count">{{ shares.length }} links</span>
    </div>

    <div v-if="!loading && shares.length > 0" class="share-list">
      <div v-for="item in shares" :key="item.share_code" class="share-card">
        <div class="share-info">
          <span class="share-name">{{ item.file_name || 'FILE' }}</span>
          <div class="share-meta">
            <span class="code">CODE: {{ item.share_code }}</span>
            <span v-if="item.code" class="pwd">PWD: {{ item.code }}</span>
            <span class="views">VIEWS: {{ item.view_count || 0 }}</span>
          </div>
          <div class="share-date">{{ formatDate(item.created_at) }}</div>
        </div>
        <div class="share-status">
          <span class="badge" :class="statusBadge(item).cls">{{ statusBadge(item).text }}</span>
        </div>
        <div class="share-actions">
          <button class="btn btn-sm" @click="copyLink(item, false)">[&lt;] COPY</button>
          <button v-if="item.code" class="btn btn-sm" @click="copyLink(item, true)">[&lt;] COPY+PWD</button>
          <button class="btn btn-sm btn-danger" @click="revokeShare(item)">[x] REVOKE</button>
        </div>
      </div>
    </div>

    <div v-else-if="loading" class="empty-state">
      <span class="cursor-blink">_</span> LOADING…
    </div>

    <div v-else class="empty-state">
      <span class="icon">&lt;&gt;</span>
      <span>NO SHARES YET</span>
      <span style="font-size:var(--fs-xs);color:var(--text-muted)">Share files from the file browser</span>
    </div>

    <ConfirmDialog
      :visible="confirmData.visible"
      :title="confirmData.title"
      :message="confirmData.message"
      :danger="confirmData.danger"
      :confirm-text="confirmData.confirmText"
      @confirm="onConfirm"
      @close="confirmData.visible = false"
    />
  </div>
</template>

<style scoped>
.shares-view {
  display: flex;
  flex-direction: column;
  gap: var(--gap-4);
}
.view-header {
  display: flex;
  align-items: baseline;
  gap: var(--gap-3);
  padding-bottom: var(--gap-3);
  border-bottom: 1px solid var(--border-faint);
}
h2 {
  font-size: var(--fs-lg);
  font-weight: 600;
  color: var(--cyan);
}
.count {
  font-size: var(--fs-xs);
  color: var(--text-dim);
}
.share-list {
  display: flex;
  flex-direction: column;
  gap: var(--gap-2);
}
.share-card {
  display: flex;
  align-items: center;
  gap: var(--gap-4);
  padding: var(--gap-3) var(--gap-4);
  background: var(--bg-surface);
  border: 1px solid var(--border);
}
.share-info {
  flex: 1;
}
.share-name {
  font-size: var(--fs-sm);
  font-weight: 500;
}
.share-meta {
  display: flex;
  gap: var(--gap-4);
  margin-top: var(--gap-1);
  font-size: var(--fs-xs);
  color: var(--text-dim);
}
.share-date {
  font-size: var(--fs-xs);
  color: var(--text-muted);
  margin-top: 2px;
}
.share-actions {
  display: flex;
  gap: var(--gap-2);
}
</style>
