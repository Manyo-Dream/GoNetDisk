<script setup>
import { ref, computed, watch } from 'vue'
import { formatSize, formatDate } from '../utils/format.js'
import { useI18n } from '../i18n/index.js'

const { t } = useI18n()

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
          <th style="width:80px"></th>
          <th>{{ t('common.name') }}</th>
          <th style="width:100px">{{ t('common.size') }}</th>
          <th style="width:140px">{{ t('common.modified') }}</th>
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
                :title="t('action.download')"
                aria-label="Download file"
                @click.stop="emit('download', item)"
              >[v]</button>
              <button
                v-if="!item.is_dir"
                class="btn btn-icon btn-sm"
                :title="t('action.share')"
                aria-label="Share file"
                @click.stop="emit('share', item)"
              >[<>]</button>
              <button
                class="btn btn-icon btn-sm"
                :title="t('action.rename')"
                aria-label="Rename item"
                @click.stop="emit('rename', item)"
              >[r]</button>
              <button
                class="btn btn-icon btn-sm"
                :title="t('action.move')"
                aria-label="Move item"
                @click.stop="emit('move', item)"
              >[m]</button>
              <button
                class="btn btn-icon btn-sm btn-rm"
                :title="t('action.delete')"
                aria-label="Delete item"
                @click.stop="emit('remove', item)"
              >[x]</button>
            </span>
          </td>
        </tr>
      </tbody>
    </table>

    <div v-else-if="loading" class="empty-state">
      <span class="cursor-blink">_</span> {{ t('common.loading') }}
    </div>

    <div v-else class="empty-state">
      <span class="icon">[--]</span>
      <span>{{ t('files.emptyDirectory') }}</span>
      <span style="font-size:var(--fs-xs);color:var(--text-muted)">{{ t('files.dropHint') }}</span>
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
  width: 80px;
  white-space: nowrap;
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
