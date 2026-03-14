<template>
  <div class="space-y-4">
    <!-- Header + toolbar -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
      <h1 class="text-xl font-bold text-gray-900">评课管理</h1>
      <el-button type="primary" plain @click="handleExport">
        <template #icon><el-icon><Download /></el-icon></template>
        导出数据
      </el-button>
    </div>

    <!-- Batch action bar -->
    <el-alert
      v-if="selectedIds.length > 0"
      :title="`已选择 ${selectedIds.length} 条`"
      type="info"
      :closable="false"
      class="!mb-0"
    >
      <template #default>
        <div class="flex items-center gap-2 mt-2">
          <el-button size="small" type="warning" @click="handleBatch('hide')">批量隐藏</el-button>
          <el-button size="small" type="success" @click="handleBatch('restore')">批量恢复</el-button>
          <el-button size="small" type="danger" @click="handleBatch('delete')">批量删除</el-button>
        </div>
      </template>
    </el-alert>

    <!-- Filters -->
    <div class="flex flex-wrap items-center gap-3">
      <el-radio-group v-model="status" @change="onStatusChange">
        <el-radio-button value="all">全部</el-radio-button>
        <el-radio-button value="published">已发布</el-radio-button>
        <el-radio-button value="hidden">已隐藏</el-radio-button>
      </el-radio-group>
      <el-input
        v-model="searchQuery"
        placeholder="搜索内容/课程/教师"
        clearable
        style="width: 240px"
        @input="onSearch"
      >
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
    </div>

    <!-- Table -->
    <el-table
      v-loading="loading"
      :data="filteredReviews"
      row-key="id"
      @selection-change="onSelectionChange"
    >
      <el-table-column type="selection" width="50" />
      <el-table-column type="expand">
        <template #default="{ row }">
          <div class="px-4 py-3 bg-gray-50 grid grid-cols-1 md:grid-cols-2 gap-3 text-sm">
            <div>
              <span class="text-gray-400 text-xs">标题：</span>
              <span>{{ row.title || '(无标题)' }}</span>
            </div>
            <div>
              <span class="text-gray-400 text-xs">学期：</span>
              <span>{{ row.termName || row.termID || '(未知)' }}</span>
            </div>
            <div>
              <span class="text-gray-400 text-xs">成绩：</span>
              <span>{{ row.grade || '(未填写)' }}</span>
            </div>
            <div>
              <span class="text-gray-400 text-xs">互动：</span>
              <span>点赞 {{ row.likeCount }} · 踩 {{ row.dislikeCount }} · 回复 {{ row.replyCount }}</span>
            </div>
            <div class="md:col-span-2">
              <span class="text-gray-400 text-xs">正文：</span>
              <p class="mt-1 p-2 bg-white border rounded text-gray-700 whitespace-pre-wrap break-words max-h-40 overflow-y-auto">{{ row.content }}</p>
            </div>
            <div v-if="row.moderationReason" class="md:col-span-2">
              <span class="text-gray-400 text-xs">屏蔽原因：</span>
              <span class="text-yellow-600">{{ row.moderationReason }}</span>
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="内容" min-width="200">
        <template #default="{ row }">
          <span class="text-gray-700 text-sm">{{ row.content.length > 80 ? row.content.slice(0, 80) + '...' : row.content }}</span>
        </template>
      </el-table-column>
      <el-table-column label="课程" prop="courseName" width="140" show-overflow-tooltip />
      <el-table-column label="教师" prop="teacherName" width="100" show-overflow-tooltip />
      <el-table-column label="评分" width="80">
        <template #default="{ row }">
          <span v-if="avgRating(row) !== null">
            <el-tag :type="ratingTagType(avgRating(row)!)" size="small">{{ avgRating(row)!.toFixed(1) }}</el-tag>
          </span>
          <span v-else class="text-gray-400 text-xs">-</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="时间" width="120">
        <template #default="{ row }">
          <span class="text-xs text-gray-400">{{ formatTime(row.createdAt) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100" fixed="right">
        <template #default="{ row }">
          <el-button
            v-if="row.status === 'published'"
            size="small"
            type="warning"
            link
            @click="handleUpdate(row.id, 'hide')"
          >隐藏</el-button>
          <el-button
            v-else
            size="small"
            type="success"
            link
            @click="handleUpdate(row.id, 'restore')"
          >恢复</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Pagination -->
    <div class="flex justify-end">
      <el-pagination
        v-if="total > 0"
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        background
        @current-change="fetchReviews"
        @size-change="onSizeChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Download, Search } from '@element-plus/icons-vue'
import { api } from '@/api'
import type { components } from '@/api'

type Review = components['schemas']['Review']

const loading = ref(false)
const reviews = ref<Review[]>([])
const selectedIds = ref<string[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const searchQuery = ref('')
const status = ref<'all' | 'published' | 'hidden'>('all')

const filteredReviews = computed(() => {
  if (!searchQuery.value.trim()) return reviews.value
  const q = searchQuery.value.toLowerCase()
  return reviews.value.filter(r =>
    r.content.toLowerCase().includes(q) ||
    (r.courseName && r.courseName.toLowerCase().includes(q)) ||
    (r.teacherName && r.teacherName.toLowerCase().includes(q))
  )
})

function avgRating(r: Review): number | null {
  const vals = Object.values(r.ratings || {})
  if (vals.length === 0) return null
  return vals.reduce((a, b) => a + b, 0) / vals.length
}

function ratingTagType(rating: number): 'success' | 'warning' | 'danger' {
  if (rating >= 4) return 'success'
  if (rating >= 3) return 'warning'
  return 'danger'
}

function statusTagType(s: string): 'success' | 'warning' | 'danger' | 'info' {
  if (s === 'published') return 'success'
  if (s === 'hidden') return 'warning'
  return 'info'
}

function statusLabel(s: string) {
  if (s === 'published') return '已发布'
  if (s === 'hidden') return '已隐藏'
  return s
}

function formatTime(dateStr: string) {
  return new Date(dateStr).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function onSelectionChange(rows: Review[]) {
  selectedIds.value = rows.map(r => r.id)
}

function onStatusChange() {
  page.value = 1
  fetchReviews()
}

let searchTimer: ReturnType<typeof setTimeout>
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    fetchReviews()
  }, 300)
}

function onSizeChange() {
  page.value = 1
  fetchReviews()
}

async function fetchReviews() {
  loading.value = true
  try {
    const res = await api.admin.getReviews({ status: status.value, page: page.value, pageSize: pageSize.value })
    reviews.value = res.data?.data?.list || []
    total.value = res.data?.data?.total || 0
  } catch {
    ElMessage.error('加载失败，请重试')
  } finally {
    loading.value = false
  }
}

async function handleUpdate(id: string, action: 'hide' | 'restore' | 'delete') {
  try {
    await api.admin.updateReview(id, { action })
    ElMessage.success('操作成功')
    fetchReviews()
  } catch {
    ElMessage.error('操作失败，请重试')
  }
}

async function handleBatch(action: 'hide' | 'restore' | 'delete') {
  if (selectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(`确定要对 ${selectedIds.value.length} 条评课执行${action === 'hide' ? '隐藏' : action === 'restore' ? '恢复' : '删除'}操作吗？`, '批量操作', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: action === 'delete' ? 'warning' : 'info',
    })
    await api.admin.batchUpdateReviews({ ids: selectedIds.value, action })
    ElMessage.success('批量操作成功')
    selectedIds.value = []
    fetchReviews()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error('操作失败，请重试')
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

onMounted(fetchReviews)
</script>
