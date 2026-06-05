<template>
  <div class="roles-view-container">
    <!-- 侧边栏：角色列表 -->
    <aside class="sidebar">
      <div class="sidebar-header">
        <h2>角色</h2>
        <button
          class="icon-btn"
          :disabled="loading || Boolean(loadError) || Boolean(roleOperation)"
          @click="createRole"
          :title="roleOperation === 'create' ? '创建中' : '创建角色'"
        >
          {{ roleOperation === 'create' ? '…' : '＋' }}
        </button>
      </div>
      
      <div class="role-list">
        <div
          v-for="role in roles"
          :key="role.id"
          class="role-item"
          :class="{ active: currentRole?.id === role.id }"
          @click="selectRole(role)"
          draggable="true"
          @dragstart="onDragStart($event, role)"
          @dragover.prevent
          @drop="onDrop($event, role)"
        >
          <span class="role-color" :style="{ backgroundColor: role.color || '#999' }"></span>
          <span class="role-name">{{ role.name }}</span>
          <k-icon v-if="role.builtin" name="lock" class="builtin-icon" title="内置角色" />
          <k-icon v-else name="grip-vertical" class="drag-handle" />
        </div>
      </div>
    </aside>

    <!-- 主内容区：编辑面板 -->
    <main class="main-content" v-if="currentRole">
      <div v-if="loadError" class="roles-load-banner" role="alert">
        <div class="roles-load-banner__body">
          <strong>刷新角色权限数据失败</strong>
          <span>{{ loadError }}</span>
        </div>
        <button class="secondary-btn" @click="fetchData">重试</button>
      </div>

      <div v-if="actionError" class="roles-action-banner" role="alert">
        <div class="roles-action-banner__body">
          <strong>{{ actionErrorTitle }}</strong>
          <span>{{ actionError }}</span>
        </div>
        <button class="secondary-btn" type="button" @click="clearActionError">关闭</button>
      </div>

      <div class="content-header">
        <h1>
          {{ currentRole.name }}
          <span v-if="currentRole.builtin" class="builtin-badge">内置</span>
        </h1>
        <div class="header-actions" v-if="!currentRole.builtin">
            <button class="clone-btn" :disabled="Boolean(roleOperation)" @click="cloneRole">
              {{ roleOperation === 'clone' ? '克隆中…' : '克隆角色' }}
            </button>
            <span style="width: 12px; display: inline-block;"></span>
           <button class="danger-btn" :disabled="Boolean(roleOperation)" @click="deleteRole">
             {{ roleOperation === 'delete' ? '删除中…' : '删除角色' }}
           </button>
        </div>
      </div>

      <div class="tabs">
        <div class="tab-item" :class="{ active: activeTab === 'display' }" @click="activeTab = 'display'">基础</div>
        <div class="tab-item" :class="{ active: activeTab === 'permissions' }" @click="activeTab = 'permissions'">权限</div>
        <div class="tab-item" :class="{ active: activeTab === 'members' }" @click="activeTab = 'members'">成员</div>
      </div>

      <div class="tab-content">
        <!-- 基础设置 -->
        <div v-if="activeTab === 'display'" class="display-settings">
          <!-- 内置角色提示 -->
          <div v-if="currentRole.builtin" class="builtin-notice">
            <k-icon name="info" />
            <span>
              <template v-if="currentRole.id === 'guild-admin'">
                <strong>群管理员</strong> - 当用户在群内拥有管理员或群主身份时，自动获得此角色的权限（仅在当前群生效）
              </template>
              <template v-else-if="currentRole.id.startsWith('authority')">
                <strong>{{ currentRole.name }}</strong> - 基于 Koishi 的 authority 权限等级自动分配（全局生效）
              </template>
              <template v-else>
                此为内置角色，不可删除
              </template>
            </span>
          </div>

          <div class="form-group" v-if="!currentRole.builtin">
            <label>角色 ID</label>
            <div class="id-display">
              <code class="role-id-code">{{ currentRole.id }}</code>
              <button class="copy-btn" @click="copyRoleId" title="复制角色 ID">
                📋
              </button>
            </div>
            <div class="field-hint">用于命令：gauth.add @用户 {{ currentRole.name }}</div>
          </div>

          <div class="form-group">
            <label>角色名称</label>
            <input type="text" v-model="editingRole.name" :disabled="currentRole.builtin" class="form-input">
          </div>

          <div class="form-group" v-if="!currentRole.builtin">
            <label>角色别名</label>
            <input type="text" v-model="editingRole.alias" placeholder="用于命令查找的简短名称" class="form-input">
            <div class="field-hint">命令示例：gauth.add @用户 {{ editingRole.alias || editingRole.name || '别名' }}</div>
          </div>

          <div class="form-group">
            <label>角色颜色</label>
            <div class="color-picker-wrapper">
              <input type="color" v-model="editingRole.color" class="color-input">
              <input type="text" v-model="editingRole.color" placeholder="#RRGGBB" class="form-input color-text">
            </div>
          </div>

          <div class="form-group">
            <label>生效范围</label>
            <div v-if="currentRole.builtin" class="scope-readonly">
              <span class="scope-badge">
                <template v-if="currentRole.id === 'guild-admin'">仅当前群生效（由系统自动识别群管理员身份）</template>
                <template v-else>全局生效（基于 Koishi authority 权限等级）</template>
              </span>
            </div>
            <div v-else class="scope-options">
              <label class="radio-label">
                <input type="radio" v-model="scopeMode" value="global">
                全局生效（所有群组）
              </label>
              <label class="radio-label">
                <input type="radio" v-model="scopeMode" value="guilds">
                仅指定群组生效
              </label>
            </div>
          </div>

          <div class="form-group" v-if="!currentRole.builtin && scopeMode === 'guilds'">
            <label>指定群组 ID（每行一个）</label>
            <textarea
              v-model="guildIdsText"
              placeholder="输入群号，每行一个&#10;例如：&#10;123456789&#10;987654321"
              class="form-textarea"
              rows="4"
            ></textarea>
          </div>
        </div>

        <!-- 权限设置 -->
        <div v-if="activeTab === 'permissions'" class="permissions-settings">
          <div class="permissions-layout">
            <!-- 左侧主内容 -->
            <div class="permissions-main" ref="permissionsMainRef">
              <div class="search-bar">
                <input type="text" v-model="permSearch" placeholder="搜索权限..." class="form-input search-input">
                <button class="secondary-btn" @click="clearPermissions">清除所有</button>
              </div>

              <!-- 当前选中权限列表 -->
              <div class="current-perms" v-if="editingRole.permissions && editingRole.permissions.length > 0">
                <span class="perms-label">已选权限:</span>
                <span class="perm-tag" v-for="p in editingRole.permissions" :key="p">{{ p }}</span>
              </div>

              <!-- 权限为空的提示 -->
              <div v-if="permissions.length === 0" class="empty-tip">
                权限列表加载中或为空...
              </div>

              <div v-else class="permission-groups">
                <div
                  v-for="(group, name) in groupedPermissions"
                  :key="name"
                  class="perm-group"
                  :id="`perm-group-${name}`"
                  :ref="el => setGroupRef(name as string, el)"
                >
                  <div class="group-header">{{ name }}</div>
                  <div class="group-items">
                    <div v-for="node in group" :key="node.id" class="permission-item" :class="{ 'covered': isCoveredByWildcard(node.id) }">
                      <div class="perm-info">
                        <div class="perm-name">{{ node.name }}</div>
                        <div class="perm-desc">{{ node.description }}</div>
                        <div class="perm-id">
                          {{ node.id }}
                          <span v-if="isCoveredByWildcard(node.id)" class="covered-hint">
                            (由 {{ isCoveredByWildcard(node.id) }} 覆盖)
                          </span>
                        </div>
                      </div>
                      <div
                        class="toggle-switch"
                        :class="{ active: hasPermission(node.id), locked: isCoveredByWildcard(node.id) }"
                        @click.stop="!isCoveredByWildcard(node.id) && togglePermission(node.id)"
                        :title="isCoveredByWildcard(node.id) ? `已被 ${isCoveredByWildcard(node.id)} 通配符覆盖` : ''"
                      >
                        <span class="slider"></span>
                        <span v-if="isCoveredByWildcard(node.id)" class="lock-icon">🔒</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 右侧快速导航 -->
            <nav class="permissions-nav" v-if="Object.keys(groupedPermissions).length > 0">
              <div class="nav-title">快速导航</div>
              <div class="nav-list">
                <div
                  v-for="(group, name) in groupedPermissions"
                  :key="name"
                  class="nav-item"
                  :class="{ active: activeGroup === name }"
                  @click="scrollToGroup(name as string)"
                >
                  <span class="nav-dot"></span>
                  <span class="nav-name">{{ name }}</span>
                  <span class="nav-count">{{ group.length }}</span>
                </div>
              </div>
            </nav>
          </div>
        </div>
        
        <!-- 成员管理 -->
        <div v-if="activeTab === 'members'" class="members-settings">
             <!-- 内置角色提示：不可手动添加成员 -->
             <div v-if="currentRole.builtin" class="builtin-notice">
               <k-icon name="info" />
               <span>
                 <template v-if="currentRole.id === 'guild-admin'">
                   此角色的成员由系统自动识别（群管理员/群主身份），不支持手动添加。
                 </template>
                 <template v-else-if="currentRole.id.startsWith('authority')">
                   此角色的成员由 Koishi 的 authority 权限等级自动分配，不支持手动添加。
                 </template>
                 <template v-else>
                   内置角色不支持手动添加成员。
                 </template>
               </span>
             </div>

             <!-- 自定义角色：可以添加成员 -->
             <template v-else>
               <div class="add-member">
                   <input
                     type="text"
                     v-model="newMemberId"
                     placeholder="输入用户 ID 添加..."
                     class="form-input"
                     :disabled="addingMember || memberLoading"
                     @keyup.enter="addMember"
                   >
                   <button class="primary-btn" :disabled="memberLoading || addingMember || !newMemberId.trim()" @click.stop="handleAddMember">
                     {{ addingMember ? '添加中…' : '添加成员' }}
                   </button>
                   <button class="secondary-btn" :disabled="memberLoading || addingMember" @click.stop="openImportDialog">导入成员</button>
               </div>

               <!-- 成员搜索框 -->
               <div class="member-search" v-if="currentRoleMembers.length > 5">
                 <input
                   type="text"
                   v-model="memberSearchQuery"
                   placeholder="搜索成员..."
                   class="form-input"
                 >
                 <span class="member-count" v-if="memberSearchQuery">
                   {{ filteredRoleMembers.length }} / {{ currentRoleMembers.length }}
                 </span>
                 <span class="member-count" v-else>
                   共 {{ currentRoleMembers.length }} 人
                 </span>
               </div>

               <div v-if="memberLoading" class="member-loading">加载角色成员中...</div>
               <div class="member-list" v-else-if="filteredRoleMembers.length > 0">
                   <div v-for="member in filteredRoleMembers" :key="member.id" class="member-item">
                       <div class="member-info">
                          <img v-if="member.avatar" :src="member.avatar" class="member-avatar">
                          <div v-else class="member-icon">👤</div>
                          <div class="member-text">
                            <span class="member-name">{{ member.name || member.id }}</span>
                            <span class="member-id-sub">{{ member.id }}</span>
                          </div>
                       </div>
                       <button
                         class="danger-btn"
                         :disabled="removingMemberIds.has(member.id)"
                         @click.stop="handleRemoveMember(member.id)"
                       >
                         {{ removingMemberIds.has(member.id) ? '移除中…' : '移除' }}
                       </button>
                   </div>
               </div>
               <div v-else-if="memberSearchQuery && currentRoleMembers.length > 0" class="empty-tip">未找到匹配的成员</div>
               <div v-else class="empty-tip">暂无成员（输入用户 QQ 号添加）</div>
             </template>
        </div>

      </div>
      
      <!-- 底部浮动保存栏 -->
      <transition name="slide-up">
        <div class="save-bar" v-if="hasChanges">
          <span>检测到未保存的修改</span>
          <div class="save-actions">
            <button class="reset-btn" :disabled="savingChanges" @click="resetChanges">重置</button>
            <button class="save-btn" :disabled="savingChanges" @click="saveChanges">
              {{ savingChanges ? '保存中…' : '保存更改' }}
            </button>
          </div>
        </div>
      </transition>
    </main>

    <main v-else-if="loadError" class="main-content roles-load-error" role="alert">
      <div class="roles-load-error__body">
        <strong>加载角色权限数据失败</strong>
        <span>{{ loadError }}</span>
      </div>
      <button class="secondary-btn" @click="fetchData">重试</button>
    </main>

    <main v-else class="main-content roles-load-empty">
      <div v-if="actionError" class="roles-action-banner" role="alert">
        <div class="roles-action-banner__body">
          <strong>{{ actionErrorTitle }}</strong>
          <span>{{ actionError }}</span>
        </div>
        <button class="secondary-btn" type="button" @click="clearActionError">关闭</button>
      </div>
      <div class="empty-tip">
        {{ loading ? '角色权限数据加载中...' : '请选择或创建一个角色' }}
      </div>
    </main>

    <ConfirmDialog
      :open="confirmDialog.open"
      :title="confirmDialog.title"
      :message="confirmDialog.message"
      :tone="confirmDialog.tone"
      :confirm-text="confirmDialog.confirmText"
      :cancel-text="confirmDialog.cancelText"
      @confirm="acceptConfirm"
      @cancel="cancelConfirm"
    />

    <!-- 导入成员对话框 -->
    <transition name="fade">
      <div class="modal-overlay" v-if="showImportDialog" @click="closeImportDialog">
        <div class="modal-dialog import-dialog" @click.stop>
          <div class="modal-header">
            <h3>导入成员</h3>
          </div>
          <div class="modal-body">
            <div v-if="importError" class="roles-action-banner roles-action-banner--modal" role="alert">
              <div class="roles-action-banner__body">
                <strong>{{ importErrorTitle }}</strong>
                <span>{{ importError }}</span>
              </div>
              <button class="secondary-btn" type="button" @click="clearImportError">关闭</button>
            </div>

            <!-- 导入来源选择 -->
            <div class="form-group">
              <label>导入来源</label>
              <div class="import-source-options">
                <label class="radio-label">
                  <input type="radio" v-model="importSource" value="role">
                  从其他角色导入
                </label>
                <label class="radio-label">
                  <input type="radio" v-model="importSource" value="authority">
                  从 Authority 等级导入
                </label>
                <label class="radio-label">
                  <input type="radio" v-model="importSource" value="guild-admin">
                  从群管理员导入
                </label>
              </div>
            </div>

            <!-- 角色选择 -->
            <div class="form-group" v-if="importSource === 'role'">
              <label>选择角色</label>
              <select v-model="importSourceRoleId" class="form-input" @change="loadImportPreview">
                <option value="">请选择角色...</option>
                <option
                  v-for="role in otherRoles"
                  :key="role.id"
                  :value="role.id"
                >{{ role.name }}</option>
              </select>
            </div>

            <!-- Authority 等级选择 -->
            <div class="form-group" v-if="importSource === 'authority'">
              <label>选择权限等级</label>
              <select v-model="importAuthorityLevel" class="form-input" @change="loadAuthorityUsers">
                <option :value="1">Authority 1</option>
                <option :value="2">Authority 2</option>
                <option :value="3">Authority 3</option>
                <option :value="4">Authority 4</option>
                <option :value="5">Authority 5</option>
              </select>
              <div class="field-hint">将从 Koishi 数据库中获取对应权限等级的用户</div>
            </div>

            <!-- 群号输入 -->
            <div class="form-group" v-if="importSource === 'guild-admin'">
              <label>输入群号</label>
              <div class="guild-input-row">
                <input
                  type="text"
                  v-model="importGuildId"
                  placeholder="请输入群号..."
                  class="form-input"
                  @keyup.enter="loadGuildAdmins"
                >
                <button class="primary-btn" @click="loadGuildAdmins" :disabled="importLoading">
                  {{ importLoading ? '加载中...' : '获取' }}
                </button>
              </div>
              <div class="field-hint">将获取该群的管理员和群主</div>
            </div>

            <!-- 预览列表 -->
            <div class="import-preview" v-if="importPreviewMembers.length > 0">
              <div class="preview-header">
                <label class="checkbox-label select-all">
                  <input type="checkbox" :checked="isAllSelected" @change="toggleSelectAll">
                  <span>全选</span>
                </label>
                <span class="preview-count">已选 {{ selectedImportIds.size }} / {{ importPreviewMembers.length }}</span>
              </div>
              <div class="preview-list">
                <div
                  v-for="member in importPreviewMembers"
                  :key="member.id"
                  class="preview-item"
                  :class="{ selected: selectedImportIds.has(member.id) }"
                  @click="toggleMemberSelect(member.id)"
                >
                  <input
                    type="checkbox"
                    :checked="selectedImportIds.has(member.id)"
                    @click.stop
                    @change="toggleMemberSelect(member.id)"
                  >
                  <img v-if="member.avatar" :src="member.avatar" class="preview-avatar">
                  <div v-else class="preview-icon">👤</div>
                  <span class="preview-name">{{ member.name || member.id }}</span>
                  <span class="preview-id">{{ member.id }}</span>
                </div>
              </div>
            </div>

            <!-- 空状态 -->
            <div class="import-empty" v-else-if="!importLoading && (importSourceRoleId || importGuildId || importSource === 'authority')">
              <span>暂无可导入的成员</span>
            </div>

            <!-- 加载中 -->
            <div class="import-loading" v-if="importLoading">
              <span>加载中...</span>
            </div>
          </div>
          <div class="modal-footer">
            <button class="secondary-btn" @click="closeImportDialog">取消</button>
            <button
              class="primary-btn"
              @click="doImportMembers"
              :disabled="selectedImportIds.size === 0 || importLoading"
            >导入 ({{ selectedImportIds.size }})</button>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, watch } from 'vue'
