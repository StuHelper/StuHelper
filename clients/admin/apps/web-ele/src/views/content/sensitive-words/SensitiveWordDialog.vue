<script setup lang="ts">
import { reactive, watch } from 'vue';

import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElOption,
  ElSelect,
  ElSwitch,
} from 'element-plus';

import { $t } from '#/locales';

type Level = 'block' | 'review' | 'warn';
type SensitiveWordForm = {
  category: string;
  id: string;
  isActive: boolean;
  level: Level;
  word: string;
};

const props = defineProps<{
  form: SensitiveWordForm;
  isEdit: boolean;
}>();

const emit = defineEmits<{
  (e: 'submit', form: SensitiveWordForm): void;
}>();
const visible = defineModel<boolean>('visible', { required: true });
const draft = reactive<SensitiveWordForm>({
  category: '',
  id: '',
  isActive: true,
  level: 'block',
  word: '',
});

watch(
  () => props.form,
  (form) => {
    Object.assign(draft, form);
  },
  { deep: true, immediate: true },
);

function submit() {
  emit('submit', { ...draft });
}
</script>

<template>
  <ElDialog
    v-model="visible"
    :title="
      isEdit
        ? $t('admin.content.sensitiveWords.editTitle')
        : $t('admin.content.sensitiveWords.createTitle')
    "
    width="480px"
  >
    <ElForm label-width="80px">
      <ElFormItem :label="$t('admin.content.sensitiveWords.word')">
        <ElInput
          v-model="draft.word"
          :placeholder="$t('admin.content.sensitiveWords.wordPlaceholder')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('admin.content.sensitiveWords.category')">
        <ElInput
          v-model="draft.category"
          :placeholder="$t('admin.content.sensitiveWords.categoryPlaceholder')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('admin.content.sensitiveWords.level')">
        <ElSelect
          v-model="draft.level"
          :placeholder="$t('admin.content.sensitiveWords.levelPlaceholder')"
          style="width: 100%"
        >
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
      </ElFormItem>
      <ElFormItem :label="$t('admin.content.sensitiveWords.active')">
        <ElSwitch v-model="draft.isActive" />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="visible = false">
        {{ $t('admin.common.cancel') }}
      </ElButton>
      <ElButton type="primary" @click="submit">
        {{ $t('admin.common.confirm') }}
      </ElButton>
    </template>
  </ElDialog>
</template>
