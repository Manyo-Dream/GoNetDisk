<script setup>
import { ref, watch } from 'vue'
import { chunkApi, fileApi } from '../api/index.js'
import { formatSize, formatChunkProgress } from '../utils/format.js'
import Modal from './Modal.vue'
import SparkMD5 from 'spark-md5'

const CHUNK_SIZE = 10 * 1024 * 1024
const CONCURRENCY = 3

const props = defineProps({
  visible: Boolean,
  currentPath: { type: Array, default: () => [{ id: 0, name: '~' }] },
  currentParentId: { type: Number, default: 0 },
})

const emit = defineEmits(['close', 'uploaded'])

const targetParentId = ref(0)
const targetBreadcrumb = ref([{ id: 0, name: '~' }])
const targetFolders = ref([])

const uploading = ref(false)
const tasks = ref([])
const fileInput = ref(null)

watch(() => props.visible, (v) => {
  if (v) {
    targetParentId.value = props.currentParentId
    targetBreadcrumb.value = props.currentPath.map(s => ({ ...s }))
    tasks.value = []
    uploading.value = false
    fetchTargetFolders()
  }
})

async function fetchTargetFolders() {
  try {
    const res = await fileApi.list({
      parent_id: targetParentId.value,
      page: 1,
      page_size: 100,
    })
    targetFolders.value = (res.list || []).filter(f => f.is_dir)
  } catch (e) { /* ignore */ }
}

function folderNavigateTo(item) {
  targetParentId.value = item.id
  targetBreadcrumb.value.push({ id: item.id, name: item.file_name })
  fetchTargetFolders()
}

function folderNavBreadcrumb(seg) {
  const idx = targetBreadcrumb.value.findIndex(s => s.id === seg.id)
  if (idx === -1) {
    targetParentId.value = 0
    targetBreadcrumb.value = [{ id: 0, name: '~' }]
  } else {
    targetParentId.value = seg.id
    targetBreadcrumb.value = targetBreadcrumb.value.slice(0, idx + 1)
  }
  fetchTargetFolders()
}

function onFileSelect(e) {
  const files = Array.from(e.target.files)
  if (files.length) startUpload(files)
  e.target.value = ''
}

function onDrop(e) {
  e.preventDefault()
  const files = Array.from(e.dataTransfer.files)
  if (files.length) startUpload(files)
}

function onDragOver(e) {
  e.preventDefault()
  e.currentTarget.classList.add('dragging')
}

function onDragLeave(e) {
  e.currentTarget.classList.remove('dragging')
}

async function startUpload(files) {
  uploading.value = true
  const parentId = targetParentId.value
  for (const file of files) {
    const task = {
      name: file.name,
      size: file.size,
      progress: 0,
      status: 'init',
      log: [],
      error: '',
    }
    tasks.value.push(task)

    try {
      if (file.size < CHUNK_SIZE) {
        await smallUpload(file, task, parentId)
      } else {
        await chunkUpload(file, task, parentId)
      }
      task.status = 'done'
    } catch (e) {
      task.status = 'error'
      task.error = e.message || 'Upload failed'
    }
    task.progress = 100
  }
  uploading.value = false
  emit('uploaded')
}

async function smallUpload(file, task, parentId) {
  task.log.push('> uploading…')
  const fd = new FormData()
  fd.append('file', file)
  fd.append('parent_id', String(parentId))
  await fileApi.upload(fd)
  task.log.push('< OK')
}

function computeMD5(file) {
  return new Promise((resolve) => {
    const spark = new SparkMD5.ArrayBuffer()
    const reader = new FileReader()
    let offset = 0

    reader.onload = function (e) {
      spark.append(e.target.result)
      offset += 2 * 1024 * 1024
      if (offset < file.size) {
        readNext()
      } else {
        resolve(spark.end())
      }
    }

    reader.onerror = function () {
      resolve('')
    }

    function readNext() {
      const blob = file.slice(offset, offset + 2 * 1024 * 1024)
      reader.readAsArrayBuffer(blob)
    }

    readNext()
  })
}

