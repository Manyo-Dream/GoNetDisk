<script setup>
import { ref, watch } from 'vue'
import { chunkApi } from '../api/index.js'
import { formatSize } from '../utils/format.js'
import { useI18n } from '../i18n/index.js'
import Modal from './Modal.vue'
import SparkMD5 from 'spark-md5'

const CHUNK_SIZE = 10 * 1024 * 1024
const CONCURRENCY = 3

const { t } = useI18n()

const props = defineProps({
  visible: Boolean,
  currentParentId: { type: Number, default: 0 },
})

const emit = defineEmits(['close', 'uploaded'])

const phase = ref('idle')
const task = ref(null)
const error = ref('')
const fileInput = ref(null)

watch(() => props.visible, (v) => {
  if (v) {
    phase.value = 'idle'
    task.value = null
    error.value = ''
  }
})

function onFileSelect(e) {
  const file = e.target.files[0]
  if (file) startUpload(file)
  e.target.value = ''
}

function onDrop(e) {
  e.preventDefault()
  const file = e.dataTransfer.files[0]
  if (file) startUpload(file)
}

function onDragOver(e) {
  e.preventDefault()
  e.currentTarget.classList.add('dragging')
}
function onDragLeave(e) {
  e.currentTarget.classList.remove('dragging')
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
      reader.readAsArrayBuffer(file.slice(offset, offset + 2 * 1024 * 1024))
    }
    readNext()
  })
}

async function startUpload(file) {
  task.value = {
    name: file.name,
    size: file.size,
    progress: 0,
    chunkCount: 0,
    completed: 0,
    skipped: 0,
    uploadId: '',
  }
  error.value = ''
  phase.value = 'hashing'

  const hash = await computeMD5(file)
  if (!hash) {
    error.value = t('error.hashFailed')
    phase.value = 'error'
    return
  }

  try {
    const initRes = await chunkApi.init({
      file_name: file.name,
      file_size: file.size,
      file_hash: hash,
      parent_id: props.currentParentId,
      chunk_size: CHUNK_SIZE,
    })

    if (initRes.instant_upload) {
      phase.value = 'done'
      task.value.progress = 100
      emit('uploaded')
      return
    }

    const { upload_id, chunk_count } = initRes
    task.value.chunkCount = chunk_count
    task.value.uploadId = upload_id
    phase.value = 'uploading'

    let uploadedSet = new Set()
    try {
      const status = await chunkApi.status(upload_id)
      if (status && status.uploaded_chunks) {
        uploadedSet = new Set(status.uploaded_chunks)
      }
    } catch (e) {
      /* status unavailable, start fresh */
    }
    task.value.skipped = uploadedSet.size
    task.value.completed = uploadedSet.size
    task.value.progress = chunk_count > 0 ? Math.round((uploadedSet.size / chunk_count) * 100) : 0

    const chunkQueue = []
    for (let i = 0; i < chunk_count; i++) {
      if (uploadedSet.has(i + 1)) continue
      const start = i * CHUNK_SIZE
      const end = Math.min(start + CHUNK_SIZE, file.size)
      chunkQueue.push({ index: i + 1, blob: file.slice(start, end) })
    }

    if (chunkQueue.length === 0) {
      task.value.completed = chunk_count
      task.value.progress = 100
    } else {
      async function uploadOne() {
        while (chunkQueue.length > 0) {
          const chunk = chunkQueue.shift()
          const fd = new FormData()
          fd.append('upload_id', upload_id)
          fd.append('chunk_index', String(chunk.index))
          fd.append('chunk_hash', '')
          fd.append('chunk', chunk.blob, 'blob')

          await chunkApi.upload(fd)
          task.value.completed++
          task.value.progress = Math.round((task.value.completed / chunk_count) * 95)
        }
      }

      const workers = Array(Math.min(CONCURRENCY, chunkQueue.length)).fill(null).map(() => uploadOne())
      await Promise.all(workers)
    }

    await chunkApi.complete({ upload_id })
    phase.value = 'done'
    task.value.progress = 100
    emit('uploaded')
  } catch (e) {
    error.value = e.message || t('error.uploadFailed')
    phase.value = 'error'
  }
}
</script>

<template>
  <Modal :visible="visible" :title="t('chunk.title')" @close="emit('close')">
    <div class="chunk-modal">
      <div
        class="dropzone"
        :class="{ dragging: false, disabled: phase === 'uploading' || phase === 'hashing' }"
        @dragover="onDragOver"
        @dragleave="onDragLeave"
        @drop="onDrop"
      >
        <input
          ref="fileInput"
          type="file"
          :disabled="phase === 'uploading' || phase === 'hashing'"
          @change="onFileSelect"
        />
        <div class="dropzone-inner">
          <span v-if="phase === 'hashing'">[..] {{ t('chunk.computingHash') }}</span>
          <span v-else-if="phase === 'uploading'">[..] {{ t('chunk.uploading') }}</span>
          <span v-else>[+] {{ t('chunk.selectFile') }}</span>
        </div>
      </div>

      <div v-if="task" class="task-panel">
        <div class="task-header">
          <span class="task-name">{{ task.name }}</span>
          <span class="task-size">{{ formatSize(task.size) }}</span>
          <span
            class="badge"
            :class="phase === 'done' ? 'badge-green' : phase === 'error' ? 'badge-red' : 'badge-accent'"
          >{{ t('chunk.' + phase) || phase.toUpperCase() }}</span>
        </div>
        <div class="progress-bar">
          <div class="fill" :style="{ width: task.progress + '%' }"></div>
        </div>
        <div v-if="phase === 'uploading'" class="task-stats">
          <span>{{ t('chunk.chunks') }}: {{ task.completed }} / {{ task.chunkCount }}</span>
          <span v-if="task.skipped > 0">({{ task.skipped }} {{ t('chunk.skipped') }})</span>
        </div>
      </div>

      <div v-if="error" class="error-msg">! {{ error }}</div>
    </div>
  </Modal>
</template>

<style scoped>
.chunk-modal {
  display: flex;
  flex-direction: column;
  gap: var(--gap-4);
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
.task-panel {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  padding: var(--gap-3);
}
.task-header {
  display: flex;
  align-items: center;
  gap: var(--gap-3);
  margin-bottom: var(--gap-2);
}
.task-name {
  flex: 1;
  font-size: var(--fs-sm);
  font-weight: 500;
}
.task-size {
  font-size: var(--fs-xs);
  color: var(--text-dim);
}
.task-stats {
  margin-top: var(--gap-2);
  font-size: var(--fs-xs);
  color: var(--text-dim);
}
.error-msg {
  color: var(--red);
  font-size: var(--fs-xs);
  padding: var(--gap-2);
  border: 1px solid var(--red-dim);
  background: var(--red-dim);
}
</style>
