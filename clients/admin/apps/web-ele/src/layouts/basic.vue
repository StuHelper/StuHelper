<script lang="ts" setup>
import type { NotificationItem } from '@vben/layouts';

import { computed, ref, watch } from 'vue';
import { useRouter } from 'vue-router';

import { AuthenticationLoginExpiredModal } from '@vben/common-ui';
import { VBEN_DOC_URL, VBEN_GITHUB_URL } from '@vben/constants';
import { useWatermark } from '@vben/hooks';
import { BookOpenText, CircleHelp, SvgGithubIcon } from '@vben/icons';
import {
  BasicLayout,
  LockScreen,
  Notification,
  UserDropdown,
} from '@vben/layouts';
import { preferences } from '@vben/preferences';
import { useAccessStore, useUserStore } from '@vben/stores';
import { openWindow } from '@vben/utils';

import { $t } from '#/locales';
import { useAuthStore } from '#/store';
import LoginForm from '#/views/_core/authentication/login.vue';

function createNotifications(): NotificationItem[] {
  return [
    {
      id: 1,
      avatar: 'https://avatar.vercel.sh/vercel.svg?text=VB',
      date: $t('admin.layout.notifications.time.threeHoursAgo'),
      isRead: true,
      message: $t('admin.layout.notifications.items.weeklyReports.message'),
      title: $t('admin.layout.notifications.items.weeklyReports.title'),
    },
    {
      id: 2,
      avatar: 'https://avatar.vercel.sh/1',
      date: $t('admin.layout.notifications.time.justNow'),
      isRead: false,
      message: $t('admin.layout.notifications.items.reply.message'),
      title: $t('admin.layout.notifications.items.reply.title'),
    },
    {
      id: 3,
      avatar: 'https://avatar.vercel.sh/1',
      date: '2024-01-01',
      isRead: false,
      message: $t('admin.layout.notifications.items.comment.message'),
      title: $t('admin.layout.notifications.items.comment.title'),
    },
    {
      id: 4,
      avatar: 'https://avatar.vercel.sh/satori',
      date: $t('admin.layout.notifications.time.oneDayAgo'),
      isRead: false,
      message: $t('admin.layout.notifications.items.todo.message'),
      title: $t('admin.layout.notifications.items.todo.title'),
    },
    {
      id: 5,
      avatar: 'https://avatar.vercel.sh/satori',
      date: $t('admin.layout.notifications.time.oneDayAgo'),
      isRead: false,
      message: $t('admin.layout.notifications.items.workspace.message'),
      title: $t('admin.layout.notifications.items.workspace.title'),
      link: '/workspace',
    },
    {
      id: 6,
      avatar: 'https://avatar.vercel.sh/satori',
      date: $t('admin.layout.notifications.time.oneDayAgo'),
      isRead: false,
      message: $t('admin.layout.notifications.items.external.message'),
      title: $t('admin.layout.notifications.items.external.title'),
      link: 'https://doc.vben.pro',
    },
  ];
}

const notifications = ref<NotificationItem[]>(createNotifications());

const router = useRouter();
const userStore = useUserStore();
const authStore = useAuthStore();
const accessStore = useAccessStore();
const { destroyWatermark, updateWatermark } = useWatermark();
const showDot = computed(() =>
  notifications.value.some((item) => !item.isRead),
);

const menus = computed(() => [
  {
    handler: () => {
      router.push({ name: 'Profile' });
    },
    icon: 'lucide:user',
    text: $t('page.auth.profile'),
  },
  {
    handler: () => {
      openWindow(VBEN_DOC_URL, {
        target: '_blank',
      });
    },
    icon: BookOpenText,
    text: $t('ui.widgets.document'),
  },
  {
    handler: () => {
      openWindow(VBEN_GITHUB_URL, {
        target: '_blank',
      });
    },
    icon: SvgGithubIcon,
    text: 'GitHub',
  },
  {
    handler: () => {
      openWindow(`${VBEN_GITHUB_URL}/issues`, {
        target: '_blank',
      });
    },
    icon: CircleHelp,
    text: $t('ui.widgets.qa'),
  },
]);

const avatar = computed(() => {
  return userStore.userInfo?.avatar ?? preferences.app.defaultAvatar;
});

async function handleLogout() {
  await authStore.logout(false);
}

function handleNoticeClear() {
  notifications.value = [];
}

function markRead(id: number | string) {
  const item = notifications.value.find((item) => item.id === id);
  if (item) {
    item.isRead = true;
  }
}

function remove(id: number | string) {
  notifications.value = notifications.value.filter((item) => item.id !== id);
}

function handleMakeAll() {
  notifications.value.forEach((item) => (item.isRead = true));
}
watch(
  () => ({
    enable: preferences.app.watermark,
    content: preferences.app.watermarkContent,
  }),
  async ({ enable, content }) => {
    if (enable) {
      await updateWatermark({
        content:
          content ||
          `${userStore.userInfo?.username} - ${userStore.userInfo?.realName}`,
      });
    } else {
      destroyWatermark();
    }
  },
  {
    immediate: true,
  },
);
</script>

<template>
  <BasicLayout @clear-preferences-and-logout="handleLogout">
    <template #user-dropdown>
      <UserDropdown
        :avatar
        :menus
        :text="userStore.userInfo?.realName"
        description="ann.vben@gmail.com"
        tag-text="Pro"
        @logout="handleLogout"
      />
    </template>
    <template #notification>
      <Notification
        :dot="showDot"
        :notifications="notifications"
        @clear="handleNoticeClear"
        @read="(item) => item.id && markRead(item.id)"
        @remove="(item) => item.id && remove(item.id)"
        @make-all="handleMakeAll"
      />
    </template>
    <template #extra>
      <AuthenticationLoginExpiredModal
        v-model:open="accessStore.loginExpired"
        :avatar
      >
        <LoginForm />
      </AuthenticationLoginExpiredModal>
    </template>
    <template #lock-screen>
      <LockScreen :avatar @to-login="handleLogout" />
    </template>
  </BasicLayout>
</template>
