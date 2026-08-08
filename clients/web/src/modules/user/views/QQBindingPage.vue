<template>
  <QQBindingPanel @bound="finish" />
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'

import {
  isCrossOriginAccountFlowRedirect,
  readAccountFlowRedirect,
} from '@/utils/internalRedirect'

import QQBindingPanel from './QQBindingPanel.vue'

const route = useRoute()
const router = useRouter()

function finish(): void {
  const target = readAccountFlowRedirect(route.query.redirect)
  if (isCrossOriginAccountFlowRedirect(target)) {
    window.location.assign(target)
    return
  }
  void router.push(target)
}
</script>