import { authApi } from '../api'
import type { Role, PermissionNode, RoleMember } from '../types'
import { message } from '@koishijs/client'
import { errorMessage } from '../utils/error-message'
import { copyTextToClipboard } from '../utils/clipboard'
import { useActionError } from '../composables/use-action-error'
import { useConfirm } from '../composables/use-confirm'
import ConfirmDialog from './primitives/ConfirmDialog.vue'

type RoleOperation = '' | 'create' | 'clone' | 'delete'

// 创建默认角色对象
const createDefaultRole = (): Role => ({
  id: '',
  name: '',
  alias: '',
  color: '#999999',
  priority: 0,
  permissions: [],
  guildIds: []
})

// 状态
const roles = ref<Role[]>([])
const permissions = ref<PermissionNode[]>([])
const currentRole = ref<Role | null>(null)
const editingRole = ref<Role>(createDefaultRole())
const activeTab = ref<'display' | 'permissions' | 'members'>('display')
const permSearch = ref('')
const newMemberId = ref('')
const currentRoleMembers = ref<RoleMember[]>([])
const loading = ref(false)
const memberLoading = ref(false)
const loadError = ref('')
const {
  actionError,
  actionErrorTitle,
  setActionError,
  clearActionError,
} = useActionError({
  onError: (title, details) => message.error(`${title}: ${details}`),
})
const roleOperation = ref<RoleOperation>('')
const savingChanges = ref(false)
const addingMember = ref(false)
const removingMemberIds = ref(new Set<string>())
const memberSearchQuery = ref('')
let loadRequestSeq = 0
let membersLoadRequestSeq = 0

