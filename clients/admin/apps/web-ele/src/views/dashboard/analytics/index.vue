<script lang="ts" setup>
import type { AdminStats } from '#/api/admin';

import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';

import { IconifyIcon } from '@vben/icons';
import { useAccessStore } from '@vben/stores';

import { ElAlert, ElButton, ElSkeleton } from 'element-plus';

import { getAdminStats } from '#/api/admin';
import { $t } from '#/locales';

import './analytics.css';

const router = useRouter();
const accessStore = useAccessStore();
const loading = ref(true);
const loadError = ref('');
const stats = ref<AdminStats | null>(null);
let fetchRequestSeq = 0;

interface DashboardShortcut {
  authority: string[];
  icon: string;
  label: string;
  path: string;
}

const allQuickActions: DashboardShortcut[] = [
  {
    authority: ['admin:reviews:manage'],
    icon: 'lucide:message-square',
    label: $t('admin.dashboard.quickActions.reviews'),
    path: '/content/reviews',
  },
  {
    authority: ['admin:reports:manage'],
    icon: 'lucide:flag',
    label: $t('admin.dashboard.quickActions.reports'),
    path: '/content/reports',
  },
  {
    authority: ['admin:teachers:manage'],
    icon: 'lucide:graduation-cap',
    label: $t('admin.dashboard.quickActions.teachers'),
    path: '/content/teachers',
  },
  {
    authority: ['student:manual_review:read', 'student:manual_review:decide'],
    icon: 'lucide:badge-check',
    label: $t('admin.dashboard.quickActions.students'),
    path: '/users/student-verification',
  },
  {
    authority: ['student:credential:read', 'student:subject_conflict:read'],
    icon: 'lucide:shield-alert',
    label: $t('admin.routes.userSystem.studentCredentials'),
    path: '/users/student-credentials',
  },
];

function canAccess(authority: string[]) {
  return authority.some((code) => accessStore.accessCodes.includes(code));
}

const quickActions = computed(() =>
  allQuickActions.filter((action) => canAccess(action.authority)),
);

const overviewItems = computed(() => {
  const current = stats.value;
  return [
    {
      icon: 'lucide:message-square-text',
      tone: 'blue',
      title: $t('admin.dashboard.analytics.overview.users.title'),
      totalTitle: $t('admin.dashboard.analytics.overview.users.totalTitle'),
      totalValue: current?.totalReviews ?? 0,
      value: current?.todayReviews ?? 0,
    },
    {
      icon: 'lucide:flag',
      tone: 'amber',
      title: $t('admin.dashboard.analytics.overview.visits.title'),
      totalTitle: $t('admin.dashboard.analytics.overview.visits.totalTitle'),
      totalValue: current?.totalReports ?? 0,
      value: current?.pendingReports ?? 0,
    },
    {
      icon: 'lucide:eye-off',
      tone: 'slate',
      title: $t('admin.dashboard.analytics.overview.downloads.title'),
      totalTitle: $t('admin.dashboard.analytics.overview.downloads.totalTitle'),
      totalValue: current?.publishedReviews ?? 0,
      value: current?.hiddenReviews ?? 0,
    },
    {
      icon: 'lucide:calendar-plus',
      tone: 'green',
      title: $t('admin.dashboard.analytics.overview.usage.title'),
      totalTitle: $t('admin.dashboard.analytics.overview.usage.totalTitle'),
      totalValue: current?.deletedReviews ?? 0,
      value: current?.weekReviews ?? 0,
    },
  ];
});

const summaryItems = computed(() => {
  const current = stats.value;
  return [
    {
      label: $t('admin.dashboard.summary.publishedReviews'),
      value: current?.publishedReviews ?? 0,
    },
    {
      label: $t('admin.dashboard.summary.deletedReviews'),
      value: current?.deletedReviews ?? 0,
    },
    {
      label: $t('admin.dashboard.summary.totalReports'),
      value: current?.totalReports ?? 0,
    },
    {
      label: $t('admin.dashboard.summary.pendingReports'),
      value: current?.pendingReports ?? 0,
    },
  ];
});

const moderationLoad = computed(() => {
  const current = stats.value;
  if (!current?.totalReports) return 0;
  return Math.round((current.pendingReports / current.totalReports) * 100);
});

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

async function navTo(path: string) {
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
  <div class="admin-dashboard">
    <ElSkeleton :loading="loading" animated>
      <template #template>
        <div class="admin-dashboard__skeleton">
          <div
            v-for="item in 4"
            :key="item"
            class="admin-dashboard__block"
          ></div>
          <div class="admin-dashboard__wide"></div>
          <div class="admin-dashboard__wide"></div>
        </div>
      </template>

      <section class="admin-dashboard__header">
        <div>
          <h1>{{ $t('page.dashboard.analytics') }}</h1>
          <p>{{ $t('admin.dashboard.summary.title') }}</p>
        </div>
        <ElButton :loading="loading" @click="fetchStats">
          {{ $t('admin.common.query') }}
        </ElButton>
      </section>

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

      <section class="admin-dashboard__kpis">
        <article
          v-for="item in overviewItems"
          :key="item.title"
          class="admin-dashboard__kpi"
          :data-tone="item.tone"
        >
          <div class="admin-dashboard__kpi-top">
            <span>{{ item.title }}</span>
            <IconifyIcon :icon="item.icon" />
          </div>
          <strong>{{ item.value }}</strong>
          <div class="admin-dashboard__kpi-foot">
            <span>{{ item.totalTitle }}</span>
            <span>{{ item.totalValue }}</span>
          </div>
        </article>
      </section>

      <section class="admin-dashboard__main">
        <div v-if="quickActions.length > 0" class="admin-dashboard__panel">
          <div class="admin-dashboard__panel-title">
            {{ $t('admin.dashboard.quickActions.title') }}
          </div>
          <button
            v-for="action in quickActions"
            :key="action.path"
            class="admin-dashboard__shortcut"
            type="button"
            @click="navTo(action.path)"
          >
            <IconifyIcon :icon="action.icon" />
            <span>{{ action.label }}</span>
          </button>
        </div>

        <div class="admin-dashboard__panel">
          <div class="admin-dashboard__panel-title">
            {{ $t('admin.dashboard.summary.title') }}
          </div>
          <div class="admin-dashboard__summary-list">
            <div
              v-for="item in summaryItems"
              :key="item.label"
              class="admin-dashboard__summary-row"
            >
              <span>{{ item.label }}</span>
              {{ item.value }}
            </div>
          </div>
          <div class="admin-dashboard__meter">
            <span>{{ $t('admin.dashboard.summary.pendingReports') }}</span>
            <strong>{{ moderationLoad }}%</strong>
            <div>
              <i :style="{ width: `${moderationLoad}%` }"></i>
            </div>
          </div>
        </div>
      </section>
    </ElSkeleton>
  </div>
</template>
