<script setup lang="ts">
import type {
  OpenPlatformImportCasdoorAppRequest,
  OpenPlatformScope,
} from '#/api/admin';

import { reactive, watch } from 'vue';

import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElOption,
  ElSelect,
} from 'element-plus';

import { $t } from '#/locales';

type ScopeDraft = {
  id: number;
  reason: string;
  scope: OpenPlatformScope;
};

type ImportDraft = {
  casdoorApplicationName: string;
  clientSecret: string;
  description: string;
  displayName: string;
  homepageURL: string;
  privacyPolicyURL: string;
  redirectURIsText: string;
  scopes: ScopeDraft[];
};

defineProps<{
  submitting: boolean;
}>();

const emit = defineEmits<{
  (e: 'submit', form: OpenPlatformImportCasdoorAppRequest): void;
}>();

const visible = defineModel<boolean>('visible', { required: true });

const scopeOptions: OpenPlatformScope[] = [
  'profile.basic.read',
  'email.read',
  'phone.read',
  'stu.identity.status.read',
  'stu.identity.type.read',
  'stu.student.status.read',
  'stu.student.school.read',
  'resource.read',
  'resource.write',
  'offline_access',
];

let nextScopeID = 1;
const draft = reactive<ImportDraft>(createDefaultDraft());

watch(
  visible,
  (isVisible) => {
    if (isVisible) {
      resetDraft();
    }
  },
  { immediate: true },
);

function createDefaultDraft(): ImportDraft {
  return {
    casdoorApplicationName: '',
    clientSecret: '',
    description: '',
    displayName: '',
    homepageURL: '',
    privacyPolicyURL: '',
    redirectURIsText: '',
    scopes: [createScopeDraft('profile.basic.read')],
  };
}

function createScopeDraft(scope: OpenPlatformScope): ScopeDraft {
  return {
    id: nextScopeID++,
    reason: '',
    scope,
  };
}

function resetDraft() {
  Object.assign(draft, createDefaultDraft());
}

function addScope() {
  const used = new Set(draft.scopes.map((item) => item.scope));
  const nextScope = scopeOptions.find((scope) => !used.has(scope));
  draft.scopes.push(createScopeDraft(nextScope ?? 'profile.basic.read'));
}

function removeScope(id: number) {
  if (draft.scopes.length <= 1) {
    return;
  }
  draft.scopes = draft.scopes.filter((item) => item.id !== id);
}

function submit() {
  const casdoorApplicationName = draft.casdoorApplicationName.trim();
  const privacyPolicyURL = draft.privacyPolicyURL.trim();
  const scopes = draft.scopes.map((item) => ({
    reason: item.reason.trim(),
    scope: item.scope,
  }));
  if (!casdoorApplicationName || !privacyPolicyURL || scopes.length === 0) {
    ElMessage.warning($t('admin.openPlatform.apps.importRequired'));
    return;
  }
  const payload: OpenPlatformImportCasdoorAppRequest = {
    casdoorApplicationName,
    privacyPolicyURL,
    scopes,
  };
  assignOptionalString(payload, 'displayName', draft.displayName);
  assignOptionalString(payload, 'description', draft.description);
  assignOptionalString(payload, 'homepageURL', draft.homepageURL);
  assignOptionalString(payload, 'clientSecret', draft.clientSecret);

  const redirectURIs = parseRedirectURIs(draft.redirectURIsText);
  if (redirectURIs.length > 0) {
    payload.redirectURIs = redirectURIs;
  }
  emit('submit', payload);
}

function assignOptionalString<
  Key extends keyof Pick<
    OpenPlatformImportCasdoorAppRequest,
    'clientSecret' | 'description' | 'displayName' | 'homepageURL'
  >,
>(payload: OpenPlatformImportCasdoorAppRequest, key: Key, value: string) {
  const normalized = value.trim();
  if (normalized) {
    payload[key] = normalized;
  }
}

function parseRedirectURIs(value: string) {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const line of value.split(/\r?\n/)) {
    const uri = line.trim();
    if (!uri || seen.has(uri)) {
      continue;
    }
    seen.add(uri);
    result.push(uri);
  }
  return result;
}
</script>

<template>
  <ElDialog
    v-model="visible"
    :title="$t('admin.openPlatform.apps.importTitle')"
    width="720px"
  >
    <ElForm label-width="132px">
      <ElFormItem :label="$t('admin.openPlatform.apps.casdoorApplicationName')">
        <ElInput
          v-model="draft.casdoorApplicationName"
          :placeholder="$t('admin.openPlatform.apps.casdoorNamePlaceholder')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('admin.openPlatform.apps.displayName')">
        <ElInput
          v-model="draft.displayName"
          :placeholder="$t('admin.openPlatform.apps.displayNamePlaceholder')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('admin.common.description')">
        <ElInput
          v-model="draft.description"
          :rows="2"
          :placeholder="$t('admin.openPlatform.apps.descriptionPlaceholder')"
          type="textarea"
        />
      </ElFormItem>
      <ElFormItem :label="$t('admin.openPlatform.apps.homepage')">
        <ElInput
          v-model="draft.homepageURL"
          :placeholder="$t('admin.openPlatform.apps.homepagePlaceholder')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('admin.openPlatform.apps.privacy')">
        <ElInput
          v-model="draft.privacyPolicyURL"
          :placeholder="$t('admin.openPlatform.apps.privacyPlaceholder')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('admin.openPlatform.apps.redirects')">
        <ElInput
          v-model="draft.redirectURIsText"
          :rows="3"
          :placeholder="$t('admin.openPlatform.apps.redirectsPlaceholder')"
          type="textarea"
        />
      </ElFormItem>
      <ElFormItem :label="$t('admin.openPlatform.apps.clientSecret')">
        <ElInput
          v-model="draft.clientSecret"
          :placeholder="$t('admin.openPlatform.apps.clientSecretPlaceholder')"
          show-password
          type="password"
        />
      </ElFormItem>

      <ElFormItem :label="$t('admin.openPlatform.apps.scopes')">
        <div class="open-platform-import-scopes">
          <div
            v-for="scope in draft.scopes"
            :key="scope.id"
            class="open-platform-import-scope"
          >
            <ElSelect
              v-model="scope.scope"
              filterable
              :placeholder="$t('admin.openPlatform.apps.scopePlaceholder')"
              :teleported="false"
            >
              <ElOption
                v-for="option in scopeOptions"
                :key="option"
                :label="option"
                :value="option"
              />
            </ElSelect>
            <ElInput
              v-model="scope.reason"
              :placeholder="
                $t('admin.openPlatform.apps.scopeReasonPlaceholder')
              "
            />
            <ElButton
              plain
              type="danger"
              :disabled="draft.scopes.length <= 1"
              @click="removeScope(scope.id)"
            >
              {{ $t('admin.common.delete') }}
            </ElButton>
          </div>
          <ElButton plain type="primary" @click="addScope">
            {{ $t('admin.openPlatform.apps.addScope') }}
          </ElButton>
        </div>
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="visible = false">
        {{ $t('admin.common.cancel') }}
      </ElButton>
      <ElButton :loading="submitting" type="primary" @click="submit">
        {{ $t('admin.openPlatform.apps.importCasdoor') }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<style scoped>
.open-platform-import-scopes {
  display: grid;
  gap: 10px;
  width: 100%;
}

.open-platform-import-scope {
  display: grid;
  grid-template-columns: minmax(180px, 0.8fr) minmax(220px, 1fr) auto;
  gap: 8px;
  align-items: center;
}

@media (max-width: 768px) {
  .open-platform-import-scope {
    grid-template-columns: 1fr;
  }
}
</style>
