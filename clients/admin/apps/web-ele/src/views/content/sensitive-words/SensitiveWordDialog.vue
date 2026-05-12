<script setup lang="ts">
import { ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElOption, ElSelect, ElSwitch } from 'element-plus';

import { $t } from '#/locales';

type Level = 'block' | 'review' | 'warn';

defineProps<{
  form: {
    category: string;
    id: string;
    isActive: boolean;
    level: Level;
    word: string;
  };
  isEdit: boolean;
}>();

const visible = defineModel<boolean>('visible', { required: true });
const emit = defineEmits<{
  (e: 'submit'): void;
}>();
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
          v-model="form.word"
          :placeholder="$t('admin.content.sensitiveWords.wordPlaceholder')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('admin.content.sensitiveWords.category')">
        <ElInput
          v-model="form.category"
          :placeholder="$t('admin.content.sensitiveWords.categoryPlaceholder')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('admin.content.sensitiveWords.level')">
        <ElSelect
          v-model="form.level"
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
        <ElSwitch v-model="form.isActive" />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="visible = false">
        {{ $t('admin.common.cancel') }}
      </ElButton>
      <ElButton type="primary" @click="emit('submit')">
        {{ $t('admin.common.confirm') }}
      </ElButton>
    </template>
  </ElDialog>
</template>
