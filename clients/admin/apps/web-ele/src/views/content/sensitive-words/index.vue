<script setup lang="ts">
import type { SensitiveWord } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElAlert,
  ElButton,
  ElInput,
  ElMessage,
  ElOption,
  ElPagination,
  ElPopconfirm,
  ElSelect,
  ElTag,
} from 'element-plus';

import {
  createSensitiveWord,
  deleteSensitiveWord,
  getSensitiveWordList,
  updateSensitiveWord,
} from '#/api/admin';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import {
  compactID,
  formatAdminDateTime,
  formatNullableText,
} from '../../shared/display';
import SensitiveWordDialog from './SensitiveWordDialog.vue';

const loading = ref(false);
const actionLoading = ref(false);
const loadError = ref('');
const actionError = ref('');
const words = ref<SensitiveWord[]>([]);
const total = ref(0);
const query = reactive({
  page: 1,
  pageSize: 20,
  category: '',
  level: '' as '' | 'block' | 'review' | 'warn',
});
let fetchRequestSeq = 0;

async function fetchData() {
  const requestSeq = ++fetchRequestSeq;
  loading.value = true;
  loadError.value = '';
  try {
    const data = await getSensitiveWordList({
      category: query.category || undefined,
      level: query.level || undefined,
      page: query.page,
      pageSize: query.pageSize,
    });
    if (requestSeq !== fetchRequestSeq) return;
    words.value = data.items;
    total.value = data.total;
  } catch (error) {
    if (requestSeq !== fetchRequestSeq) return;
    loadError.value = adminErrorMessage(error);
  } finally {
    if (requestSeq === fetchRequestSeq) {
      loading.value = false;
    }
  }
}

function resetPageAndFetch() {
  query.page = 1;
  void fetchData();
}

// ── 弹窗 ──

const dialogVisible = ref(false);
const isEdit = ref(false);
const form = reactive({
  id: '',
  word: '',
  category: '',
  level: 'block' as 'block' | 'review' | 'warn',
  isActive: true,
});

type SensitiveWordForm = typeof form;

function resetForm() {
  form.id = '';
  form.word = '';
  form.category = '';
  form.level = 'block';
  form.isActive = true;
}

function openCreate() {
  resetForm();
  isEdit.value = false;
  dialogVisible.value = true;
}

function openEdit(row: SensitiveWord) {
  form.id = row.id;
  form.word = row.word;
  form.category = row.category;
  form.level = row.level;
  form.isActive = row.isActive;
  isEdit.value = true;
  dialogVisible.value = true;
}

async function handleSubmit(submitted: SensitiveWordForm) {
  if (actionLoading.value) {
    return;
  }
  if (!submitted.word.trim()) {
    ElMessage.warning(
      $t('admin.content.sensitiveWords.validation.wordRequired'),
    );
    return;
  }
  const payload = {
    category: submitted.category,
    isActive: submitted.isActive,
    level: submitted.level,
    word: submitted.word,
  };
  actionLoading.value = true;
  actionError.value = '';
  try {
    if (isEdit.value) {
      await updateSensitiveWord(submitted.id, payload);
      ElMessage.success($t('admin.content.sensitiveWords.updated'));
    } else {
      await createSensitiveWord(payload);
      ElMessage.success($t('admin.content.sensitiveWords.created'));
    }
    dialogVisible.value = false;
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    actionLoading.value = false;
  }
}

async function handleDelete(id: string) {
  if (actionLoading.value) {
    return;
  }
  actionLoading.value = true;
  actionError.value = '';
  try {
    await deleteSensitiveWord(id);
    ElMessage.success($t('admin.content.sensitiveWords.deleted'));
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    actionLoading.value = false;
  }
}

function handleActionError(error: unknown) {
  actionError.value = adminErrorMessage(error);
  ElMessage.error(actionError.value);
}

function adminErrorMessage(error: unknown): string {
  return error instanceof Error && error.message
    ? error.message
    : $t('admin.result.requestFailed');
}

onMounted(fetchData);
</script>