async function chunkUpload(file, task, parentId) {
  task.log.push('> computing hash…')
  const hash = await computeMD5(file)
  if (!hash) {
    throw new Error('Hash computation failed')
  }
  task.log.push(`  hash: ${hash.slice(0, 16)}…`)

  task.log.push('> init chunk upload…')
  const initRes = await chunkApi.init({
    file_name: file.name,
    file_size: file.size,
    file_hash: hash,
    parent_id: parentId,
    chunk_size: CHUNK_SIZE,
  })

  if (initRes.instant_upload) {
    task.log.push('< instant upload OK (dedup)')
    return
  }

  const { upload_id, chunk_count } = initRes
  task.log.push(`  chunks: ${chunk_count}`)

  const chunkQueue = []
  for (let i = 0; i < chunk_count; i++) {
    const start = i * CHUNK_SIZE
    const end = Math.min(start + CHUNK_SIZE, file.size)
    chunkQueue.push({ index: i + 1, blob: file.slice(start, end) })
  }

  let completed = 0

  async function uploadOne() {
    while (chunkQueue.length > 0) {
      const chunk = chunkQueue.shift()
      const fd = new FormData()
      fd.append('upload_id', upload_id)
      fd.append('chunk_index', String(chunk.index))
      fd.append('chunk_hash', '')
      fd.append('chunk', chunk.blob, 'blob')

      await chunkApi.upload(fd)
      completed++
      task.progress = Math.round((completed / chunk_count) * 90)
      task.log.push(`  ${formatChunkProgress(completed, chunk_count)}`)
    }
  }

  const workers = Array(Math.min(CONCURRENCY, chunkQueue.length))
    .fill(null)
    .map(() => uploadOne())
  await Promise.all(workers)

  task.log.push('> completing…')
  await chunkApi.complete({ upload_id })
  task.log.push('< DONE')
}

function targetPath() {
  return targetBreadcrumb.value.length > 0
    ? targetBreadcrumb.value.map(s => s.name).join('/')
    : '~'
}
</script>

