<script setup>
import { ref, computed, watch, onMounted, inject } from 'vue'
import { trashApi } from '../api/index.js'
import { formatSize, formatDate } from '../utils/format.js'
import { useI18n } from '../i18n/index.js'
import ConfirmDialog from '../components/ConfirmDialog.vue'

const toast = inject('toast')
const { t } = useI18n()

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
    confirmText: confirmText || t('common.confirm'),
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
    t('trash.restoreTitle', { name: item.file_name }),
    t('trash.restoreMessage', { name: item.file_name }),
    false,
    t('trash.restoreBtn'),
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
    t('trash.batchRestoreTitle', { count: selected.length }),
    t('trash.batchRestoreMessage', { count: selected.length }),
    false,
    t('trash.restoreBtn'),
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
    t('trash.batchDeleteTitle', { count: selected.length }),
    t('trash.batchDeleteMessage', { count: selected.length }),
    true,
    t('trash.deleteForeverBtn'),
    async () => {
      try {
        for (const item of selected) {
          if (item.is_dir) await trashApi.removeFolder(item.id)
          else await trashApi.removeFile(item.id)
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
    t('trash.emptyTitle'),
    t('trash.emptyMessage', { count: items.value.length }),
    true,
    t('trash.deleteAllBtn'),
    async () => {
      try {
        for (const item of items.value) {
          if (item.is_dir) await trashApi.removeFolder(item.id)
          else await trashApi.removeFile(item.id)
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
    t('trash.permanentDeleteTitle', { name: item.file_name }),
    t('trash.permanentDeleteMessage', { name: item.file_name }),
    true,
    t('trash.deleteForeverBtn'),
    async () => {
      try {
        if (item.is_dir) await trashApi.removeFolder(item.id)
        else await trashApi.removeFile(item.id)
        fetchTrash()
      } catch (e) {
        toast('! ' + e.message)
      }
    }
  )
}

function fileIcon(item) {
  if (item.is_dir) return { label: '[' + t('files.fileType.dir') + ']', cls: 'fi-dir' }
  const ext = (item.file_ext || '').toLowerCase()
  if (['.jpg','.jpeg','.png','.gif','.svg','.webp','.bmp','.ico'].includes(ext)) return { label: '[' + t('files.fileType.img') + ']', cls: 'fi-img' }
  if (['.mp4','.mkv','.avi','.mov','.webm','.flv','.wmv'].includes(ext)) return { label: '[' + t('files.fileType.vid') + ']', cls: 'fi-vid' }
  if (['.mp3','.wav','.flac','.ogg','.aac','.wma','.m4a'].includes(ext)) return { label: '[' + t('files.fileType.aud') + ']', cls: 'fi-aud' }
  if (['.zip','.rar','.7z','.tar','.gz','.bz2','.xz'].includes(ext)) return { label: '[' + t('files.fileType.zip') + ']', cls: 'fi-zip' }
  if (['.pdf'].includes(ext)) return { label: '[' + t('files.fileType.pdf') + ']', cls: 'fi-pdf' }
  if (['.doc','.docx','.xls','.xlsx','.ppt','.pptx'].includes(ext)) return { label: '[' + t('files.fileType.doc') + ']', cls: 'fi-doc' }
  if (['.txt','.md','.log','.csv'].includes(ext)) return { label: '[' + t('files.fileType.txt') + ']', cls: 'fi-txt' }
  if (['.js','.ts','.jsx','.tsx','.vue','.py','.go','.rs','.java','.c','.cpp','.h','.json','.xml','.html','.css','.scss','.yaml','.toml','.sh','.bat'].includes(ext)) return { label: '[' + t('files.fileType.code') + ']', cls: 'fi-code' }
  return { label: '[   ]', cls: 'fi-other' }
}

onMounted(fetchTrash)
</script>

<template>
  <div class="trash-view">
    <div class="view-header">
      <h2>[x] {{ t('trash.title') }}</h2>
      <span class="count">{{ t('trash.items', { count: total }) }}</span>
      <div class="header-actions">
        <button v-if="items.length > 0" class="btn btn-sm btn-danger" @click="handleDeleteAll">[X] {{ t('trash.emptyTrash') }}</button>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="selectedCount > 0" class="fixed-batch-bar">
      <span class="count">{{ t('files.batchBar', { count: selectedCount }) }}</span>
      <button class="btn btn-sm" @click="handleBatchRestore">[+] {{ t('trash.restoreBtn') }}</button>
      <button class="btn btn-sm btn-danger" @click="handleBatchDelete">[x] {{ t('trash.deleteForeverBtn') }}</button>
      <button class="btn btn-sm" @click="selectedSet = new Set(); selectAll = false">[X] {{ t('trash.cancelBtn') }}</button>
    </div>
    </Teleport>

    <div class="table-wrap" v-if="!loading && items.length > 0">
      <table>
        <thead>
          <tr>
            <th style="width:30px"><input type="checkbox" class="cb" :checked="selectAll" @click.stop="toggleSelectAll" /></th>
            <th style="width:80px"></th>
            <th>{{ t('common.name') }}</th>
            <th style="width:100px">{{ t('common.size') }}</th>
            <th style="width:140px">{{ t('common.deleted') }}</th>
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
                <button class="btn btn-icon btn-sm" :title="t('trash.restoreBtn')" aria-label="Restore item" @click="handleRestore(item)">[+]</button>
                <button class="btn btn-icon btn-sm btn-rm" :title="t('trash.deleteForeverBtn')" aria-label="Permanently delete item" @click="handlePermanentDelete(item)">[x]</button>
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else-if="loading" class="empty-state">
      <span class="cursor-blink">_</span> {{ t('common.loading') }}
    </div>

    <div v-else class="empty-state">
      <span class="icon">[ok]</span>
      <span>{{ t('trash.emptyState') }}</span>
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
  white-space: nowrap;
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
