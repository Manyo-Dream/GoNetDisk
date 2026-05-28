<script setup>
defineProps({
  pathStack: { type: Array, default: () => [] },
})
const emit = defineEmits(['nav'])

const ROOT = { id: 0, name: '~' }
</script>

<template>
  <div class="breadcrumb">
    <span class="sep">/</span>
    <button class="crumb" @click="emit('nav', ROOT)">~</button>
    <template v-for="(seg, i) in pathStack" :key="i">
      <span class="sep">/</span>
      <button class="crumb" @click="emit('nav', seg)">{{ seg.name }}</button>
    </template>
  </div>
</template>

<style scoped>
.breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--gap-1);
  font-size: var(--fs-sm);
  color: var(--text-dim);
  padding-bottom: var(--gap-3);
  flex-wrap: wrap;
  user-select: none;
}
.sep {
  color: var(--text-muted);
}
.crumb {
  cursor: pointer;
  padding: 1px var(--gap-1);
}
.crumb:hover {
  color: var(--accent);
}
</style>
