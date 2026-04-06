<script lang="ts" setup>
import type {
  WorkbenchProjectItem,
  WorkbenchQuickNavItem,
  WorkbenchTodoItem,
  WorkbenchTrendItem,
} from '@vben/common-ui';

import { ref } from 'vue';
import { useRouter } from 'vue-router';

import {
  AnalysisChartCard,
  WorkbenchHeader,
  WorkbenchProject,
  WorkbenchQuickNav,
  WorkbenchTodo,
  WorkbenchTrends,
} from '@vben/common-ui';
import { preferences } from '@vben/preferences';
import { useUserStore } from '@vben/stores';
import { openWindow } from '@vben/utils';

import { $t } from '#/locales';
import AnalyticsVisitsSource from '../analytics/analytics-visits-source.vue';

const userStore = useUserStore();

// 这是一个示例数据，实际项目中需要根据实际情况进行调整
// url 也可以是内部路由，在 navTo 方法中识别处理，进行内部跳转
// 例如：url: /dashboard/workspace
const projectItems: WorkbenchProjectItem[] = [
  {
    color: '',
    content: $t('admin.dashboard.workspace.projects.github.content'),
    date: '2021-04-01',
    group: $t('admin.dashboard.workspace.projects.github.group'),
    icon: 'carbon:logo-github',
    title: 'Github',
    url: 'https://github.com',
  },
  {
    color: '#3fb27f',
    content: $t('admin.dashboard.workspace.projects.vue.content'),
    date: '2021-04-01',
    group: $t('admin.dashboard.workspace.projects.vue.group'),
    icon: 'ion:logo-vue',
    title: 'Vue',
    url: 'https://vuejs.org',
  },
  {
    color: '#e18525',
    content: $t('admin.dashboard.workspace.projects.html.content'),
    date: '2021-04-01',
    group: $t('admin.dashboard.workspace.projects.html.group'),
    icon: 'ion:logo-html5',
    title: 'Html5',
    url: 'https://developer.mozilla.org/zh-CN/docs/Web/HTML',
  },
  {
    color: '#bf0c2c',
    content: $t('admin.dashboard.workspace.projects.angular.content'),
    date: '2021-04-01',
    group: 'UI',
    icon: 'ion:logo-angular',
    title: 'Angular',
    url: 'https://angular.io',
  },
  {
    color: '#00d8ff',
    content: $t('admin.dashboard.workspace.projects.react.content'),
    date: '2021-04-01',
    group: $t('admin.dashboard.workspace.projects.react.group'),
    icon: 'bx:bxl-react',
    title: 'React',
    url: 'https://reactjs.org',
  },
  {
    color: '#EBD94E',
    content: $t('admin.dashboard.workspace.projects.javascript.content'),
    date: '2021-04-01',
    group: $t('admin.dashboard.workspace.projects.javascript.group'),
    icon: 'ion:logo-javascript',
    title: 'Js',
    url: 'https://developer.mozilla.org/zh-CN/docs/Web/JavaScript',
  },
];

// 同样，这里的 url 也可以使用以 http 开头的外部链接
const quickNavItems: WorkbenchQuickNavItem[] = [
  {
    color: '#1fdaca',
    icon: 'ion:home-outline',
    title: $t('admin.dashboard.workspace.quickNav.home'),
    url: '/',
  },
  {
    color: '#bf0c2c',
    icon: 'ion:grid-outline',
    title: $t('admin.dashboard.workspace.quickNav.dashboard'),
    url: '/dashboard',
  },
  {
    color: '#e18525',
    icon: 'ion:layers-outline',
    title: $t('admin.dashboard.workspace.quickNav.components'),
    url: '/demos/features/icons',
  },
  {
    color: '#3fb27f',
    icon: 'ion:settings-outline',
    title: $t('admin.dashboard.workspace.quickNav.system'),
    url: '/demos/features/login-expired', // 这里的 URL 是示例，实际项目中需要根据实际情况进行调整
  },
  {
    color: '#4daf1bc9',
    icon: 'ion:key-outline',
    title: $t('admin.dashboard.workspace.quickNav.access'),
    url: '/demos/access/page-control',
  },
  {
    color: '#00d8ff',
    icon: 'ion:bar-chart-outline',
    title: $t('admin.dashboard.workspace.quickNav.charts'),
    url: '/analytics',
  },
];

