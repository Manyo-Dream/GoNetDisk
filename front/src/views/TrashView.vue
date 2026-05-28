<script setup>
import { ref, computed, watch, onMounted, inject } from 'vue'
import { trashApi, fileApi, folderApi } from '../api/index.js'
import { formatSize, formatDate } from '../utils/format.js'
import ConfirmDialog from '../components/ConfirmDialog.vue'

const toast = inject('toast')

const items = ref([])
const loading = ref(false)
const page = ref(1)
const total = ref(0)

const selectedSet = ref(new Set())
const selectAll = ref(false)

watch(items, () => {
  selectedSet.value = new Set()
  selectAll.value = false
})

const selectedCount = computed(() => selectedSet.value.size)

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

function toggleSelect(id, item) {
  const next = new Set(selectedSet.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedSet.value = next
  selectAll.value = selectedSet.value.size === items.value.length
}

function toggleSelectAll() {
  if (selectAll.value) {
    selectedSet.value = new Set()
    selectAll.value = false
  } else {
    selectedSet.value = new Set(items.value.map(f => f.id))
    selectAll.value = true
  }
}

async function fetchTrash() {
  loading.value = true
  try {
    const res = await trashApi.list({ page: page.value, page_size: 50 })
    items.value = res.list || []
    total.value = res.total || 0
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

async function handleRestore(item) {
  showConfirm(
    `[+] RESTORE: ${item.file_name}`,
    `Restore "${item.file_name}" back to its original location?`,
    false,
    'RESTORE',
    async () => {
      try {
        if (item.is_dir) await trashApi.restoreFolder(item.id)
        else await trashApi.restoreFile(item.id)
        fetchTrash()
      } catch (e) {
        toast('! ' + e.message)
      }
    }
  )
}

async function handleBatchRestore() {
  const selected = items.value.filter(f => selectedSet.value.has(f.id))
  if (selected.length === 0) return
  showConfirm(
    `[+] BATCH RESTORE: ${selected.length} items`,
    `Restore ${selected.length} items back to their original locations?`,
    false,
    'RESTORE',
    async () => {
      try {
        for (const item of selected) {
          if (item.is_dir) await trashApi.restoreFolder(item.id)
          else await trashApi.restoreFile(item.id)
        }
        fetchTrash()
      } catch (e) {
        toast('! ' + e.message)
      }
    }
  )
}

async function handleBatchDelete() {
  const selected = items.value.filter(f => selectedSet.value.has(f.id))
  if (selected.length === 0) return
  showConfirm(
    `[x] PERMANENT DELETE: ${selected.length} items`,
    `This will permanently delete ${selected.length} items.\nThis action CANNOT be undone.`,
    true,
    'DELETE FOREVER',
    async () => {
      try {
        for (const item of selected) {
          if (item.is_dir) await folderApi.remove(item.id)
          else await fileApi.remove(item.id)
        }
        fetchTrash()
      } catch (e) {
        toast('! ' + e.message)
      }
    }
  )
}

async function handleDeleteAll() {
  if (items.value.length === 0) return
  showConfirm(
    `[x] EMPTY TRASH`,
    `This will permanently delete ALL ${items.value.length} items in trash.\nThis action CANNOT be undone.`,
    true,
    'DELETE ALL',
    async () => {
      try {
        for (const item of items.value) {
          if (item.is_dir) await folderApi.remove(item.id)
          else await fileApi.remove(item.id)
        }
        fetchTrash()
      } catch (e) {
        toast('! ' + e.message)
      }
    }
  )
}

async function handlePermanentDelete(item) {
  showConfirm(
    `[x] PERMANENTLY DELETE: ${item.file_name}`,
    `This will permanently delete "${item.file_name}".\nThis action CANNOT be undone. Continue?`,
    true,
    'DELETE FOREVER',
    async () => {
      try {
        if (item.is_dir) await folderApi.remove(item.id)
        else await fileApi.remove(item.id)
        fetchTrash()
      } catch (e) {
        toast('! ' + e.message)
      }
    }
  )
}

function fileIcon(item) {
  if (item.is_dir) return { label: '[DIR]', cls: 'fi-dir' }
  const ext = (item.file_ext || '').toLowerCase()
  if (['.jpg','.jpeg','.png','.gif','.svg','.webp','.bmp','.ico'].includes(ext)) return { label: '[IMG]', cls: 'fi-img' }
  if (['.mp4','.mkv','.avi','.mov','.webm','.flv','.wmv'].includes(ext)) return { label: '[VID]', cls: 'fi-vid' }
  if (['.mp3','.wav','.flac','.ogg','.aac','.wma','.m4a'].includes(ext)) return { label: '[AUD]', cls: 'fi-aud' }
  if (['.zip','.rar','.7z','.tar','.gz','.bz2','.xz'].includes(ext)) return { label: '[ZIP]', cls: 'fi-zip' }
  if (['.pdf'].includes(ext)) return { label: '[PDF]', cls: 'fi-pdf' }
  if (['.doc','.docx','.xls','.xlsx','.ppt','.pptx'].includes(ext)) return { label: '[DOC]', cls: 'fi-doc' }
  if (['.txt','.md','.log','.csv'].includes(ext)) return { label: '[TXT]', cls: 'fi-txt' }
  if (['.js','.ts','.jsx','.tsx','.vue','.py','.go','.rs','.java','.c','.cpp','.h','.json','.xml','.html','.css','.scss','.yaml','.toml','.sh','.bat'].includes(ext)) return { label: '[COD]', cls: 'fi-code' }
  return { label: '[   ]', cls: 'fi-other' }
}

onMounted(fetchTrash)
</script>

<template>
  <div class="trash-view">
    <div class="view-header">
      <h2>[x] TRASH</h2>
      <span class="count">{{ total }} items</span>
      <div class="header-actions">
        <button v-if="items.length > 0" class="btn btn-sm btn-danger" @click="handleDeleteAll">[X] EMPTY TRASH</button>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="selectedCount > 0" class="fixed-batch-bar">
      <span class="count">{{ selectedCount }} selected</span>
      <button class="btn btn-sm" @click="handleBatchRestore">[+] RESTORE</button>
      <button class="btn btn-sm btn-danger" @click="handleBatchDelete">[x] DELETE FOREVER</button>
      <button class="btn btn-sm" @click="selectedSet = new Set(); selectAll = false">[X] CANCEL</button>
    </div>
    </Teleport>

    <div class="table-wrap" v-if="!loading && items.length > 0">
      <table>
        <thead>
          <tr>
            <th style="width:30px"><input type="checkbox" class="cb" :checked="selectAll" @click.stop="toggleSelectAll" /></th>
            <th style="width:56px"></th>
            <th>NAME</th>
            <th style="width:100px">SIZE</th>
            <th style="width:140px">DELETED</th>
            <th style="width:120px"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in items" :key="item.id">
            <td class="td-cb">
              <input type="checkbox" class="cb" :checked="selectedSet.has(item.id)" @click.stop="toggleSelect(item.id, item)" />
            </td>
            <td class="td-icon" :class="fileIcon(item).cls">{{ fileIcon(item).label }}</td>
            <td class="td-name">{{ item.file_name }}</td>
            <td class="td-size">{{ item.is_dir ? '--' : formatSize(item.file_size) }}</td>
            <td class="td-date">{{ formatDate(item.deleted_at || item.updated_at) }}</td>
            <td class="td-actions">
              <span class="actions">
                <button class="btn btn-icon btn-sm" title="RESTORE" aria-label="Restore item" @click="handleRestore(item)">[+]</button>
                <button class="btn btn-icon btn-sm btn-rm" title="DELETE FOREVER" aria-label="Permanently delete item" @click="handlePermanentDelete(item)">[x]</button>
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else-if="loading" class="empty-state">
      <span class="cursor-blink">_</span> LOADING…
    </div>

    <div v-else class="empty-state">
      <span class="icon">[ok]</span>
      <span>TRASH IS EMPTY</span>
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
.trash-view {
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
  color: var(--red);
}
.count {
  font-size: var(--fs-xs);
  color: var(--text-dim);
}
.header-actions {
  margin-left: auto;
}
.td-cb {
  width: 30px;
  text-align: center;
}
.td-icon {
  font-size: var(--fs-xs);
  font-weight: 600;
}
.td-name {
  font-size: var(--fs);
  font-weight: 500;
}
.td-size, .td-date {
  font-size: var(--fs-sm);
  color: var(--text-dim);
}
.td-size {
  font-variant-numeric: tabular-nums;
}
.actions {
  display: flex;
  gap: 2px;
  opacity: 0;
}
tr:hover .actions {
  opacity: 1;
}
.btn-rm:hover {
  color: var(--red) !important;
}
.fixed-batch-bar {
  position: fixed;
  bottom: 0;
  left: 200px;
  right: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  gap: var(--gap-3);
  padding: var(--gap-3) var(--gap-6);
  background: var(--red-dim);
  backdrop-filter: blur(4px);
  border-top: 1px solid var(--red);
  font-size: var(--fs-xs);
}
.fixed-batch-bar .count {
  color: var(--red);
  font-weight: 600;
}
.fixed-batch-bar .btn {
  border-color: var(--red);
  color: var(--red);
}
.fixed-batch-bar .btn-sm {
  padding-top: 3px;
  padding-bottom: 3px;
  line-height: 1.1;
}
</style>
