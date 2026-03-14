<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-bold text-gray-900">仪表盘</h1>
      <el-button :loading="loading" @click="fetchData">
        <template #icon><el-icon><Refresh /></el-icon></template>
        刷新
      </el-button>
    </div>

    <!-- Stats cards -->
    <el-row :gutter="16" v-loading="loading">
      <el-col :xs="12" :sm="12" :md="6" v-for="card in statCards" :key="card.key" class="mb-4">
        <el-card shadow="never" class="h-full">
          <el-statistic :title="card.label" :value="card.value">
            <template #prefix>
              <el-icon :color="card.color"><component :is="card.icon" /></el-icon>
            </template>
          </el-statistic>
        </el-card>
      </el-col>
    </el-row>

    <!-- Quick actions -->
    <div>
      <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">快捷操作</h2>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="8" class="mb-3">
          <el-card shadow="hover" class="cursor-pointer" @click="$router.push({ name: 'admin-reviews' })">
            <div class="flex items-center gap-3">
              <el-icon :size="20" color="#409eff"><ChatDotRound /></el-icon>
              <span class="text-sm font-medium">评课管理</span>
            </div>
          </el-card>
        </el-col>
        <el-col :xs="24" :sm="8" class="mb-3">
          <el-card shadow="hover" class="cursor-pointer relative" @click="$router.push({ name: 'admin-reports' })">
            <div class="flex items-center gap-3">
              <el-icon :size="20" color="#f56c6c"><Flag /></el-icon>
              <span class="text-sm font-medium">举报处理</span>
              <el-badge v-if="stats.pendingReports > 0" :value="stats.pendingReports" class="ml-auto" type="danger" />
            </div>
          </el-card>
        </el-col>
        <el-col :xs="24" :sm="8" class="mb-3">
          <el-card shadow="hover" class="cursor-pointer" @click="handleExport">
            <div class="flex items-center gap-3">
              <el-icon :size="20" color="#67c23a"><Download /></el-icon>
              <span class="text-sm font-medium">导出数据</span>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <!-- Recent logs -->
    <div>
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wider">近期操作日志</h2>
        <el-link type="primary" @click="$router.push({ name: 'admin-logs' })">查看全部</el-link>
      </div>
      <el-card shadow="never">
        <el-table :data="recentLogs" size="small" :show-header="false">
          <el-table-column width="80">
            <template #default="{ row }">
              <el-tag :type="actionTagType(row.action)" size="small">{{ row.action }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column>
            <template #default="{ row }">
              <span class="font-medium text-gray-700">{{ row.adminUsername }}</span>
              <span class="text-gray-400 ml-2 text-xs">{{ row.resourceType }} #{{ row.resourceID.slice(0, 8) }}</span>
            </template>
          </el-table-column>
          <el-table-column width="160" align="right">
            <template #default="{ row }">
              <span class="text-xs text-gray-400">{{ formatTime(row.createdAt) }}</span>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="recentLogs.length === 0 && !loading" description="暂无日志" :image-size="60" />
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Refresh, Download, Flag, ChatDotRound,
  TrendCharts, SuccessFilled, Warning, CircleClose,
} from '@element-plus/icons-vue'
import { api } from '@/api'
import type { components } from '@/api'

type AdminStats = components['schemas']['AdminStats']
type AdminOperationLog = components['schemas']['AdminOperationLog']

const loading = ref(false)
const stats = ref<AdminStats>({
  totalReviews: 0,
  publishedReviews: 0,
  hiddenReviews: 0,
  deletedReviews: 0,
  pendingReports: 0,
  totalReports: 0,
  todayReviews: 0,
  weekReviews: 0,
})
const recentLogs = ref<AdminOperationLog[]>([])

const statCards = computed(() => [
  { key: 'total', label: '全部评课', value: stats.value.totalReviews, icon: TrendCharts, color: '#409eff' },
  { key: 'published', label: '已发布', value: stats.value.publishedReviews, icon: SuccessFilled, color: '#67c23a' },
  { key: 'hidden', label: '已隐藏', value: stats.value.hiddenReviews, icon: Warning, color: '#e6a23c' },
  { key: 'deleted', label: '已删除', value: stats.value.deletedReviews, icon: CircleClose, color: '#f56c6c' },
  { key: 'pending', label: '待处理举报', value: stats.value.pendingReports, icon: Flag, color: stats.value.pendingReports > 0 ? '#f56c6c' : '#909399' },
  { key: 'reports', label: '全部举报', value: stats.value.totalReports, icon: Flag, color: '#909399' },
  { key: 'today', label: '今日新增', value: stats.value.todayReviews, icon: TrendCharts, color: '#409eff' },
  { key: 'week', label: '本周新增', value: stats.value.weekReviews, icon: TrendCharts, color: '#9b59b6' },
])

function actionTagType(action: string): '' | 'success' | 'warning' | 'danger' | 'info' {
  const map: Record<string, '' | 'success' | 'warning' | 'danger' | 'info'> = {
    hide: 'warning',
    restore: 'success',
    delete: 'danger',
    edit: 'info',
  }
  return map[action] || ''
}

function formatTime(dateStr: string) {
  return new Date(dateStr).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

async function fetchData() {
  loading.value = true
  try {
    const [statsRes, logsRes] = await Promise.all([
      api.admin.getStats(),
      api.admin.getLogs({ page: 1, pageSize: 5 }),
    ])
    if (statsRes.data?.data) {
      stats.value = statsRes.data.data
    }
    recentLogs.value = logsRes.data?.data?.list || []
  } catch {
    ElMessage.error('加载失败，请重试')
  } finally {
    loading.value = false
  }
}

async function handleExport() {
  try {
    const res = await api.admin.exportReviews()
    if (!res.data) return
    const url = URL.createObjectURL(new Blob([res.data as unknown as BlobPart]))
    const a = document.createElement('a')
    a.href = url
    a.download = `reviews-export-${new Date().toISOString().slice(0, 10)}.ndjson`
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('导出成功')
  } catch {
    ElMessage.error('导出失败，请重试')
  }
}

onMounted(fetchData)
</script>
