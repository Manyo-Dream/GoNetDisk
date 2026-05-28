<script setup>
import { useRoute } from 'vue-router'

defineProps({
  navItems: Array,
})

const route = useRoute()

function isActive(item) {
  return route.path.startsWith(item.to)
}
</script>

<template>
  <nav class="sidebar">
    <div class="logo">
      <span class="logo-mark">[gnd]</span>
      <span class="logo-text">GoNetDisk</span>
    </div>
    <ul>
      <li
        v-for="item in navItems"
        :key="item.to"
        :class="{ active: isActive(item) }"
      >
        <router-link :to="item.to" class="nav-link">
          <span class="icon">{{ item.icon }}</span>
          <span>{{ item.label }}</span>
        </router-link>
      </li>
    </ul>
    <div class="sidebar-footer">
      <span class="ver">v1.0.0</span>
    </div>
  </nav>
</template>

<style scoped>
.sidebar {
  width: 200px;
  min-width: 200px;
  background: var(--bg-surface);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  user-select: none;
}
.logo {
  padding: var(--gap-4) var(--gap-4);
  border-bottom: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: var(--gap-1);
}
.logo-mark {
  font-size: var(--fs-xs);
  color: var(--accent);
  letter-spacing: 0.1em;
}
.logo-text {
  font-size: var(--fs-sm);
  font-weight: 600;
  color: var(--text-dim);
  letter-spacing: 0.05em;
}
ul {
  list-style: none;
  padding: var(--gap-2) 0;
  flex: 1;
}
li {
  border-left: 2px solid transparent;
}
li.active {
  border-left-color: var(--accent);
}
.nav-link {
  display: flex;
  align-items: center;
  gap: var(--gap-3);
  padding: var(--gap-2) var(--gap-4);
  cursor: pointer;
  font-size: var(--fs-sm);
  color: var(--text-dim);
  letter-spacing: 0.04em;
  font-weight: 600;
  text-decoration: none;
}
.nav-link:hover {
  background: var(--bg-hover);
  color: var(--text);
}
li.active .nav-link {
  color: var(--accent);
  background: var(--bg-hover);
}
li .icon {
  font-size: var(--fs-xs);
  width: 24px;
  text-align: center;
  color: var(--text-muted);
}
li.active .icon {
  color: var(--accent);
}
.sidebar-footer {
  padding: var(--gap-3) var(--gap-4);
  border-top: 1px solid var(--border);
}
.ver {
  font-size: var(--fs-xs);
  color: var(--text-muted);
}
</style>
