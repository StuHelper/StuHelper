<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h2 class="text-lg font-semibold text-gray-900">学生认证审核</h2>
    </div>

    <!-- Filters -->
    <div class="mb-4 flex items-center gap-4 flex-wrap">
      <el-radio-group v-model="filterStatus" @change="handleFilterChange">
        <el-radio-button value="">全部</el-radio-button>
        <el-radio-button value="pending">待审核</el-radio-button>
        <el-radio-button value="verified">已认证</el-radio-button>
        <el-radio-button value="rejected">已拒绝</el-radio-button>
      </el-radio-group>
      <el-input
        v-model="filterSchoolId"
        placeholder="按学校ID筛选"
        clearable
        style="width: 200px"
        @change="handleFilterChange"
        @clear="handleFilterChange"
      />
    </div>

    <!-- Table -->
    <el-table v-loading="loading" :data="list" border stripe>
      <el-table-column prop="userID" label="用户ID" width="100" />
      <el-table-column prop="schoolID" label="学校" width="120" />
      <el-table-column label="学号列表" min-width="160">
        <template #default="{ row }">
          <el-tag
            v-for="sid in row.studentIDs"
            :key="sid"
            size="small"
            class="mr-1 mb-1"
          >
            {{ sid }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="activeStudentID" label="当前学号" width="130" />
      <el-table-column label="认证状态" width="110">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.verificationStatus)" size="small">
            {{ statusLabel(row.verificationStatus) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="认证方式" width="110">
        <template #default="{ row }">
          {{ verificationMethodLabel(row.verificationMethod) }}
        </template>
      </el-table-column>
      <el-table-column prop="phone" label="手机号" width="130" />
      <el-table-column label="提交时间" width="180">
        <template #default="{ row }">
          {{ formatDate(row.createdAt) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <template v-if="row.verificationStatus === 'pending'">
            <el-button type="success" size="small" @click="handleApprove(row)">通过</el-button>
            <el-button type="danger" size="small" @click="openRejectDialog(row)">拒绝</el-button>
          </template>
          <span v-else class="text-gray-400 text-sm">—</span>
        </template>
      </el-table-column>
    </el-table>

    <!-- Pagination -->
    <div class="mt-4 flex justify-end">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        @current-change="fetchList"
        @size-change="handleSizeChange"
      />
    </div>

    <!-- Reject Dialog -->
    <el-dialog v-model="rejectDialogVisible" title="拒绝认证" width="460px" @closed="rejectReason = ''">
      <el-form label-width="80px">
        <el-form-item label="拒绝原因">
          <el-input
            v-model="rejectReason"
            type="textarea"
            :rows="3"
            placeholder="请输入拒绝原因"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="rejectDialogVisible = false">取消</el-button>
        <el-button type="danger" :loading="submitting" @click="handleReject">确认拒绝</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api'

interface StudentVerification {
  userID: number
  schoolID: string
  studentIDs: string[]
  activeStudentID: string
  verificationStatus: string
  verificationMethod: string
  phone: string
  createdAt: string
}

const loading = ref(false)
const submitting = ref(false)
const list = ref<StudentVerification[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const filterStatus = ref('')
const filterSchoolId = ref('')

const rejectDialogVisible = ref(false)
const rejectReason = ref('')
const currentRow = ref<StudentVerification | null>(null)

function statusLabel(status: string): string {
  const map: Record<string, string> = {
    unverified: '未认证',
    pending: '待审核',
    verified: '已认证',
    rejected: '已拒绝',
  }
  return map[status] ?? status
}

function statusTagType(status: string): 'success' | 'warning' | 'info' | 'danger' {
  const map: Record<string, 'success' | 'warning' | 'info' | 'danger'> = {
    unverified: 'info',
    pending: 'warning',
    verified: 'success',
    rejected: 'danger',
  }
  return map[status] ?? 'info'
}

function verificationMethodLabel(method: string): string {
  const map: Record<string, string> = {
    ldap: 'LDAP',
    manual: '人工审核',
  }
  return map[method] ?? method
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '—'
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN', { hour12: false })
}

async function fetchList() {
  loading.value = true
  try {
    const res = await api.userSystem.listStudentVerifications({
      status: filterStatus.value || undefined,
      schoolId: filterSchoolId.value || undefined,
      page: page.value,
      pageSize: pageSize.value,
    })
    const data = (res as { data?: { list?: StudentVerification[]; total?: number } }).data
    list.value = data?.list ?? []
    total.value = data?.total ?? 0
  } catch {
    ElMessage.error('获取数据失败')
  } finally {
    loading.value = false
  }
}

function handleFilterChange() {
  page.value = 1
  fetchList()
}

function handleSizeChange() {
  page.value = 1
  fetchList()
}

async function handleApprove(row: StudentVerification) {
  try {
    await api.userSystem.reviewStudentVerification(row.userID, { status: 'verified' })
    ElMessage.success('审核通过')
    fetchList()
  } catch {
    ElMessage.error('操作失败')
  }
}

function openRejectDialog(row: StudentVerification) {
  currentRow.value = row
  rejectReason.value = ''
  rejectDialogVisible.value = true
}

async function handleReject() {
  if (!currentRow.value) return
  submitting.value = true
  try {
    await api.userSystem.reviewStudentVerification(currentRow.value.userID, {
      status: 'rejected',
      rejectionReason: rejectReason.value,
    })
    ElMessage.success('已拒绝')
    rejectDialogVisible.value = false
    fetchList()
  } catch {
    ElMessage.error('操作失败')
  } finally {
    submitting.value = false
  }
}

onMounted(fetchList)
</script>
