<template>
  <div class="space-y-4">
    <!-- Header + filter -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
      <h1 class="text-xl font-bold text-gray-900">举报管理</h1>
      <span class="text-sm text-gray-400">共 {{ total }} 条</span>
    </div>

    <!-- Status filter -->
    <el-radio-group v-model="status" @change="onStatusChange">
      <el-radio-button value="all">全部</el-radio-button>
      <el-radio-button value="pending">待处理</el-radio-button>
      <el-radio-button value="resolved">已处理</el-radio-button>
      <el-radio-button value="rejected">已驳回</el-radio-button>
    </el-radio-group>

    <!-- Table -->
    <el-table v-loading="loading" :data="reports" row-key="id">
      <el-table-column label="举报原因" width="120">
        <template #default="{ row }">
          <el-tag type="danger" size="small">{{ reasonLabel(row.reason) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="被举报内容" min-width="200">
        <template #default="{ row }">
          <div v-if="row.review" class="text-sm text-gray-600">
            <div class="font-medium text-xs text-gray-400 mb-0.5">{{ row.review.courseName }}<span v-if="row.review.teacherName"> · {{ row.review.teacherName }}</span></div>
            <span>{{ row.review.content.length > 100 ? row.review.content.slice(0, 100) + '...' : row.review.content }}</span>
          </div>
          <span v-else class="text-xs text-gray-400">无关联内容</span>
        </template>
      </el-table-column>
      <el-table-column label="描述" prop="description" min-width="120" show-overflow-tooltip />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="时间" width="130">
        <template #default="{ row }">
          <span class="text-xs text-gray-400">{{ formatTime(row.createdAt) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <template v-if="row.status === 'pending'">
            <el-button size="small" type="danger" @click="openProcess(row, 'hide')">隐藏评课</el-button>
            <el-button size="small" @click="openProcess(row, 'reject')">驳回</el-button>
          </template>
          <template v-else>
            <span class="text-xs text-gray-400">{{ row.resolvedBy ? `由 ${row.resolvedBy} 处理` : '已处理' }}</span>
          </template>
        </template>
      </el-table-column>
    </el-table>

    <!-- Pagination -->
    <div class="flex justify-end">
      <el-pagination
        v-if="total > 0"
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        background
        @current-change="fetchReports"
      />
    </div>

    <!-- Process dialog -->
    <el-dialog v-model="dialogVisible" :title="processAction === 'hide' ? '隐藏评课' : '驳回举报'" width="440px">
      <div class="space-y-3">
        <div v-if="processingReport?.review" class="p-3 bg-gray-50 rounded text-sm text-gray-600">
          <p class="font-medium text-xs text-gray-400 mb-1">{{ processingReport.review.courseName }}</p>
          <p>{{ processingReport.review.content.length > 120 ? processingReport.review.content.slice(0, 120) + '...' : processingReport.review.content }}</p>
        </div>
        <el-form label-position="top">
          <el-form-item label="备注（可选）">
            <el-input v-model="processNote" type="textarea" :rows="3" placeholder="请输入处理备注..." />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button
          :type="processAction === 'hide' ? 'danger' : 'default'"
          :loading="saving"
          @click="confirmProcess"
        >
          {{ processAction === 'hide' ? '确认隐藏' : '确认驳回' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api'
import type { components } from '@/api'

type ReviewReport = components['schemas']['ReviewReport']

const loading = ref(false)
const reports = ref<ReviewReport[]>([])
const page = ref(1)
const pageSize = 20
const total = ref(0)
const status = ref<'all' | 'pending' | 'resolved' | 'rejected'>('pending')

// Dialog state
const dialogVisible = ref(false)
const processingReport = ref<ReviewReport | null>(null)
const processAction = ref<'hide' | 'reject'>('hide')
const processNote = ref('')
const saving = ref(false)

function reasonLabel(reason: string) {
  const map: Record<string, string> = {
    spam: '垃圾信息',
    inappropriate: '不当内容',
    harassment: '骚扰',
    false_info: '虚假信息',
    other: '其他',
  }
  return map[reason] || reason
}

function statusTagType(s: string): 'success' | 'warning' | 'danger' | 'info' {
  if (s === 'pending') return 'warning'
  if (s === 'resolved') return 'success'
  return 'info'
}

function statusLabel(s: string) {
  if (s === 'pending') return '待处理'
  if (s === 'resolved') return '已处理'
  return '已驳回'
}

function formatTime(dateStr: string) {
  return new Date(dateStr).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function onStatusChange() {
  page.value = 1
  fetchReports()
}

function openProcess(report: ReviewReport, action: 'hide' | 'reject') {
  processingReport.value = report
  processAction.value = action
  processNote.value = ''
  dialogVisible.value = true
}

async function fetchReports() {
  loading.value = true
  try {
    const res = await api.admin.getReports({ status: status.value, page: page.value, pageSize })
    reports.value = res.data?.data?.list || []
    total.value = res.data?.data?.total || 0
  } catch {
    ElMessage.error('加载失败，请重试')
  } finally {
    loading.value = false
  }
}

async function confirmProcess() {
  if (!processingReport.value) return
  saving.value = true
  try {
    await api.admin.processReport(processingReport.value.id, {
      action: processAction.value,
      note: processNote.value || undefined,
    })
    ElMessage.success('处理成功')
    dialogVisible.value = false
    fetchReports()
  } catch {
    ElMessage.error('处理失败，请重试')
  } finally {
    saving.value = false
  }
}

onMounted(fetchReports)
</script>
