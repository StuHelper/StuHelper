<template>
  <div class="space-y-4">
    <!-- Header + toolbar -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
      <h1 class="text-xl font-bold text-gray-900">敏感词管理</h1>
      <div class="flex items-center gap-2 flex-wrap">
        <el-select v-model="filterCategory" placeholder="全部分类" clearable style="width: 140px" @change="onFilterChange">
          <el-option value="" label="全部分类" />
          <el-option v-for="c in categories" :key="c" :value="c" :label="c" />
        </el-select>
        <el-select v-model="filterLevel" placeholder="全部级别" clearable style="width: 130px" @change="onFilterChange">
          <el-option value="" label="全部级别" />
          <el-option value="block" label="屏蔽" />
          <el-option value="warn" label="警告" />
          <el-option value="review" label="审核" />
        </el-select>
        <el-button type="primary" @click="openForm()">
          <template #icon><el-icon><Plus /></el-icon></template>
          添加敏感词
        </el-button>
      </div>
    </div>

    <!-- Table -->
    <el-table v-loading="loading" :data="words" row-key="id">
      <el-table-column label="词语" prop="word" min-width="120" />
      <el-table-column label="分类" prop="category" width="120" show-overflow-tooltip />
      <el-table-column label="级别" width="90">
        <template #default="{ row }">
          <el-tag :type="levelTagType(row.level)" size="small">{{ levelLabel(row.level) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-switch
            :model-value="row.isActive"
            size="small"
            @change="toggleActive(row)"
          />
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
        @current-change="fetchWords"
        @size-change="onSizeChange"
      />
    </div>

    <!-- Add/Edit dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingWord ? '编辑敏感词' : '添加敏感词'"
      width="480px"
      @closed="resetForm"
    >
      <el-form ref="formRef" :model="form" :rules="formRules" label-position="top">
        <el-form-item label="词语" prop="word">
          <el-input v-model="form.word" placeholder="请输入敏感词" />
        </el-form-item>
        <el-form-item label="分类">
          <el-input v-model="form.category" placeholder="请输入分类（如：政治、广告等）" />
        </el-form-item>
        <el-form-item label="级别" prop="level">
          <el-radio-group v-model="form.level">
            <el-radio-button value="block">屏蔽</el-radio-button>
            <el-radio-button value="warn">警告</el-radio-button>
            <el-radio-button value="review">审核</el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">
          {{ editingWord ? '保存修改' : '添加' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { api } from '@/api'
import type { components } from '@/api'

type AdminSensitiveWord = components['schemas']['AdminSensitiveWord']

const loading = ref(false)
const words = ref<AdminSensitiveWord[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const filterCategory = ref('')
const filterLevel = ref<'' | 'block' | 'warn' | 'review'>('')

const categories = computed(() => {
  return [...new Set(words.value.map(w => w.category).filter((c): c is string => !!c))].sort()
})

// Dialog state
const dialogVisible = ref(false)
const editingWord = ref<AdminSensitiveWord | null>(null)
const saving = ref(false)
const formRef = ref<FormInstance>()
const form = reactive({
  word: '',
  category: '',
  level: 'block' as 'block' | 'warn' | 'review',
})
const formRules: FormRules = {
  word: [{ required: true, message: '请输入敏感词', trigger: 'blur' }],
  level: [{ required: true, message: '请选择级别', trigger: 'change' }],
}

function levelTagType(level: string): 'success' | 'warning' | 'danger' | 'info' {
  if (level === 'block') return 'danger'
  if (level === 'warn') return 'warning'
  return 'info'
}

function levelLabel(level: string) {
  if (level === 'block') return '屏蔽'
  if (level === 'warn') return '警告'
  return '审核'
}

function formatTime(dateStr: string) {
  return new Date(dateStr).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

function onFilterChange() {
  page.value = 1
  fetchWords()
}

function onSizeChange() {
  page.value = 1
  fetchWords()
}

function openForm(word?: AdminSensitiveWord) {
  editingWord.value = word || null
  form.word = word?.word || ''
  form.category = word?.category || ''
  form.level = word?.level || 'block'
  dialogVisible.value = true
}

function resetForm() {
  formRef.value?.resetFields()
  editingWord.value = null
}

async function fetchWords() {
  loading.value = true
  try {
    const params: { page: number; pageSize: number; category?: string; level?: 'block' | 'warn' | 'review' } = {
      page: page.value,
      pageSize: pageSize.value,
    }
    if (filterCategory.value) params.category = filterCategory.value
    if (filterLevel.value) params.level = filterLevel.value
    const res = await api.admin.getSensitiveWords(params)
    words.value = res.data?.data?.list || []
    total.value = res.data?.data?.total || 0
  } catch {
    ElMessage.error('加载失败，请重试')
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      if (editingWord.value) {
        await api.admin.updateSensitiveWord(editingWord.value.id, {
          word: form.word.trim(),
          category: form.category.trim() || undefined,
          level: form.level,
        })
        ElMessage.success('修改成功')
      } else {
        await api.admin.createSensitiveWord({
          word: form.word.trim(),
          category: form.category.trim() || 'general',
          level: form.level,
        })
        ElMessage.success('添加成功')
      }
      dialogVisible.value = false
      fetchWords()
    } catch {
      ElMessage.error('保存失败，请重试')
    } finally {
      saving.value = false
    }
  })
}

async function toggleActive(word: AdminSensitiveWord) {
  try {
    await api.admin.updateSensitiveWord(word.id, { isActive: !word.isActive })
    ElMessage.success('状态已更新')
    fetchWords()
  } catch {
    ElMessage.error('操作失败，请重试')
  }
}

async function handleDelete(word: AdminSensitiveWord) {
  try {
    await ElMessageBox.confirm(`确定要删除敏感词「${word.word}」吗？`, '删除确认', {
      confirmButtonText: '确认删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await api.admin.deleteSensitiveWord(word.id)
    ElMessage.success('删除成功')
    fetchWords()
  } catch (err) {
    if (err !== 'cancel') ElMessage.error('删除失败，请重试')
  }
}

onMounted(fetchWords)
</script>
