<template>
  <div class="space-y-4">
    <!-- Toolbar -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary">{{ t('admin.sensitiveWords.title') }}</h1>
      <div class="flex items-center gap-2 w-full sm:w-auto">
        <select v-model="filterCategory" class="px-3 py-2 text-sm bg-bg-card border border-border rounded-lg text-text-primary">
          <option value="">{{ t('admin.sensitiveWords.allCategories') }}</option>
          <option v-for="c in categories" :key="c" :value="c">{{ c }}</option>
        </select>
        <select v-model="filterLevel" class="px-3 py-2 text-sm bg-bg-card border border-border rounded-lg text-text-primary">
          <option value="">{{ t('admin.sensitiveWords.allLevels') }}</option>
          <option value="block">{{ t('admin.sensitiveWords.levelBlock') }}</option>
          <option value="warn">{{ t('admin.sensitiveWords.levelWarn') }}</option>
          <option value="review">{{ t('admin.sensitiveWords.levelReview') }}</option>
        </select>
        <button
          class="flex items-center gap-1.5 px-3 py-2 text-sm text-text-inverse bg-primary rounded-lg transition-colors duration-fast hover:bg-primary/90"
          @click="openForm()"
        >
          <Plus class="size-3.5" />
          <span class="hidden sm:inline">{{ t('admin.sensitiveWords.add') }}</span>
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="bg-bg-card border border-border rounded-xl shadow-card overflow-hidden">
      <div v-for="i in 5" :key="i" class="h-14 border-b border-border animate-pulse" />
    </div>

    <!-- Table -->
    <div v-else-if="words.length > 0" class="bg-bg-card border border-border rounded-xl shadow-card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-bg-secondary">
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted">{{ t('admin.sensitiveWords.word') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted hidden sm:table-cell">{{ t('admin.sensitiveWords.category') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted">{{ t('admin.sensitiveWords.level') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted hidden md:table-cell">{{ t('admin.sensitiveWords.status') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted">{{ t('admin.sensitiveWords.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="w in words" :key="w.id" class="border-t border-border transition-colors duration-fast hover:bg-bg-hover">
              <td class="p-3 text-sm text-text-primary font-medium">{{ w.word }}</td>
              <td class="p-3 text-sm text-text-secondary hidden sm:table-cell">{{ w.category }}</td>
              <td class="p-3">
                <span class="inline-flex px-2 py-0.5 text-xs font-medium rounded-md" :class="levelClass(w.level)">
                  {{ levelLabel(w.level) }}
                </span>
              </td>
              <td class="p-3 hidden md:table-cell">
                <button
                  class="inline-flex px-2 py-0.5 text-xs font-medium rounded-md cursor-pointer transition-colors duration-fast"
                  :class="w.isActive ? 'bg-success/10 text-success' : 'bg-bg-secondary text-text-muted'"
                  @click="toggleActive(w)"
                >
                  {{ w.isActive ? t('admin.sensitiveWords.active') : t('admin.sensitiveWords.inactive') }}
                </button>
              </td>
              <td class="p-3">
                <div class="flex items-center gap-1">
                  <button
                    class="p-1.5 text-text-muted rounded-md transition-colors duration-fast hover:bg-primary/10 hover:text-primary"
                    :title="t('admin.sensitiveWords.edit')"
                    @click="openForm(w)"
                  ><Pencil class="size-4" /></button>
                  <button
                    class="p-1.5 text-text-muted rounded-md transition-colors duration-fast hover:bg-danger/10 hover:text-danger"
                    :title="t('admin.sensitiveWords.delete')"
                    @click="handleDelete(w)"
                  ><Trash2 class="size-4" /></button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <EmptyState v-else :title="t('admin.sensitiveWords.empty')" />

    <!-- Pagination -->
    <div v-if="total > 0" class="flex flex-col sm:flex-row items-center justify-between gap-3 text-sm">
      <span class="text-text-muted">{{ t('admin.pagination.total', { total }) }}</span>
      <div class="flex items-center gap-1">
        <button
          class="px-3 py-1.5 border border-border rounded-lg text-text-secondary transition-colors duration-fast hover:bg-bg-hover disabled:opacity-40 disabled:cursor-not-allowed"
          :disabled="page <= 1" @click="page--"
        >{{ t('admin.pagination.prev') }}</button>
        <template v-for="p in visiblePages" :key="p">
          <span v-if="p === '...'" class="px-2 text-text-muted">...</span>
          <button v-else
            class="min-w-[36px] h-9 px-2 rounded-lg text-sm font-medium transition-colors duration-fast"
            :class="p === page ? 'bg-primary text-text-inverse' : 'text-text-secondary hover:bg-bg-hover'"
            @click="page = p as number"
          >{{ p }}</button>
        </template>
        <button
          class="px-3 py-1.5 border border-border rounded-lg text-text-secondary transition-colors duration-fast hover:bg-bg-hover disabled:opacity-40 disabled:cursor-not-allowed"
          :disabled="page >= totalPages" @click="page++"
        >{{ t('admin.pagination.next') }}</button>
      </div>
    </div>

    <!-- Add/Edit Modal -->
    <div v-if="formOpen" class="fixed inset-0 bg-bg-overlay z-[var(--z-modal-backdrop)] flex items-center justify-center p-4" @click.self="formOpen = false">
      <div class="bg-bg-card border border-border rounded-xl shadow-lg w-full max-w-md p-6 space-y-4">
        <h2 class="text-lg font-bold text-text-primary">{{ editing ? t('admin.sensitiveWords.edit') : t('admin.sensitiveWords.add') }}</h2>
        <div class="space-y-3">
          <div>
            <label class="block text-sm text-text-muted mb-1">{{ t('admin.sensitiveWords.word') }}</label>
            <input v-model="formWord" type="text" :placeholder="t('admin.sensitiveWords.wordPlaceholder')"
              class="w-full px-3 py-2 text-sm bg-bg-card border border-border rounded-lg text-text-primary placeholder:text-text-muted focus:border-primary focus:outline-none" />
          </div>
          <div>
            <label class="block text-sm text-text-muted mb-1">{{ t('admin.sensitiveWords.category') }}</label>
            <input v-model="formCategory" type="text" :placeholder="t('admin.sensitiveWords.categoryPlaceholder')"
              class="w-full px-3 py-2 text-sm bg-bg-card border border-border rounded-lg text-text-primary placeholder:text-text-muted focus:border-primary focus:outline-none" />
          </div>
          <div>
            <label class="block text-sm text-text-muted mb-1">{{ t('admin.sensitiveWords.level') }}</label>
            <select v-model="formLevel" class="w-full px-3 py-2 text-sm bg-bg-card border border-border rounded-lg text-text-primary">
              <option value="block">{{ t('admin.sensitiveWords.levelBlock') }}</option>
              <option value="warn">{{ t('admin.sensitiveWords.levelWarn') }}</option>
              <option value="review">{{ t('admin.sensitiveWords.levelReview') }}</option>
            </select>
          </div>
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <button class="px-4 py-2 text-sm text-text-secondary border border-border rounded-lg transition-colors duration-fast hover:bg-bg-hover"
            @click="formOpen = false">{{ t('common.actions.cancel') }}</button>
          <button
            class="px-4 py-2 text-sm text-text-inverse bg-primary rounded-lg transition-colors duration-fast hover:bg-primary/90 disabled:opacity-50"
            :disabled="!formWord.trim() || saving" @click="handleSave"
          >{{ saving ? '...' : (editing ? t('admin.sensitiveWords.edit') : t('admin.sensitiveWords.add')) }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, Pencil, Trash2 } from 'lucide-vue-next'
import { getSensitiveWords, createSensitiveWord, updateSensitiveWord, deleteSensitiveWord } from '@/api/admin'
import type { AdminSensitiveWord } from '@/types/admin'
import EmptyState from '@/components/common/EmptyState.vue'
import { useToast } from '@/composables/useToast'
import { useAsyncData } from '@/composables/useAsyncData'

const { t } = useI18n()
const toast = useToast()

const page = ref(1)
const pageSize = ref(20)
const filterCategory = ref('')
const filterLevel = ref('')

const { data: wordsData, loading, execute: fetchWords } = useAsyncData(async () => {
  const res = await getSensitiveWords(page.value, pageSize.value, filterCategory.value, filterLevel.value)
  return {
    words: res.data?.list || [],
    total: res.data?.total || 0
  }
})

const words = computed(() => wordsData.value?.words || [])
const total = computed(() => wordsData.value?.total || 0)

// Distinct categories extracted from loaded data
const categories = computed(() => [...new Set(words.value.map(w => w.category))].sort())

// Form state
const formOpen = ref(false)
const formWord = ref('')
const formCategory = ref('')
const formLevel = ref<'block' | 'warn' | 'review'>('block')
const editing = ref<AdminSensitiveWord | null>(null)
const saving = ref(false)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

const visiblePages = computed(() => {
  const tp = totalPages.value
  if (tp <= 5) return Array.from({ length: tp }, (_, i) => i + 1)
  const pages: (number | string)[] = []
  if (page.value <= 3) pages.push(1, 2, 3, 4, '...', tp)
  else if (page.value >= tp - 2) pages.push(1, '...', tp - 3, tp - 2, tp - 1, tp)
  else pages.push(1, '...', page.value - 1, page.value, page.value + 1, '...', tp)
  return pages
})

function levelClass(level: string) {
  if (level === 'block') return 'bg-danger/10 text-danger'
  if (level === 'warn') return 'bg-warning/10 text-warning'
  return 'bg-primary/10 text-primary'
}

function levelLabel(level: string) {
  if (level === 'block') return t('admin.sensitiveWords.levelBlock')
  if (level === 'warn') return t('admin.sensitiveWords.levelWarn')
  return t('admin.sensitiveWords.levelReview')
}

function openForm(w?: AdminSensitiveWord) {
  editing.value = w || null
  formWord.value = w?.word || ''
  formCategory.value = w?.category || ''
  formLevel.value = w?.level || 'block'
  formOpen.value = true
}

async function handleSave() {
  if (!formWord.value.trim()) return
  saving.value = true
  try {
    if (editing.value) {
      await updateSensitiveWord(editing.value.id, {
        word: formWord.value.trim(),
        category: formCategory.value.trim() || undefined,
        level: formLevel.value
      })
      toast.success(t('admin.sensitiveWords.updateSuccess'))
    } else {
      await createSensitiveWord({
        word: formWord.value.trim(),
        category: formCategory.value.trim() || undefined,
        level: formLevel.value
      })
      toast.success(t('admin.sensitiveWords.createSuccess'))
    }
    formOpen.value = false
    fetchWords()
  } catch {
    toast.error(t('admin.sensitiveWords.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function toggleActive(w: AdminSensitiveWord) {
  try {
    await updateSensitiveWord(w.id, { isActive: !w.isActive })
    toast.success(t('admin.sensitiveWords.updateSuccess'))
    fetchWords()
  } catch {
    toast.error(t('admin.sensitiveWords.saveFailed'))
  }
}

async function handleDelete(w: AdminSensitiveWord) {
  if (!confirm(t('admin.sensitiveWords.deleteConfirm'))) return
  try {
    await deleteSensitiveWord(w.id)
    toast.success(t('admin.sensitiveWords.deleteSuccess'))
    fetchWords()
  } catch {
    toast.error(t('admin.sensitiveWords.deleteFailed'))
  }
}

watch(filterCategory, () => { page.value = 1; fetchWords() })
watch(filterLevel, () => { page.value = 1; fetchWords() })
watch(page, () => fetchWords())
</script>
