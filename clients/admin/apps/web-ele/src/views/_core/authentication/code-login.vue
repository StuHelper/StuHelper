<script lang="ts" setup>
import { onMounted } from 'vue';

import { useAuthStore } from '#/store';

defineOptions({ name: 'CodeLogin' });

const authStore = useAuthStore();

onMounted(async () => {
  await authStore.redirectToLogin();
});
</script>

<template>
  <div class="flex h-full items-center justify-center">
    <p class="text-muted-foreground animate-pulse text-lg">
      Redirecting to login...
    </p>
  </div>
</template>
