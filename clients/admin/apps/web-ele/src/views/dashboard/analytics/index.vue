<script lang="ts" setup>
import type { AnalysisOverviewItem } from '@vben/common-ui';

import type { AdminStats } from '#/api/admin';

import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';

import { AnalysisChartCard, AnalysisOverview } from '@vben/common-ui';
import {
  SvgBellIcon,
  SvgCakeIcon,
  SvgCardIcon,
  SvgDownloadIcon,
} from '@vben/icons';

import {
  ElButton,
  ElDescriptions,
  ElDescriptionsItem,
  ElSkeleton,
} from 'element-plus';

import { getAdminStats } from '#/api/admin';
import { $t } from '#/locales';

const router = useRouter();
const loading = ref(true);
const stats = ref<AdminStats | null>(null);

const quickActions = [
  {
    label: $t('admin.dashboard.quickActions.reviews'),
    path: '/content/reviews',
  },
  {
    label: $t('admin.dashboard.quickActions.reports'),
    path: '/content/reports',
  },
  {
    label: $t('admin.dashboard.quickActions.teachers'),
    path: '/content/teachers',
  },
  {
    label: $t('admin.dashboard.quickActions.identity'),
    path: '/users/identity-review',
  },
  {
    label: $t('admin.dashboard.quickActions.students'),
    path: '/users/student-verification',
  },
];

const overviewItems = computed<AnalysisOverviewItem[]>(() => {
  const current = stats.value;
  return [
    {
      icon: SvgCardIcon,
      title: $t('admin.dashboard.analytics.overview.users.title'),
      totalTitle: $t('admin.dashboard.analytics.overview.users.totalTitle'),
      totalValue: current?.totalReviews ?? 0,
      value: current?.todayReviews ?? 0,
    },
    {
      icon: SvgCakeIcon,
      title: $t('admin.dashboard.analytics.overview.visits.title'),
      totalTitle: $t('admin.dashboard.analytics.overview.visits.totalTitle'),
      totalValue: current?.totalReports ?? 0,
      value: current?.pendingReports ?? 0,
    },
    {
      icon: SvgDownloadIcon,
      title: $t('admin.dashboard.analytics.overview.downloads.title'),
      totalTitle: $t('admin.dashboard.analytics.overview.downloads.totalTitle'),
      totalValue: current?.publishedReviews ?? 0,
      value: current?.hiddenReviews ?? 0,
    },
    {
      icon: SvgBellIcon,
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

async function fetchStats() {
  loading.value = true;
  try {
    stats.value = await getAdminStats();
  } finally {
    loading.value = false;
  }
}

async function navTo(path: string) {
  await router.push(path);
}

onMounted(fetchStats);
</script>

<template>
  <div class="p-5">
    <ElSkeleton :loading="loading" animated>
      <template #template>
        <div class="space-y-5">
          <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <div
              v-for="item in 4"
              :key="item"
              class="h-28 rounded-xl bg-[var(--el-fill-color-lighter)]"
            ></div>
          </div>
          <div class="grid gap-5 md:grid-cols-2">
            <div
              class="h-56 rounded-xl bg-[var(--el-fill-color-lighter)]"
            ></div>
            <div
              class="h-56 rounded-xl bg-[var(--el-fill-color-lighter)]"
            ></div>
          </div>
        </div>
      </template>

      <AnalysisOverview :items="overviewItems" />

      <div class="mt-5 grid gap-5 md:grid-cols-2">
        <AnalysisChartCard :title="$t('admin.dashboard.quickActions.title')">
          <div class="flex flex-wrap gap-3">
            <ElButton
              v-for="action in quickActions"
              :key="action.path"
              plain
              type="primary"
              @click="navTo(action.path)"
            >
              {{ action.label }}
            </ElButton>
          </div>
        </AnalysisChartCard>

        <AnalysisChartCard :title="$t('admin.dashboard.summary.title')">
          <ElDescriptions :column="1" border>
            <ElDescriptionsItem
              v-for="item in summaryItems"
              :key="item.label"
              :label="item.label"
            >
              {{ item.value }}
            </ElDescriptionsItem>
          </ElDescriptions>
        </AnalysisChartCard>
      </div>
    </ElSkeleton>
  </div>
</template>
