<script setup lang="ts">
import type { SensitiveWord } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElOption,
  ElPagination,
  ElPopconfirm,
  ElSelect,
  ElSwitch,
  ElTable,
  ElTableColumn,
} from 'element-plus';

import { $t } from '#/locales';
import {
  createSensitiveWord,
  deleteSensitiveWord,
  getSensitiveWordList,
  updateSensitiveWord,
} from '#/api/admin';

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
    ElMessage.warning($t('admin.content.sensitiveWords.validation.wordRequired'));
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
  <div class="p-4">
    <div class="mb-4 flex items-center gap-3">
      <ElInput
        v-model="query.category"
        clearable
        :placeholder="$t('admin.content.sensitiveWords.filterByCategory')"
        style="width: 160px"
        @clear="fetchData"
        @keyup.enter="fetchData"
      />
      <ElSelect
        v-model="query.level"
        clearable
        :placeholder="$t('admin.content.sensitiveWords.level')"
        style="width: 120px"
        @change="fetchData"
      >
        <ElOption :label="$t('admin.common.all')" value="" />
        <ElOption :label="$t('admin.content.sensitiveWords.levels.block')" value="block" />
        <ElOption :label="$t('admin.content.sensitiveWords.levels.warn')" value="warn" />
        <ElOption :label="$t('admin.content.sensitiveWords.levels.review')" value="review" />
      </ElSelect>
      <ElButton type="primary" @click="fetchData">{{ $t('admin.common.query') }}</ElButton>
      <ElButton type="success" @click="openCreate">{{ $t('admin.content.sensitiveWords.create') }}</ElButton>
    </div>

    <ElTable v-loading="loading" :data="words" stripe>
      <ElTableColumn :label="$t('admin.common.id')" prop="id" width="70" />
      <ElTableColumn :label="$t('admin.content.sensitiveWords.word')" min-width="150" prop="word" />
      <ElTableColumn :label="$t('admin.content.sensitiveWords.category')" prop="category" width="120" />
      <ElTableColumn :label="$t('admin.content.sensitiveWords.level')" prop="level" width="90" />
      <ElTableColumn :label="$t('admin.content.sensitiveWords.active')" width="80">
        <template #default="{ row }">
          {{ row.isActive ? $t('admin.common.yes') : $t('admin.common.no') }}
        </template>
      </ElTableColumn>
      <ElTableColumn :label="$t('admin.common.createdAt')" prop="createdAt" width="170" />
      <ElTableColumn fixed="right" :label="$t('admin.common.actions')" width="140">
        <template #default="{ row }">
          <ElButton link size="small" type="primary" @click="openEdit(row)">
            {{ $t('admin.common.edit') }}
          </ElButton>
          <ElPopconfirm
            :title="$t('admin.content.sensitiveWords.confirmDelete')"
            @confirm="handleDelete(row.id)"
          >
            <template #reference>
              <ElButton link size="small" type="danger">{{ $t('admin.common.delete') }}</ElButton>
            </template>
          </ElPopconfirm>
        </template>
      </ElTableColumn>
    </ElTable>

    <div class="mt-4 flex justify-end">
      <ElPagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="fetchData"
      />
    </div>

    <!-- 新增/编辑弹窗 -->
    <ElDialog
      v-model="dialogVisible"
      :title="isEdit ? $t('admin.content.sensitiveWords.editTitle') : $t('admin.content.sensitiveWords.createTitle')"
      width="480px"
    >
      <ElForm label-width="80px">
        <ElFormItem :label="$t('admin.content.sensitiveWords.word')">
          <ElInput v-model="form.word" :placeholder="$t('admin.content.sensitiveWords.wordPlaceholder')" />
        </ElFormItem>
        <ElFormItem :label="$t('admin.content.sensitiveWords.category')">
          <ElInput v-model="form.category" :placeholder="$t('admin.content.sensitiveWords.categoryPlaceholder')" />
        </ElFormItem>
        <ElFormItem :label="$t('admin.content.sensitiveWords.level')">
          <ElSelect
            v-model="form.level"
            :placeholder="$t('admin.content.sensitiveWords.levelPlaceholder')"
            style="width: 100%"
          >
            <ElOption :label="$t('admin.content.sensitiveWords.levels.block')" value="block" />
            <ElOption :label="$t('admin.content.sensitiveWords.levels.warn')" value="warn" />
            <ElOption :label="$t('admin.content.sensitiveWords.levels.review')" value="review" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem :label="$t('admin.content.sensitiveWords.active')">
          <ElSwitch v-model="form.isActive" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">{{ $t('admin.common.cancel') }}</ElButton>
        <ElButton type="primary" @click="handleSubmit">{{ $t('admin.common.confirm') }}</ElButton>
      </template>
    </ElDialog>
  </div>
</template>
