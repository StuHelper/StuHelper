<template>
  <k-layout class="stuhelperGroupCenter-app">
    <AppShell :version="pkg.version" />
  </k-layout>
</template>

<script setup lang="ts">
import pkg from '../../package.json'
import AppShell from '../components/shell/AppShell.vue'
</script>

<style>
.stuhelperGroupCenter-app {
  background: var(--sh-surface-1);
  height: 100vh;
  min-height: 0;
  font-family: var(--sh-font-ui);
}

.stuhelperGroupCenter-app .layout-header {
  display: none !important;
}

.stuhelperGroupCenter-app .layout-container {
  padding: 0;
  height: 100%;
  overflow: hidden;
}

.stuhelperGroupCenter-app ::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

.stuhelperGroupCenter-app ::-webkit-scrollbar-track {
  background: transparent;
}

.stuhelperGroupCenter-app ::-webkit-scrollbar-thumb {
  background-color: var(--sh-border);
  border-radius: 3px;
}

.stuhelperGroupCenter-app ::-webkit-scrollbar-thumb:hover {
  background-color: var(--sh-fg-3);
}
</style>

