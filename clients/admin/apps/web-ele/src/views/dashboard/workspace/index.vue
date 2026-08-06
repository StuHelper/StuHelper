<script lang="ts" setup>
import type { AdminStats } from '#/api/admin';

import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';

import { IconifyIcon } from '@vben/icons';
import { useAccessStore } from '@vben/stores';

import { ElAlert, ElButton, ElSkeleton } from 'element-plus';

import { getAdminStats } from '#/api/admin';
import { $t } from '#/locales';

import AdminContentLayout from '../../shared/AdminContentLayout.vue';

import './workspace.css';

const router = useRouter();
const accessStore = useAccessStore();
const loading = ref(true);
const loadError = ref('');
const stats = ref<AdminStats | null>(null);
let fetchRequestSeq = 0;

interface WorkspaceLink {
  authority: string[];
  icon: string;
  label: string;
  path: string;
}

interface WorkspaceQueueItem extends WorkspaceLink {
  tone: string;
  value: number;
}

function canAccess(authority: string[]) {
  return authority.some((code) => accessStore.accessCodes.includes(code));
}

const queueItems = computed(() => {
  const current = stats.value;
  return [
    {
      authority: ['admin:reports:manage'],
      icon: 'lucide:flag',
      label: '待处理举报',
      path: '/content/reports',
      tone: 'amber',
      value: current?.pendingReports ?? 0,
    },
    {
      authority: ['admin:reviews:manage'],
      icon: 'lucide:eye-off',
      label: '隐藏评课',
      path: '/content/reviews',
      tone: 'slate',
      value: current?.hiddenReviews ?? 0,
    },
    {
      authority: ['admin:reviews:manage'],
      icon: 'lucide:calendar-plus',
      label: '本周新增评课',
      path: '/content/reviews',
      tone: 'green',
      value: current?.weekReviews ?? 0,
    },
  ] satisfies WorkspaceQueueItem[];
});

const visibleQueueItems = computed(() =>
  queueItems.value.filter((item) => canAccess(item.authority)),
);

const overviewItems = computed(() => {
  const current = stats.value;
  return [
    {
      label: '评课总量',
      value: current?.totalReviews ?? 0,
    },
    {
      label: '已发布评课',
      value: current?.publishedReviews ?? 0,
    },
    {
      label: '举报总量',
      value: current?.totalReports ?? 0,
    },
    {
      label: '今日新增评课',
      value: current?.todayReviews ?? 0,
    },
  ];
});

const shortcuts: WorkspaceLink[] = [
  {
    authority: ['admin:reviews:manage'],
    icon: 'lucide:message-square-text',
    label: '评课管理',
    path: '/content/reviews',
  },
  {
    authority: ['admin:reports:manage'],
    icon: 'lucide:flag',
    label: '举报管理',
    path: '/content/reports',
  },
  {
    authority: ['student:manual_review:read', 'student:manual_review:decide'],
    icon: 'lucide:badge-check',
    label: '学生材料审核',
    path: '/users/student-verification',
  },
  {
    authority: ['member_blacklist:read', 'member_blacklist:manage'],
    icon: 'lucide:shield-check',
    label: '成员黑名单',
    path: '/users/member-blacklist',
  },
];

const visibleShortcuts = computed(() =>
  shortcuts.filter((item) => canAccess(item.authority)),
);

async function fetchStats() {
  const requestSeq = ++fetchRequestSeq;
  loading.value = true;
  loadError.value = '';
  try {
    const data = await getAdminStats();
    if (requestSeq !== fetchRequestSeq) return;
    stats.value = data;
  } catch (error) {
    if (requestSeq !== fetchRequestSeq) return;
    loadError.value = adminErrorMessage(error);
  } finally {
    if (requestSeq === fetchRequestSeq) {
      loading.value = false;
    }
  }
}

async function goTo(path: string) {
  await router.push(path);
}

function adminErrorMessage(error: unknown): string {
  return error instanceof Error && error.message
    ? error.message
    : $t('admin.result.requestFailed');
}

onMounted(fetchStats);
</script>

<template>
  <AdminContentLayout
    :description="$t('admin.dashboard.summary.title')"
    :title="$t('page.dashboard.workspace')"
  >
    <template #actions>
      <ElButton :loading="loading" @click="fetchStats">
        {{ $t('admin.common.query') }}
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
      <ElButton size="small" :loading="loading" @click="fetchStats">
        {{ $t('admin.common.retry') }}
      </ElButton>
    </ElAlert>

    <ElSkeleton :loading="loading" animated>
      <template #template>
        <div class="admin-workspace__skeleton">
          <div
            v-for="item in 4"
            :key="item"
            class="admin-workspace__block"
          ></div>
          <div class="admin-workspace__wide"></div>
        </div>
      </template>

      <section class="admin-workspace__overview" aria-label="后台统计">
        <article
          v-for="item in overviewItems"
          :key="item.label"
          class="admin-workspace__metric"
        >
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}</strong>
        </article>
      </section>

      <section class="admin-workspace__main">
        <div v-if="visibleQueueItems.length > 0" class="admin-workspace__panel">
          <div class="admin-workspace__panel-header">
            <h2>处理队列</h2>
          </div>
          <button
            v-for="item in visibleQueueItems"
            :key="item.label"
            class="admin-workspace__queue"
            :data-tone="item.tone"
            type="button"
            @click="goTo(item.path)"
          >
            <span>
              <IconifyIcon :icon="item.icon" />
              {{ item.label }}
            </span>
            <strong>{{ item.value }}</strong>
          </button>
        </div>

        <div v-if="visibleShortcuts.length > 0" class="admin-workspace__panel">
          <div class="admin-workspace__panel-header">
            <h2>常用入口</h2>
          </div>
          <div class="admin-workspace__shortcuts">
            <button
              v-for="item in visibleShortcuts"
              :key="item.path"
              class="admin-workspace__shortcut"
              type="button"
              @click="goTo(item.path)"
            >
              <IconifyIcon :icon="item.icon" />
              <span>{{ item.label }}</span>
            </button>
          </div>
        </div>
      </section>
    </ElSkeleton>
  </AdminContentLayout>
</template>
