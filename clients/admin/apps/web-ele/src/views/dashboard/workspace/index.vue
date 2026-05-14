<script lang="ts" setup>
import type { AdminStats } from '#/api/admin';

import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';

import { IconifyIcon } from '@vben/icons';

import { ElButton, ElSkeleton } from 'element-plus';

import { getAdminStats } from '#/api/admin';
import { $t } from '#/locales';

import AdminContentLayout from '../../shared/AdminContentLayout.vue';

import './workspace.css';

const router = useRouter();
const loading = ref(true);
const stats = ref<AdminStats | null>(null);

const queueItems = computed(() => {
  const current = stats.value;
  return [
    {
      icon: 'lucide:flag',
      label: '待处理举报',
      path: '/content/reports',
      tone: 'amber',
      value: current?.pendingReports ?? 0,
    },
    {
      icon: 'lucide:eye-off',
      label: '隐藏评课',
      path: '/content/reviews',
      tone: 'slate',
      value: current?.hiddenReviews ?? 0,
    },
    {
      icon: 'lucide:calendar-plus',
      label: '本周新增评课',
      path: '/content/reviews',
      tone: 'green',
      value: current?.weekReviews ?? 0,
    },
  ];
});

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

const shortcuts = [
  {
    icon: 'lucide:message-square-text',
    label: '评课管理',
    path: '/content/reviews',
  },
  {
    icon: 'lucide:flag',
    label: '举报管理',
    path: '/content/reports',
  },
  {
    icon: 'lucide:id-card',
    label: '实名审核',
    path: '/users/identity-review',
  },
  {
    icon: 'lucide:shield-check',
    label: '成员黑名单',
    path: '/users/member-blacklist',
  },
];

async function fetchStats() {
  loading.value = true;
  try {
    stats.value = await getAdminStats();
  } finally {
    loading.value = false;
  }
}

async function goTo(path: string) {
  await router.push(path);
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
        <div class="admin-workspace__panel">
          <div class="admin-workspace__panel-header">
            <h2>处理队列</h2>
          </div>
          <button
            v-for="item in queueItems"
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

        <div class="admin-workspace__panel">
          <div class="admin-workspace__panel-header">
            <h2>常用入口</h2>
          </div>
          <div class="admin-workspace__shortcuts">
            <button
              v-for="item in shortcuts"
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