<template>
  <Modal :visible="visible" title="[^] UPLOAD FILES" @close="emit('close')">
    <div class="upload-modal">
      <div class="section">
        <div class="section-label">TARGET FOLDER</div>
        <div class="folder-breadcrumb">
          <span
            class="fcrumb"
            :class="{ active: targetParentId === 0 }"
            @click="folderNavBreadcrumb({ id: 0, name: '~' })"
          >~</span>
          <template v-for="seg in targetBreadcrumb.slice(1)" :key="seg.id">
            <span class="fsep">/</span>
            <span
              class="fcrumb"
              :class="{ active: targetParentId === seg.id }"
              @click="folderNavBreadcrumb(seg)"
            >{{ seg.name }}</span>
          </template>
        </div>
        <div class="folder-list">
          <div
            v-for="f in targetFolders"
            :key="f.id"
            class="folder-item"
            @click="folderNavigateTo(f)"
          >
            [DIR] {{ f.file_name }}
          </div>
          <div v-if="targetFolders.length === 0" class="folder-empty">(empty)</div>
        </div>
      </div>

      <div class="section">
        <div class="section-label">SELECT FILES</div>
        <div
          class="dropzone"
          :class="{ disabled: uploading }"
          @dragover="onDragOver"
          @dragleave="onDragLeave"
          @drop="onDrop"
        >
          <input
            ref="fileInput"
            type="file"
            multiple
            :disabled="uploading"
            @change="onFileSelect"
          />
          <div class="dropzone-inner">
            <span v-if="uploading">[..] UPLOADING…</span>
            <span v-else>[+] DROP FILES HERE OR CLICK TO SELECT</span>
          </div>
        </div>
      </div>

      <div v-if="tasks.length > 0" class="upload-log">
        <div v-for="(task, i) in tasks" :key="i" class="log-entry">
          <div class="log-header">
            <span class="log-name">{{ task.name }}</span>
            <span class="log-size">{{ formatSize(task.size) }}</span>
            <span
              class="badge"
              :class="task.status === 'done' ? 'badge-green' : task.status === 'error' ? 'badge-red' : 'badge-accent'"
            >{{ task.status.toUpperCase() }}</span>
          </div>
          <div class="progress-bar">
            <div class="fill" :style="{ width: task.progress + '%' }"></div>
          </div>
          <div class="log-lines" v-if="task.log.length">
            <div v-for="(line, j) in task.log" :key="j" class="log-line">{{ line }}</div>
          </div>
          <div v-if="task.error" class="log-error">! {{ task.error }}</div>
        </div>
      </div>

      <div class="modal-actions">
        <div class="actions-left">
          <span class="target-hint">TO: {{ targetPath() }}</span>
        </div>
        <div class="actions-right">
          <button class="btn btn-sm" @click="emit('close')">CLOSE</button>
        </div>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.upload-modal {
  display: flex;
  flex-direction: column;
  gap: var(--gap-4);
}
.section {
  display: flex;
  flex-direction: column;
  gap: var(--gap-3);
}
.section-label {
  font-size: var(--fs-xs);
  color: var(--text-dim);
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}
.folder-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--gap-1);
  font-size: var(--fs-xs);
  color: var(--text-dim);
  padding-bottom: var(--gap-2);
  border-bottom: 1px solid var(--border-faint);
}
.fsep { color: var(--text-muted); }
.fcrumb {
  cursor: pointer;
  padding: 1px var(--gap-1);
}
.fcrumb:hover { color: var(--accent); }
.fcrumb.active { color: var(--accent); }
.folder-list {
  max-height: 140px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.folder-item {
  padding: var(--gap-1) var(--gap-2);
  cursor: pointer;
  font-size: var(--fs-sm);
  border: 1px solid transparent;
}
.folder-item:hover {
  background: var(--bg-hover);
  border-color: var(--border);
}
.folder-empty {
  text-align: center;
  padding: var(--gap-2);
  color: var(--text-muted);
  font-size: var(--fs-xs);
}
.dropzone {
  position: relative;
  border: 1px dashed var(--border);
}
.dropzone.disabled {
  opacity: 0.5;
  pointer-events: none;
}
.dropzone.dragging {
  border-color: var(--accent);
  background: var(--accent-dim);
}
.dropzone input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}
.dropzone-inner {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--gap-6);
  font-size: var(--fs-sm);
  color: var(--text-dim);
  letter-spacing: 0.04em;
  user-select: none;
  pointer-events: none;
}
.dropzone.dragging .dropzone-inner {
  color: var(--accent);
}
.upload-log {
  display: flex;
  flex-direction: column;
  gap: var(--gap-3);
  max-height: 240px;
  overflow-y: auto;
}
.log-entry {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  padding: var(--gap-3);
}
.log-header {
  display: flex;
  align-items: center;
  gap: var(--gap-3);
  margin-bottom: var(--gap-2);
}
.log-name {
  flex: 1;
  font-size: var(--fs-sm);
  font-weight: 500;
}
.log-size {
  font-size: var(--fs-xs);
  color: var(--text-dim);
}
.log-lines {
  margin-top: var(--gap-2);
  max-height: 100px;
  overflow-y: auto;
}
.log-line {
  font-size: var(--fs-xs);
  color: var(--text-dim);
  line-height: 1.5;
}
.log-error {
  color: var(--red);
  font-size: var(--fs-xs);
  margin-top: var(--gap-1);
}
.modal-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.actions-left {
  font-size: var(--fs-xs);
  color: var(--text-muted);
}
.actions-right {
  display: flex;
  gap: var(--gap-2);
}
.target-hint {
  font-size: var(--fs-xs);
  color: var(--text-dim);
}
</style>
