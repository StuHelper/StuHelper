<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h2 class="text-lg font-semibold text-gray-900">学校配置</h2>
    </div>

    <!-- Table -->
    <el-table v-loading="loading" :data="list" border stripe>
      <el-table-column prop="schoolID" label="学校ID" width="140" />
      <el-table-column prop="schoolName" label="学校名称" min-width="160" />
      <el-table-column label="认证方式" width="130">
        <template #default="{ row }">
          {{ verificationMethodLabel(row.verificationMethod) }}
        </template>
      </el-table-column>
      <el-table-column label="启用状态" width="110">
        <template #default="{ row }">
          <el-switch
            v-model="row.enabled"
            :loading="row._toggling"
            @change="handleToggleEnabled(row)"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openEditDialog(row)">编辑</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Edit Dialog -->
    <el-dialog
      v-model="editDialogVisible"
      title="编辑学校配置"
      width="520px"
      @closed="resetForm"
    >
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="100px">
        <el-form-item label="学校名称" prop="schoolName">
          <el-input v-model="form.schoolName" placeholder="请输入学校名称" />
        </el-form-item>
        <el-form-item label="认证方式" prop="verificationMethod">
          <el-select v-model="form.verificationMethod" style="width: 100%">
            <el-option label="LDAP" value="ldap" />
            <el-option label="人工审核" value="manual" />
          </el-select>
        </el-form-item>
        <el-form-item label="同意条款" prop="consentText">
          <el-input
            v-model="form.consentText"
            type="textarea"
            :rows="4"
            placeholder="用户同意条款文本（可选）"
          />
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { components } from '@stuhelper/shared'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { api } from '@/api'

type SchoolConfigRow = components['schemas']['AdminSchoolConfig'] & { _toggling?: boolean }
type VerificationMethod = components['schemas']['AdminSchoolConfig']['verificationMethod']
type SchoolConfigForm = {
  schoolName: string
  verificationMethod: VerificationMethod
  consentText: string
  enabled: boolean
}

const loading = ref(false)
const submitting = ref(false)
const list = ref<SchoolConfigRow[]>([])

const editDialogVisible = ref(false)
const editingSchoolID = ref('')
const formRef = ref<FormInstance>()
const form = ref<SchoolConfigForm>({
  schoolName: '',
  verificationMethod: 'manual',
  consentText: '',
  enabled: true,
})
const formRules: FormRules = {
  schoolName: [{ required: true, message: '请输入学校名称', trigger: 'blur' }],
  verificationMethod: [{ required: true, message: '请选择认证方式', trigger: 'change' }],
}

function verificationMethodLabel(method: string): string {
  const map: Record<string, string> = {
    ldap: 'LDAP',
    manual: '人工审核',
  }
  return map[method] ?? method
}

async function fetchList() {
  loading.value = true
  try {
    const res = await api.userAdmin.listSchoolConfigs()
    list.value = (res.data?.data ?? []).map((item) => ({ ...item, _toggling: false }))
  } catch {
    ElMessage.error('获取数据失败')
  } finally {
    loading.value = false
  }
}

async function handleToggleEnabled(row: SchoolConfigRow) {
  row._toggling = true
  try {
    await api.userAdmin.updateSchoolConfig(row.schoolID, {
      schoolName: row.schoolName,
      verificationMethod: row.verificationMethod,
      consentText: row.consentText ?? undefined,
      enabled: row.enabled,
    })
    ElMessage.success(row.enabled ? '已启用' : '已禁用')
  } catch {
    row.enabled = !row.enabled
    ElMessage.error('操作失败')
  } finally {
    row._toggling = false
  }
}

function openEditDialog(row: SchoolConfigRow) {
  editingSchoolID.value = row.schoolID
  form.value = {
    schoolName: row.schoolName,
    verificationMethod: row.verificationMethod,
    consentText: row.consentText ?? '',
    enabled: row.enabled,
  }
  editDialogVisible.value = true
}

function resetForm() {
  editingSchoolID.value = ''
  form.value = { schoolName: '', verificationMethod: 'manual', consentText: '', enabled: true }
  formRef.value?.clearValidate()
}

async function handleSave() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    await api.userAdmin.updateSchoolConfig(editingSchoolID.value, {
      schoolName: form.value.schoolName,
      verificationMethod: form.value.verificationMethod,
      consentText: form.value.consentText || undefined,
      enabled: form.value.enabled,
    })
    ElMessage.success('保存成功')
    editDialogVisible.value = false
    fetchList()
  } catch {
    ElMessage.error('操作失败')
  } finally {
    submitting.value = false
  }
}

onMounted(fetchList)
</script>
