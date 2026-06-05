<template>
  <div class="sh-view sh-system">
    <WorkspaceHead
      title="系统 / 缓存"
      description="群组、用户、成员名称的本地缓存。仅影响显示名解析速度，不影响功能逻辑。"
      :chips="headerChips"
    >
      <template #actions>
        <el-button
          class="sh-button sh-button--ghost"
          :disabled="loading"
          @click="load"
        >
          {{ loading ? '加载中…' : '刷新统计' }}
        </el-button>
      </template>
    </WorkspaceHead>

    <ConsolePageSkeleton v-if="loading && !stats" />
    <EmptyState
      v-else-if="error && !stats"
      title="加载缓存统计失败"
      :body="error"
      tone="error"
    >
      <template #action>
        <el-button class="sh-button sh-button--ghost" @click="load">重试</el-button>
      </template>
    </EmptyState>

    <template v-else-if="stats">
      <div v-if="error" class="sh-load-error" role="alert">
        <div class="sh-load-error__body">
          <strong>刷新缓存统计失败</strong>
          <span>{{ error }}</span>
        </div>
        <el-button class="sh-button sh-button--ghost" @click="load">重试</el-button>
      </div>

      <section class="sh-dashboard-metrics sh-system__metrics">
        <article class="sh-stat sh-dashboard-metric sh-stat--primary">
          <span class="sh-stat__label">群组缓存</span>
          <span class="sh-stat__value sh-num">{{ stats.guilds }}</span>
          <span class="sh-stat__note">已记录的群信息条目数</span>
        </article>
        <article class="sh-stat sh-dashboard-metric">
          <span class="sh-stat__label">用户缓存</span>
          <span class="sh-stat__value sh-num">{{ stats.users }}</span>
          <span class="sh-stat__note">已记录的用户名称条目数</span>
        </article>
        <article class="sh-stat sh-dashboard-metric">
          <span class="sh-stat__label">成员缓存</span>
          <span class="sh-stat__value sh-num">{{ stats.members }}</span>
          <span class="sh-stat__note">群成员名称条目数</span>
        </article>
        <article class="sh-stat sh-dashboard-metric">
          <span class="sh-stat__label">最近刷新</span>
          <span class="sh-stat__value sh-mono sh-system__time">
            {{ stats.lastFullRefreshTime || '—' }}
          </span>
          <span class="sh-stat__note">7 天后自动失效</span>
        </article>
      </section>

      <WorkspaceSection
        title="缓存操作"
        description="强制刷新会从 Bot 重新拉取所有名称；清空只删除本地缓存条目。"
      >
        <div v-if="actionError" class="sh-load-error sh-system__action-error" role="alert">
          <div class="sh-load-error__body">
            <strong>{{ actionErrorTitle }}</strong>
            <span>{{ actionError }}</span>
          </div>
          <el-button class="sh-button sh-button--ghost" @click="clearActionError">关闭</el-button>
        </div>

        <div class="sh-btn-row">
          <el-button
            type="primary"
            class="sh-button sh-button--primary"
            :disabled="refreshing || clearing"
            @click="onRefresh"
          >
            {{ refreshing ? '刷新中…' : '强制刷新' }}
          </el-button>
          <el-button
            class="sh-button sh-button--danger"
            :disabled="refreshing || clearing"
            @click="onClear"
          >
            {{ clearing ? '清空中…' : '清空缓存' }}
          </el-button>
        </div>
        <ul class="sh-system__notes">
          <li>插件启动时会自动预热缓存。日常无需手动刷新。</li>
          <li>缓存失效后下一次访问该群 / 用户时会自动重拉。</li>
          <li>刷新所有缓存可能耗时数秒至数十秒，期间 UI 仍可用。</li>
        </ul>
      </WorkspaceSection>
    </template>

    <NoticeStack :items="notices" @dismiss="dismissNotice" />

    <ConfirmDialog
      :open="confirmState.open"
      :title="confirmState.title"
      :message="confirmState.message"
      :tone="confirmState.tone"
      :confirm-text="confirmState.confirmText"
      :cancel-text="confirmState.cancelText"
      @confirm="accept"
      @cancel="cancel"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