// 过滤后的角色成员列表
const filteredRoleMembers = computed(() => {
  if (!memberSearchQuery.value.trim()) {
    return currentRoleMembers.value
  }
  const query = memberSearchQuery.value.toLowerCase().trim()
  return currentRoleMembers.value.filter(m =>
    m.id.toLowerCase().includes(query) ||
    (m.name && m.name.toLowerCase().includes(query))
  )
})

// 导入对话框相关状态
const showImportDialog = ref(false)
const importSource = ref<'role' | 'authority' | 'guild-admin'>('role')
const importSourceRoleId = ref('')
const importAuthorityLevel = ref(1)
const importGuildId = ref('')
const importPreviewMembers = ref<RoleMember[]>([])
const importLoading = ref(false)
const {
  actionError: importError,
  actionErrorTitle: importErrorTitle,
  setActionError: setImportError,
  clearActionError: clearImportError,
} = useActionError({
  defaultTitle: '导入失败',
  onError: (title, details) => message.error(`${title}: ${details}`),
})
const selectedImportIds = ref<Set<string>>(new Set())
let importPreviewRequestSeq = 0

// 全选状态
const isAllSelected = computed(() => {
  if (importPreviewMembers.value.length === 0) return false
  return importPreviewMembers.value.every(m => selectedImportIds.value.has(m.id))
})

// 切换单个成员选择
const toggleMemberSelect = (id: string) => {
  const newSet = new Set(selectedImportIds.value)
  if (newSet.has(id)) {
    newSet.delete(id)
  } else {
    newSet.add(id)
  }
  selectedImportIds.value = newSet
}

// 全选/取消全选
const toggleSelectAll = () => {
  if (isAllSelected.value) {
    selectedImportIds.value = new Set()
  } else {
    selectedImportIds.value = new Set(importPreviewMembers.value.map(m => m.id))
  }
}

// 快速导航相关
const permissionsMainRef = ref<HTMLElement | null>(null)
const groupRefs = ref<Record<string, HTMLElement | null>>({})
const activeGroup = ref<string>('')

const setGroupRef = (name: string, el: unknown) => {
  groupRefs.value[name] = el as HTMLElement | null
}

const scrollToGroup = (name: string) => {
  const el = groupRefs.value[name]
  if (el && permissionsMainRef.value) {
    // 使用平滑滚动到目标分组
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    activeGroup.value = name
  }
}

// 监听滚动以更新当前激活的分组
const handlePermissionsScroll = () => {
  if (!permissionsMainRef.value) return
  
  const container = permissionsMainRef.value
  const scrollTop = container.scrollTop
  
  let currentActive = ''
  for (const [name, el] of Object.entries(groupRefs.value)) {
    if (el) {
      const offsetTop = el.offsetTop - container.offsetTop
      if (scrollTop >= offsetTop - 50) {
        currentActive = name
      }
    }
  }
  
  if (currentActive) {
    activeGroup.value = currentActive
  }
}

// 群组范围模式 - 使用独立的 ref 避免空数组时状态回弹
const scopeMode = ref<'global' | 'guilds'>('global')

// 群组 ID 文本（用于编辑）
const guildIdsText = computed({
  get: () => (editingRole.value.guildIds || []).join('\n'),
  set: (val: string) => {
    const ids = val.split('\n').map(s => s.trim()).filter(Boolean)
    editingRole.value = { ...editingRole.value, guildIds: ids }
  }
})

// 监听 scopeMode 变化，同步 guildIds
watch(scopeMode, (newVal) => {
  if (newVal === 'global') {
    // 切换到全局时清空群组列表
    editingRole.value = { ...editingRole.value, guildIds: [] }
  }
})

const {
  state: confirmDialog,
  confirm,
  accept: acceptConfirm,
  cancel: cancelConfirm,
} = useConfirm()

function setLoadError(title: string, cause: unknown, fallback: string): void {
  const details = errorMessage(cause, fallback)
  loadError.value = details
  message.error(`${title}: ${details}`)
}

// 获取数据
const fetchData = async () => {
  const requestSeq = ++loadRequestSeq
  loading.value = true
  loadError.value = ''
  clearActionError()
  try {
    const [nextRoles, nextPermissions] = await Promise.all([
      authApi.getRoles(),
      authApi.getPermissions(),
    ])
    if (requestSeq !== loadRequestSeq) return
    roles.value = nextRoles
    permissions.value = nextPermissions
  } catch (e) {
    if (requestSeq !== loadRequestSeq) return
    setLoadError(
      currentRole.value ? '刷新角色权限数据失败' : '加载角色权限数据失败',
      e,
      '加载角色权限数据失败',
    )
  } finally {
    if (requestSeq === loadRequestSeq) {
      loading.value = false
    }
  }
}

onMounted(() => {
  fetchData()
  
  // 延迟添加滚动监听器（等待 DOM 渲染）
  setTimeout(() => {
    if (permissionsMainRef.value) {
      permissionsMainRef.value.addEventListener('scroll', handlePermissionsScroll)
    }
  }, 100)
})

// 计算属性
const hasChanges = computed(() => {
  if (!currentRole.value) return false
  // 使用更可靠的比较方式
  const original = JSON.stringify({
    name: currentRole.value.name,
    alias: currentRole.value.alias || '',
    color: currentRole.value.color,
    priority: currentRole.value.priority,
    permissions: currentRole.value.permissions || [],
    guildIds: currentRole.value.guildIds || []
  })
  const current = JSON.stringify({
    name: editingRole.value.name,
    alias: editingRole.value.alias || '',
    color: editingRole.value.color,
    priority: editingRole.value.priority,
    permissions: editingRole.value.permissions || [],
    guildIds: editingRole.value.guildIds || []
  })
  return original !== current
})

const groupedPermissions = computed(() => {
  const result: Record<string, PermissionNode[]> = {}
  const lower = permSearch.value.toLowerCase()
  
  const filtered = permissions.value.filter(p =>
    !lower ||
    p.name.toLowerCase().includes(lower) ||
    p.id.toLowerCase().includes(lower) ||
    p.description.toLowerCase().includes(lower)
  )

  filtered.forEach(p => {
    // 使用权限节点自带的 group 属性，如果没有则根据 id 前缀推断
    let groupName = p.group || '通用'
    if (!p.group) {
      if (p.id === '*') groupName = '系统'
      else if (p.id.endsWith('.*')) groupName = '通配符'
    }
    
    if (!result[groupName]) result[groupName] = []
    result[groupName].push(p)
  })
  
  // 对每个分组内的权限进行排序：通配符权限排在前面
  for (const group in result) {
    result[group].sort((a, b) => {
      const aIsWildcard = a.id.endsWith('.*') || a.id === '*'
      const bIsWildcard = b.id.endsWith('.*') || b.id === '*'
      if (aIsWildcard && !bIsWildcard) return -1
      if (!aIsWildcard && bIsWildcard) return 1
      return a.id.localeCompare(b.id)
    })
  }
  
  return result
})

// 方法
const fetchRoleMembers = async (roleId: string) => {
  const requestSeq = ++membersLoadRequestSeq
  memberLoading.value = true
  clearActionError()
  try {
    const nextMembers = await authApi.getRoleMembers(roleId, true)
    if (requestSeq !== membersLoadRequestSeq || currentRole.value?.id !== roleId) return
    currentRoleMembers.value = nextMembers
  } catch (e) {
    if (requestSeq !== membersLoadRequestSeq || currentRole.value?.id !== roleId) return
    currentRoleMembers.value = []
    setActionError('加载角色成员失败', e, '加载角色成员失败')
  } finally {
    if (requestSeq === membersLoadRequestSeq && currentRole.value?.id === roleId) {
      memberLoading.value = false
    }
  }
}

