<script setup>
import { ref, computed, watch } from 'vue'
import { formatSize, formatDate } from '../utils/format.js'

const props = defineProps({
  files: { type: Array, default: () => [] },
  loading: Boolean,
})
const emit = defineEmits(['open-dir', 'download', 'remove', 'share', 'rename', 'move', 'batch-delete', 'batch-move', 'batch-share'])

const selectedSet = ref(new Set())
const selectAll = ref(false)

watch(() => props.files, () => {
  selectedSet.value = new Set()
  selectAll.value = false
})

const selectedCount = computed(() => selectedSet.value.size)
const selectedItems = computed(() => props.files.filter(f => selectedSet.value.has(f.id)))

function toggleSelect(id, item) {
  if (item.is_dir) return
  const next = new Set(selectedSet.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedSet.value = next
  selectAll.value = selectedSet.value.size === props.files.filter(f => !f.is_dir).length
}

function toggleSelectAll() {
  if (selectAll.value) {
    selectedSet.value = new Set()
    selectAll.value = false
  } else {
    selectedSet.value = new Set(props.files.filter(f => !f.is_dir).map(f => f.id))
    selectAll.value = true
  }
}

function clearSelection() {
  selectedSet.value = new Set()
  selectAll.value = false
}

defineExpose({ selectedItems, selectedCount, selectedSet, selectAll, clearSelection })

function rowClick(item) {
  if (item.is_dir) {
    emit('open-dir', item)
  }
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

function formatTime(d) {
  return formatDate(d).slice(5)
}
</script>

<template>
  <div class="table-wrap">
    <table v-if="!loading && files.length > 0">
      <thead>
        <tr>
          <th style="width:30px"><input type="checkbox" class="cb" :checked="selectAll" @click.stop="toggleSelectAll" /></th>
          <th style="width:56px"></th>
          <th>NAME</th>
          <th style="width:100px">SIZE</th>
          <th style="width:140px">MODIFIED</th>
          <th style="width:80px"></th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="item in files"
          :key="item.id"
          :class="{ clickable: item.is_dir }"
          @click="rowClick(item)"
        >
          <td class="td-cb">
            <input
              v-if="!item.is_dir"
              type="checkbox"
              class="cb"
              :checked="selectedSet.has(item.id)"
              @click.stop="toggleSelect(item.id, item)"
            />
          </td>
          <td class="td-icon" :class="fileIcon(item).cls">{{ fileIcon(item).label }}</td>
          <td class="td-name">
            <span class="name">{{ item.file_name }}</span>
          </td>
          <td class="td-size">{{ item.is_dir ? '--' : formatSize(item.file_size) }}</td>
          <td class="td-date">{{ formatTime(item.created_at || item.updated_at) }}</td>
          <td class="td-actions">
            <span class="actions">
              <button
                v-if="!item.is_dir"
                class="btn btn-icon btn-sm"
                title="DOWNLOAD"
                aria-label="Download file"
                @click.stop="emit('download', item)"
              >[v]</button>
              <button
                v-if="!item.is_dir"
                class="btn btn-icon btn-sm"
                title="SHARE"
                aria-label="Share file"
                @click.stop="emit('share', item)"
              >[<>]</button>
              <button
                class="btn btn-icon btn-sm"
                title="RENAME"
                aria-label="Rename item"
                @click.stop="emit('rename', item)"
              >[r]</button>
              <button
                class="btn btn-icon btn-sm"
                title="MOVE"
                aria-label="Move item"
                @click.stop="emit('move', item)"
              >[m]</button>
              <button
                class="btn btn-icon btn-sm btn-rm"
                title="DELETE"
                aria-label="Delete item"
                @click.stop="emit('remove', item)"
              >[x]</button>
            </span>
          </td>
        </tr>
      </tbody>
    </table>

    <div v-else-if="loading" class="empty-state">
      <span class="cursor-blink">_</span> LOADING…
    </div>

    <div v-else class="empty-state">
      <span class="icon">[--]</span>
      <span>EMPTY DIRECTORY</span>
      <span style="font-size:var(--fs-xs);color:var(--text-muted)">Drop files or create folders</span>
    </div>
  </div>
</template>

<style scoped>
.table-wrap {
  min-height: 200px;
}
.td-cb {
  width: 30px;
  text-align: center;
}
.td-icon {
  width: 56px;
  font-size: var(--fs-xs);
  font-weight: 600;
}
.td-name .name {
  color: var(--text);
  font-weight: 500;
}
td {
  font-size: var(--fs);
  padding: var(--gap-2) var(--gap-3);
}
tr.clickable {
  cursor: pointer;
}
tr.clickable:hover .td-name .name {
  color: var(--accent);
}
.td-size, .td-date {
  font-size: var(--fs-sm);
  color: var(--text-dim);
}
.td-size {
  font-variant-numeric: tabular-nums;
}
.td-actions {
  width: 80px;
}
.actions {
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.1s;
}
tr:hover .actions {
  opacity: 1;
}
.btn-rm:hover {
  color: var(--red) !important;
  border-color: var(--red-dim) !important;
}
</style>
