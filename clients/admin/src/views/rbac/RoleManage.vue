<template>
  <div>
    <div class="mb-4 flex items-center justify-between">
      <h2 class="text-lg font-semibold text-gray-900">角色管理</h2>
      <el-button v-if="canCreateRole" type="primary" @click="openCreateDialog">新增角色</el-button>
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
          <template v-if="canManageRolePermissions || canUpdateRole || canDeleteRole">
            <el-button
              v-if="canManageRolePermissions"
              size="small"
              :disabled="row.isSystem"
              @click="openPermissionDialog(row)"
            >
              配置权限
            </el-button>
            <el-button
              v-if="canUpdateRole"
              size="small"
              :disabled="row.isSystem"
              @click="openEditDialog(row)"
            >
              编辑
            </el-button>
            <el-button
              v-if="canDeleteRole"
              size="small"
              type="danger"
              :disabled="row.isSystem"
              @click="handleDelete(row)"
            >
              删除
            </el-button>
          </template>
          <span v-else class="text-sm text-gray-400">—</span>
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
      <el-alert
        v-if="permissionLoadFailed"
        class="mb-4"
        type="error"
        :closable="false"
        title="加载角色权限失败，请关闭弹窗后重试。"
      />
      <div v-loading="permissionLoading">
        <div class="mb-3 text-sm text-gray-500">
          为角色 <span class="font-semibold text-gray-800">{{ permissionTargetRole?.displayName }}</span> 配置权限
        </div>
        <el-scrollbar max-height="400px">
          <div v-for="(perms, module) in groupedPermissions" :key="module" class="mb-4">
            <div class="text-sm font-semibold text-gray-700 mb-2">{{ module }}</div>
            <el-checkbox-group
              v-model="selectedPermissionIDs"
              class="flex flex-wrap gap-2"
              :disabled="permissionLoadFailed"
            >
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
        <el-button type="primary" :loading="submitting" :disabled="!canSavePermissions" @click="handleSavePermissions">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { api } from '@/api'
import type { components } from '@stuhelper/shared'
import { useAuthStore } from '@/stores/auth'
import {
  RBAC_PERMISSION_READ,
  RBAC_ROLE_CREATE,
  RBAC_ROLE_DELETE,
  RBAC_ROLE_READ,
  RBAC_ROLE_UPDATE,
  hasCapability,
} from '@stuhelper/shared/constants'

type Role = components['schemas']['Role']
type Permission = components['schemas']['Permission']

const authStore = useAuthStore()
const loading = ref(false)
const submitting = ref(false)
const roles = ref<Role[]>([])
const allPermissions = ref<Permission[]>([])

const canReadRoles = computed(() =>
  hasCapability(authStore.globalCapabilities, RBAC_ROLE_READ),
)
const canReadPermissions = computed(() =>
  hasCapability(authStore.globalCapabilities, RBAC_PERMISSION_READ),
)
const canCreateRole = computed(() =>
  hasCapability(authStore.globalCapabilities, RBAC_ROLE_CREATE),
)
const canUpdateRole = computed(() =>
  hasCapability(authStore.globalCapabilities, RBAC_ROLE_UPDATE),
)
const canDeleteRole = computed(() =>
  hasCapability(authStore.globalCapabilities, RBAC_ROLE_DELETE),
)
const canManageRolePermissions = computed(() =>
  canReadPermissions.value && canUpdateRole.value,
)

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
const permissionLoadFailed = ref(false)
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

const canSavePermissions = computed(() =>
  !permissionLoading.value && !permissionLoadFailed.value && permissionTargetRole.value !== null,
)

async function fetchRoles() {
  if (!canReadRoles.value) {
    roles.value = []
    return
  }
  loading.value = true
  try {
    const res = await api.rbac.listRoles()
    roles.value = res.data?.data ?? []
  } catch {
    ElMessage.error('获取角色列表失败')
  } finally {
    loading.value = false
  }
}

async function fetchPermissions() {
  if (!canReadPermissions.value || allPermissions.value.length > 0) {
    return
  }
  try {
    const res = await api.rbac.listPermissions()
    allPermissions.value = res.data?.data ?? []
  } catch {
    ElMessage.error('获取权限列表失败')
  }
}

function openCreateDialog() {
  if (!canCreateRole.value) return
  editingRole.value = null
  roleDialogVisible.value = true
}

function openEditDialog(row: Role) {
  if (!canUpdateRole.value) return
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
      await api.rbac.updateRole(editingRole.value.id, {
        displayName: roleForm.value.displayName,
        description: roleForm.value.description,
      })
      ElMessage.success('更新成功')
    } else {
      await api.rbac.createRole({
        name: roleForm.value.name,
        displayName: roleForm.value.displayName,
        description: roleForm.value.description || undefined,
      })
      ElMessage.success('创建成功')
    }
    roleDialogVisible.value = false
    await fetchRoles()
  } catch {
    ElMessage.error('操作失败')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row: Role) {
  if (!canDeleteRole.value) return
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
    await api.rbac.deleteRole(row.id)
    ElMessage.success('删除成功')
    await fetchRoles()
  } catch {
    ElMessage.error('操作失败')
  }
}

async function openPermissionDialog(row: Role) {
  if (!canManageRolePermissions.value) return
  permissionTargetRole.value = row
  selectedPermissionIDs.value = []
  permissionLoadFailed.value = false
  permissionDialogVisible.value = true
  permissionLoading.value = true
  try {
    await fetchPermissions()
    const res = await api.rbac.getRolePermissions(row.id)
    selectedPermissionIDs.value = res.data?.data.permissionIDs ?? []
  } catch {
    permissionLoadFailed.value = true
    ElMessage.error('获取权限失败')
  } finally {
    permissionLoading.value = false
  }
}

function resetPermissionState() {
  permissionTargetRole.value = null
  selectedPermissionIDs.value = []
  permissionLoadFailed.value = false
}

async function handleSavePermissions() {
  if (!permissionTargetRole.value || !canSavePermissions.value) return

  const clearAll = selectedPermissionIDs.value.length === 0
  if (clearAll) {
    try {
      await ElMessageBox.confirm(
        '您即将清空该角色的所有权限，确定继续吗？',
        '清空权限确认',
        {
          type: 'warning',
          confirmButtonText: '确认清空',
          cancelButtonText: '取消',
        },
      )
    } catch {
      return
    }
  }

  submitting.value = true
  try {
    await api.rbac.assignRolePermissions(permissionTargetRole.value.id, {
      permissionIDs: selectedPermissionIDs.value,
      clearAll,
    })
    ElMessage.success('权限配置已保存')
    permissionDialogVisible.value = false
  } catch {
    ElMessage.error('操作失败')
  } finally {
    submitting.value = false
  }
}

onMounted(fetchRoles)
</script>
