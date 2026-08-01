<script setup lang="ts">
import type {
  OpenPlatformAppWithScopes,
  OpenPlatformResourceAccessAction,
  OpenPlatformResourceGrant,
  OpenPlatformResourceGrantRequest,
  OpenPlatformResourceType,
  OpenPlatformScope,
} from '#/api/admin';

import { computed, reactive, ref, watch } from 'vue';

import {
  ElAlert,
  ElButton,
  ElCheckbox,
  ElCheckboxGroup,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElMessageBox,
  ElOption,
  ElSelect,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import {
  getOpenPlatformResourceGrants,
  grantOpenPlatformResourceAccess,
  revokeOpenPlatformResourceAccess,
} from '#/api/admin';
import { $t } from '#/locales';

const props = defineProps<{
  app: null | OpenPlatformAppWithScopes;
}>();

const visible = defineModel<boolean>('visible', { required: true });

const resourceIDPattern = /^[A-Za-z0-9_-]{1,128}$/;
const resourceTypes: OpenPlatformResourceType[] = [
  'resource_item',
  'user_profile',
];
const actionScopeMap: Record<
  OpenPlatformResourceAccessAction,
  OpenPlatformScope
> = {
  read: 'resource.read',
  write: 'resource.write',
};

const loading = ref(false);
const actionLoading = ref(false);
const loadError = ref('');
const actionError = ref('');
const grants = ref<OpenPlatformResourceGrant[]>([]);
let fetchRequestSeq = 0;
const selectedResourceType = ref<OpenPlatformResourceType>('resource_item');
const draft = reactive<{
  actions: OpenPlatformResourceAccessAction[];
  reason: string;
  resourceID: string;
}>({
  actions: [],
  reason: '',
  resourceID: '',
});

const dialogTitle = computed(() => {
  const name = props.app?.app.displayName ?? '';
  return name
    ? $t('admin.openPlatform.apps.resourceGrantsTitle', { app: name })
    : $t('admin.openPlatform.apps.resourceGrants');
});

const availableActions = computed<OpenPlatformResourceAccessAction[]>(() =>
  selectedResourceType.value === 'user_profile' ? ['read'] : ['read', 'write'],
);

const grantableActions = computed(() =>
  availableActions.value.filter((action) => hasApprovedResourceScope(action)),
);

const canGrantResourceAccess = computed(
  () =>
    props.app?.app.status === 'approved' && grantableActions.value.length > 0,
);

const canSubmitGrant = computed(
  () => canGrantResourceAccess.value && !loadError.value,
);

watch(visible, (isVisible) => {
  if (!isVisible) {
    fetchRequestSeq += 1;
    loading.value = false;
    return;
  }
  selectedResourceType.value = 'resource_item';
  grants.value = [];
  loadError.value = '';
  actionError.value = '';
  resetDraft();
  void fetchGrants();
});

watch(selectedResourceType, () => {
  if (!visible.value) {
    return;
  }
  resetDraft();
  loadError.value = '';
  actionError.value = '';
  void fetchGrants();
});

function resetDraft() {
  draft.resourceID = '';
  draft.reason = '';
  draft.actions = grantableActions.value.slice(0, 1);
}

function hasApprovedResourceScope(action: OpenPlatformResourceAccessAction) {
  const requiredScope = actionScopeMap[action];
  return (
    props.app?.scopes.some(
      (scope) => scope.scope === requiredScope && scope.status === 'approved',
    ) ?? false
  );
}

function actionLabel(action: OpenPlatformResourceAccessAction) {
  return $t(`admin.openPlatform.apps.resourceAction.${action}`);
}

function resourceTypeLabel(resourceType: OpenPlatformResourceType) {
  return $t(`admin.openPlatform.apps.resourceType.${resourceType}`);
}

function grantUnavailableMessage() {
  if (props.app?.app.status !== 'approved') {
    return $t('admin.openPlatform.apps.resourceGrantUnavailableApp');
  }
  if (grantableActions.value.length === 0) {
    return $t('admin.openPlatform.apps.resourceGrantMissingScope');
  }
  return '';
}

async function fetchGrants() {
  const requestSeq = ++fetchRequestSeq;
  if (!props.app) {
    grants.value = [];
    loading.value = false;
    return;
  }
  loading.value = true;
  loadError.value = '';
  try {
    const result = await getOpenPlatformResourceGrants(
      props.app.app.id,
      selectedResourceType.value,
    );
    if (requestSeq !== fetchRequestSeq) return;
    grants.value = result.grants;
  } catch (error) {
    if (requestSeq !== fetchRequestSeq) return;
    loadError.value = adminErrorMessage(error);
  } finally {
    if (requestSeq === fetchRequestSeq) {
      loading.value = false;
    }
  }
}

async function submitGrant() {
  if (!props.app || actionLoading.value) {
    return;
  }
  if (!canSubmitGrant.value) {
    ElMessage.warning(loadError.value || grantUnavailableMessage());
    return;
  }
  const resourceID = draft.resourceID.trim();
  if (!resourceIDPattern.test(resourceID)) {
    ElMessage.warning($t('admin.openPlatform.apps.resourceGrantInvalid'));
    return;
  }
  const actions = draft.actions.filter(
    (action: OpenPlatformResourceAccessAction) =>
      grantableActions.value.includes(action),
  );
  if (actions.length === 0) {
    ElMessage.warning(
      $t('admin.openPlatform.apps.resourceGrantActionRequired'),
    );
    return;
  }
  const reason = draft.reason.trim();
  if (!reason) {
    ElMessage.warning($t('admin.openPlatform.apps.reasonRequired'));
    return;
  }
  const payload: OpenPlatformResourceGrantRequest = {
    actions,
    reason,
    resourceID,
    resourceType: selectedResourceType.value,
  };

  actionLoading.value = true;
  actionError.value = '';
  try {
    const result = await grantOpenPlatformResourceAccess(
      props.app.app.id,
      payload,
    );
    grants.value = result.grants;
    ElMessage.success($t('admin.openPlatform.apps.resourceGranted'));
    resetDraft();
    await fetchGrants();
  } catch (error) {
    handleActionError(error);
  } finally {
    actionLoading.value = false;
  }
}

async function revokeGrant(grant: OpenPlatformResourceGrant) {
  if (!props.app || actionLoading.value) {
    return;
  }
  const reason = await promptRevokeReason(grant);
  if (reason === null) {
    return;
  }
  const payload: OpenPlatformResourceGrantRequest = {
    actions: [grant.action],
    reason,
    resourceID: grant.resourceID,
    resourceType: grant.resourceType,
  };

  actionLoading.value = true;
  actionError.value = '';
  try {
    const result = await revokeOpenPlatformResourceAccess(
      props.app.app.id,
      payload,
    );
    grants.value = result.grants;
    ElMessage.success($t('admin.openPlatform.apps.resourceRevoked'));
    await fetchGrants();
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

async function promptRevokeReason(grant: OpenPlatformResourceGrant) {
  try {
    const result = await ElMessageBox.prompt(
      $t('admin.openPlatform.apps.revokeResourcePrompt', {
        action: actionLabel(grant.action),
        resource: grant.resourceID,
      }),
      $t('admin.openPlatform.apps.revokeResourceTitle'),
      {
        cancelButtonText: $t('admin.common.cancel'),
        confirmButtonText: $t('admin.common.confirm'),
        inputPlaceholder: $t(
          'admin.openPlatform.apps.resourceGrantReasonPlaceholder',
        ),
        inputValidator: (value) =>
          value.trim().length > 0 ||
          $t('admin.openPlatform.apps.reasonRequired'),
      },
    );
    return result.value.trim();
  } catch (_error) {
    void _error;
    return null;
  }
}
</script>

<template>
  <ElDialog v-model="visible" :title="dialogTitle" width="860px">
    <div v-if="app" class="open-platform-resource-grants">
      <div class="open-platform-resource-grants__summary">
        <div>
          <div class="admin-cell-title">{{ app.app.displayName }}</div>
          <div class="admin-cell-muted">{{ app.app.clientID }}</div>
        </div>
        <ElSelect
          v-model="selectedResourceType"
          class="open-platform-resource-grants__type"
          :teleported="false"
        >
          <ElOption
            v-for="resourceType in resourceTypes"
            :key="resourceType"
            :label="resourceTypeLabel(resourceType)"
            :value="resourceType"
          />
        </ElSelect>
      </div>

      <ElAlert
        class="open-platform-resource-grants__hint"
        show-icon
        type="info"
        :closable="false"
        :title="$t('admin.openPlatform.apps.resourceGrantHint')"
      />

      <ElAlert
        v-if="!canGrantResourceAccess"
        class="open-platform-resource-grants__hint"
        show-icon
        type="warning"
        :closable="false"
        :title="grantUnavailableMessage()"
      />

      <ElAlert
        v-if="loadError"
        class="open-platform-resource-grants__hint"
        show-icon
        type="error"
        :closable="false"
        :title="loadError"
      >
        <ElButton size="small" :loading="loading" @click="fetchGrants">
          {{ $t('admin.common.retry') }}
        </ElButton>
      </ElAlert>

      <ElAlert
        v-if="actionError"
        class="open-platform-resource-grants__hint"
        show-icon
        type="error"
        :closable="true"
        :title="actionError"
        @close="actionError = ''"
      />

      <ElForm class="open-platform-resource-grants__form" label-position="top">
        <ElFormItem :label="$t('admin.openPlatform.apps.resourceID')">
          <ElInput
            v-model="draft.resourceID"
            :disabled="!canSubmitGrant || actionLoading"
            :placeholder="$t('admin.openPlatform.apps.resourceIDPlaceholder')"
          />
        </ElFormItem>
        <ElFormItem :label="$t('admin.openPlatform.apps.resourceActions')">
          <ElCheckboxGroup v-model="draft.actions">
            <ElCheckbox
              v-for="action in availableActions"
              :key="action"
              :label="action"
              :value="action"
              :disabled="
                !canSubmitGrant ||
                !hasApprovedResourceScope(action) ||
                actionLoading
              "
            >
              {{ actionLabel(action) }}
            </ElCheckbox>
          </ElCheckboxGroup>
        </ElFormItem>
        <ElFormItem :label="$t('admin.openPlatform.apps.reasonTitle')" required>
          <ElInput
            v-model="draft.reason"
            :disabled="!canSubmitGrant || actionLoading"
            :rows="2"
            :placeholder="
              $t('admin.openPlatform.apps.resourceGrantReasonPlaceholder')
            "
            type="textarea"
          />
        </ElFormItem>
        <ElFormItem>
          <ElButton
            type="primary"
            :disabled="!canSubmitGrant"
            :loading="actionLoading"
            @click="submitGrant"
          >
            {{ $t('admin.openPlatform.apps.grantResource') }}
          </ElButton>
          <ElButton :disabled="loading" @click="fetchGrants">
            {{ $t('admin.common.query') }}
          </ElButton>
        </ElFormItem>
      </ElForm>

      <ElTable
        v-loading="loading"
        border
        class="open-platform-resource-grants__table"
        :data="grants"
        row-key="resourceID"
      >
        <ElTableColumn
          prop="resourceID"
          :label="$t('admin.openPlatform.apps.resourceID')"
          min-width="180"
        />
        <ElTableColumn
          prop="action"
          :label="$t('admin.openPlatform.apps.resourceActions')"
          width="116"
        >
          <template #default="{ row }">
            <ElTag size="small">
              {{ actionLabel(row.action) }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="relation"
          label="OpenFGA relation"
          min-width="180"
        />
        <ElTableColumn
          fixed="right"
          :label="$t('admin.common.actions')"
          width="120"
        >
          <template #default="{ row }">
            <ElButton
              plain
              size="small"
              type="danger"
              :disabled="actionLoading || Boolean(loadError)"
              @click="revokeGrant(row as OpenPlatformResourceGrant)"
            >
              {{ $t('admin.openPlatform.apps.revokeGrant') }}
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
    </div>
  </ElDialog>
</template>

<style scoped>
.open-platform-resource-grants {
  display: grid;
  gap: 14px;
}

.open-platform-resource-grants__summary {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;
  min-width: 0;
}

.open-platform-resource-grants__type {
  flex: 0 0 auto;
  width: 180px;
}

.open-platform-resource-grants__hint {
  margin: 0;
}

.open-platform-resource-grants__form {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) minmax(160px, 0.8fr) minmax(
      220px,
      1fr
    );
  gap: 12px;
  align-items: end;
}

.open-platform-resource-grants__form :deep(.el-form-item) {
  margin-bottom: 0;
}

.open-platform-resource-grants__table {
  width: 100%;
}

@media (max-width: 760px) {
  .open-platform-resource-grants__summary,
  .open-platform-resource-grants__form {
    display: grid;
    grid-template-columns: 1fr;
  }

  .open-platform-resource-grants__type {
    width: 100%;
  }
}
</style>
