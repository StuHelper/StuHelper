<script setup lang="ts">
import type { SensitiveWord } from '#/api/admin';

import { reactive, ref } from 'vue';

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
import { useAdminAction } from '#/composables/use-admin-action';
import { useAdminList } from '#/composables/use-admin-list';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import {
  compactID,
  formatAdminDateTime,
  formatNullableText,
} from '../../shared/display';
import {
  ADMIN_DEFAULT_PAGE_SIZE,
  ADMIN_PAGE_SIZES,
  ADMIN_PAGINATION_LAYOUT,
} from '../../shared/pagination';
import SensitiveWordDialog from './SensitiveWordDialog.vue';

type SensitiveWordLevelFilter = '' | 'block' | 'review' | 'warn';

const {
  fetchData,
  items: words,
  loadError,
  loading,
  query,
  resetPageAndFetch,
  total,
} = useAdminList<
  SensitiveWord,
  {
    category: string;
    level: SensitiveWordLevelFilter;
    page: number;
    pageSize: number;
  }
>({
  fetcher: (listQuery) =>
    getSensitiveWordList({
      category: listQuery.category || undefined,
      level: listQuery.level || undefined,
      page: listQuery.page,
      pageSize: listQuery.pageSize,
    }),
  initialQuery: {
    category: '',
    level: '',
    page: 1,
    pageSize: ADMIN_DEFAULT_PAGE_SIZE,
  },
});

const {
  actionError,
  actionPending,
  clearActionError,
  isActionPending,
  runAction,
} = useAdminAction();

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
  if (!submitted.word.trim()) {
    ElMessage.warning(
      $t('admin.content.sensitiveWords.validation.wordRequired'),
    );
    return;
  }
  const payload = {
    // 空 category 交给后端落默认分类（general），不要发送空白字符串。
    category: submitted.category.trim(),
    isActive: submitted.isActive,
    level: submitted.level,
    word: submitted.word.trim(),
  };
  const succeeded = await runAction(
    () =>
      isEdit.value
        ? updateSensitiveWord(submitted.id, payload)
        : createSensitiveWord(payload),
    {
      successMessage: isEdit.value
        ? $t('admin.content.sensitiveWords.updated')
        : $t('admin.content.sensitiveWords.created'),
    },
  );
  if (succeeded) {
    dialogVisible.value = false;
    await fetchData();
  }
}

async function handleDelete(id: string) {
  const succeeded = await runAction(() => deleteSensitiveWord(id), {
    id,
    successMessage: $t('admin.content.sensitiveWords.deleted'),
  });
  if (succeeded) {
    await fetchData();
  }
}
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
      <ElButton type="success" :disabled="actionPending" @click="openCreate">
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
      @close="clearActionError"
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
              :disabled="isActionPending(row.id)"
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
                  :disabled="isActionPending(row.id)"
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
        background
        :layout="ADMIN_PAGINATION_LAYOUT"
        :page-sizes="ADMIN_PAGE_SIZES"
        :total="total"
        @current-change="fetchData"
        @size-change="resetPageAndFetch"
      />
    </template>

    <SensitiveWordDialog
      v-model:visible="dialogVisible"
      :form="form"
      :is-edit="isEdit"
      :loading="actionPending"
      @submit="handleSubmit"
    />
  </AdminContentLayout>
</template>
