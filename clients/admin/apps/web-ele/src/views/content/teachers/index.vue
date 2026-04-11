<script setup lang="ts">
import type { Teacher } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElPagination,
  ElPopconfirm,
  ElTable,
  ElTableColumn,
} from 'element-plus';

import {
  createTeacher,
  deleteTeacher,
  getTeacherList,
  updateTeacher,
} from '#/api/admin';
import { $t } from '#/locales';

const loading = ref(false);
const teachers = ref<Teacher[]>([]);
const total = ref(0);
const query = reactive({
  page: 1,
  pageSize: 20,
  keyword: '',
  departmentID: null as null | number,
});

async function fetchData() {
  loading.value = true;
  try {
    const data = await getTeacherList({
      departmentID: query.departmentID ?? undefined,
      keyword: query.keyword || undefined,
      page: query.page,
      pageSize: query.pageSize,
    });
    teachers.value = data.items;
    total.value = data.total;
  } finally {
    loading.value = false;
  }
}

// ── 弹窗 ──

const dialogVisible = ref(false);
const isEdit = ref(false);
const form = reactive({
  id: 0,
  name: '',
  departmentID: null as null | number,
});

function resetForm() {
  form.id = 0;
  form.name = '';
  form.departmentID = null;
}

function openCreate() {
  resetForm();
  isEdit.value = false;
  dialogVisible.value = true;
}

function openEdit(row: Teacher) {
  form.id = row.id;
  form.name = row.name;
  form.departmentID = row.departmentID ?? null;
  isEdit.value = true;
  dialogVisible.value = true;
}

async function handleSubmit() {
  if (!form.name.trim()) {
    ElMessage.warning($t('admin.content.teachers.validation.nameRequired'));
    return;
  }
  const payload = {
    departmentID: form.departmentID || undefined,
    name: form.name,
  };
  if (isEdit.value) {
    await updateTeacher(form.id, payload);
    ElMessage.success($t('admin.content.teachers.updated'));
  } else {
    await createTeacher(payload);
    ElMessage.success($t('admin.content.teachers.created'));
  }
  dialogVisible.value = false;
  await fetchData();
}

async function handleDelete(teacherId: number) {
  await deleteTeacher(teacherId);
  ElMessage.success($t('admin.content.teachers.deleted'));
  await fetchData();
}

onMounted(fetchData);
</script>

<template>
  <div class="p-4">
    <div class="mb-4 flex items-center gap-3">
      <ElInput
        v-model="query.keyword"
        clearable
        :placeholder="$t('admin.content.teachers.searchByName')"
        style="width: 200px"
        @clear="fetchData"
        @keyup.enter="fetchData"
      />
      <ElInput
        v-model.number="query.departmentID"
        clearable
        :placeholder="$t('admin.content.teachers.filterByDepartmentId')"
        style="width: 200px"
        @clear="fetchData"
        @keyup.enter="fetchData"
      />
      <ElButton type="primary" @click="fetchData">
        {{ $t('admin.common.query') }}
      </ElButton>
      <ElButton type="success" @click="openCreate">
        {{ $t('admin.content.teachers.create') }}
      </ElButton>
    </div>

    <ElTable v-loading="loading" :data="teachers" stripe>
      <ElTableColumn :label="$t('admin.common.id')" prop="id" width="70" />
      <ElTableColumn
        :label="$t('admin.content.teachers.name')"
        prop="name"
        width="120"
      />
      <ElTableColumn
        :label="$t('admin.content.teachers.department')"
        min-width="150"
      >
        <template #default="{ row }">
          {{
            row.departmentName ||
            (row.departmentID
              ? `${$t('admin.content.teachers.departmentPrefix')} #${row.departmentID}`
              : $t('admin.common.notSet'))
          }}
        </template>
      </ElTableColumn>
      <ElTableColumn
        :label="$t('admin.content.teachers.reviewCount')"
        prop="reviewCount"
        width="90"
      />
      <ElTableColumn
        :label="$t('admin.common.createdAt')"
        prop="createdAt"
        width="170"
      />
      <ElTableColumn
        fixed="right"
        :label="$t('admin.common.actions')"
        width="140"
      >
        <template #default="{ row }">
          <ElButton link size="small" type="primary" @click="openEdit(row)">
            {{ $t('admin.common.edit') }}
          </ElButton>
          <ElPopconfirm
            :title="$t('admin.content.teachers.confirmDelete')"
            @confirm="handleDelete(row.id)"
          >
            <template #reference>
              <ElButton link size="small" type="danger">
                {{ $t('admin.common.delete') }}
              </ElButton>
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
      :title="
        isEdit
          ? $t('admin.content.teachers.editTitle')
          : $t('admin.content.teachers.createTitle')
      "
      width="480px"
    >
      <ElForm label-width="80px">
        <ElFormItem :label="$t('admin.content.teachers.name')">
          <ElInput
            v-model="form.name"
            :placeholder="$t('admin.content.teachers.namePlaceholder')"
          />
        </ElFormItem>
        <ElFormItem :label="$t('admin.content.teachers.departmentId')">
          <ElInput
            v-model.number="form.departmentID"
            :placeholder="$t('admin.content.teachers.departmentIdPlaceholder')"
            type="number"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">
          {{ $t('admin.common.cancel') }}
        </ElButton>
        <ElButton type="primary" @click="handleSubmit">
          {{ $t('admin.common.confirm') }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>
