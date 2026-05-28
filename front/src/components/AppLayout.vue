<script setup>
import Sidebar from './Sidebar.vue'
import Topbar from './Topbar.vue'

const props = defineProps({
  userInfo: Object,
  navItems: Array,
})
const emit = defineEmits(['logout'])
</script>

<template>
  <div class="shell">
    <Topbar :user-info="userInfo" @logout="emit('logout')" />
    <div class="main">
      <Sidebar :nav-items="navItems" />
      <div class="content">
        <slot />
      </div>
    </div>
  </div>
</template>

<style scoped>
.shell {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.main {
  display: flex;
  flex: 1;
  overflow: hidden;
}
.content {
  flex: 1;
  overflow-y: auto;
  padding: var(--gap-6);
  background-color: var(--bg-root);
}
</style>
