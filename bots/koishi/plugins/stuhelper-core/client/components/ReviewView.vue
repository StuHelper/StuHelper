<template>
  <div class="console-view">
    <section class="console-header">
      <div>
        <h2 class="console-header__title">处置中心</h2>
        <div class="console-header__meta">
          <span class="console-chip">统一工作项列表</span>
          <span class="console-chip" v-if="data">更新于 {{ formatTimestamp(data.generatedAt) }}</span>
        </div>
      </div>
      <div class="console-toolbar">
        <select v-model="kindFilter" class="console-select" @change="syncSelection()">
          <option value="">全部类型</option>
          <option value="review">复核</option>
          <option value="admission">准入</option>
          <option value="report">举报</option>
        </select>
        <input v-model.trim="keyword" class="console-input" type="text" placeholder="搜索成员、群号、原因" @input="syncSelection()" />
        <button class="console-button" @click="loadData">刷新</button>
      </div>
    </section>

    <div v-if="error" class="console-error">{{ error }}</div>
    <div v-else-if="loading" class="console-empty">正在加载处置工作项…</div>
    <template v-else-if="data">
      <section class="console-metrics">
        <article v-for="metric in metrics" :key="metric.label" class="console-metric">
          <div class="console-metric__label">{{ metric.label }}</div>
          <div class="console-metric__value">{{ metric.value }}</div>
          <div class="console-metric__note">{{ metric.note }}</div>
        </article>
      </section>

      <section class="console-split">
        <section class="console-panel">
          <div>
            <h3 class="console-panel__title">工作项列表</h3>
            <div class="console-panel__subtitle">复核、准入、举报统一展示。</div>
          </div>
          <div v-if="filteredItems.length === 0" class="console-empty">没有匹配的工作项。</div>
          <div v-else class="console-list">
            <button
              v-for="item in filteredItems"
              :key="item.id"
              type="button"
              class="console-list__item console-list__item--interactive"
              :class="{ 'console-list__item--active': selectedItem?.id === item.id }"
              @click="selectItem(item.id)"
            >
              <span class="console-list__title">{{ item.subjectLabel }}</span>
              <span class="console-list__meta">{{ REVIEW_KIND_LABELS[item.kind] }} · {{ item.status }} · {{ item.reason }}</span>
            </button>
          </div>
        </section>

        <section class="console-panel">
          <div>
            <h3 class="console-panel__title">详情与上下文</h3>
            <div class="console-panel__subtitle">优先接通 review 工作项的执行与驳回动作。</div>
          </div>
          <div v-if="!selectedItem" class="console-empty">请选择左侧工作项。</div>
          <div v-else class="console-stack">
            <div class="console-list__item">
              <span class="console-list__title">{{ selectedItem.subjectLabel }}</span>
              <span class="console-list__meta">
                {{ REVIEW_KIND_LABELS[selectedItem.kind] }} · {{ selectedItem.priority }} · {{ formatTimestamp(selectedItem.createdAt) }}
              </span>
            </div>
            <div class="console-list__item">
              <span class="console-list__title">处理原因</span>
              <span class="console-list__meta">{{ selectedItem.reason }}</span>
            </div>
            <div class="console-list__item">
              <span class="console-list__title">可用动作</span>
              <span class="console-list__meta">{{ selectedItem.availableActions.map((action) => REVIEW_ACTION_LABELS[action]).join('、') }}</span>
            </div>
            <div class="console-list__item">
              <span class="console-list__title">关联事件</span>
              <span class="console-list__meta">{{ relatedEvents.map((item) => item.summary).join(' / ') || '暂无' }}</span>
            </div>
            <div class="console-toolbar">
              <input v-model.trim="reviewNote" class="console-input" type="text" placeholder="可选处理备注" />
              <button
                v-for="action in selectedItem.availableActions"
                :key="action"
                class="console-button"
                :class="{ 'console-button--primary': action === selectedItem.availableActions[0] }"
                :disabled="actionLoading"
                @click="submitAction(action)"
              >
                {{ REVIEW_ACTION_LABELS[action] }}
              </button>
            </div>
          </div>
        </section>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

import type { ConsoleNavigationController } from '../composables/use-console-navigation'
import { consolePageApi } from '../page-api'
import type { ReviewPageData, ReviewWorkItem } from '../page-types'
import { formatTimestamp } from '../models/formatters'
import { buildReviewModel, REVIEW_ACTION_LABELS, REVIEW_KIND_LABELS } from '../models/review'

const props = defineProps<{
  navigation?: ConsoleNavigationController
}>()

const loading = ref(false)
const error = ref('')
const data = ref<ReviewPageData | null>(null)
const kindFilter = ref('')
const keyword = ref('')
const selectedItemId = ref('')
const reviewNote = ref('')
const actionLoading = ref(false)

const model = computed(() => {
  if (!data.value) {
    return null
  }
  return buildReviewModel(data.value, {
    workspace: kindFilter.value,
    keyword: keyword.value,
    itemId: selectedItemId.value,
  })
})
const metrics = computed(() => model.value?.metrics ?? [])
const filteredItems = computed(() => model.value?.filteredItems ?? [])
const selectedItem = computed(() => model.value?.selectedItem ?? null)
const relatedEvents = computed(() => model.value?.relatedEvents ?? [])

loadData()

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    data.value = await consolePageApi.review()
    kindFilter.value = props.navigation?.state.value.workspace || ''
    keyword.value = props.navigation?.state.value.keyword || ''
    selectedItemId.value = props.navigation?.state.value.itemId || data.value.items[0]?.id || ''
    syncSelection()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function selectItem(itemId: string) {
  selectedItemId.value = itemId
  syncSelection()
}

function syncSelection() {
  const item = filteredItems.value.find((entry) => entry.id === selectedItemId.value) ?? filteredItems.value[0] ?? null
  if (!item) {
    props.navigation?.replaceState({ workspace: kindFilter.value || null, guildId: null, memberId: null, itemId: null, keyword: keyword.value })
    return
  }
  selectedItemId.value = item.id
  props.navigation?.replaceState({
    workspace: item.kind,
    guildId: item.guildId,
    memberId: item.memberId,
    itemId: item.id,
    keyword: keyword.value,
  })
}

async function submitAction(action: ReviewWorkItem['availableActions'][number]) {
  if (!selectedItem.value) {
    return
  }
  actionLoading.value = true
  error.value = ''
  try {
    await consolePageApi.workItemAction({
      kind: selectedItem.value.kind,
      itemId: selectedItem.value.id,
      action,
      note: reviewNote.value || undefined,
    })
    reviewNote.value = ''
    await loadData()
    syncSelection()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    actionLoading.value = false
  }
}
</script>