import { cacheApi, type CacheStats } from '../api'
import { useActionFeedback } from '../composables/use-action-feedback'
import { useConfirm } from '../composables/use-confirm'
import ConfirmDialog from './primitives/ConfirmDialog.vue'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import EmptyState from './primitives/EmptyState.vue'
import NoticeStack from './primitives/NoticeStack.vue'
import WorkspaceHead, { type WorkspaceHeadChip } from './primitives/WorkspaceHead.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'

const loading = ref(false)
const refreshing = ref(false)
const clearing = ref(false)
const stats = ref<CacheStats | null>(null)
const error = ref('')
let loadRequestSeq = 0

const { state: confirmState, confirm, accept, cancel } = useConfirm()
const {
  actionError,
  actionErrorTitle,
  notices,
  pushSuccess,
  setActionError,
  clearActionError,
  dismissNotice,
  errorMessage,
} = useActionFeedback()

const headerChips = computed<WorkspaceHeadChip[]>(() => {
  const chips: WorkspaceHeadChip[] = []
  if (stats.value) {
    const total = stats.value.guilds + stats.value.users + stats.value.members
    chips.push({ text: `${total} 条`, numeric: true })
  }
  if (stats.value?.lastFullRefreshTime) {
    chips.push({ text: `更新 · ${stats.value.lastFullRefreshTime}`, mono: true })
  }
  return chips
})

onMounted(load)

async function load(): Promise<void> {
  const requestSeq = ++loadRequestSeq
  loading.value = true
  error.value = ''
  clearActionError()
  try {
    const next = await cacheApi.stats()
    if (requestSeq !== loadRequestSeq) return
    stats.value = next
  } catch (cause) {
    if (requestSeq !== loadRequestSeq) return
    error.value = errorMessage(cause, '加载缓存统计失败')
  } finally {
    if (requestSeq === loadRequestSeq) {
      loading.value = false
    }
  }
}

async function onRefresh(): Promise<void> {
  const confirmed = await confirm({
    title: '强制刷新缓存',
    message: '确认从 Bot 重新拉取所有群组、用户、成员信息？这可能耗时数十秒。',
    confirmText: '开始刷新',
  })
  if (!confirmed) return

  refreshing.value = true
  clearActionError()
  try {
    const result = await cacheApi.refresh()
    stats.value = result.stats
    pushSuccess('缓存刷新完成')
  } catch (cause) {
    setActionError('刷新缓存失败', cause, '刷新缓存失败')
  } finally {
    refreshing.value = false
  }
}

async function onClear(): Promise<void> {
  const confirmed = await confirm({
    title: '清空缓存',
    message: '确认清空所有本地缓存？下次访问相关群 / 用户时会重新拉取。',
    tone: 'danger',
    confirmText: '清空',
  })
  if (!confirmed) return

  clearing.value = true
  clearActionError()
  try {
    await cacheApi.clear()
    pushSuccess('缓存已清空')
    await load()
  } catch (cause) {
    setActionError('清空缓存失败', cause, '清空缓存失败')
  } finally {
    clearing.value = false
  }
}

</script>

<style scoped>
.sh-system__metrics {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

@media (max-width: 1080px) {
  .sh-system__metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .sh-system__metrics {
    grid-template-columns: minmax(0, 1fr);
  }
}

.sh-system__time {
  font-size: var(--sh-t-body-lg);
}

.sh-system__notes {
  margin: var(--sh-s-3) 0 0;
  padding-left: 18px;
  font-size: var(--sh-t-meta);
  color: var(--sh-fg-2);
  line-height: var(--sh-l-loose);
}

.sh-system__action-error {
  margin-bottom: var(--sh-s-3);
}
</style>
