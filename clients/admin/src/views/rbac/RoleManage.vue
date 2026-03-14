<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h2 class="text-lg font-semibold text-gray-900">角色管理</h2>
      <el-button type="primary" @click="openCreateDialog">新增角色</el-button>
    </div>

    <!-- Table -->
    <el-table v-loading="loading" :data="roles" border stripe>
      <el-table-column prop="name" label="角色名称" width="160" />
      <el-table-column prop="displayName" label="显示名称" width="160" />
      <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
      <el-table-column label="系统角色" width="100">
        <template #default="{ row }">
          <el-tag :type="row.isSystem ? 'warning' : 'info'" size="small">
            {{ row.isSystem ? '是' : '否' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="240" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openPermissionDialog(row)">配置权限</el-button>
          <el-button size="small" :disabled="row.isSystem" @click="openEditDialog(row)">编辑</el-button>
          <el-button size="small" type="danger" :disabled="row.isSystem" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create / Edit Dialog -->
    <el-dialog
      v-model="roleDialogVisible"
      :title="editingRole ? '编辑角色' : '新增角色'"
      width="480px"
      @closed="resetRoleForm"
    >
      <el-form ref="roleFormRef" :model="roleForm" :rules="roleRules" label-width="90px">
        <el-form-item label="角色名称" prop="name">
          <el-input v-model="roleForm.name" :disabled="!!editingRole" placeholder="英文标识符，如 editor" />
        </el-form-item>
        <el-form-item label="显示名称" prop="displayName">
          <el-input v-model="roleForm.displayName" placeholder="如 编辑员" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="roleForm.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="roleDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSaveRole">保存</el-button>
      </template>
    </el-dialog>

    <!-- Permission Dialog -->
    <el-dialog
      v-model="permissionDialogVisible"
      title="配置权限"
      width="600px"
      @closed="resetPermissionState"
    >
      <div v-loading="permissionLoading">
        <div class="mb-3 text-sm text-gray-500">
          为角色 <span class="font-semibold text-gray-800">{{ permissionTargetRole?.displayName }}</span> 配置权限
        </div>
        <el-scrollbar max-height="400px">
          <div v-for="(perms, module) in groupedPermissions" :key="module" class="mb-4">
            <div class="text-sm font-semibold text-gray-700 mb-2">{{ module }}</div>
            <el-checkbox-group v-model="selectedPermissionIDs" class="flex flex-wrap gap-2">
              <el-checkbox
                v-for="perm in perms"
                :key="perm.id"
                :value="perm.id"
              >
                {{ perm.displayName || perm.name }}
              </el-checkbox>
            </el-checkbox-group>
          </div>
        </el-scrollbar>
      </div>
      <template #footer>
        <el-button @click="permissionDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSavePermissions">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { api } from '@/api'

interface Role {
  id: number
  name: string
  displayName: string
  description?: string
  isSystem: boolean
}

interface Permission {
  id: number
  name: string
  displayName?: string
  module?: string
}

const loading = ref(false)
const submitting = ref(false)
const roles = ref<Role[]>([])
const allPermissions = ref<Permission[]>([])

// Role dialog state
const roleDialogVisible = ref(false)
const editingRole = ref<Role | null>(null)
const roleFormRef = ref<FormInstance>()
const roleForm = ref({
  name: '',
  displayName: '',
  description: '',
})
const roleRules: FormRules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  displayName: [{ required: true, message: '请输入显示名称', trigger: 'blur' }],
}

// Permission dialog state
const permissionDialogVisible = ref(false)
const permissionLoading = ref(false)
const permissionTargetRole = ref<Role | null>(null)
const selectedPermissionIDs = ref<number[]>([])

const groupedPermissions = computed(() => {
  const groups: Record<string, Permission[]> = {}
  for (const perm of allPermissions.value) {
    const mod = perm.module || '其他'
    if (!groups[mod]) groups[mod] = []
    groups[mod].push(perm)
  }
  return groups
})

async function fetchRoles() {
  loading.value = true
  try {
    const res = await api.userSystem.listRoles()
    const data = (res as { data?: { list?: Role[] } }).data
    roles.value = data?.list ?? (Array.isArray(data) ? (data as Role[]) : [])
  } catch {
    ElMessage.error('获取角色列表失败')
  } finally {
    loading.value = false
  }
}

async function fetchPermissions() {
  try {
    const res = await api.userSystem.listPermissions()
    const data = (res as { data?: { list?: Permission[] } }).data
    allPermissions.value = data?.list ?? (Array.isArray(data) ? (data as Permission[]) : [])
  } catch {
    ElMessage.error('获取权限列表失败')
  }
}

function openCreateDialog() {
  editingRole.value = null
  roleDialogVisible.value = true
}

function openEditDialog(row: Role) {
  editingRole.value = row
  roleForm.value = {
    name: row.name,
    displayName: row.displayName,
    description: row.description ?? '',
  }
  roleDialogVisible.value = true
}

function resetRoleForm() {
  editingRole.value = null
  roleForm.value = { name: '', displayName: '', description: '' }
  roleFormRef.value?.clearValidate()
}

async function handleSaveRole() {
  const valid = await roleFormRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (editingRole.value) {
      await api.userSystem.updateRole(editingRole.value.id, {
        displayName: roleForm.value.displayName,
        description: roleForm.value.description || undefined,
      })
      ElMessage.success('更新成功')
    } else {
      await api.userSystem.createRole({
        name: roleForm.value.name,
        displayName: roleForm.value.displayName,
        description: roleForm.value.description || undefined,
      })
      ElMessage.success('创建成功')
    }
    roleDialogVisible.value = false
    fetchRoles()
  } catch {
    ElMessage.error('操作失败')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row: Role) {
  try {
    await ElMessageBox.confirm(`确定要删除角色 "${row.displayName}" 吗？`, '删除确认', {
      type: 'warning',
      confirmButtonText: '确认删除',
      cancelButtonText: '取消',
      confirmButtonClass: 'el-button--danger',
    })
  } catch {
    return
  }

  try {
    await api.userSystem.deleteRole(row.id)
    ElMessage.success('删除成功')
    fetchRoles()
  } catch {
    ElMessage.error('操作失败')
  }
}

async function openPermissionDialog(row: Role) {
  permissionTargetRole.value = row
  selectedPermissionIDs.value = []
  permissionDialogVisible.value = true
  permissionLoading.value = true
  try {
    const res = await api.userSystem.getRolePermissions(row.id)
    const data = (res as { data?: { list?: { id: number }[] } }).data
    const assigned = data?.list ?? (Array.isArray(data) ? (data as { id: number }[]) : [])
    selectedPermissionIDs.value = assigned.map((p) => p.id)
  } catch {
    ElMessage.error('获取权限失败')
  } finally {
    permissionLoading.value = false
  }
}

function resetPermissionState() {
  permissionTargetRole.value = null
  selectedPermissionIDs.value = []
}

async function handleSavePermissions() {
  if (!permissionTargetRole.value) return
  submitting.value = true
  try {
    await api.userSystem.setRolePermissions(permissionTargetRole.value.id, selectedPermissionIDs.value)
    ElMessage.success('权限配置已保存')
    permissionDialogVisible.value = false
  } catch {
    ElMessage.error('操作失败')
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  await Promise.all([fetchRoles(), fetchPermissions()])
})
</script>