const todoItems = ref<WorkbenchTodoItem[]>([
  {
    completed: false,
    content: $t('admin.dashboard.workspace.todos.reviewFrontend.content'),
    date: '2024-07-30 11:00:00',
    title: $t('admin.dashboard.workspace.todos.reviewFrontend.title'),
  },
  {
    completed: true,
    content: $t('admin.dashboard.workspace.todos.optimizePerformance.content'),
    date: '2024-07-30 11:00:00',
    title: $t('admin.dashboard.workspace.todos.optimizePerformance.title'),
  },
  {
    completed: false,
    content: $t('admin.dashboard.workspace.todos.securityCheck.content'),
    date: '2024-07-30 11:00:00',
    title: $t('admin.dashboard.workspace.todos.securityCheck.title'),
  },
  {
    completed: false,
    content: $t('admin.dashboard.workspace.todos.updateDependencies.content'),
    date: '2024-07-30 11:00:00',
    title: $t('admin.dashboard.workspace.todos.updateDependencies.title'),
  },
  {
    completed: false,
    content: $t('admin.dashboard.workspace.todos.fixUi.content'),
    date: '2024-07-30 11:00:00',
    title: $t('admin.dashboard.workspace.todos.fixUi.title'),
  },
]);
const trendItems: WorkbenchTrendItem[] = [
  {
    avatar: 'svg:avatar-1',
    content: $t('admin.dashboard.workspace.trends.createProject.content'),
    date: $t('admin.dashboard.workspace.trends.time.justNow'),
    title: $t('admin.dashboard.workspace.trends.people.william'),
  },
  {
    avatar: 'svg:avatar-2',
    content: $t('admin.dashboard.workspace.trends.follow.content'),
    date: $t('admin.dashboard.workspace.trends.time.oneHourAgo'),
    title: $t('admin.dashboard.workspace.trends.people.ivan'),
  },
  {
    avatar: 'svg:avatar-3',
    content: $t('admin.dashboard.workspace.trends.publishUpdate.content'),
    date: $t('admin.dashboard.workspace.trends.time.oneDayAgo'),
    title: $t('admin.dashboard.workspace.trends.people.chris'),
  },
  {
    avatar: 'svg:avatar-4',
    content: $t('admin.dashboard.workspace.trends.publishArticle.content'),
    date: $t('admin.dashboard.workspace.trends.time.twoDaysAgo'),
    title: 'Vben',
  },
  {
    avatar: 'svg:avatar-1',
    content: $t('admin.dashboard.workspace.trends.replyQuestion.content'),
    date: $t('admin.dashboard.workspace.trends.time.threeDaysAgo'),
    title: $t('admin.dashboard.workspace.trends.people.peter'),
  },
  {
    avatar: 'svg:avatar-2',
    content: $t('admin.dashboard.workspace.trends.closeIssue.content'),
    date: $t('admin.dashboard.workspace.trends.time.oneWeekAgo'),
    title: $t('admin.dashboard.workspace.trends.people.jack'),
  },
  {
    avatar: 'svg:avatar-3',
    content: $t('admin.dashboard.workspace.trends.publishUpdate.content'),
    date: $t('admin.dashboard.workspace.trends.time.oneWeekAgo'),
    title: $t('admin.dashboard.workspace.trends.people.william'),
  },
  {
    avatar: 'svg:avatar-4',
    content: $t('admin.dashboard.workspace.trends.pushGithub.content'),
    date: '2021-04-01 20:00',
    title: $t('admin.dashboard.workspace.trends.people.william'),
  },
  {
    avatar: 'svg:avatar-4',
    content: $t('admin.dashboard.workspace.trends.adminArticle.content'),
    date: '2021-03-01 20:00',
    title: 'Vben',
  },
];

const router = useRouter();

// 这是一个示例方法，实际项目中需要根据实际情况进行调整
// This is a sample method, adjust according to the actual project requirements
function navTo(nav: WorkbenchProjectItem | WorkbenchQuickNavItem) {
  if (nav.url?.startsWith('http')) {
    openWindow(nav.url);
    return;
  }
  if (nav.url?.startsWith('/')) {
    router.push(nav.url).catch((error) => {
      console.error('Navigation failed:', error);
    });
  } else {
    console.warn(`Unknown URL for navigation item: ${nav.title} -> ${nav.url}`);
  }
}
</script>

<template>
  <div class="p-5">
    <WorkbenchHeader
      :avatar="userStore.userInfo?.avatar || preferences.app.defaultAvatar"
    >
      <template #title>
        {{ $t('admin.dashboard.workspace.header.title', { name: userStore.userInfo?.realName ?? '' }) }}
      </template>
      <template #description>{{ $t('admin.dashboard.workspace.header.description') }}</template>
    </WorkbenchHeader>

    <div class="mt-5 flex flex-col lg:flex-row">
      <div class="mr-4 w-full lg:w-3/5">
        <WorkbenchProject :items="projectItems" :title="$t('admin.dashboard.workspace.cards.projects')" @click="navTo" />
        <WorkbenchTrends :items="trendItems" class="mt-5" :title="$t('admin.dashboard.workspace.cards.trends')" />
      </div>
      <div class="w-full lg:w-2/5">
        <WorkbenchQuickNav
          :items="quickNavItems"
          class="mt-5 lg:mt-0"
          :title="$t('admin.dashboard.workspace.cards.quickNav')"
          @click="navTo"
        />
        <WorkbenchTodo :items="todoItems" class="mt-5" :title="$t('admin.dashboard.workspace.cards.todo')" />
        <AnalysisChartCard class="mt-5" :title="$t('admin.dashboard.workspace.cards.visitsSource')">
          <AnalyticsVisitsSource />
        </AnalysisChartCard>
      </div>
    </div>
  </div>
</template>
