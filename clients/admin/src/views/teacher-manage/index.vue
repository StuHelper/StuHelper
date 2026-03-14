<template>
  <div class="space-y-4">
    <!-- Header + toolbar -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
      <h1 class="text-xl font-bold text-gray-900">教师管理</h1>
      <div class="flex items-center gap-2">
        <el-input
          v-model="searchQuery"
          placeholder="搜索教师姓名"
          clearable
          style="width: 200px"
          @input="onSearch"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select v-model="filterDeptID" placeholder="全部院系" clearable style="width: 160px" @change="onFilterChange">
          <el-option :value="0" label="全部院系" />
          <el-option v-for="d in departments" :key="d.id" :value="d.id" :label="d.name" />
        </el-select>
        <el-button type="primary" @click="openForm()">
          <template #icon><el-icon><Plus /></el-icon></template>
          添加教师
        </el-button>
      </div>
    </div>

    <!-- Table -->
    <el-table v-loading="loading" :data="teachers" row-key="id">
      <el-table-column label="姓名" prop="name" min-width="120" />
      <el-table-column label="所属院系" min-width="160">
        <template #default="{ row }">
          <span>{{ row.departmentName || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="课程评价数" width="120" align="center">
        <template #default="{ row }">
          <el-tag type="info" size="small">{{ row.reviewCount ?? 0 }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="160">
        <template #default="{ row }">
          <span class="text-xs text-gray-400">{{ formatTime(row.createdAt) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button size="small" link type="primary" @click="openForm(row)">编辑</el-button>
          <el-button size="small" link type="danger" @click="handleDelete(row)">删除</el-button>
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
        @current-change="fetchTeachers"
        @size-change="onSizeChange"
      />
    </div>

    <!-- Add/Edit dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingTeacher ? '编辑教师' : '添加教师'"
      width="480px"
      @closed="resetForm"
    >
      <el-form ref="formRef" :model="form" :rules="formRules" label-position="top">
        <el-form-item label="姓名" prop="name">
          <el-input v-model="form.name" placeholder="请输入教师姓名" />
        </el-form-item>
        <el-form-item label="所属院系">
          <el-select v-model="form.departmentID" placeholder="请选择院系（可选）" clearable class="w-full">
            <el-option v-for="d in departments" :key="d.id" :value="d.id" :label="d.name" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">
          {{ editingTeacher ? '保存修改' : '添加' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Search, Plus } from '@element-plus/icons-vue'
import { api } from '@/api'
import type { components } from '@/api'

type AdminTeacher = components['schemas']['AdminTeacher']

// Minimal department type for the filter
interface Department {
  id: number
  name: string
}

const loading = ref(false)
const teachers = ref<AdminTeacher[]>([])
const departments = ref<Department[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const searchQuery = ref('')
const filterDeptID = ref<number>(0)

// Dialog state
const dialogVisible = ref(false)
const editingTeacher = ref<AdminTeacher | null>(null)
const saving = ref(false)
const formRef = ref<FormInstance>()
const form = reactive({
  name: '',
  departmentID: undefined as number | undefined,
})
const formRules: FormRules = {
  name: [{ required: true, message: '请输入教师姓名', trigger: 'blur' }],
}

function formatTime(dateStr: string) {
  return new Date(dateStr).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

function onFilterChange() {
  page.value = 1
  fetchTeachers()
}

let searchTimer: ReturnType<typeof setTimeout>
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    fetchTeachers()
  }, 300)
}

function onSizeChange() {
  page.value = 1
  fetchTeachers()
}

function openForm(teacher?: AdminTeacher) {
  editingTeacher.value = teacher || null
  form.name = teacher?.name || ''
  form.departmentID = teacher?.departmentID ?? undefined
  dialogVisible.value = true
}

function resetForm() {
  formRef.value?.resetFields()
  editingTeacher.value = null
}

async function fetchTeachers() {
  loading.value = true
  try {
    const params: { page: number; pageSize: number; search?: string; departmentID?: number } = {
      page: page.value,
      pageSize: pageSize.value,
    }
    if (searchQuery.value.trim()) params.search = searchQuery.value.trim()
    if (filterDeptID.value) params.departmentID = filterDeptID.value
    const res = await api.admin.getTeachers(params)
    teachers.value = res.data?.data?.list || []
    total.value = res.data?.data?.total || 0
  } catch {
    ElMessage.error('加载失败，请重试')
  } finally {
    loading.value = false
  }
}

async function fetchDepartments() {
  try {
    // Try to get departments from teachers data if available
    // The admin API doesn't expose a departments endpoint directly, so we use what we have
    const res = await api.admin.getTeachers({ pageSize: 200 })
    const list = res.data?.data?.list || []
    const seen = new Set<number>()
    const depts: Department[] = []
    for (const t of list) {
      if (t.departmentID && t.departmentName && !seen.has(t.departmentID)) {
        seen.add(t.departmentID)
        depts.push({ id: t.departmentID, name: t.departmentName })
      }
    }
    departments.value = depts
  } catch { /* ignore */ }
}

async function handleSave() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      if (editingTeacher.value) {
        await api.admin.updateTeacher(editingTeacher.value.id, {
          name: form.name.trim(),
          departmentID: form.departmentID ?? null,
        })
        ElMessage.success('修改成功')
      } else {
        await api.admin.createTeacher({
          name: form.name.trim(),
          departmentID: form.departmentID,
        })
        ElMessage.success('添加成功')
      }
      dialogVisible.value = false
      fetchTeachers()
      fetchDepartments()
    } catch {
      ElMessage.error('保存失败，请重试')
    } finally {
      saving.value = false
    }
  })
}

async function handleDelete(teacher: AdminTeacher) {
  try {
    await ElMessageBox.confirm(`确定要删除教师「${teacher.name}」吗？此操作不可撤销。`, '删除确认', {
      confirmButtonText: '确认删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await api.admin.deleteTeacher(teacher.id)
    ElMessage.success('删除成功')
    fetchTeachers()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error('删除失败，请重试')
  }
}

onMounted(() => {
  fetchTeachers()
  fetchDepartments()
})
</script>
