<script setup lang="ts">
import type { SensitiveWord } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
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

import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import {
  compactID,
  formatAdminDateTime,
  formatNullableText,
} from '../../shared/display';
import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import SensitiveWordDialog from './SensitiveWordDialog.vue';

const loading = ref(false);
const words = ref<SensitiveWord[]>([]);
const total = ref(0);
const query = reactive({
  page: 1,
  pageSize: 20,
  category: '',
  level: '' as '' | 'block' | 'review' | 'warn',
});

async function fetchData() {
  loading.value = true;
  try {
    const data = await getSensitiveWordList({
      category: query.category || undefined,
      level: query.level || undefined,
      page: query.page,
      pageSize: query.pageSize,
    });
    words.value = data.items;
    total.value = data.total;
  } finally {
    loading.value = false;
  }
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

async function handleSubmit() {
  if (!form.word.trim()) {
    ElMessage.warning(
      $t('admin.content.sensitiveWords.validation.wordRequired'),
    );
    return;
  }
  const payload = {
    category: form.category,
    isActive: form.isActive,
    level: form.level,
    word: form.word,
  };
  if (isEdit.value) {
    await updateSensitiveWord(form.id, payload);
    ElMessage.success($t('admin.content.sensitiveWords.updated'));
  } else {
    await createSensitiveWord(payload);
    ElMessage.success($t('admin.content.sensitiveWords.created'));
  }
  dialogVisible.value = false;
  await fetchData();
}

async function handleDelete(id: string) {
  await deleteSensitiveWord(id);
  ElMessage.success($t('admin.content.sensitiveWords.deleted'));
  await fetchData();
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
        @clear="fetchData"
        @keyup.enter="fetchData"
      />
      <ElSelect
        v-model="query.level"
        class="admin-toolbar-control"
        clearable
        :placeholder="$t('admin.content.sensitiveWords.level')"
        :teleported="false"
        @change="fetchData"
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
      <ElButton type="primary" @click="fetchData">
        {{ $t('admin.common.query') }}
      </ElButton>
      <ElButton type="success" @click="openCreate">
        {{ $t('admin.content.sensitiveWords.create') }}
      </ElButton>
    </template>

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
            <ElButton plain size="small" type="primary" @click="openEdit(row)">
              {{ $t('admin.common.edit') }}
            </ElButton>
            <ElPopconfirm
              :title="$t('admin.content.sensitiveWords.confirmDelete')"
              @confirm="handleDelete(row.id)"
            >
              <template #reference>
                <ElButton plain size="small" type="danger">
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
      @submit="handleSubmit"
    />
  </AdminContentLayout>
</template>
