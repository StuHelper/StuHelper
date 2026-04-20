<template>
  <div class="sh-dashboard">
    <DashboardMetrics :metrics="model.metrics" />

    <div class="sh-dashboard-grid">
      <section class="sh-section sh-dashboard-section">
        <header class="sh-section__head">
          <div class="sh-section__head-copy">
            <h2 class="sh-section__title">待处理事项</h2>
          </div>
        </header>
        <div class="sh-section__body sh-section__body--flush">
          <DashboardTodoTable :rows="model.todoRows" @open="openSection" />
        </div>
      </section>

      <div class="sh-dashboard-side">
        <section class="sh-section sh-dashboard-section">
          <header class="sh-section__head">
            <div class="sh-section__head-copy">
              <h2 class="sh-section__title">快捷入口</h2>
            </div>
          </header>
          <div class="sh-section__body">
            <div class="sh-shortcut-list">
              <button
                v-for="item in model.shortcuts"
                :key="item.label"
                type="button"
                class="sh-shortcut"
                @click="emit('open-target', item.target)"
              >
                <span class="sh-shortcut__label">{{ item.label }}</span>
                <span class="sh-shortcut__description">{{ item.description }}</span>
              </button>
            </div>
          </div>
        </section>

        <section class="sh-section sh-dashboard-section">
          <header class="sh-section__head">
            <div class="sh-section__head-copy">
              <h2 class="sh-section__title">策略摘要</h2>
            </div>
          </header>
          <div class="sh-section__body">
            <div class="sh-dashboard-policy">
              <div
                v-for="item in model.policySummary"
                :key="item.label"
                class="sh-dashboard-policy__row"
              >
                <span>{{ item.label }}</span>
                <strong class="sh-num">{{ item.value }}</strong>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>

    <section class="sh-section sh-dashboard-section">
      <header class="sh-section__head">
        <div class="sh-section__head-copy">
          <h2 class="sh-section__title">最近动态</h2>
        </div>
      </header>
      <div v-if="model.recentActivity.length > 0" class="sh-section__body sh-section__body--flush">
        <div class="sh-lane">
          <div v-for="item in model.recentActivity" :key="item.id" class="sh-lane__row">
            <span class="sh-lane__dot" :class="activityDotClass(item.kind)"></span>
            <div>
              <div class="sh-lane__title">{{ item.title }}</div>
              <div class="sh-lane__subtitle">{{ item.meta }}</div>
            </div>
            <span class="sh-lane__time">{{ item.time }}</span>
          </div>
        </div>
      </div>
      <div v-else class="sh-section__body">
        <div class="sh-dashboard-empty">当前没有最近动态。</div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import type { DashboardActivityRow, DashboardModel, DashboardTarget } from '../../dashboard/model'
import type { ConsoleSectionId } from '../../sections'
import DashboardMetrics from './DashboardMetrics.vue'
import DashboardTodoTable from './DashboardTodoTable.vue'

defineProps<{
  model: DashboardModel
}>()

const emit = defineEmits<{
  'open-target': [target: DashboardTarget]
}>()

function openSection(section: ConsoleSectionId) {
  emit('open-target', { section })
}

function activityDotClass(kind: DashboardActivityRow['kind']) {
  return kind === 'report' ? 'sh-lane__dot--warning' : 'sh-lane__dot--primary'
}
</script>