const selectRole = async (role: Role) => {
  if (hasChanges.value) {
    const confirmed = await confirm({
      title: '未保存的修改',
      message: '当前有未保存的修改，是否放弃这些更改？',
      tone: 'danger'
    })
    if (!confirmed) return
  }
  clearActionError()
  membersLoadRequestSeq += 1
  importPreviewRequestSeq += 1
  memberLoading.value = false
  importLoading.value = false
  currentRole.value = role
  currentRoleMembers.value = []
  importPreviewMembers.value = []
  selectedImportIds.value = new Set()
  // 确保 role 有所有必要的字段
  const normalizedRole: Role = {
    ...createDefaultRole(),
    ...role,
    permissions: Array.isArray(role.permissions) ? [...role.permissions] : [],
    guildIds: Array.isArray(role.guildIds) ? [...role.guildIds] : []
  }
  editingRole.value = normalizedRole
  // 同步 scopeMode
  scopeMode.value = (normalizedRole.guildIds && normalizedRole.guildIds.length > 0) ? 'guilds' : 'global'
  activeTab.value = 'display'
  memberSearchQuery.value = '' // 重置成员搜索
  fetchRoleMembers(role.id)
}

const createRole = async () => {
  if (roleOperation.value) return
  if (loading.value || loadError.value) {
    message.warning('角色权限数据尚未加载，无法创建角色')
    return
  }
  clearActionError()
  roleOperation.value = 'create'

  const newRole: Role = {
    id: Date.now().toString(),
    name: '新角色',
    alias: '',
    color: '#999999',
    priority: 1,
    permissions: [],
    guildIds: []
  }
  try {
    await authApi.updateRole(newRole)
    await fetchData()
    // 从刷新后的列表中找到新角色
    const created = roles.value.find(r => r.id === newRole.id)
    if (created) {
      selectRole(created)
    }
    message.success('角色创建成功')
  } catch (e) {
    setActionError('创建角色失败', e, '创建角色失败')
  } finally {
    roleOperation.value = ''
  }
}

const saveChanges = async () => {
  if (!currentRole.value || savingChanges.value) return
  
  clearActionError()
  savingChanges.value = true
  try {
    await authApi.updateRole(editingRole.value)
    message.success('保存成功')
    await fetchData()
    // 直接更新 currentRole，不调用 selectRole（避免触发 hasChanges 检查）
    const updated = roles.value.find(r => r.id === editingRole.value.id)
    if (updated) {
      currentRole.value = updated
      // 同步 editingRole
      editingRole.value = {
        ...createDefaultRole(),
        ...updated,
        permissions: Array.isArray(updated.permissions) ? [...updated.permissions] : [],
        guildIds: Array.isArray(updated.guildIds) ? [...updated.guildIds] : []
      }
      // 同步 scopeMode
      scopeMode.value = (updated.guildIds && updated.guildIds.length > 0) ? 'guilds' : 'global'
    }
  } catch (e) {
    setActionError('保存失败', e, '保存失败')
  } finally {
    savingChanges.value = false
  }
}

const resetChanges = async () => {
  if (!currentRole.value || savingChanges.value) return
  
  const confirmed = await confirm({
    title: '重置更改',
    message: '确定要放弃当前所有未保存的修改吗？',
    tone: 'normal'
  })
  
  if (confirmed) {
    clearActionError()
    // 直接重置 editingRole，不调用 selectRole（会触发重复确认）
    const normalizedRole: Role = {
      ...createDefaultRole(),
      ...currentRole.value,
      permissions: Array.isArray(currentRole.value.permissions) ? [...currentRole.value.permissions] : [],
      guildIds: Array.isArray(currentRole.value.guildIds) ? [...currentRole.value.guildIds] : []
    }
    editingRole.value = normalizedRole
    // 同步 scopeMode
    scopeMode.value = (normalizedRole.guildIds && normalizedRole.guildIds.length > 0) ? 'guilds' : 'global'
    message.success('已重置更改')
  }
}

const cloneRole = async () => {
  if (roleOperation.value) return
  if (!currentRole.value) {
    message.warning('请先选择一个角色')
    return
  }
  if (currentRole.value.builtin) {
    message.warning('内置角色无法克隆')
    return
  }

  const sourceRole = currentRole.value
  roleOperation.value = 'clone'
  try {
    const confirmed = await confirm({
      title: '克隆角色',
      message: `确定要克隆角色"${sourceRole.name}"吗？`,
      tone: 'normal'
    })
    if (!confirmed) return

    clearActionError()
    const baseName = sourceRole.name || '角色'
    let newName = `${baseName}（克隆）`
    const existingNames = new Set(roles.value.map(r => r.name))
    let idx = 1
    while (existingNames.has(newName)) {
      idx += 1
      newName = `${baseName}（克隆 ${idx}）`
    }

    const newRole: Role = {
      ...createDefaultRole(),
      ...sourceRole,
      id: Date.now().toString() + Math.floor(Math.random() * 10000).toString(),
      name: newName,
      alias: '', // 克隆时清空别名，避免冲突
      // 确保数组被复制，避免引用同一对象
      permissions: Array.isArray(sourceRole.permissions) ? [...sourceRole.permissions] : [],
      guildIds: Array.isArray(sourceRole.guildIds) ? [...sourceRole.guildIds] : [],
      builtin: false
    }

    await authApi.updateRole(newRole)
    message.success('克隆成功')
    await fetchData()
    const created = roles.value.find(r => r.id === newRole.id)
    if (created) {
      await selectRole(created)
    }
  } catch (e) {
    setActionError('克隆失败', e, '克隆失败')
  } finally {
    roleOperation.value = ''
  }
}

const deleteRole = async () => {
  if (!currentRole.value || roleOperation.value) return
  const roleToDelete = currentRole.value
  
  roleOperation.value = 'delete'
  try {
    const confirmed = await confirm({
      title: '删除角色',
      message: `确定要删除角色"${roleToDelete.name}"吗？此操作不可撤销。`,
      tone: 'danger'
    })
    if (!confirmed) return

    clearActionError()
    await authApi.deleteRole(roleToDelete.id)
    message.success('删除成功')
    if (currentRole.value?.id === roleToDelete.id) {
      currentRole.value = null
      editingRole.value = createDefaultRole()
    }
    await fetchData()
  } catch (e) {
    setActionError('删除失败', e, '删除失败')
  } finally {
    roleOperation.value = ''
  }
}

// 权限操作
/**
 * 检查是否拥有指定权限节点
 * 支持通配符匹配：
 * - `*` 匹配所有权限
 * - `warn.*` 匹配所有 `warn.xxx` 权限
 */
const hasPermission = (nodeId: string): boolean => {
  const perms = editingRole.value?.permissions
  if (!Array.isArray(perms)) return false
  
  // 1. 精确匹配
  if (perms.includes(nodeId)) return true
  
  // 2. 超级通配符
  if (perms.includes('*')) return true
  
  // 3. 模块级通配符匹配 (e.g. "warn.*" matches "warn.add")
  const parts = nodeId.split('.')
  if (parts.length > 1) {
    const moduleName = parts[0]
    if (perms.includes(`${moduleName}.*`)) return true
  }
  
  return false
}

/**
 * 检查是否被通配符覆盖（用于禁用开关）
 * 如果权限被通配符覆盖，返回通配符名称
 */
const isCoveredByWildcard = (nodeId: string): string | null => {
  const perms = editingRole.value?.permissions
  if (!Array.isArray(perms)) return null
  
  // 超级通配符
  if (perms.includes('*') && nodeId !== '*') return '*'
  
  // 模块级通配符
  const parts = nodeId.split('.')
  if (parts.length > 1 && !nodeId.endsWith('.*')) {
    const wildcardId = `${parts[0]}.*`
    if (perms.includes(wildcardId)) return wildcardId
  }
  
  return null
}

const togglePermission = (nodeId: string) => {
  // 确保 permissions 是数组
  const currentPerms = Array.isArray(editingRole.value.permissions)
    ? editingRole.value.permissions
    : []
  
  let newPerms: string[]
  if (currentPerms.includes(nodeId)) {
    // 移除权限
    newPerms = currentPerms.filter(p => p !== nodeId)
  } else {
    // 添加权限
    newPerms = [...currentPerms, nodeId]
  }
  
  // 使用新的对象替换整个 editingRole 以确保响应式更新
  editingRole.value = {
    ...editingRole.value,
    permissions: newPerms
  }
}

const clearPermissions = () => {
  editingRole.value = {
    ...editingRole.value,
    permissions: []
  }
}

// 成员操作
const addMember = async () => {
  if (addingMember.value) return
  if (memberLoading.value) {
    message.warning('角色成员正在加载，请稍后再操作')
    return
  }
  if (!newMemberId.value.trim()) {
    message.warning('请输入用户 ID')
    return
  }
  if (!currentRole.value) {
    message.warning('请先选择一个角色')
    return
  }
  
  const userId = newMemberId.value.trim()
  const roleId = currentRole.value.id
  
  clearActionError()
  addingMember.value = true
  try {
    await authApi.assignRole(userId, roleId)
    message.success('添加成员成功')
    newMemberId.value = ''
    await fetchRoleMembers(roleId)
  } catch (e) {
    setActionError('添加成员失败', e, '添加成员失败')
  } finally {
    addingMember.value = false
  }
}

