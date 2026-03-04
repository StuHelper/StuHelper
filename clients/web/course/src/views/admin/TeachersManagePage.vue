<template>
  <div class="space-y-4">
    <!-- Toolbar -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary">{{ t('admin.teachers.title') }}</h1>
      <div class="flex items-center gap-2 w-full sm:w-auto">
        <div class="relative flex-1 sm:flex-initial">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-text-muted pointer-events-none" />
          <input
            v-model="searchQuery"
            type="text"
            :placeholder="t('admin.teachers.searchPlaceholder')"
            class="w-full sm:w-56 pl-9 pr-3 py-2 text-sm bg-bg-card border border-border rounded-lg text-text-primary placeholder:text-text-muted transition-colors duration-fast focus:border-primary focus:outline-none"
          />
        </div>
        <select
          v-model="filterDeptID"
          class="px-3 py-2 text-sm bg-bg-card border border-border rounded-lg text-text-primary"
        >
          <option :value="0">{{ t('admin.teachers.allDepartments') }}</option>
          <option v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</option>
        </select>
        <button
          class="flex items-center gap-1.5 px-3 py-2 text-sm text-text-inverse bg-primary rounded-lg transition-colors duration-fast hover:bg-primary/90"
          @click="openForm()"
        >
          <Plus class="size-3.5" />
          <span class="hidden sm:inline">{{ t('admin.teachers.add') }}</span>
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="bg-bg-card border border-border rounded-xl shadow-card overflow-hidden">
      <div v-for="i in 5" :key="i" class="h-14 border-b border-border animate-pulse" />
    </div>

    <!-- Table -->
    <div v-else-if="teachers.length > 0" class="bg-bg-card border border-border rounded-xl shadow-card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-bg-secondary">
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted">{{ t('admin.teachers.name') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted hidden sm:table-cell">{{ t('admin.teachers.department') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted hidden md:table-cell">{{ t('admin.teachers.reviewCount') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted hidden md:table-cell">{{ t('admin.teachers.time') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted">{{ t('admin.teachers.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="teacher in teachers"
              :key="teacher.id"
              class="border-t border-border transition-colors duration-fast hover:bg-bg-hover"
            >
              <td class="p-3 text-sm text-text-primary font-medium">{{ teacher.name }}</td>
              <td class="p-3 text-sm text-text-secondary hidden sm:table-cell">{{ teacher.departmentName || '-' }}</td>
              <td class="p-3 text-sm text-text-secondary hidden md:table-cell">{{ teacher.reviewCount }}</td>
              <td class="p-3 text-xs text-text-muted hidden md:table-cell whitespace-nowrap">{{ formatAbsolute(teacher.createdAt) }}</td>
              <td class="p-3">
                <div class="flex items-center gap-1">
                  <button
                    class="p-1.5 text-text-muted rounded-md transition-colors duration-fast hover:bg-primary/10 hover:text-primary"
                    :title="t('admin.teachers.edit')"
                    @click="openForm(teacher)"
                  >
                    <Pencil class="size-4" />
                  </button>
                  <button
                    class="p-1.5 text-text-muted rounded-md transition-colors duration-fast hover:bg-danger/10 hover:text-danger"
                    :title="t('admin.teachers.delete')"
                    @click="handleDelete(teacher)"
                  >
                    <Trash2 class="size-4" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <EmptyState v-else :title="t('admin.teachers.empty')" />

    <!-- Pagination -->
    <div v-if="total > 0" class="flex flex-col sm:flex-row items-center justify-between gap-3 text-sm">
      <span class="text-text-muted">{{ t('admin.pagination.total', { total }) }}</span>
      <div class="flex items-center gap-1">
        <button
          class="px-3 py-1.5 border border-border rounded-lg text-text-secondary transition-colors duration-fast hover:bg-bg-hover disabled:opacity-40 disabled:cursor-not-allowed"
          :disabled="page <= 1"
          @click="page--"
        >{{ t('admin.pagination.prev') }}</button>
        <template v-for="p in visiblePages" :key="p">
          <span v-if="p === '...'" class="px-2 text-text-muted">...</span>
          <button
            v-else
            class="min-w-[36px] h-9 px-2 rounded-lg text-sm font-medium transition-colors duration-fast"
            :class="p === page ? 'bg-primary text-text-inverse' : 'text-text-secondary hover:bg-bg-hover'"
            @click="page = p as number"
          >{{ p }}</button>
        </template>
        <button
          class="px-3 py-1.5 border border-border rounded-lg text-text-secondary transition-colors duration-fast hover:bg-bg-hover disabled:opacity-40 disabled:cursor-not-allowed"
          :disabled="page >= totalPages"
          @click="page++"
        >{{ t('admin.pagination.next') }}</button>
      </div>
    </div>

    <!-- Add/Edit Modal -->
    <div v-if="formOpen" class="fixed inset-0 bg-bg-overlay z-[var(--z-modal-backdrop)] flex items-center justify-center p-4" @click.self="formOpen = false">
      <div class="bg-bg-card border border-border rounded-xl shadow-lg w-full max-w-md p-6 space-y-4">
        <h2 class="text-lg font-bold text-text-primary">{{ editingTeacher ? t('admin.teachers.edit') : t('admin.teachers.add') }}</h2>
        <div class="space-y-3">
          <div>
            <label class="block text-sm text-text-muted mb-1">{{ t('admin.teachers.name') }}</label>
            <input
              v-model="formName"
              type="text"
              :placeholder="t('admin.teachers.namePlaceholder')"
              class="w-full px-3 py-2 text-sm bg-bg-card border border-border rounded-lg text-text-primary placeholder:text-text-muted focus:border-primary focus:outline-none"
            />
          </div>
          <div>
            <label class="block text-sm text-text-muted mb-1">{{ t('admin.teachers.department') }}</label>
            <select
              v-model="formDeptID"
              class="w-full px-3 py-2 text-sm bg-bg-card border border-border rounded-lg text-text-primary"
            >
              <option :value="undefined">{{ t('admin.teachers.departmentPlaceholder') }}</option>
              <option v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</option>
            </select>
          </div>
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <button
            class="px-4 py-2 text-sm text-text-secondary border border-border rounded-lg transition-colors duration-fast hover:bg-bg-hover"
            @click="formOpen = false"
          >{{ t('common.actions.cancel') }}</button>
          <button
            class="px-4 py-2 text-sm text-text-inverse bg-primary rounded-lg transition-colors duration-fast hover:bg-primary/90 disabled:opacity-50"
            :disabled="!formName.trim() || saving"
            @click="handleSave"
          >{{ saving ? '...' : (editingTeacher ? t('admin.teachers.edit') : t('admin.teachers.add')) }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Search, Plus, Pencil, Trash2 } from 'lucide-vue-next'
import { getAdminTeachers, createTeacher, updateTeacher, deleteTeacher } from '@/api/admin'
import { getDepartments } from '@/api/course'
import type { AdminTeacher } from '@/types/admin'
import type { Department } from '@/types/course'
import EmptyState from '@/components/common/EmptyState.vue'
import { useToast } from '@/composables/useToast'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatAbsoluteTime } from '@/utils/date'

const { t, locale } = useI18n()
const toast = useToast()

const departments = ref<Department[]>([])
const page = ref(1)
const pageSize = ref(20)
const searchQuery = ref('')
const filterDeptID = ref(0)

const { data: teachersData, loading, execute: fetchTeachers } = useAsyncData(async () => {
  const res = await getAdminTeachers(page.value, pageSize.value, searchQuery.value, filterDeptID.value || undefined)
  return {
    teachers: res.data?.list || [],
    total: res.data?.total || 0
  }
})

const teachers = computed(() => teachersData.value?.teachers || [])
const total = computed(() => teachersData.value?.total || 0)

// Form state
const formOpen = ref(false)
const formName = ref('')
const formDeptID = ref<number | undefined>(undefined)
const editingTeacher = ref<AdminTeacher | null>(null)
const saving = ref(false)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

const visiblePages = computed(() => {
  const tp = totalPages.value
  if (tp <= 5) return Array.from({ length: tp }, (_, i) => i + 1)
  const pages: (number | string)[] = []
  if (page.value <= 3) {
    pages.push(1, 2, 3, 4, '...', tp)
  } else if (page.value >= tp - 2) {
    pages.push(1, '...', tp - 3, tp - 2, tp - 1, tp)
  } else {
    pages.push(1, '...', page.value - 1, page.value, page.value + 1, '...', tp)
  }
  return pages
})

const formatAbsolute = (dateStr: string) => formatAbsoluteTime(dateStr, locale.value)

async function fetchDepartments() {
  try {
    const res = await getDepartments()
    departments.value = res.data || []
  } catch { /* ignore */ }
}

function openForm(teacher?: AdminTeacher) {
  editingTeacher.value = teacher || null
  formName.value = teacher?.name || ''
  formDeptID.value = teacher?.departmentID
  formOpen.value = true
}

async function handleSave() {
  if (!formName.value.trim()) return
  saving.value = true
  try {
    if (editingTeacher.value) {
      await updateTeacher(editingTeacher.value.id, {
        name: formName.value.trim(),
        departmentID: formDeptID.value ?? null
      })
      toast.success(t('admin.teachers.updateSuccess'))
    } else {
      await createTeacher({
        name: formName.value.trim(),
        departmentID: formDeptID.value
      })
      toast.success(t('admin.teachers.createSuccess'))
    }
    formOpen.value = false
    fetchTeachers()
  } catch {
    toast.error(t('admin.teachers.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function handleDelete(teacher: AdminTeacher) {
  if (!confirm(t('admin.teachers.deleteConfirm'))) return
  try {
    await deleteTeacher(teacher.id)
    toast.success(t('admin.teachers.deleteSuccess'))
    fetchTeachers()
  } catch {
    toast.error(t('admin.teachers.deleteFailed'))
  }
}

// Debounced search
let searchTimer: ReturnType<typeof setTimeout>
watch(searchQuery, () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    fetchTeachers()
  }, 300)
})

watch(filterDeptID, () => {
  page.value = 1
  fetchTeachers()
})
watch(page, () => fetchTeachers())

onMounted(() => {
  fetchTeachers()
  fetchDepartments()
})
</script>
