<script setup lang="ts">
import type { Teacher } from '#/api/admin';

import { reactive, ref } from 'vue';

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
import { useAdminAction } from '#/composables/use-admin-action';
import { useAdminList } from '#/composables/use-admin-list';
import { $t } from '#/locales';

import PersistentAdminTable from '../../shared/admin-table/PersistentAdminTable.vue';
import PersistentAdminTableColumn from '../../shared/admin-table/PersistentAdminTableColumn.vue';
import AdminContentLayout from '../../shared/AdminContentLayout.vue';
import { formatAdminDateTime } from '../../shared/display';
import {
  ADMIN_DEFAULT_PAGE_SIZE,
  ADMIN_PAGE_SIZES,
  ADMIN_PAGINATION_LAYOUT,
} from '../../shared/pagination';

const {
  fetchData,
  items: teachers,
  loadError,
  loading,
  query,
  resetPageAndFetch,
  total,
} = useAdminList<
  Teacher,
  {
    departmentID: null | number;
    keyword: string;
    page: number;
    pageSize: number;
  }
>({
  fetcher: (listQuery) =>
    getTeacherList({
      departmentID: listQuery.departmentID ?? undefined,
      keyword: listQuery.keyword || undefined,
      page: listQuery.page,
      pageSize: listQuery.pageSize,
    }),
  initialQuery: {
    departmentID: null,
    keyword: '',
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
  id: 0,
  name: '',
  departmentID: null as null | number,
});

type TeacherSubmitAction =
  | {
      data: { departmentID: null | number; name: string };
      id: number;
      kind: 'edit';
    }
  | {
      data: { departmentID: number; name: string };
      kind: 'create';
    };

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
  let action: TeacherSubmitAction;
  if (isEdit.value) {
    action = {
      data: {
        departmentID: form.departmentID,
        name: form.name,
      },
      id: form.id,
      kind: 'edit',
    };
  } else {
    const departmentID = form.departmentID;
    if (departmentID === null) {
      ElMessage.warning(
        $t('admin.content.teachers.validation.departmentRequired'),
      );
      return;
    }
    action = {
      data: {
        departmentID,
        name: form.name,
      },
      kind: 'create',
    };
  }
  const succeeded = await runAction(
    () =>
      action.kind === 'edit'
        ? updateTeacher(action.id, action.data)
        : createTeacher(action.data),
    {
      successMessage:
        action.kind === 'edit'
          ? $t('admin.content.teachers.updated')
          : $t('admin.content.teachers.created'),
    },
  );
  if (succeeded) {
    dialogVisible.value = false;
    await fetchData();
  }
}

async function handleDelete(teacherId: number) {
  const succeeded = await runAction(() => deleteTeacher(teacherId), {
    id: teacherId,
    successMessage: $t('admin.content.teachers.deleted'),
  });
  if (succeeded) {
    await fetchData();
  }
}
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
      <ElButton type="success" :disabled="actionPending" @click="openCreate">
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
      @close="clearActionError"
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
              :disabled="isActionPending(row.id)"
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
        <ElButton type="primary" :loading="actionPending" @click="handleSubmit">
          {{ $t('admin.common.confirm') }}
        </ElButton>
      </template>
    </ElDialog>
  </AdminContentLayout>
</template>
