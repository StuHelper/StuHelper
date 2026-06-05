<script setup lang="ts">
import type { Teacher } from '#/api/admin';

import { onMounted, reactive, ref } from 'vue';

import {
  ElAlert,
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElPagination,
  ElPopconfirm,
} from 'element-plus';

import {
  createTeacher,
  deleteTeacher,
  getTeacherList,
  updateTeacher,
} from '#/api/admin';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';

const loading = ref(false);
const actionLoading = ref(false);
const loadError = ref('');
const actionError = ref('');
const teachers = ref<Teacher[]>([]);
const total = ref(0);
const query = reactive({
  page: 1,
  pageSize: 20,
  keyword: '',
  departmentID: null as null | number,
});
let fetchRequestSeq = 0;

async function fetchData() {
  const requestSeq = ++fetchRequestSeq;
  loading.value = true;
  loadError.value = '';
  try {
    const data = await getTeacherList({
      departmentID: query.departmentID ?? undefined,
      keyword: query.keyword || undefined,
      page: query.page,
      pageSize: query.pageSize,
    });
    if (requestSeq !== fetchRequestSeq) return;
    teachers.value = data.items;
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
  if (actionLoading.value) {
    return;
  }
  if (!form.name.trim()) {
    ElMessage.warning($t('admin.content.teachers.validation.nameRequired'));
    return;
  }
  const payload = {
    departmentID: form.departmentID || undefined,
    name: form.name,
  };
  actionLoading.value = true;
  actionError.value = '';
  try {
    if (isEdit.value) {
      await updateTeacher(form.id, payload);
      ElMessage.success($t('admin.content.teachers.updated'));
    } else {
      await createTeacher(payload);
      ElMessage.success($t('admin.content.teachers.created'));
    }
    dialogVisible.value = false;
    await fetchData();
  } catch (error) {
    handleActionError(error);
  } finally {
    actionLoading.value = false;
  }
}

async function handleDelete(teacherId: number) {
  if (actionLoading.value) {
    return;
  }
  actionLoading.value = true;
  actionError.value = '';
  try {
    await deleteTeacher(teacherId);
    ElMessage.success($t('admin.content.teachers.deleted'));
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
    :title="$t('admin.routes.content.teachers')"
    :total="total"
  >
    <template #toolbar>
      <ElInput
        v-model="query.keyword"
        class="admin-toolbar-control admin-toolbar-control--wide"
        clearable
        :placeholder="$t('admin.content.teachers.searchByName')"
        @clear="resetPageAndFetch"
        @keyup.enter="resetPageAndFetch"
      />
      <ElInput
        v-model.number="query.departmentID"
        class="admin-toolbar-control admin-toolbar-control--wide"
        clearable
        :placeholder="$t('admin.content.teachers.filterByDepartmentId')"
        @clear="resetPageAndFetch"
        @keyup.enter="resetPageAndFetch"
      />
      <ElButton type="primary" @click="resetPageAndFetch">
        {{ $t('admin.common.query') }}
      </ElButton>
      <ElButton type="success" :disabled="actionLoading" @click="openCreate">
        {{ $t('admin.content.teachers.create') }}
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
      table-key="content.teachers"
      :loading="loading"
      :data="teachers"
      row-key="id"
      stripe
    >
      <PersistentAdminTableColumn
        column-key="id"
        :label="$t('admin.common.id')"
        prop="id"
        :default-width="88"
      />
      <PersistentAdminTableColumn
        column-key="name"
        :label="$t('admin.content.teachers.name')"
        prop="name"
        :default-width="140"
      />
      <PersistentAdminTableColumn
        column-key="department"
        :label="$t('admin.content.teachers.department')"
        :default-min-width="180"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          {{
            row.departmentName ||
            (row.departmentID
              ? `${$t('admin.content.teachers.departmentPrefix')} #${row.departmentID}`
              : $t('admin.common.notSet'))
          }}
        </template>
      </PersistentAdminTableColumn>
      <PersistentAdminTableColumn
        column-key="reviewCount"
        :label="$t('admin.content.teachers.reviewCount')"
        prop="reviewCount"
        :default-width="100"
      />
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
              :title="$t('admin.content.teachers.confirmDelete')"
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
        <ElButton type="primary" :loading="actionLoading" @click="handleSubmit">
          {{ $t('admin.common.confirm') }}
        </ElButton>
      </template>
    </ElDialog>
  </AdminContentLayout>
</template>
