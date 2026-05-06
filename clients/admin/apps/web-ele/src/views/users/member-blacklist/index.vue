<script setup lang="ts">
import type {
  ListMemberBlacklistParams,
  MemberBlacklistEntry,
} from '#/api/admin';

import { computed, onMounted, reactive, ref } from 'vue';

import { useAccessStore } from '@vben/stores';

import {
  ElButton,
  ElDatePicker,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElOption,
  ElPagination,
  ElPopconfirm,
  ElSelect,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import {
  createMemberBlacklist,
  listMemberBlacklist,
  releaseMemberBlacklist,
} from '#/api/admin';

type ScopeType = 'global' | 'guild';
type StatusFilter = 'active' | 'all' | 'expired' | 'released';
type SourceFilter = '' | NonNullable<NonNullable<ListMemberBlacklistParams>['source']>;
type ReleaseReasonCode =
  | 'admission_appeal_passed'
  | 'manual_pardon'
  | 'release_only';

const SOURCE_LABELS: Record<string, string> = {
  admission_failure: '认证失败',
  kick_blacklist: '踢出拉黑',
  manual_admin: '管理员手动',
  migration_admission_failure: '迁移·认证失败',
  migration_legacy_koishi: '迁移·Koishi 旧库',
  moderation_action: '审核处置',
};

const CREATED_FROM_LABELS: Record<string, string> = {
  admin_console: 'Admin 后台',
  admission_worker: 'Admission Worker',
  koishi_console: 'Koishi 控制台',
  migration_script: '迁移脚本',
  moderation_review: '审核流程',
  qq_command: 'QQ 命令',
};

const RELEASE_REASON_OPTIONS: Array<{ label: string; value: ReleaseReasonCode }> =
  [
    { label: '宽恕（重置 admission 失败计数）', value: 'manual_pardon' },
    { label: '仅解除（保留失败计数）', value: 'release_only' },
    { label: '申诉通过', value: 'admission_appeal_passed' },
  ];

const STATUS_OPTIONS: Array<{ label: string; value: StatusFilter }> = [
  { label: '生效中', value: 'active' },
  { label: '已解除', value: 'released' },
  { label: '已过期', value: 'expired' },
  { label: '全部', value: 'all' },
];

const SCOPE_OPTIONS: Array<{ label: string; value: '' | ScopeType }> = [
  { label: '全部范围', value: '' },
  { label: '单群', value: 'guild' },
  { label: '全局', value: 'global' },
];

const SOURCE_OPTIONS: Array<{ label: string; value: SourceFilter }> = [
  { label: '全部来源', value: '' },
  { label: '管理员手动', value: 'manual_admin' },
  { label: 'QQ 踢出', value: 'kick_blacklist' },
  { label: '审核处置', value: 'moderation_action' },
  { label: '认证失败', value: 'admission_failure' },
];

const loading = ref(false);
const items = ref<MemberBlacklistEntry[]>([]);
const total = ref(0);

const accessStore = useAccessStore();
const canManage = computed(() =>
  accessStore.accessCodes.includes('member_blacklist:manage'),
);

const query = reactive({
  page: 1,
  pageSize: 20,
  platform: '',
  scopeType: '' as '' | ScopeType,
  source: '' as SourceFilter,
  status: 'active' as StatusFilter,
  guildID: '',
  subjectID: '',
});

const createDialogVisible = ref(false);
const creating = ref(false);
const createDraft = reactive({
  platform: 'qq',
  subjectID: '',
  scopeType: 'guild' as ScopeType,
  guildID: '',
  reasonText: '',
  expiresAt: '' as Date | string,
});

const releaseDialogVisible = ref(false);
const releasing = ref(false);
const releaseTarget = ref<MemberBlacklistEntry | null>(null);
const releaseDraft = reactive({
  releaseReasonCode: 'manual_pardon' as ReleaseReasonCode,
  releaseReason: '',
});

const canSubmitCreate = computed(() => {
  if (!createDraft.platform.trim() || !createDraft.subjectID.trim()) return false;
  if (createDraft.scopeType === 'guild' && !createDraft.guildID.trim()) return false;
  return Boolean(createDraft.reasonText.trim());
});

const canSubmitRelease = computed(() => Boolean(releaseTarget.value));

async function fetchData() {
  loading.value = true;
  try {
    const params: ListMemberBlacklistParams = {
      page: query.page,
      pageSize: query.pageSize,
      status: query.status,
    };
    if (query.platform.trim()) params.platform = query.platform.trim();
    if (query.scopeType) params.scopeType = query.scopeType;
    if (query.source) params.source = query.source;
    if (query.guildID.trim()) params.guildID = query.guildID.trim();
    if (query.subjectID.trim()) params.subjectID = query.subjectID.trim();

    const data = await listMemberBlacklist(params);
    items.value = data.items;
    total.value = data.total;
  } finally {
    loading.value = false;
  }
}

function resetQuery() {
  query.platform = '';
  query.scopeType = '';
  query.source = '';
  query.status = 'active';
  query.guildID = '';
  query.subjectID = '';
  query.page = 1;
  void fetchData();
}

function openCreateDialog() {
  createDraft.platform = 'qq';
  createDraft.subjectID = '';
  createDraft.scopeType = 'guild';
  createDraft.guildID = '';
  createDraft.reasonText = '';
  createDraft.expiresAt = '';
  createDialogVisible.value = true;
}

async function submitCreate() {
  if (!canSubmitCreate.value) return;
  creating.value = true;
  try {
    const expiresAtIso = toIsoString(createDraft.expiresAt);
    await createMemberBlacklist({
      platform: createDraft.platform.trim(),
      subjectType: 'qq_user',
      subjectID: createDraft.subjectID.trim(),
      scopeType: createDraft.scopeType,
      guildID:
        createDraft.scopeType === 'guild'
          ? createDraft.guildID.trim()
          : undefined,
      source: 'manual_admin',
      reasonCode: 'manual_blacklist',
      reasonText: createDraft.reasonText.trim(),
      expiresAt: expiresAtIso,
      metadata: {
        operatorInput: createDraft.subjectID.trim(),
        scopeSelectionContext:
          createDraft.scopeType === 'global'
            ? 'admin_console_form_global'
            : 'admin_console_form_guild',
      },
    });
    ElMessage.success(`已将 ${createDraft.subjectID.trim()} 加入黑名单`);
    createDialogVisible.value = false;
    query.page = 1;
    await fetchData();
  } finally {
    creating.value = false;
  }
}

function openReleaseDialog(entry: MemberBlacklistEntry) {
  releaseTarget.value = entry;
  releaseDraft.releaseReasonCode =
    entry.source === 'admission_failure' ? 'manual_pardon' : 'release_only';
  releaseDraft.releaseReason = '';
  releaseDialogVisible.value = true;
}

async function submitRelease() {
  if (!releaseTarget.value) return;
  releasing.value = true;
  try {
    await releaseMemberBlacklist(releaseTarget.value.id, {
      releaseReasonCode: releaseDraft.releaseReasonCode,
      ...(releaseDraft.releaseReason.trim()
        ? { releaseReason: releaseDraft.releaseReason.trim() }
        : {}),
    });
    ElMessage.success(`已解除黑名单：${releaseTarget.value.subjectID}`);
    releaseDialogVisible.value = false;
    releaseTarget.value = null;
    await fetchData();
  } finally {
    releasing.value = false;
  }
}

function entryStatus(
  entry: MemberBlacklistEntry,
): 'active' | 'expired' | 'released' {
  if (entry.releasedAt) return 'released';
  if (entry.expiresAt && Date.parse(entry.expiresAt) <= Date.now()) {
    return 'expired';
  }
  return 'active';
}

function statusType(status: 'active' | 'expired' | 'released') {
  if (status === 'active') return 'danger';
  if (status === 'expired') return 'info';
  return 'success';
}

function statusLabel(status: 'active' | 'expired' | 'released') {
  if (status === 'active') return '生效中';
  if (status === 'expired') return '已过期';
  return '已解除';
}

function scopeLabel(entry: MemberBlacklistEntry) {
  return entry.scopeType === 'global' ? '全局' : `群 ${entry.guildID ?? '—'}`;
}

function sourceLabel(entry: MemberBlacklistEntry) {
  return SOURCE_LABELS[entry.source] ?? entry.source;
}

function createdFromLabel(entry: MemberBlacklistEntry) {
  return CREATED_FROM_LABELS[entry.createdFrom] ?? entry.createdFrom;
}

function createdByLabel(entry: MemberBlacklistEntry) {
  return `${entry.createdByType} · ${entry.createdByID}`;
}

function formatDateTime(value?: null | string) {
  if (!value) return '—';
  return new Date(value).toLocaleString('zh-CN', { hour12: false });
}

function toIsoString(value: Date | string): string | undefined {
  if (!value) return undefined;
  if (value instanceof Date) return value.toISOString();
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return undefined;
  return parsed.toISOString();
}

onMounted(fetchData);
</script>

<template>
  <div class="p-4">
    <div class="mb-4 flex flex-wrap items-end gap-3">
      <ElInput
        v-model="query.subjectID"
        clearable
        data-field="subjectID"
        placeholder="QQ / 主体 ID"
        style="width: 180px"
        @keyup.enter="fetchData"
      />
      <ElInput
        v-model="query.guildID"
        clearable
        data-field="guildID"
        placeholder="群号"
        style="width: 160px"
        @keyup.enter="fetchData"
      />
      <ElInput
        v-model="query.platform"
        clearable
        data-field="platform"
        placeholder="平台"
        style="width: 120px"
      />
      <ElSelect
        v-model="query.scopeType"
        data-field="scopeType"
        style="width: 140px"
      >
        <ElOption
          v-for="opt in SCOPE_OPTIONS"
          :key="opt.value || 'all'"
          :label="opt.label"
          :value="opt.value"
        />
      </ElSelect>
      <ElSelect
        v-model="query.source"
        data-field="source"
        style="width: 160px"
      >
        <ElOption
          v-for="opt in SOURCE_OPTIONS"
          :key="opt.value || 'all'"
          :label="opt.label"
          :value="opt.value"
        />
      </ElSelect>
      <ElSelect
        v-model="query.status"
        data-field="status"
        style="width: 140px"
      >
        <ElOption
          v-for="opt in STATUS_OPTIONS"
          :key="opt.value"
          :label="opt.label"
          :value="opt.value"
        />
      </ElSelect>
      <ElButton type="primary" @click="fetchData">查询</ElButton>
      <ElButton @click="resetQuery">重置</ElButton>
      <div class="flex-1" />
      <ElButton
        v-if="canManage"
        data-action="openCreate"
        type="success"
        @click="openCreateDialog"
      >
        新增黑名单
      </ElButton>
    </div>

    <ElTable v-loading="loading" :data="items" stripe>
      <ElTableColumn label="状态" width="96">
        <template #default="{ row }">
          <ElTag :type="statusType(entryStatus(row))" data-field="status">
            {{ statusLabel(entryStatus(row)) }}
          </ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn label="主体" min-width="160">
        <template #default="{ row }">
          <div class="font-mono">{{ row.subjectID }}</div>
          <div class="text-xs text-slate-500">{{ row.subjectType }}</div>
        </template>
      </ElTableColumn>
      <ElTableColumn label="范围" width="160">
        <template #default="{ row }">{{ scopeLabel(row) }}</template>
      </ElTableColumn>
      <ElTableColumn label="来源" width="120">
        <template #default="{ row }">{{ sourceLabel(row) }}</template>
      </ElTableColumn>
      <ElTableColumn label="原因" min-width="200">
        <template #default="{ row }">
          <div>{{ row.reasonText || '—' }}</div>
          <div class="text-xs text-slate-500">{{ row.reasonCode }}</div>
        </template>
      </ElTableColumn>
      <ElTableColumn label="创建入口" width="140">
        <template #default="{ row }">{{ createdFromLabel(row) }}</template>
      </ElTableColumn>
      <ElTableColumn label="创建人" width="200">
        <template #default="{ row }">
          <span class="font-mono text-xs">{{ createdByLabel(row) }}</span>
        </template>
      </ElTableColumn>
      <ElTableColumn label="创建时间" width="180">
        <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
      </ElTableColumn>
      <ElTableColumn label="过期时间" width="180">
        <template #default="{ row }">
          {{ row.expiresAt ? formatDateTime(row.expiresAt) : '永久' }}
        </template>
      </ElTableColumn>
      <ElTableColumn label="解除时间" width="180">
        <template #default="{ row }">{{ formatDateTime(row.releasedAt) }}</template>
      </ElTableColumn>
      <ElTableColumn fixed="right" label="操作" width="120">
        <template #default="{ row }">
          <ElButton
            v-if="canManage && entryStatus(row) === 'active'"
            data-action="release"
            link
            type="warning"
            @click="openReleaseDialog(row)"
          >
            解除
          </ElButton>
          <span v-else class="text-slate-400">—</span>
        </template>
      </ElTableColumn>
    </ElTable>

    <div class="mt-4 flex justify-end">
      <ElPagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :total="total"
        layout="total, prev, pager, next, sizes"
        @current-change="fetchData"
        @size-change="fetchData"
      />
    </div>

    <ElDialog
      v-model="createDialogVisible"
      data-dialog="create"
      title="新增成员黑名单"
      width="520px"
    >
      <ElForm label-position="top">
        <ElFormItem label="平台">
          <ElInput v-model="createDraft.platform" data-field="platform" />
        </ElFormItem>
        <ElFormItem label="主体 ID（QQ 号）">
          <ElInput v-model="createDraft.subjectID" data-field="subjectID" />
        </ElFormItem>
        <ElFormItem label="范围">
          <ElSelect v-model="createDraft.scopeType" data-field="scopeType">
            <ElOption label="单群" value="guild" />
            <ElOption label="全局" value="global" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem v-if="createDraft.scopeType === 'guild'" label="群号">
          <ElInput v-model="createDraft.guildID" data-field="guildID" />
        </ElFormItem>
        <ElFormItem label="原因（必填）">
          <ElInput
            v-model="createDraft.reasonText"
            :rows="3"
            data-field="reasonText"
            type="textarea"
          />
        </ElFormItem>
        <ElFormItem label="过期时间（留空表示永久）">
          <ElDatePicker
            v-model="createDraft.expiresAt"
            data-field="expiresAt"
            type="datetime"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="createDialogVisible = false">取消</ElButton>
        <ElPopconfirm
          v-if="createDraft.scopeType === 'global'"
          title="确认创建全局黑名单？该成员将被所有群拒绝。"
          @confirm="submitCreate"
        >
          <template #reference>
            <ElButton
              :disabled="!canSubmitCreate"
              :loading="creating"
              data-action="submitCreate"
              type="danger"
            >
              创建全局黑名单
            </ElButton>
          </template>
        </ElPopconfirm>
        <ElButton
          v-else
          :disabled="!canSubmitCreate"
          :loading="creating"
          data-action="submitCreate"
          type="primary"
          @click="submitCreate"
        >
          创建
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="releaseDialogVisible"
      data-dialog="release"
      title="解除成员黑名单"
      width="520px"
    >
      <div v-if="releaseTarget" class="mb-3 text-sm text-slate-600">
        <div>主体：<span class="font-mono">{{ releaseTarget.subjectID }}</span></div>
        <div>范围：{{ scopeLabel(releaseTarget) }}</div>
        <div>来源：{{ sourceLabel(releaseTarget) }}</div>
      </div>
      <ElForm label-position="top">
        <ElFormItem label="解除语义">
          <ElSelect
            v-model="releaseDraft.releaseReasonCode"
            data-field="releaseReasonCode"
          >
            <ElOption
              v-for="opt in RELEASE_REASON_OPTIONS"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="备注（可选）">
          <ElInput
            v-model="releaseDraft.releaseReason"
            :rows="2"
            data-field="releaseReason"
            type="textarea"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="releaseDialogVisible = false">取消</ElButton>
        <ElButton
          :disabled="!canSubmitRelease"
          :loading="releasing"
          data-action="submitRelease"
          type="warning"
          @click="submitRelease"
        >
          解除
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>