// 包装函数，用于处理按钮点击
const handleAddMember = () => {
  addMember()
}

const removeMember = async (userId: string) => {
  if (!currentRole.value || removingMemberIds.value.has(userId)) return
  
  const roleId = currentRole.value.id
  
  clearActionError()
  removingMemberIds.value = new Set([...removingMemberIds.value, userId])
  try {
    await authApi.revokeRole(userId, roleId)
    message.success('移除成员成功')
    await fetchRoleMembers(roleId)
  } catch (e) {
    setActionError('移除成员失败', e, '移除成员失败')
  } finally {
    const nextRemoving = new Set(removingMemberIds.value)
    nextRemoving.delete(userId)
    removingMemberIds.value = nextRemoving
  }
}

// 包装函数，用于处理按钮点击
const handleRemoveMember = (userId: string) => {
  removeMember(userId)
}

// 导入相关的计算属性和方法
const otherRoles = computed(() => {
  if (!currentRole.value) return roles.value
  return roles.value.filter(r => r.id !== currentRole.value!.id && !r.builtin)
})

// 监听导入来源变化，切换时清空并自动加载
watch(importSource, (newSource) => {
  // 切换来源时清空预览列表
  importPreviewRequestSeq += 1
  importLoading.value = false
  clearImportError()
  importPreviewMembers.value = []
  selectedImportIds.value = new Set()
  importSourceRoleId.value = ''
  importGuildId.value = ''

  // 如果切换到 authority，自动加载当前选中等级的用户
  if (newSource === 'authority') {
    loadAuthorityUsers()
  }
})

const openImportDialog = () => {
  if (memberLoading.value) {
    message.warning('角色成员正在加载，请稍后再导入')
    return
  }
  clearImportError()
  showImportDialog.value = true
}

const closeImportDialog = () => {
  importPreviewRequestSeq += 1
  showImportDialog.value = false
  clearImportError()
  importSource.value = 'role'
  importSourceRoleId.value = ''
  importAuthorityLevel.value = 1
  importGuildId.value = ''
  importPreviewMembers.value = []
  importLoading.value = false
  selectedImportIds.value = new Set()
}

const loadImportPreview = async () => {
  const sourceRoleId = importSourceRoleId.value
  const targetRoleId = currentRole.value?.id ?? ''
  const requestSeq = ++importPreviewRequestSeq
  if (!sourceRoleId || !targetRoleId) {
    importPreviewMembers.value = []
    selectedImportIds.value = new Set()
    importLoading.value = false
    return
  }

  importLoading.value = true
  clearImportError()
  try {
    const members = await authApi.getRoleMembers(sourceRoleId, true)
    if (
      requestSeq !== importPreviewRequestSeq ||
      !showImportDialog.value ||
      importSource.value !== 'role' ||
      importSourceRoleId.value !== sourceRoleId ||
      currentRole.value?.id !== targetRoleId
    ) return
    // 过滤掉已经是当前角色成员的用户
    const existingIds = new Set(currentRoleMembers.value.map(m => m.id))
    importPreviewMembers.value = members.filter(m => !existingIds.has(m.id))
    // 默认全选
    selectedImportIds.value = new Set(importPreviewMembers.value.map(m => m.id))
  } catch (e) {
    if (
      requestSeq !== importPreviewRequestSeq ||
      !showImportDialog.value ||
      importSource.value !== 'role' ||
      importSourceRoleId.value !== sourceRoleId ||
      currentRole.value?.id !== targetRoleId
    ) return
    setImportError('加载成员列表失败', e, '加载成员列表失败')
    importPreviewMembers.value = []
    selectedImportIds.value = new Set()
  } finally {
    if (requestSeq === importPreviewRequestSeq) {
      importLoading.value = false
    }
  }
}

const loadAuthorityUsers = async () => {
  const authorityLevel = Number(importAuthorityLevel.value)
  const targetRoleId = currentRole.value?.id ?? ''
  const requestSeq = ++importPreviewRequestSeq
  if (!targetRoleId) {
    importPreviewMembers.value = []
    selectedImportIds.value = new Set()
    importLoading.value = false
    return
  }

  importLoading.value = true
  clearImportError()
  try {
    const members = await authApi.getUsersByAuthority(authorityLevel)
    if (
      requestSeq !== importPreviewRequestSeq ||
      !showImportDialog.value ||
      importSource.value !== 'authority' ||
      Number(importAuthorityLevel.value) !== authorityLevel ||
      currentRole.value?.id !== targetRoleId
    ) return
    // 过滤掉已经是当前角色成员的用户
    const existingIds = new Set(currentRoleMembers.value.map(m => m.id))
    importPreviewMembers.value = members.filter(m => !existingIds.has(m.id))
    // 默认全选
    selectedImportIds.value = new Set(importPreviewMembers.value.map(m => m.id))
  } catch (e) {
    if (
      requestSeq !== importPreviewRequestSeq ||
      !showImportDialog.value ||
      importSource.value !== 'authority' ||
      Number(importAuthorityLevel.value) !== authorityLevel ||
      currentRole.value?.id !== targetRoleId
    ) return
    setImportError('加载用户列表失败', e, '加载用户列表失败')
    importPreviewMembers.value = []
    selectedImportIds.value = new Set()
  } finally {
    if (requestSeq === importPreviewRequestSeq) {
      importLoading.value = false
    }
  }
}

const loadGuildAdmins = async () => {
  const guildId = importGuildId.value.trim()
  const targetRoleId = currentRole.value?.id ?? ''
  const requestSeq = ++importPreviewRequestSeq
  if (!guildId) {
    message.warning('请输入群号')
    importLoading.value = false
    return
  }
  if (!targetRoleId) {
    importPreviewMembers.value = []
    selectedImportIds.value = new Set()
    importLoading.value = false
    return
  }

  importLoading.value = true
  clearImportError()
  try {
    const members = await authApi.getGuildAdmins(guildId)
    if (
      requestSeq !== importPreviewRequestSeq ||
      !showImportDialog.value ||
      importSource.value !== 'guild-admin' ||
      importGuildId.value.trim() !== guildId ||
      currentRole.value?.id !== targetRoleId
    ) return
    // 过滤掉已经是当前角色成员的用户
    const existingIds = new Set(currentRoleMembers.value.map(m => m.id))
    importPreviewMembers.value = members.filter(m => !existingIds.has(m.id))
    // 默认全选
    selectedImportIds.value = new Set(importPreviewMembers.value.map(m => m.id))
  } catch (e) {
    if (
      requestSeq !== importPreviewRequestSeq ||
      !showImportDialog.value ||
      importSource.value !== 'guild-admin' ||
      importGuildId.value.trim() !== guildId ||
      currentRole.value?.id !== targetRoleId
    ) return
    setImportError('获取群管理员失败', e, '获取群管理员失败')
    importPreviewMembers.value = []
    selectedImportIds.value = new Set()
  } finally {
    if (requestSeq === importPreviewRequestSeq) {
      importLoading.value = false
    }
  }
}

const doImportMembers = async () => {
  if (!currentRole.value || selectedImportIds.value.size === 0) return
  const roleId = currentRole.value.id

  importLoading.value = true
  clearImportError()
  try {
    const userIds = Array.from(selectedImportIds.value)
    const result = await authApi.importMembers(roleId, userIds)
    message.success(`成功导入 ${result.imported} 个成员`)
    closeImportDialog()
    // 刷新成员列表
    if (currentRole.value?.id === roleId) {
      await fetchRoleMembers(roleId)
    }
  } catch (e) {
    setImportError('导入失败', e, '导入失败')
  } finally {
    importLoading.value = false
  }
}

// 拖拽排序
const onDragStart = (e: DragEvent, role: Role) => {
    if (e.dataTransfer) {
        e.dataTransfer.setData('text/plain', role.id)
        e.dataTransfer.effectAllowed = 'move'
    }
}

const onDrop = async (e: DragEvent, targetRole: Role) => {
    const draggedId = e.dataTransfer?.getData('text/plain')
    if (!draggedId || draggedId === targetRole.id) return
    
    const draggedRole = roles.value.find(r => r.id === draggedId)
    if (draggedRole) {
      clearActionError()
      try {
        // 交换 priority
        const temp = draggedRole.priority
        draggedRole.priority = targetRole.priority
        targetRole.priority = temp
        
        await authApi.updateRole(draggedRole)
        await authApi.updateRole(targetRole)
        await fetchData()
      } catch (error) {
        await fetchData()
        setActionError('角色排序失败', error, '角色排序失败')
      }
    }
}

// 复制角色 ID
const copyRoleId = async () => {
  if (!currentRole.value) return
  const roleId = currentRole.value.id
  if (await copyTextToClipboard(roleId)) {
    message.success('角色 ID 已复制到剪贴板')
    return
  }
  message.error('复制角色 ID 失败，请手动复制')
}

</script>

