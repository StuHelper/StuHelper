<template>
  <div class="max-w-[600px] mx-auto p-6 animate-fade-in">
    <div class="flex items-center gap-3 mb-6 pb-4 border-b border-border-light">
      <router-link to="/notifications" class="text-text-muted hover:text-text-primary transition-colors duration-fast">
        <ArrowLeft :size="20" />
      </router-link>
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0">
        Notification preferences
      </h1>
    </div>

    <div class="rounded-md border border-border-light bg-bg-secondary p-4 text-sm text-text-muted leading-6">
      Notification preferences are not available in this build.
      Notifications are always delivered using the server-defined defaults.
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowLeft } from 'lucide-vue-next'
</script>
