<script setup>
import { ref } from 'vue'
import { chunkApi, fileApi } from '../api/index.js'
import { formatSize, formatChunkProgress } from '../utils/format.js'
import SparkMD5 from 'spark-md5'

const CHUNK_SIZE = 10 * 1024 * 1024
const CONCURRENCY = 3

const emit = defineEmits(['uploaded'])
const dragging = ref(false)
const uploading = ref(false)
const tasks = ref([])

function computeMD5(file) {
  return new Promise((resolve) => {
    const chunkSize = 2 * 1024 * 1024
    const spark = new SparkMD5.ArrayBuffer()
    const reader = new FileReader()
    let offset = 0

    reader.onload = function (e) {
      spark.append(e.target.result)
      offset += chunkSize
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
      const blob = file.slice(offset, offset + chunkSize)
      reader.readAsArrayBuffer(blob)
    }

    readNext()
  })
}

function onDragOver(e) {
  e.preventDefault()
  dragging.value = true
}
function onDragLeave() {
  dragging.value = false
}
function onDrop(e) {
  e.preventDefault()
  dragging.value = false
  const files = Array.from(e.dataTransfer.files)
  if (files.length) startUpload(files)
}

function onFileSelect(e) {
  const files = Array.from(e.target.files)
  if (files.length) startUpload(files)
  e.target.value = ''
}

async function startUpload(files) {
  uploading.value = true
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
        await smallUpload(file, task)
      } else {
        await chunkUpload(file, task)
      }
      task.status = 'done'
    } catch (e) {
      task.status = 'error'
      task.error = e.message
    }
    task.progress = 100
  }
  uploading.value = false
  emit('uploaded')
}

async function smallUpload(file, task) {
  task.log.push('> uploading…')
  const fd = new FormData()
  fd.append('file', file)
  fd.append('parent_id', '0')
  await fileApi.upload(fd)
  task.log.push('< OK')
}

async function chunkUpload(file, task) {
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
    parent_id: 0,
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
</script>

<template>
  <div>
    <div
      class="dropzone"
      :class="{ dragging, uploading }"
      @dragover="onDragOver"
      @dragleave="onDragLeave"
      @drop="onDrop"
    >
      <input
        type="file"
        id="file-input"
        multiple
        @change="onFileSelect"
        style="display:none"
      />
      <label for="file-input" class="dropzone-inner">
        <span v-if="uploading">[..]  UPLOADING…</span>
        <span v-else-if="dragging">[<>]  DROP FILES HERE</span>
        <span v-else>[+]  DROP FILES OR CLICK TO UPLOAD</span>
      </label>
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
  </div>
</template>

<style scoped>
.dropzone {
  border: 1px dashed var(--border);
  transition: none;
  margin-bottom: var(--gap-4);
}
.dropzone.dragging {
  border-color: var(--accent);
  background: var(--accent-dim);
}
.dropzone.uploading {
  border-color: var(--cyan);
  opacity: 0.7;
}
.dropzone-inner {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--gap-6);
  cursor: pointer;
  font-size: var(--fs-sm);
  color: var(--text-dim);
  letter-spacing: 0.04em;
  user-select: none;
}
.dropzone.dragging .dropzone-inner {
  color: var(--accent);
}
.upload-log {
  display: flex;
  flex-direction: column;
  gap: var(--gap-3);
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
  max-height: 120px;
  overflow-y: auto;
}
.log-line {
  font-size: var(--fs-xs);
  color: var(--text-dim);
  line-height: 1.5;
  font-family: var(--font);
}
.log-error {
  color: var(--red);
  font-size: var(--fs-xs);
  margin-top: var(--gap-1);
}
</style>