<template>
  <AdminContentLayout
    :title="$t('admin.routes.content.sensitiveWords')"
    :total="total"
  >
    <template #toolbar>
      <ElInput
        v-model="query.category"
        class="admin-toolbar-control"
        clearable
        :placeholder="$t('admin.content.sensitiveWords.filterByCategory')"
        @clear="resetPageAndFetch"
        @keyup.enter="resetPageAndFetch"
      />
      <ElSelect
        v-model="query.level"
        class="admin-toolbar-control"
        clearable
        :placeholder="$t('admin.content.sensitiveWords.level')"
        :teleported="false"
        @change="resetPageAndFetch"
      >
        <ElOption :label="$t('admin.common.all')" value="" />
        <ElOption
          :label="$t('admin.content.sensitiveWords.levels.block')"
          value="block"
        />
        <ElOption
          :label="$t('admin.content.sensitiveWords.levels.warn')"
          value="warn"
        />
        <ElOption
          :label="$t('admin.content.sensitiveWords.levels.review')"
          value="review"
        />
      </ElSelect>
      <ElButton type="primary" @click="resetPageAndFetch">
        {{ $t('admin.common.query') }}
      </ElButton>
      <ElButton type="success" :disabled="actionLoading" @click="openCreate">
        {{ $t('admin.content.sensitiveWords.create') }}
      </ElButton>
    </template>

    <ElAlert
      v-if="loadError"
      class="admin-load-error"
      type="error"
      :closable="false"
      show-icon
      :title="loadError"
    >
      <ElButton size="small" :loading="loading" @click="fetchData">
        {{ $t('admin.common.retry') }}
      </ElButton>
    </ElAlert>

    <ElAlert
      v-if="actionError"
      class="admin-load-error"
      type="error"
      :closable="true"
      show-icon
      :title="actionError"
      @close="actionError = ''"
    />

    <PersistentAdminTable
      table-key="content.sensitiveWords"
      :loading="loading"
      :data="words"
      row-key="id"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="id"
        :label="$t('admin.common.id')"
        :default-width="148"
      >
        <template #default="{ row }">
          <span class="admin-id-token" :title="row.id">
            {{ compactID(row.id) }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="word"
        :label="$t('admin.content.sensitiveWords.word')"
        :default-min-width="180"
        prop="word"
        show-overflow-tooltip
      />
      <PersistentAdminTableColumn
        column-key="category"
        :label="$t('admin.content.sensitiveWords.category')"
        :default-width="140"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          {{ formatNullableText(row.category) }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="level"
        :label="$t('admin.content.sensitiveWords.level')"
        prop="level"
        :default-width="104"
      />
      <PersistentAdminTableColumn
        column-key="isActive"
        :label="$t('admin.content.sensitiveWords.active')"
        :default-width="96"
      >
        <template #default="{ row }">
          <ElTag :type="row.isActive ? 'success' : 'info'" size="small">
            {{ row.isActive ? $t('admin.common.yes') : $t('admin.common.no') }}
          </ElTag>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="createdAt"
        :label="$t('admin.common.createdAt')"
        :default-width="148"
      >
        <template #default="{ row }">
          <span class="admin-cell-muted">
            {{ formatAdminDateTime(row.createdAt) }}
          </span>
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="actions"
        fixed="right"
        :label="$t('admin.common.actions')"
        :default-width="160"
      >
        <template #default="{ row }">
          <div class="admin-action-group">
            <ElButton
              plain
              size="small"
              type="primary"
              :disabled="actionLoading"
              @click="openEdit(row)"
            >
              {{ $t('admin.common.edit') }}
            </ElButton>
            <ElPopconfirm
              :title="$t('admin.content.sensitiveWords.confirmDelete')"
              @confirm="handleDelete(row.id)"
            >
              <template #reference>
                <ElButton
                  plain
                  size="small"
                  type="danger"
                  :disabled="actionLoading"
                >
                  {{ $t('admin.common.delete') }}
                </ElButton>
              </template>
            </ElPopconfirm>
          </div>
        </template>
      </PersistentAdminTableColumn>
    </PersistentAdminTable>

    <template #pagination>
      <ElPagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="fetchData"
      />
    </template>

    <SensitiveWordDialog
      v-model:visible="dialogVisible"
      :form="form"
      :is-edit="isEdit"
      :loading="actionLoading"
      @submit="handleSubmit"
    />
  </AdminContentLayout>
</template>