<style scoped>
/* ========================================
   GitHub Dimmed / Vercel 风格 RolesView
   硬核专业高信噪比
   ======================================== */

.roles-view-container {
  display: flex;
  height: 100%;
  max-height: 100%;
  background: var(--bg2, #252529);
  color: var(--fg1, rgba(255, 255, 245, .9));
  overflow: hidden;
  border-radius: 8px;
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  font-family: -apple-system, BlinkMacSystemFont, 'Inter', 'Segoe UI', sans-serif;
}

/* 侧边栏 */
.sidebar {
  width: 220px;
  background: var(--bg1, #1e1e20);
  border-right: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  padding: 0.875rem 1rem;
  border-bottom: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.sidebar-header h2 {
  margin: 0;
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--fg3, rgba(255, 255, 245, .4));
}

.role-list {
  flex: 1;
  overflow-y: auto;
  padding: 0.375rem;
}

/* 滚动条 - 细微克制 */
.role-list::-webkit-scrollbar {
  width: 4px;
}

.role-list::-webkit-scrollbar-track {
  background: transparent;
}

.role-list::-webkit-scrollbar-thumb {
  background: var(--k-color-divider, rgba(82, 82, 89, 0.5));
  border-radius: 2px;
}

.role-list::-webkit-scrollbar-thumb:hover {
  background: var(--fg3, rgba(255, 255, 245, .4));
}

/* 角色项 */
.role-item {
  display: flex;
  align-items: center;
  padding: 0.5rem 0.625rem;
  margin-bottom: 1px;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.15s ease;
}

.role-item:hover {
  background: var(--bg3, #313136);
}

.role-item.active {
  background: var(--k-color-primary-fade, rgba(116, 89, 255, 0.1));
  border-left: 2px solid var(--k-color-primary, #7459ff);
  margin-left: -2px;
}

/* 角色颜色指示器 - 实心小圆点 */
.role-color {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 8px;
  flex-shrink: 0;
}

.role-name {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 0.8rem;
  font-weight: 500;
  color: var(--fg2, rgba(255, 255, 245, .6));
}

.role-item.active .role-name {
  color: var(--fg1, rgba(255, 255, 245, .9));
}

.builtin-icon {
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-size: 10px;
}

.drag-handle {
  color: var(--fg3, rgba(255, 255, 245, .4));
  cursor: grab;
  font-size: 12px;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.role-item:hover .drag-handle {
  opacity: 1;
}

/* 主内容区 */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
  background: var(--bg2, #252529);
}

.roles-load-banner,
.roles-action-banner {
  margin: 0;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid rgba(248, 81, 73, 0.28);
  background: rgba(248, 81, 73, 0.08);
  color: var(--fg1, rgba(255, 255, 245, .9));
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.roles-action-banner--modal {
  margin-bottom: 1rem;
  border: 1px solid rgba(248, 81, 73, 0.28);
  border-radius: 6px;
}

.roles-load-banner__body,
.roles-action-banner__body,
.roles-load-error__body {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.roles-load-banner__body strong,
.roles-action-banner__body strong,
.roles-load-error__body strong {
  font-size: 0.8rem;
  color: #ff8a80;
}

.roles-load-banner__body span,
.roles-action-banner__body span,
.roles-load-error__body span {
  color: var(--fg2, rgba(255, 255, 245, .6));
  font-size: 0.75rem;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.roles-load-error {
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding: 1.5rem;
  text-align: center;
}

.roles-load-error__body {
  max-width: 520px;
  padding: 1rem;
  border: 1px solid rgba(248, 81, 73, 0.28);
  border-radius: 6px;
  background: rgba(248, 81, 73, 0.08);
}

.roles-load-empty {
  align-items: center;
  justify-content: center;
}

.content-header {
  padding: 1rem 1.25rem;
  border-bottom: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.content-header h1 {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--fg1, rgba(255, 255, 245, .9));
}

.builtin-badge {
  font-size: 0.6rem;
  padding: 2px 6px;
  background: var(--bg3, #313136);
  color: var(--fg3, rgba(255, 255, 245, .4));
  border-radius: 3px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
}

.builtin-notice {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 0.75rem 1rem;
  background: var(--bg3, #313136);
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  border-radius: 6px;
  margin-bottom: 1.25rem;
  color: var(--fg2, rgba(255, 255, 245, .6));
  font-size: 0.8rem;
  line-height: 1.5;
}

.builtin-notice k-icon {
  color: var(--k-color-primary, #7459ff);
  font-size: 14px;
  margin-top: 1px;
  flex-shrink: 0;
}

/* Tab 导航 */
.tabs {
  display: flex;
  padding: 0 1.25rem;
  border-bottom: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  background: var(--bg1, #1e1e20);
}

.tab-item {
  padding: 0.625rem 1rem;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.15s ease;
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-size: 0.8rem;
  font-weight: 500;
}

.tab-item:hover {
  color: var(--fg2, rgba(255, 255, 245, .6));
}

.tab-item.active {
  border-bottom-color: var(--k-color-primary, #7459ff);
  color: var(--fg1, rgba(255, 255, 245, .9));
}

.tab-content {
  flex: 1;
  padding: 1.25rem;
  overflow-y: auto;
  padding-bottom: 80px;
}

/* Tab 内容滚动条 */
.tab-content::-webkit-scrollbar {
  width: 4px;
}

.tab-content::-webkit-scrollbar-track {
  background: transparent;
}

.tab-content::-webkit-scrollbar-thumb {
  background: var(--k-color-divider, rgba(82, 82, 89, 0.5));
  border-radius: 2px;
}

.tab-content::-webkit-scrollbar-thumb:hover {
  background: var(--fg3, rgba(255, 255, 245, .4));
}

/* 表单组件 */
.form-group {
  margin-bottom: 1.25rem;
  max-width: 480px;
}

.form-group label {
  display: block;
  margin-bottom: 0.375rem;
  font-weight: 500;
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.form-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  border-radius: 4px;
  background: var(--bg1, #1e1e20);
  color: var(--fg1, rgba(255, 255, 245, .9));
  font-family: inherit;
  font-size: 0.8rem;
  transition: border-color 0.15s ease;
}

.form-input:focus {
  outline: none;
  border-color: var(--k-color-primary, #7459ff);
}

.form-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* 颜色选择器 */
.color-picker-wrapper {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  background: var(--bg1, #1e1e20);
  padding: 6px;
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  border-radius: 4px;
  width: fit-content;
}

.color-input {
  width: 32px;
  height: 32px;
  padding: 0;
  border: none;
  background: none;
  cursor: pointer;
  border-radius: 4px;
}

.color-text {
  border: none !important;
  background: transparent !important;
  padding: 0 !important;
  width: 80px;
  font-family: 'JetBrains Mono', 'SF Mono', Consolas, monospace;
  font-size: 0.75rem;
}

/* 范围选项 */
.scope-options {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.radio-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  color: var(--fg2, rgba(255, 255, 245, .6));
  font-size: 0.8rem;
}

.radio-label input[type="radio"] {
  margin: 0;
  accent-color: var(--k-color-primary, #7459ff);
}

.scope-readonly {
  padding: 0.5rem 0.75rem;
  background: var(--bg1, #1e1e20);
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  border-radius: 4px;
}

.scope-badge {
  font-size: 0.8rem;
  color: var(--fg3, rgba(255, 255, 245, .4));
}

.form-textarea {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  border-radius: 4px;
  background: var(--bg1, #1e1e20);
  color: var(--fg1, rgba(255, 255, 245, .9));
  font-family: 'JetBrains Mono', 'SF Mono', Consolas, monospace;
  font-size: 0.75rem;
  resize: vertical;
  min-height: 72px;
}

.form-textarea:focus {
  outline: none;
  border-color: var(--k-color-primary, #7459ff);
}

/* 角色 ID 显示 */
.id-display {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0.375rem 0.625rem;
  background: var(--bg1, #1e1e20);
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  border-radius: 4px;
  width: fit-content;
}

.role-id-code {
  font-family: 'JetBrains Mono', 'SF Mono', Consolas, monospace;
  font-size: 0.75rem;
  color: var(--k-color-primary, #7459ff);
  background: transparent;
  padding: 0;
  user-select: all;
}

.copy-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 2px;
  font-size: 12px;
  opacity: 0.5;
  transition: opacity 0.15s ease;
}

.copy-btn:hover {
  opacity: 1;
}

.field-hint {
  margin-top: 4px;
  font-size: 0.7rem;
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-family: 'JetBrains Mono', 'SF Mono', Consolas, monospace;
}

/* 当前已选权限显示 */
.current-perms {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
  margin-bottom: 1rem;
  padding: 0.75rem;
  background: var(--bg1, #1e1e20);
  border-radius: 4px;
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
}

.perms-label {
  font-size: 0.7rem;
  color: var(--fg3, rgba(255, 255, 245, .4));
  margin-right: 6px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.perm-tag {
  padding: 2px 8px;
  background: rgba(63, 185, 80, 0.15);
  color: #3fb950;
  border-radius: 3px;
  font-size: 0.7rem;
  font-family: 'JetBrains Mono', 'SF Mono', Consolas, monospace;
  border: 1px solid rgba(63, 185, 80, 0.3);
}

/* 权限列表搜索栏 */
.search-bar {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.search-input {
  flex: 1;
}

/* 权限布局 */
.permissions-layout {
  display: flex;
  gap: 1.25rem;
  height: 100%;
}

.permissions-main {
  flex: 1;
  overflow-y: auto;
  padding-right: 0.75rem;
}

.permissions-main::-webkit-scrollbar {
  width: 4px;
}

.permissions-main::-webkit-scrollbar-track {
  background: transparent;
}

.permissions-main::-webkit-scrollbar-thumb {
  background: var(--k-color-divider, rgba(82, 82, 89, 0.5));
  border-radius: 2px;
}

/* 快速导航 */
.permissions-nav {
  width: 160px;
  flex-shrink: 0;
  position: sticky;
  top: 0;
  align-self: flex-start;
  max-height: calc(100vh - 300px);
  overflow-y: auto;
}

.nav-title {
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--fg3, rgba(255, 255, 245, .4));
  margin-bottom: 0.5rem;
  padding-left: 10px;
}

.nav-list {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s ease;
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-size: 0.75rem;
}

.nav-item:hover {
  background: var(--bg3, #313136);
  color: var(--fg2, rgba(255, 255, 245, .6));
}

.nav-item.active {
  background: var(--k-color-primary-fade, rgba(116, 89, 255, 0.1));
  color: var(--k-color-primary, #7459ff);
}

.nav-dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--fg3, rgba(255, 255, 245, .4));
  flex-shrink: 0;
}

.nav-item.active .nav-dot {
  background: var(--k-color-primary, #7459ff);
}

.nav-name {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.nav-count {
  font-size: 0.65rem;
  font-family: 'JetBrains Mono', 'SF Mono', Consolas, monospace;
  padding: 1px 5px;
  background: var(--bg3, #313136);
  border-radius: 8px;
  color: var(--fg3, rgba(255, 255, 245, .4));
}

.nav-item.active .nav-count {
  background: var(--k-color-primary, #7459ff);
  color: #fff;
}

@media (max-width: 900px) {
  .permissions-nav {
    display: none;
  }
  .permissions-main {
    padding-right: 0;
  }
}

/* 权限分组 */
.permission-groups {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.group-header {
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--fg3, rgba(255, 255, 245, .4));
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 0.5rem;
  padding-bottom: 0.25rem;
  border-bottom: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
}

/* 权限项 - hover 效果 */
.permission-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem;
  background: var(--bg1, #1e1e20);
  border-radius: 6px;
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  margin-bottom: 0.375rem;
  transition: all 0.15s ease;
}

.permission-item:hover {
  border-color: var(--k-color-border, rgba(82, 82, 89, 0.8));
  background: var(--bg3, #313136);
}

.perm-info .perm-name {
  font-weight: 500;
  font-size: 0.85rem;
  color: var(--fg1, rgba(255, 255, 245, .9));
}

.perm-id {
  font-family: 'JetBrains Mono', 'SF Mono', Consolas, monospace;
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-size: 0.7rem;
  margin-top: 3px;
}

.perm-desc {
  font-size: 0.75rem;
  color: var(--fg2, rgba(255, 255, 245, .6));
  margin-top: 3px;
}

/* Toggle 开关 - 更简洁 */
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 36px;
  height: 20px;
  cursor: pointer;
  flex-shrink: 0;
}

.toggle-switch .slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--bg3, #313136);
  transition: all 0.15s ease;
  border-radius: 10px;
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
}

.toggle-switch .slider:before {
  position: absolute;
  content: "";
  height: 14px;
  width: 14px;
  left: 2px;
  bottom: 2px;
  background: var(--fg3, rgba(255, 255, 245, .4));
  transition: all 0.15s ease;
  border-radius: 50%;
}

.toggle-switch.active .slider {
  background: rgba(63, 185, 80, 0.2);
  border-color: rgba(63, 185, 80, 0.4);
}

.toggle-switch.active .slider:before {
  transform: translateX(16px);
  background: #3fb950;
}

.toggle-switch:hover .slider {
  border-color: var(--k-color-border, rgba(82, 82, 89, 0.8));
}

/* 锁定状态 */
.toggle-switch.locked {
  cursor: not-allowed;
  opacity: 0.6;
}

.toggle-switch .lock-icon {
  position: absolute;
  right: -18px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 10px;
}

.permission-item.covered {
  opacity: 0.6;
}

.covered-hint {
  color: #3fb950;
  font-size: 0.65rem;
  margin-left: 6px;
  font-weight: 400;
}

/* 成员管理 */
.add-member {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 1rem;
  max-width: 400px;
}

.member-search {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1rem;
  max-width: 400px;
}

.member-search .form-input {
  flex: 1;
}

.member-count {
  font-size: 0.75rem;
  color: var(--fg3, rgba(255, 255, 245, .4));
  white-space: nowrap;
}

.member-loading {
  padding: 1.25rem;
  text-align: center;
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-size: 0.8rem;
}

.member-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 0.75rem;
}

/* 成员项 - hover 效果 */
.member-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.625rem 0.75rem;
  background: var(--bg1, #1e1e20);
  border-radius: 6px;
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  transition: all 0.15s ease;
}

.member-item:hover {
  border-color: var(--k-color-border, rgba(82, 82, 89, 0.8));
  background: var(--bg3, #313136);
}

.member-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.member-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
}

.member-icon {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--bg3, #313136);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-size: 12px;
}

.member-text {
  display: flex;
  flex-direction: column;
}

.member-name {
  font-weight: 500;
  font-size: 0.8rem;
  color: var(--fg1, rgba(255, 255, 245, .9));
}

.member-id-sub {
  font-size: 0.65rem;
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-family: 'JetBrains Mono', 'SF Mono', Consolas, monospace;
}

/* 保存浮动条 - Discord 风格 */
.save-bar {
  position: fixed;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  width: calc(100% - 48px);
  max-width: 560px;
  background: #111214;
  color: var(--fg1, rgba(255, 255, 245, .9));
  padding: 10px 12px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.4);
  z-index: 100;
  font-size: 0.8125rem;
}

.save-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.reset-btn {
  background: transparent;
  border: none;
  color: #b5bac1;
  cursor: pointer;
  padding: 4px 8px;
  font-size: 0.8125rem;
  font-weight: 500;
  transition: text-decoration 0.1s ease;
}

.reset-btn:hover {
  text-decoration: underline;
}

.reset-btn:disabled,
.reset-btn:disabled:hover {
  color: #6b7280;
  cursor: not-allowed;
  text-decoration: none;
}

.save-btn {
  background: #248046;
  border: none;
  color: #fff;
  padding: 6px 14px;
  border-radius: 3px;
  cursor: pointer;
  font-size: 0.8125rem;
  font-weight: 500;
  transition: background 0.15s ease;
}

.save-btn:hover {
  background: #1a6334;
}

.save-btn:disabled,
.save-btn:disabled:hover {
  background: #2f4b39;
  cursor: not-allowed;
}

.slide-up-enter-active,
.slide-up-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.slide-up-enter-from,
.slide-up-leave-to {
  transform: translate(-50%, 20px);
  opacity: 0;
}

.empty-state, .empty-tip {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  height: 100%;
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-size: 0.85rem;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 0.75rem;
  opacity: 0.3;
}

/* 通用按钮 - GitHub 风格 */
.icon-btn {
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 4px;
  background: var(--k-color-primary, #7459ff);
  color: #fff;
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: opacity 0.15s ease;
}

.icon-btn:hover {
  opacity: 0.85;
}

.icon-btn:disabled,
.icon-btn:disabled:hover {
  opacity: 0.45;
  cursor: not-allowed;
}

.primary-btn {
  padding: 6px 12px;
  border: 1px solid var(--k-color-primary, #7459ff);
  border-radius: 4px;
  background: rgba(116, 89, 255, 0.15);
  color: var(--k-color-primary, #7459ff);
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
}

.primary-btn:hover {
  background: rgba(116, 89, 255, 0.25);
}

.secondary-btn {
  padding: 6px 12px;
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  border-radius: 4px;
  background: transparent;
  color: var(--fg2, rgba(255, 255, 245, .6));
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
}

.secondary-btn:hover {
  border-color: var(--k-color-border, rgba(82, 82, 89, 0.8));
  color: var(--fg1, rgba(255, 255, 245, .9));
}

.danger-btn {
  padding: 6px 12px;
  border: 1px solid rgba(248, 81, 73, 0.3);
  border-radius: 4px;
  background: rgba(248, 81, 73, 0.15);
  color: #f85149;
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
}

.danger-btn:hover {
  background: rgba(248, 81, 73, 0.25);
  border-color: rgba(248, 81, 73, 0.5);
}

.clone-btn {
  padding: 6px 12px;
  border: 1px solid rgba(88, 166, 255, 0.3);
  border-radius: 4px;
  background: rgba(88, 166, 255, 0.15);
  color: #58a6ff;
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
}

.clone-btn:hover {
  background: rgba(88, 166, 255, 0.25);
  border-color: rgba(88, 166, 255, 0.5);
}

/* 模态框 - GitHub 风格 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(2px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-dialog {
  background: var(--bg2, #252529);
  border-radius: 8px;
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.4);
  min-width: 300px;
  max-width: 420px;
  overflow: hidden;
  animation: modal-enter 0.2s ease-out;
}

@keyframes modal-enter {
  from {
    opacity: 0;
    transform: scale(0.95);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

.modal-header {
  padding: 1rem;
  border-bottom: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
}

.modal-header h3 {
  margin: 0;
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--fg1, rgba(255, 255, 245, .9));
}

.modal-body {
  padding: 1rem;
}

.modal-body p {
  margin: 0;
  color: var(--fg2, rgba(255, 255, 245, .6));
  font-size: 0.8rem;
  line-height: 1.6;
}

.modal-footer {
  padding: 0.75rem 1rem;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  border-top: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  background: var(--bg1, #1e1e20);
}

/* 淡入淡出动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* 导入成员对话框 */
.import-dialog {
  min-width: 400px;
  max-width: 520px;
}

.import-source-options {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.guild-input-row {
  display: flex;
  gap: 0.5rem;
}

.guild-input-row .form-input {
  flex: 1;
}

.import-preview {
  margin-top: 1rem;
  border: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  border-radius: 6px;
  overflow: hidden;
}

.preview-header {
  padding: 0.5rem 0.75rem;
  background: var(--bg1, #1e1e20);
  border-bottom: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
  font-size: 0.75rem;
  color: var(--fg2, rgba(255, 255, 245, .6));
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
  margin: 0;
  accent-color: var(--k-color-primary, #7459ff);
  cursor: pointer;
}

.select-all {
  font-weight: 500;
}

.preview-count {
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-size: 0.7rem;
}

.preview-list {
  max-height: 200px;
  overflow-y: auto;
}

.preview-list::-webkit-scrollbar {
  width: 4px;
}

.preview-list::-webkit-scrollbar-track {
  background: transparent;
}

.preview-list::-webkit-scrollbar-thumb {
  background: var(--k-color-divider, rgba(82, 82, 89, 0.5));
  border-radius: 2px;
}

.preview-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border-bottom: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.3));
  cursor: pointer;
  transition: background 0.15s ease;
}

.preview-item:hover {
  background: var(--bg3, #313136);
}

.preview-item.selected {
  background: rgba(116, 89, 255, 0.1);
}

.preview-item.selected:hover {
  background: rgba(116, 89, 255, 0.15);
}

.preview-item:last-child {
  border-bottom: none;
}

.preview-item input[type="checkbox"] {
  margin: 0;
  accent-color: var(--k-color-primary, #7459ff);
  cursor: pointer;
  flex-shrink: 0;
}

.preview-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  object-fit: cover;
}

.preview-icon {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--bg3, #313136);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-size: 10px;
}

.preview-name {
  flex: 1;
  font-size: 0.8rem;
  color: var(--fg1, rgba(255, 255, 245, .9));
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.preview-id {
  font-size: 0.7rem;
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-family: 'JetBrains Mono', 'SF Mono', Consolas, monospace;
}

.import-empty,
.import-loading {
  padding: 2rem;
  text-align: center;
  color: var(--fg3, rgba(255, 255, 245, .4));
  font-size: 0.8rem;
}

.primary-btn:disabled,
.secondary-btn:disabled,
.clone-btn:disabled,
.danger-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ========================================
   移动端适配 (< 768px)
   ======================================== */
@media (max-width: 768px) {
  .roles-view-container {
    flex-direction: column;
    height: auto;
    max-height: none;
    min-height: calc(100vh - 80px);
  }

  /* 侧边栏变为顶部横向滚动列表 */
  .sidebar {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid var(--k-color-divider, rgba(82, 82, 89, 0.5));
    flex-shrink: 0;
  }

  .sidebar-header {
    padding: 0.75rem;
  }

  .sidebar-header h2 {
    font-size: 0.75rem;
  }

  .role-list {
    display: flex;
    flex-direction: row;
    overflow-x: auto;
    overflow-y: hidden;
    padding: 0.25rem 0.5rem;
    gap: 0.25rem;
    -webkit-overflow-scrolling: touch;
  }

  .role-list::-webkit-scrollbar {
    height: 3px;
  }

  .role-item {
    flex-shrink: 0;
    padding: 0.5rem 0.75rem;
    margin-bottom: 0;
    white-space: nowrap;
  }

  .role-item.active {
    border-left: none;
    margin-left: 0;
    border-bottom: 2px solid var(--k-color-primary, #7459ff);
  }

  .drag-handle {
    display: none;
  }

  /* 主内容区 */
  .main-content {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
  }

  .content-header {
    padding: 0.75rem 1rem;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .content-header h1 {
    font-size: 1rem;
    width: 100%;
  }

  .header-actions {
    width: 100%;
    display: flex;
    gap: 0.5rem;
  }

  .header-actions .clone-btn,
  .header-actions .danger-btn {
    flex: 1;
    text-align: center;
  }

  .header-actions span {
    display: none;
  }

  /* Tab 导航 */
  .tabs {
    padding: 0 0.75rem;
  }

  .tab-item {
    padding: 0.5rem 0.75rem;
    font-size: 0.75rem;
  }

  .tab-content {
    padding: 1rem;
    padding-bottom: 100px;
  }

  /* 表单组件 */
  .form-group {
    max-width: 100%;
  }

  .form-input {
    font-size: 16px; /* 防止 iOS 缩放 */
  }

  .form-textarea {
    font-size: 16px;
  }

  /* 颜色选择器 */
  .color-picker-wrapper {
    width: 100%;
  }

  .color-text {
    flex: 1;
  }

  /* 权限布局 */
  .permissions-layout {
    flex-direction: column;
  }

  .permissions-nav {
    display: none;
  }

  .permissions-main {
    padding-right: 0;
  }

  .search-bar {
    flex-direction: column;
  }

  .search-bar .form-input {
    width: 100%;
  }

  .search-bar .secondary-btn {
    width: 100%;
  }

  /* 当前已选权限 */
  .current-perms {
    padding: 0.5rem;
  }

  .perm-tag {
    font-size: 0.65rem;
    padding: 2px 6px;
  }

  /* 权限项 */
  .permission-item {
    padding: 0.625rem;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .perm-info {
    flex: 1;
    min-width: 0;
  }

  .perm-info .perm-name {
    font-size: 0.8rem;
  }

  .perm-desc {
    font-size: 0.7rem;
  }

  .perm-id {
    font-size: 0.65rem;
  }

  /* 成员管理 */
  .add-member {
    flex-direction: column;
    max-width: 100%;
  }

  .add-member .form-input {
    width: 100%;
  }

  .add-member .primary-btn,
  .add-member .secondary-btn {
    width: 100%;
  }

  .member-search {
    max-width: 100%;
  }

  .member-list {
    grid-template-columns: 1fr;
  }

  .member-item {
    padding: 0.5rem;
  }

  /* 保存浮动条 */
  .save-bar {
    bottom: 12px;
    width: calc(100% - 24px);
    max-width: none;
    padding: 8px 10px;
    font-size: 0.75rem;
  }

  .save-actions {
    gap: 6px;
  }

  .reset-btn {
    font-size: 0.75rem;
  }

  .save-btn {
    padding: 5px 10px;
    font-size: 0.75rem;
  }

  /* 模态框 */
  .modal-dialog {
    min-width: auto;
    max-width: calc(100vw - 32px);
    margin: 16px;
  }

  .import-dialog {
    min-width: auto;
    max-width: calc(100vw - 32px);
  }

  .modal-header {
    padding: 0.75rem;
  }

  .modal-header h3 {
    font-size: 0.85rem;
  }

  .modal-body {
    padding: 0.75rem;
  }

  .modal-footer {
    padding: 0.625rem 0.75rem;
  }

  /* 导入对话框 */
  .guild-input-row {
    flex-direction: column;
  }

  .preview-list {
    max-height: 150px;
  }

  .preview-item {
    padding: 0.375rem 0.5rem;
  }

  .preview-avatar,
  .preview-icon {
    width: 20px;
    height: 20px;
  }

  .preview-name {
    font-size: 0.75rem;
  }

  .preview-id {
    font-size: 0.65rem;
  }
}

/* 小屏手机适配 (< 480px) */
@media (max-width: 480px) {
  .role-item {
    padding: 0.4rem 0.5rem;
    font-size: 0.75rem;
  }

  .role-color {
    width: 6px;
    height: 6px;
    margin-right: 6px;
  }

  .role-name {
    font-size: 0.75rem;
  }

  .content-header h1 {
    font-size: 0.9rem;
  }

  .tab-content {
    padding: 0.75rem;
    padding-bottom: 90px;
  }

  .builtin-notice {
    padding: 0.5rem 0.75rem;
    font-size: 0.75rem;
  }

  .form-group label {
    font-size: 0.7rem;
  }

  .toggle-switch {
    width: 32px;
    height: 18px;
  }

  .toggle-switch .slider:before {
    height: 12px;
    width: 12px;
  }

  .toggle-switch.active .slider:before {
    transform: translateX(14px);
  }
}
</style>
