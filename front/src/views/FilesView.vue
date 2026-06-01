<script setup>
import { ref, onMounted, inject } from 'vue'
import { fileApi, folderApi, shareApi } from '../api/index.js'
import { formatSize } from '../utils/format.js'
import { useI18n } from '../i18n/index.js'
import Breadcrumb from '../components/Breadcrumb.vue'
import FileTable from '../components/FileTable.vue'
import UploadModal from '../components/UploadModal.vue'
import ChunkUploadModal from '../components/ChunkUploadModal.vue'
import ShareModal from '../components/ShareModal.vue'
import Modal from '../components/Modal.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'

const toast = inject('toast')
const { t } = useI18n()

const files = ref([])
const loading = ref(false)
const currentParentId = ref(0)
const breadcrumb = ref([{ id: 0, name: '~' }])
const showUpload = ref(false)
const showChunkUpload = ref(false)
const showShareModal = ref(false)
const shareTarget = ref(null)
const batchMoveItems = ref([])
const fileTableRef = ref(null)

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

const modalType = ref('')
const modalTitle = ref('')
const modalTarget = ref(null)
const modalInput = ref('')

async function fetchFiles() {
  loading.value = true
  try {
    const res = await fileApi.list({
      parent_id: currentParentId.value,
      page: 1,
      page_size: 100,
    })
    files.value = res.list || []
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function navigateTo(item) {
  currentParentId.value = item.id
  breadcrumb.value.push({ id: item.id, name: item.file_name })
  fetchFiles()
}

function navBreadcrumb(seg) {
  const idx = breadcrumb.value.findIndex((s) => s.id === seg.id)
  if (idx >= 0) {
    breadcrumb.value = breadcrumb.value.slice(0, idx + 1)
    currentParentId.value = seg.id
    fetchFiles()
  }
}

function onUploaded() {
  fetchFiles()
}

function openRename(item) {
  modalTarget.value = item
  modalInput.value = item.file_name
  modalType.value = 'rename'
  modalTitle.value = item.is_dir ? t('files.renameFolder') : t('files.renameFile')
}

function openNewFolder() {
  modalTarget.value = null
  modalInput.value = ''
  modalType.value = 'newfolder'
  modalTitle.value = t('files.newFolder')
}

async function confirmModal() {
  try {
    if (modalType.value === 'newfolder') {
      await folderApi.create({
        folder_name: modalInput.value,
        parent_id: currentParentId.value,
      })
    } else if (modalType.value === 'rename') {
      if (modalTarget.value.is_dir) {
        await folderApi.rename({
          user_folder_id: modalTarget.value.id,
          new_folder_name: modalInput.value,
        })
      } else {
        await fileApi.rename({
          user_file_id: modalTarget.value.id,
          new_file_name: modalInput.value,
        })
      }
    }
    modalType.value = ''
    fetchFiles()
  } catch (e) {
    toast('! ' + e.message)
  }
}

async function handleDownload(item) {
  const url = fileApi.download(item.id)
  try {
    const token = localStorage.getItem('access_token')
    const res = await fetch(url, {
      headers: token ? { Authorization: 'Bearer ' + token } : {},
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error(data.error || t('error.downloadFailed'))
    }
    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = blobUrl
    a.download = item.file_name
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(blobUrl)
  } catch (e) {
    toast('! ' + e.message)
  }
}

async function handleRemove(item) {
  showConfirm(
    t('files.deleteTitle', { name: item.file_name }),
    t('files.deleteMessage', { name: item.file_name }),
    true,
    t('common.delete'),
    async () => {
      try {
        if (item.is_dir) {
          await folderApi.toTrash(item.id)
        } else {
          await fileApi.toTrash(item.id)
        }
        fetchFiles()
      } catch (e) {
        toast('! ' + e.message)
      }
    }
  )
}

async function handleBatchDelete(selected) {
  if (!selected || selected.length === 0) return
  const names = selected.map(s => s.file_name).join('\n')
  showConfirm(
    t('files.batchDeleteTitle', { count: selected.length }),
    t('files.batchDeleteMessage', { names }),
    true,
    t('common.delete'),
    async () => {
      try {
        for (const item of selected) {
          if (item.is_dir) await folderApi.toTrash(item.id)
          else await fileApi.toTrash(item.id)
        }
        fetchFiles()
      } catch (e) {
        toast('! ' + e.message)
      }
    }
  )
}

async function handleBatchShare(selected) {
  if (!selected || selected.length === 0) return
  shareTarget.value = { item: selected[0], isBatch: true, batchItems: selected }
  showShareModal.value = true
}

function handleShare(item) {
  shareTarget.value = { item, isBatch: false }
  showShareModal.value = true
}

async function handleShareCreate({ expire_days, code, include_in_link }) {
  if (!shareTarget.value) return
  const st = shareTarget.value
  showShareModal.value = false

  try {
    if (st.isBatch) {
      const links = []
      for (const item of st.batchItems) {
        if (item.is_dir) continue
        const res = await shareApi.create({ user_file_id: item.id, expire_days, code })
        const pwdParam = code && include_in_link ? `&pwd=${encodeURIComponent(code)}` : ''
        links.push(`${window.location.origin}/#/share?code=${res.share_code}${pwdParam}`)
      }
      if (links.length > 0) {
        await navigator.clipboard.writeText(links.join('\n'))
        toast(`[<>] ${links.length} share links copied${code ? ' [code required]' : ''}`)
      }
    } else {
      const res = await shareApi.create({ user_file_id: st.item.id, expire_days, code })
      const pwdParam = code && include_in_link ? `&pwd=${encodeURIComponent(code)}` : ''
      const link = `${window.location.origin}/#/share?code=${res.share_code}${pwdParam}`
      await navigator.clipboard.writeText(link)
      toast(`[<>] Link copied: …/${res.share_code}${code && !include_in_link ? ` [code required]` : ''}`)
    }
  } catch (e) {
    toast(`! ${e.message}`)
  }
}

function handleBatchMove(selected) {
  if (!selected || selected.length === 0) return
  modalTarget.value = { is_dir: false, file_name: `${selected.length} items`, id: null }
  batchMoveItems.value = selected
  modalType.value = 'batchmove'
  modalTitle.value = t('files.batchMoveTitle', { count: selected.length })
  moveParentId.value = 0
  moveBreadcrumb.value = []
  moveFolders.value = []
  fetchMoveFolders()
}

function handleMove(item) {
  modalTarget.value = item
  modalType.value = 'move'
  modalTitle.value = t('files.moveTitle', { name: item.file_name })
  moveParentId.value = 0
  moveBreadcrumb.value = []
  moveFolders.value = []
  fetchMoveFolders()
}

const moveParentId = ref(0)
const moveBreadcrumb = ref([])
const moveFolders = ref([])

async function fetchMoveFolders() {
  try {
    const res = await fileApi.list({ parent_id: moveParentId.value, page: 1, page_size: 100 })
    moveFolders.value = (res.list || []).filter(f => f.is_dir)
  } catch (e) { /* ignore */ }
}

function moveNavigateTo(item) {
  moveParentId.value = item.id
  moveBreadcrumb.value.push({ id: item.id, name: item.file_name })
  fetchMoveFolders()
}

function moveNavBreadcrumb(seg) {
  const idx = moveBreadcrumb.value.findIndex(s => s.id === seg.id)
  if (idx === -1) {
    moveParentId.value = 0
    moveBreadcrumb.value = []
  } else {
    moveParentId.value = seg.id
    moveBreadcrumb.value = moveBreadcrumb.value.slice(0, idx + 1)
  }
  fetchMoveFolders()
}

async function confirmMove() {
  try {
    const targetId = moveParentId.value
    if (modalType.value === 'batchmove') {
      for (const item of batchMoveItems.value) {
        if (item.is_dir) await folderApi.move({ user_folder_id: item.id, target_parent_id: targetId })
        else await fileApi.move({ user_file_id: item.id, target_parent_id: targetId })
      }
    } else if (modalTarget.value.is_dir) {
      await folderApi.move({ user_folder_id: modalTarget.value.id, target_parent_id: targetId })
    } else {
      await fileApi.move({ user_file_id: modalTarget.value.id, target_parent_id: targetId })
    }
    modalType.value = ''
    batchMoveItems.value = []
    fetchFiles()
  } catch (e) {
    toast('! ' + e.message)
  }
}

onMounted(fetchFiles)
</script>

<template>
  <div class="files-view">
    <div class="toolbar">
      <Breadcrumb :path-stack="breadcrumb" @nav="navBreadcrumb" />
      <div class="toolbar-actions">
        <button class="btn btn-sm" @click="openNewFolder">[+] {{ t('files.newFolder') }}</button>
        <button class="btn btn-sm btn-primary" @click="showUpload = !showUpload">
          {{ showUpload ? '[-] ' + t('files.hideUpload') : '[^] ' + t('files.upload') }}
        </button>
        <button class="btn btn-sm" @click="showChunkUpload = true">[#] {{ t('files.bigFile') }}</button>
      </div>
    </div>

    <UploadModal
      :visible="showUpload"
      :current-path="breadcrumb"
      :current-parent-id="currentParentId"
      @uploaded="onUploaded"
      @close="showUpload = false"
    />

    <ChunkUploadModal
      :visible="showChunkUpload"
      :current-parent-id="currentParentId"
      @uploaded="onUploaded"
      @close="showChunkUpload = false"
    />

    <ShareModal
      :visible="showShareModal"
      :file-name="shareTarget?.isBatch ? `${shareTarget.batchItems.length} files` : shareTarget?.item?.file_name || ''"
      @close="showShareModal = false"
      @create="handleShareCreate"
    />

    <FileTable
      ref="fileTableRef"
      :files="files"
      :loading="loading"
      @open-dir="navigateTo"
      @download="handleDownload"
      @remove="handleRemove"
      @share="handleShare"
      @rename="openRename"
      @move="handleMove"
    />

    <Teleport to="body">
      <div v-if="fileTableRef?.selectedCount > 0" class="sticky-batch-bar">
      <span class="count">{{ t('files.batchBar', { count: fileTableRef.selectedCount }) }}</span>
      <button class="btn btn-sm" @click="handleBatchMove([...fileTableRef.selectedItems])">[m] {{ t('common.move') }}</button>
      <button class="btn btn-sm" @click="handleBatchShare([...fileTableRef.selectedItems])">[<>] {{ t('common.share') }}</button>
      <button class="btn btn-sm" @click="handleBatchDelete([...fileTableRef.selectedItems]); fileTableRef.clearSelection()">[x] {{ t('common.delete') }}</button>
      <button class="btn btn-sm" @click="fileTableRef.clearSelection()">[X] {{ t('files.batchCancel') }}</button>
    </div>
    </Teleport>

    <Modal
      :visible="!!modalType && modalType !== 'move' && modalType !== 'batchmove'"
      :title="modalTitle"
      @close="modalType = ''"
    >
      <div class="modal-form">
        <label>
          <span>{{ t('common.name') }}</span>
          <input
            v-model="modalInput"
            @keyup.enter="confirmModal"
          />
        </label>
        <div class="modal-actions">
          <button class="btn btn-sm" @click="modalType = ''">{{ t('common.cancel') }}</button>
          <button class="btn btn-sm btn-primary" @click="confirmModal">{{ t('common.confirm') }}</button>
        </div>
      </div>
    </Modal>

    <Modal
      :visible="modalType === 'move' || modalType === 'batchmove'"
      :title="modalTitle"
      @close="modalType = ''"
    >
      <div class="modal-form">
        <div class="move-breadcrumb">
          <span class="mcrumb" @click="moveNavBreadcrumb({ id: 0, name: '~' })">~</span>
          <template v-for="(seg, i) in moveBreadcrumb" :key="i">
            <span class="msep">/</span>
            <span class="mcrumb" @click="moveNavBreadcrumb(seg)">{{ seg.name }}</span>
          </template>
        </div>
        <div class="move-folder-list">
          <div
            v-for="f in moveFolders"
            :key="f.id"
            class="move-folder"
            @click="moveNavigateTo(f)"
          >
            [DIR] {{ f.file_name }}
          </div>
          <div v-if="moveFolders.length === 0" class="move-empty">{{ t('common.empty') }}</div>
        </div>
        <div style="margin-top:var(--gap-3);font-size:var(--fs-xs);color:var(--text-dim)">
          {{ t('files.targetLabel') }} {{ moveBreadcrumb.length > 0 ? moveBreadcrumb.map(s=>s.name).join('/') : '~' }}
        </div>
        <div class="modal-actions">
          <button class="btn btn-sm" @click="modalType = ''">{{ t('common.cancel') }}</button>
          <button
            class="btn btn-sm btn-primary"
            :disabled="modalTarget && modalTarget.parent_id === moveParentId"
            @click="confirmMove"
          >{{ t('files.moveHere') }}</button>
        </div>
      </div>
    </Modal>

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
.files-view {
  display: flex;
  flex-direction: column;
  gap: var(--gap-4);
}
.toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--gap-4);
}
.toolbar-actions {
  display: flex;
  gap: var(--gap-2);
  flex-shrink: 0;
}
.modal-form {
  display: flex;
  flex-direction: column;
  gap: var(--gap-4);
}
.modal-form label {
  display: flex;
  flex-direction: column;
  gap: var(--gap-1);
}
.modal-form label span {
  font-size: var(--fs-xs);
  color: var(--text-dim);
  font-weight: 600;
  letter-spacing: 0.06em;
}
.modal-form input {
  padding: var(--gap-2) var(--gap-3);
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--gap-2);
}
.move-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--gap-1);
  font-size: var(--fs-xs);
  color: var(--text-dim);
  padding-bottom: var(--gap-2);
  margin-bottom: var(--gap-2);
  border-bottom: 1px solid var(--border-faint);
}
.msep { color: var(--text-muted); }
.mcrumb {
  cursor: pointer;
  padding: 1px var(--gap-1);
}
.mcrumb:hover { color: var(--accent); }
.move-folder-list {
  max-height: 200px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.move-folder {
  padding: var(--gap-2);
  cursor: pointer;
  font-size: var(--fs-sm);
  border: 1px solid transparent;
}
.move-folder:hover {
  background: var(--bg-hover);
  border-color: var(--border);
}
.move-empty {
  text-align: center;
  padding: var(--gap-4);
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.sticky-batch-bar {
  position: fixed;
  bottom: 0;
  left: 200px;
  right: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  gap: var(--gap-3);
  padding: var(--gap-3) var(--gap-6);
  background: var(--accent-dim);
  backdrop-filter: blur(4px);
  border-top: 1px solid var(--accent);
  font-size: var(--fs-xs);
}
.sticky-batch-bar .count {
  color: var(--accent);
  font-weight: 600;
}
.sticky-batch-bar .btn {
  border-color: var(--accent);
  color: var(--accent);
}
.sticky-batch-bar .btn-sm {
  padding-top: 3px;
  padding-bottom: 3px;
  line-height: 1.1;
}
</style>
