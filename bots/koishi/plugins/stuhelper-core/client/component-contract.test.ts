import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const clientDir = dirname(fileURLToPath(import.meta.url))

function readClientFile(relativePath: string): string {
  return readFileSync(resolve(clientDir, relativePath), 'utf8')
}

test('Drawer forwards the closed event to parent consumers', () => {
  const source = readClientFile('./components/primitives/Drawer.vue')

  assert.match(source, /@closed="handleClosed"/)
  assert.match(source, /emit\('closed'\)/)
})

test('SubscriptionView resets draft state on Drawer closed without a stray timeout', () => {
  const source = readClientFile('./components/SubscriptionView.vue')

  assert.match(source, /@closed="handleFormClosed"/)
  assert.match(source, /function handleFormClosed\(\) \{\s*resetDraft\(\)\s*\}/)
  assert.doesNotMatch(source, /setTimeout\(resetDraft,\s*280\)/)
})

test('WarnsView refreshes server state after successful count updates', () => {
  const source = readClientFile('./components/WarnsView.vue')

  assert.match(source, /await warnsApi\.update\(key, next\)/)
  assert.match(source, /pushSuccess\(next <= 0 \? '警告已清除' : '警告次数已更新'\)/)
  assert.match(source, /await refresh\(\)/)
})

test('WarnsView reads table cells through typed helpers instead of template any casts', () => {
  const source = readClientFile('./components/WarnsView.vue')

  assert.doesNotMatch(source, /row\.cells\.\w+ as any/)
  assert.match(source, /interface WarnUserCell extends QueueTableCellObject/)
  assert.match(source, /interface WarnCountCell extends QueueTableCellObject/)
  assert.match(source, /function warnUserCell\(cell: QueueTableCell \| undefined\): WarnUserCell/)
  assert.match(source, /function warnCountCell\(cell: QueueTableCell \| undefined\): WarnCountCell/)
})

test('page API client uses typed Koishi send events without double casts', () => {
  const source = readClientFile('./page-api.ts')
  const typesSource = readClientFile('./types.ts')

  assert.doesNotMatch(source, /as unknown as/)
  assert.doesNotMatch(source, /query as unknown as/)
  assert.match(source, /return send\('stuhelperGroupCenter\/page\/entity-profile', query\)/)
  assert.match(typesSource, /'stuhelperGroupCenter\/page\/dashboard'\(\): Promise<DashboardPageData>/)
  assert.match(typesSource, /'stuhelperGroupGuard\/page\/admission-runtime'\(\): Promise<AdmissionRuntimePageData>/)
  assert.match(typesSource, /'stuhelperGroupGuard\/action\/admission-member'\(input: \{/)
  assert.match(typesSource, /'stuhelperGroupGuard\/action\/save-admission-runtime-settings'\(input: AdmissionRuntimeSettingsPatch\): Promise<string>/)
  assert.match(typesSource, /'stuhelperGroupCenter\/action\/save-guard-binding'\(input: \{/)
})

test('EntityOverlay ignores stale profile loads for retries and close transitions', () => {
  const source = readClientFile('./components/shell/EntityOverlay.vue')

  assert.match(source, /let loadRequestSeq = 0/)
  assert.match(source, /const requestSeq = \+\+loadRequestSeq\s*window\.setTimeout/)
  assert.match(source, /shell\.entityTarget\.value === null && requestSeq === loadRequestSeq/)
  assert.match(source, /state\.loading = false\s*state\.profile = null/)
  assert.match(source, /async function loadProfile\(target: EntityRef\) \{\s*const requestSeq = \+\+loadRequestSeq/)
  assert.match(source, /const profile = await consolePageApi\.entityProfile\(\{[\s\S]*?guildId: target\.guildId,[\s\S]*?\}\)\s*if \(requestSeq !== loadRequestSeq\) return/)
  assert.match(source, /catch \(cause\) \{\s*if \(requestSeq !== loadRequestSeq\) return/)
  assert.match(source, /finally \{\s*if \(requestSeq === loadRequestSeq && entityKey\(shell\.entityTarget\.value\) === key\) \{\s*state\.loading = false/)
})

test('legacy API client unwraps console responses through an explicit unknown boundary', () => {
  const source = readClientFile('./api.ts')
  const chatImageSource = readClientFile('./components/chat/image-proxy.ts')

  assert.doesNotMatch(source, /@ts-ignore/)
  assert.doesNotMatch(source, /keyof any/)
  assert.doesNotMatch(source, /params\?: any/)
  assert.doesNotMatch(source, /catch \(e: any\)/)
  assert.match(source, /type ConsoleSend = \(event: string, params\?: unknown\) => Promise<unknown>/)
  assert.match(source, /function unwrapApiResponse<T>\(result: unknown\): T/)
  assert.match(source, /list: \(fetchNames\?: boolean\) => call<WarnListItem\[\]>/)
  assert.match(source, /get: \(\) => call<PluginSettings>/)
  assert.doesNotMatch(chatImageSource, /result: any/)
  assert.match(chatImageSource, /result: ImageProxyResult/)
})

test('dashboard chart components use explicit chart item types', () => {
  const trendSource = readClientFile('./components/dashboard/TrendChartCard.vue')
  const distSource = readClientFile('./components/dashboard/DistChartCard.vue')
  const rankSource = readClientFile('./components/dashboard/RankChartCard.vue')

  for (const source of [trendSource, distSource, rankSource]) {
    assert.doesNotMatch(source, /data: any\[\]/)
  }
  assert.match(trendSource, /data: ChartTrendItem\[\]/)
  assert.match(distSource, /data: ChartDistributionItem\[\]/)
  assert.match(rankSource, /type RankChartItem = ChartGuildRankItem \| ChartUserRankItem/)
  assert.match(rankSource, /function rankItemId\(item: RankChartItem\): string/)
})

test('dashboard pulse exposes stable fallback errors for global badges', () => {
  const source = readClientFile('./composables/use-pulse.ts')

  assert.match(source, /pendingAdmissions: 0/)
  assert.match(source, /pendingReviews: 0/)
  assert.match(source, /state\.error = errorMessage\(cause, '脉冲数据加载失败'\)/)
  assert.match(source, /import \{ errorMessage \} from '\.\.\/utils\/error-message'/)
  assert.doesNotMatch(source, /function errorMessage\(cause: unknown, fallback: string\): string/)
  assert.doesNotMatch(source, /state\.error = cause instanceof Error \? cause\.message : '脉冲数据加载失败'/)
})

test('console views treat caught errors as unknown before showing messages', () => {
  const chatSource = readClientFile('./components/ChatView.vue')
  const configSource = readClientFile('./components/ConfigView.vue')
  const settingsSource = readClientFile('./components/SettingsView.vue')

  for (const source of [chatSource, configSource, settingsSource]) {
    assert.doesNotMatch(source, /catch \(e: any\)/)
    assert.match(source, /function errorMessage\(cause: unknown\): string|const errorMessage = \(cause: unknown\)|import \{ errorMessage \} from '\.\.\/utils\/error-message'|useActionFeedback\(\)|useActionError\(/)
  }
})

test('NoticeStack supports operation-specific titles without changing the default labels', () => {
  const source = readClientFile('./components/primitives/NoticeStack.vue')

  assert.match(source, /title\?: string/)
  assert.match(source, /item\.title \?\? \(item\.kind === 'success' \? '操作成功' : '操作失败'\)/)
})

test('LogsView keeps log search failures visible and retryable', () => {
  const source = readClientFile('./components/LogsView.vue')

  assert.match(source, /const searchError = ref\(''\)/)
  assert.match(source, /<div v-if="searchError" class="sh-logs__error" role="alert">/)
  assert.match(source, /<strong>加载日志失败<\/strong>/)
  assert.match(source, /@click="runSearch"/)
  assert.match(source, /v-if="logs\.length === 0 && !loading && !searchError"/)
  assert.match(source, /let searchRequestSeq = 0/)
  assert.match(source, /const requestSeq = \+\+searchRequestSeq/)
  assert.match(source, /if \(requestSeq !== searchRequestSeq\) return\s*logs\.value = result\.list/)
  assert.match(source, /searchError\.value = ''\s*try \{/)
  assert.match(source, /const message = errorMessage\(cause, '加载日志失败'\)/)
  assert.match(source, /catch \(cause\) \{\s*if \(requestSeq !== searchRequestSeq\) return/)
  assert.match(source, /searchError\.value = message/)
  assert.match(source, /pushError\('加载日志失败', message\)/)
  assert.match(source, /finally \{\s*if \(requestSeq === searchRequestSeq\) \{\s*loading\.value = false\s*\}\s*\}/)
  assert.match(source, /import \{ useActionFeedback \} from '\.\.\/composables\/use-action-feedback'/)
  assert.match(source, /notices,[\s\S]*pushError,[\s\S]*dismissNotice,[\s\S]*errorMessage,[\s\S]*\} = useActionFeedback\(\)/)
  assert.doesNotMatch(source, /notices\.value\.push\(\{ id: noticeId\(\), kind: 'error', title, message \}\)/)
  assert.doesNotMatch(source, /function errorMessage\(cause: unknown, fallback: string\): string/)
})

test('SubscriptionView keeps list load failures visible and retryable', () => {
  const source = readClientFile('./components/SubscriptionView.vue')

  assert.match(source, /const loadError = ref\(''\)/)
  assert.match(source, /const initialLoadBlocked = computed\(\(\) => Boolean\(loadError\.value && subscriptions\.value\.length === 0\)\)/)
  assert.match(source, /v-else-if="loadError && subscriptions\.length === 0"/)
  assert.match(source, /tone="error"/)
  assert.match(source, /title="加载订阅失败"/)
  assert.match(source, /class="sh-sub-load-error"/)
  assert.match(source, /<strong>刷新订阅失败<\/strong>/)
  assert.match(source, /<el-button class="sh-button sh-button--ghost" @click="refresh">重试<\/el-button>/)
  assert.match(source, /:disabled="loading \|\| initialLoadBlocked"/)
  assert.match(source, /let refreshRequestSeq = 0/)
  assert.match(source, /const requestSeq = \+\+refreshRequestSeq/)
  assert.match(source, /loadError\.value = ''\s*try \{/)
  assert.match(source, /const next = await subscriptionApi\.list\(fetchNames\.value\)\s*if \(requestSeq !== refreshRequestSeq\) return\s*subscriptions\.value = next/)
  assert.match(source, /const details = errorMessage\(cause, '加载订阅失败'\)/)
  assert.match(source, /catch \(cause\) \{\s*if \(requestSeq !== refreshRequestSeq\) return/)
  assert.match(source, /loadError\.value = details/)
  assert.match(source, /pushError\('加载订阅失败', details\)/)
  assert.match(source, /finally \{\s*if \(requestSeq === refreshRequestSeq\) \{\s*loading\.value = false\s*\}\s*\}/)
  assert.match(source, /pushError\('订阅尚未加载', '订阅尚未加载，无法添加订阅'\)/)
  assert.match(source, /import \{ useActionFeedback \} from '\.\.\/composables\/use-action-feedback'/)
  assert.match(source, /pushError,[\s\S]*errorMessage,[\s\S]*\} = useActionFeedback\(\)/)
})

test('SubscriptionView keeps mutation failures visible after notice timeout', () => {
  const source = readClientFile('./components/SubscriptionView.vue')

  assert.match(source, /import \{ useActionFeedback \} from '\.\.\/composables\/use-action-feedback'/)
  assert.match(source, /actionError,[\s\S]*actionErrorTitle,[\s\S]*notices,[\s\S]*setActionError,[\s\S]*clearActionError,[\s\S]*dismissNotice,[\s\S]*\} = useActionFeedback\(\)/)
  assert.match(source, /class="sh-sub-action-error"/)
  assert.match(source, /class="sh-sub-action-error sh-sub-action-error--drawer"/)
  assert.match(source, /<strong>\{\{ actionErrorTitle \}\}<\/strong>/)
  assert.match(source, /<span>\{\{ actionError \}\}<\/span>/)
  assert.match(source, /@click="clearActionError"/)
  assert.match(source, /setActionError\(isEdit\.value \? '更新失败' : '添加失败', cause, isEdit\.value \? '更新失败' : '添加失败'\)/)
  assert.match(source, /setActionError\('删除失败', cause, '删除失败'\)/)
  assert.match(source, /async function save\(\) \{\s*if \(!canSave\.value \|\| saving\.value\) return/)
  assert.ok(
    source.indexOf('if (!canSave.value || saving.value) return') < source.indexOf('saving.value = true'),
  )
  assert.doesNotMatch(source, /function setActionError\(title: string, cause: unknown, fallback: string\)/)
  assert.doesNotMatch(source, /function clearActionError\(\)/)
})

test('BlacklistView keeps list load failures visible and retryable', () => {
  const source = readClientFile('./components/BlacklistView.vue')

  assert.match(source, /const loadError = ref\(''\)/)
  assert.match(source, /const initialLoadBlocked = computed\(\(\) => Boolean\(loadError\.value && blacklist\.value\.length === 0\)\)/)
  assert.match(source, /<ConsolePageSkeleton v-if="loading && blacklist\.length === 0" \/>/)
  assert.match(source, /v-else-if="loadError && blacklist\.length === 0"/)
  assert.match(source, /title="加载黑名单失败"/)
  assert.match(source, /class="sh-blacklist-load-error"/)
  assert.match(source, /<strong>刷新黑名单失败<\/strong>/)
  assert.match(source, /<el-button class="sh-button sh-button--ghost" @click="refresh">重试<\/el-button>/)
  assert.match(source, /:disabled="loading \|\| initialLoadBlocked"/)
  assert.match(source, /let refreshRequestSeq = 0/)
  assert.match(source, /const requestSeq = \+\+refreshRequestSeq/)
  assert.match(source, /loadError\.value = ''\s*try \{/)
  assert.match(source, /const next = await blacklistApi\.list\(\)\s*if \(requestSeq !== refreshRequestSeq\) return\s*blacklist\.value = next\.list/)
  assert.match(source, /const details = errorMessage\(cause, '加载黑名单失败'\)/)
  assert.match(source, /catch \(cause\) \{\s*if \(requestSeq !== refreshRequestSeq\) return/)
  assert.match(source, /loadError\.value = details/)
  assert.match(source, /pushError\('加载黑名单失败', details\)/)
  assert.match(source, /finally \{\s*if \(requestSeq === refreshRequestSeq\) \{\s*loading\.value = false\s*\}\s*\}/)
  assert.match(source, /pushError\('黑名单尚未加载', '黑名单尚未加载，无法添加用户'\)/)
  assert.match(source, /import \{ useActionFeedback \} from '\.\.\/composables\/use-action-feedback'/)
  assert.match(source, /pushError,[\s\S]*errorMessage,[\s\S]*\} = useActionFeedback\(\)/)
  assert.doesNotMatch(source, /function pushError\(title: string, message: string\)/)
  assert.doesNotMatch(source, /function errorMessage\(cause: unknown, fallback: string\): string/)
})

test('BlacklistView keeps add and release failures visible after notice timeout', () => {
  const source = readClientFile('./components/BlacklistView.vue')

  assert.match(source, /import \{ useActionFeedback \} from '\.\.\/composables\/use-action-feedback'/)
  assert.match(source, /actionError,[\s\S]*actionErrorTitle,[\s\S]*notices,[\s\S]*pushSuccess,[\s\S]*pushError,[\s\S]*setActionError,[\s\S]*clearActionError,[\s\S]*dismissNotice,[\s\S]*errorMessage,[\s\S]*\} = useActionFeedback\(\)/)
  assert.match(source, /class="sh-blacklist-action-error"/)
  assert.match(source, /class="sh-blacklist-action-error sh-blacklist-action-error--drawer"/)
  assert.match(source, /<strong>\{\{ actionErrorTitle \}\}<\/strong>/)
  assert.match(source, /<span>\{\{ actionError \}\}<\/span>/)
  assert.match(source, /@click="clearActionError"/)
  assert.match(source, /setActionError\('添加失败', cause, '添加失败'\)/)
  assert.match(source, /setActionError\('解除失败', cause, '解除失败'\)/)
  assert.match(source, /const removingIds = ref\(new Set<string>\(\)\)/)
  assert.match(source, /actions: row\.actions\.map\(\(action\) => \(\{ \.\.\.action, disabled: true \}\)\)/)
  assert.match(source, /async function submitAdd\(\) \{\s*if \(!canSubmitAdd\.value \|\| adding\.value\) return/)
  assert.ok(
    source.indexOf('if (!canSubmitAdd.value || adding.value) return') < source.indexOf('adding.value = true'),
  )
  assert.ok(
    source.indexOf('adding.value = true') < source.indexOf('await confirmGlobalAdd(userId)'),
  )
  assert.match(source, /if \(!entry \|\| removingIds\.value\.has\(userId\)\) return/)
  assert.ok(
    source.indexOf('removingIds.value = new Set([...removingIds.value, userId])') < source.indexOf('const confirmed = await confirm({'),
  )
  assert.match(source, /finally \{\s*const nextRemoving = new Set\(removingIds\.value\)\s*nextRemoving\.delete\(userId\)\s*removingIds\.value = nextRemoving\s*\}/)
  assert.doesNotMatch(source, /notices\.value\.push\(\{ id: noticeId\(\), kind: 'error', title, message \}\)/)
  assert.doesNotMatch(source, /function clearActionError\(\)/)
})

test('WarnsView keeps list load failures visible and retryable', () => {
  const source = readClientFile('./components/WarnsView.vue')

  assert.match(source, /const loadError = ref\(''\)/)
  assert.match(source, /const sourceRecordCount = computed\(\(\) =>/)
  assert.match(source, /const initialLoadBlocked = computed\(\(\) => Boolean\(loadError\.value && sourceRecordCount\.value === 0\)\)/)
  assert.match(source, /<ConsolePageSkeleton v-if="loading && sourceRecordCount === 0" \/>/)
  assert.match(source, /v-else-if="loadError && sourceRecordCount === 0"/)
  assert.match(source, /title="加载警告记录失败"/)
  assert.match(source, /class="sh-warns-load-error"/)
  assert.match(source, /<strong>刷新警告记录失败<\/strong>/)
  assert.match(source, /<el-button class="sh-button sh-button--ghost" @click="refresh">重试<\/el-button>/)
  assert.match(source, /:disabled="loading \|\| initialLoadBlocked"/)
  assert.match(source, /let refreshRequestSeq = 0/)
  assert.match(source, /const requestSeq = \+\+refreshRequestSeq/)
  assert.match(source, /loadError\.value = ''\s*try \{/)
  assert.match(source, /if \(requestSeq !== refreshRequestSeq\) return\s*groups\.value = next/)
  assert.match(source, /const details = errorMessage\(cause, '加载警告记录失败'\)/)
  assert.match(source, /catch \(cause\) \{\s*if \(requestSeq !== refreshRequestSeq\) return/)
  assert.match(source, /loadError\.value = details/)
  assert.match(source, /pushError\('加载警告记录失败', details\)/)
  assert.match(source, /finally \{\s*if \(requestSeq === refreshRequestSeq\) \{\s*loading\.value = false\s*\}\s*\}/)
  assert.match(source, /pushError\('警告记录尚未加载', '警告记录尚未加载，无法添加警告'\)/)
  assert.match(source, /import \{ useActionFeedback \} from '\.\.\/composables\/use-action-feedback'/)
  assert.match(source, /pushError,[\s\S]*errorMessage,[\s\S]*\} = useActionFeedback\(\)/)
  assert.doesNotMatch(source, /function pushError\(title: string, message: string\)/)
  assert.doesNotMatch(source, /function errorMessage\(cause: unknown, fallback: string\): string/)
})

test('WarnsView keeps reload and mutation failures visible after notice timeout', () => {
  const source = readClientFile('./components/WarnsView.vue')

  assert.match(source, /import \{ useActionFeedback \} from '\.\.\/composables\/use-action-feedback'/)
  assert.match(source, /actionError,[\s\S]*actionErrorTitle,[\s\S]*notices,[\s\S]*pushSuccess,[\s\S]*pushError,[\s\S]*setActionError,[\s\S]*clearActionError,[\s\S]*dismissNotice,[\s\S]*errorMessage,[\s\S]*\} = useActionFeedback\(\)/)
  assert.match(source, /class="sh-warns-action-error"/)
  assert.match(source, /class="sh-warns-action-error sh-warns-action-error--drawer"/)
  assert.match(source, /<strong>\{\{ actionErrorTitle \}\}<\/strong>/)
  assert.match(source, /<span>\{\{ actionError \}\}<\/span>/)
  assert.match(source, /@click="clearActionError"/)
  assert.match(source, /setActionError\('重新加载失败', cause, '重新加载失败'\)/)
  assert.match(source, /setActionError\('添加失败', cause, '添加失败'\)/)
  assert.match(source, /setActionError\('更新警告失败', cause, '更新警告失败'\)/)
  assert.doesNotMatch(source, /notices\.value\.push\(\{ id: noticeId\(\), kind: 'error', title, message \}\)/)
  assert.doesNotMatch(source, /function clearActionError\(\)/)
})

test('top-level console pages keep load failures retryable without hiding stale data', () => {
  const primitives = readClientFile('./styles/primitives.css')
  assert.match(primitives, /\.sh-load-error \{/)
  assert.match(primitives, /\.sh-load-error__body span \{/)

  const pages = [
    {
      file: './components/AdmissionView.vue',
      empty: /v-if="error && !data"/,
      emptyTitle: /title="加载入群认证数据失败"/,
      loadFallback: /error\.value = errorMessage\(cause, '加载入群认证数据失败'\)/,
      stale: /<template v-else-if="data && model">\s*<div v-if="error" class="sh-load-error" role="alert">/,
      title: /<strong>刷新入群认证数据失败<\/strong>/,
      retry: /@click="loadData"/,
      successGuard: /const next = await consolePageApi\.admissionRuntime\(\)\s*if \(requestSeq !== loadRequestSeq\) return\s*data\.value = next/,
    },
    {
      file: './components/ReviewView.vue',
      empty: /v-if="error && !data"/,
      emptyTitle: /title="加载处置中心数据失败"/,
      loadFallback: /error\.value = errorMessage\(cause, '加载处置中心数据失败'\)/,
      stale: /<template v-else-if="data">\s*<div v-if="error" class="sh-load-error" role="alert">/,
      title: /<strong>刷新处置中心数据失败<\/strong>/,
      retry: /@click="loadData"/,
      successGuard: /const next = await consolePageApi\.review\(\)\s*if \(requestSeq !== loadRequestSeq\) return\s*data\.value = next/,
    },
    {
      file: './components/IdentityView.vue',
      empty: /v-if="error && !data"/,
      emptyTitle: /title="加载身份认证数据失败"/,
      loadFallback: /error\.value = errorMessage\(cause, '加载身份认证数据失败'\)/,
      stale: /<template v-else-if="data">\s*<div v-if="error" class="sh-load-error" role="alert">/,
      title: /<strong>刷新身份认证数据失败<\/strong>/,
      retry: /@click="loadData"/,
      successGuard: /const next = await consolePageApi\.identity\(\)\s*if \(requestSeq !== loadRequestSeq\) return\s*data\.value = next/,
    },
    {
      file: './components/DashboardHubView.vue',
      empty: /v-else-if="error && !dashboardData"/,
      emptyTitle: /title="加载控制台总览失败"/,
      loadFallback: /error\.value = errorMessage\(cause, '加载控制台总览失败'\)/,
      stale: /<template v-else-if="dashboardData">\s*<div v-if="error" class="sh-load-error" role="alert">/,
      title: /<strong>刷新控制台总览失败<\/strong>/,
      retry: /@click="loadData"/,
      successGuard: /if \(requestSeq !== loadRequestSeq\) return\s*dashboardData\.value = pageData/,
    },
    {
      file: './components/SystemView.vue',
      empty: /v-else-if="error && !stats"/,
      emptyTitle: /title="加载缓存统计失败"/,
      loadFallback: /error\.value = errorMessage\(cause, '加载缓存统计失败'\)/,
      stale: /<template v-else-if="stats">\s*<div v-if="error" class="sh-load-error" role="alert">/,
      title: /<strong>刷新缓存统计失败<\/strong>/,
      retry: /@click="load"/,
      successGuard: /const next = await cacheApi\.stats\(\)\s*if \(requestSeq !== loadRequestSeq\) return\s*stats\.value = next/,
    },
    {
      file: './components/ConfigCenterView.vue',
      helperFile: './composables/use-config-governance.ts',
      empty: /v-if="error && !data"/,
      emptyTitle: /title="加载配置治理数据失败"/,
      loadFallback: /error\.value = errorMessage\(cause, '加载配置治理数据失败'\)/,
      stale: /<template v-else-if="data">\s*<div v-if="error" class="sh-load-error" role="alert">/,
      title: /<strong>刷新配置治理数据失败<\/strong>/,
      retry: /@click="loadData"/,
      successGuard: /const next = await consolePageApi\.configGovernance\(\)\s*if \(requestSeq !== loadRequestSeq\) return\s*data\.value = next/,
    },
  ] as const

  for (const page of pages) {
    const source = readClientFile(page.file)
    assert.match(source, page.empty)
    if ('emptyTitle' in page) {
      assert.match(source, page.emptyTitle)
    }
    if ('loadFallback' in page) {
      const logicSource = 'helperFile' in page ? readClientFile(page.helperFile) : source
      assert.match(logicSource, page.loadFallback)
      if (logicSource.includes('/utils/error-message')) {
        assert.match(logicSource, /import \{ errorMessage \} from '\.\.\/utils\/error-message'|import \{ errorMessage \} from '\.\.\/\.\.\/utils\/error-message'/)
        assert.doesNotMatch(logicSource, /function errorMessage\(cause: unknown, fallback: string\): string/)
      } else if (logicSource.includes('function errorMessage(')) {
        assert.match(logicSource, /function errorMessage\(cause: unknown, fallback: string\): string/)
      } else {
        assert.match(logicSource, /errorMessage,[\s\S]*\} = useAction(?:Feedback|Error)\(/)
      }
      assert.match(logicSource, /let loadRequestSeq = 0/)
      assert.match(logicSource, /const requestSeq = \+\+loadRequestSeq/)
      assert.match(logicSource, page.successGuard)
      assert.match(logicSource, /catch \(cause\) \{\s*if \(requestSeq !== loadRequestSeq\) return/)
      assert.match(logicSource, /finally \{\s*if \(requestSeq === loadRequestSeq\) \{\s*loading\.value = false/)
    }
    assert.match(source, /<template #action>/)
    assert.match(source, page.retry)
    assert.match(source, page.stale)
    assert.match(source, page.title)
    assert.match(source, /<span>\{\{ error \}\}<\/span>/)
  }
})

test('SystemView keeps cache operation failures visible', () => {
  const source = readClientFile('./components/SystemView.vue')

  assert.match(source, /const clearing = ref\(false\)/)
  assert.match(source, /import \{ useActionFeedback \} from '\.\.\/composables\/use-action-feedback'/)
  assert.match(source, /actionError,[\s\S]*actionErrorTitle,[\s\S]*notices,[\s\S]*setActionError,[\s\S]*clearActionError,[\s\S]*dismissNotice,[\s\S]*errorMessage,[\s\S]*\} = useActionFeedback\(\)/)
  assert.match(source, /<div v-if="actionError" class="sh-load-error sh-system__action-error" role="alert">/)
  assert.match(source, /<strong>\{\{ actionErrorTitle \}\}<\/strong>/)
  assert.match(source, /<span>\{\{ actionError \}\}<\/span>/)
  assert.match(source, /@click="clearActionError"/)
  assert.match(source, /:disabled="refreshing \|\| clearing"/)
  assert.match(source, /\{\{ clearing \? '清空中…' : '清空缓存' \}\}/)
  assert.match(source, /clearActionError\(\)\s*try \{\s*const result = await cacheApi\.refresh\(\)/)
  assert.match(source, /setActionError\('刷新缓存失败', cause, '刷新缓存失败'\)/)
  assert.match(source, /clearing\.value = true\s*clearActionError\(\)\s*try \{\s*await cacheApi\.clear\(\)/)
  assert.match(source, /setActionError\('清空缓存失败', cause, '清空缓存失败'\)/)
  assert.match(source, /clearing\.value = false/)
  assert.doesNotMatch(source, /function setActionError\(title: string, cause: unknown, fallback: string\)/)
  assert.doesNotMatch(source, /function clearActionError\(\)/)
  assert.doesNotMatch(source, /pushError\(cause, '刷新缓存失败'\)/)
  assert.doesNotMatch(source, /pushError\(cause, '清空缓存失败'\)/)
})

test('ConfigView keeps config list loading failures visible and retryable', () => {
  const source = readClientFile('./components/ConfigView.vue')

  assert.match(source, /const loadError = ref\(''\)/)
  assert.match(source, /const hasFilteredConfigs = computed\(\(\) => Object\.keys\(filteredConfigs\.value\)\.length > 0\)/)
  assert.match(source, /<div v-if="!loading && loadError" class="config-load-error" role="alert">/)
  assert.match(source, /<strong>加载群组配置失败<\/strong>/)
  assert.match(source, /<span>\{\{ loadError \}\}<\/span>/)
  assert.match(source, /@click="refreshConfigs"/)
  assert.match(source, /<div v-if="!loading && \(!loadError \|\| hasFilteredConfigs\)" class="config-list">/)
  assert.match(source, /<div v-if="!hasFilteredConfigs" class="empty-state">/)
  assert.match(source, /let refreshRequestSeq = 0/)
  assert.match(source, /const refreshConfigs = async \(\): Promise<boolean> => \{/)
  assert.match(source, /const requestSeq = \+\+refreshRequestSeq/)
  assert.match(source, /loadError\.value = ''\s*clearActionError\(\)\s*try \{/)
  assert.match(source, /const next = await configApi\.list\(fetchNames\.value\)\s*if \(requestSeq !== refreshRequestSeq\) return false\s*configs\.value = next\s*return true/)
  assert.match(source, /catch \(cause\) \{\s*if \(requestSeq !== refreshRequestSeq\) return false/)
  assert.match(source, /const details = errorMessage\(cause, '加载配置失败'\)/)
  assert.match(source, /loadError\.value = details/)
  assert.match(source, /message\.error\(details\)/)
  assert.match(source, /return false\s*\} finally \{\s*if \(requestSeq === refreshRequestSeq\) \{\s*loading\.value = false\s*\}/)
  assert.match(source, /void refreshConfigs\(\)\.then\(\(loaded\) => \{\s*if \(loaded\) applyNavigationState\(\)\s*\}\)/)
  assert.match(source, /import \{ useActionError \} from '\.\.\/composables\/use-action-error'/)
  assert.doesNotMatch(source, /function errorMessage\(cause: unknown\): string/)
})

test('ConfigView keeps config operation failures visible', () => {
  const source = readClientFile('./components/ConfigView.vue')

  assert.match(source, /import \{ useActionError \} from '\.\.\/composables\/use-action-error'/)
  assert.match(source, /actionError,[\s\S]*actionErrorTitle,[\s\S]*setActionError,[\s\S]*clearActionError,[\s\S]*errorMessage,[\s\S]*\} = useActionError\(\{/)
  assert.match(source, /onError: \(_title, details\) => message\.error\(details\)/)
  assert.match(source, /<button class="btn btn-primary" @click="openCreateDialog">/)
  assert.match(source, /<div v-if="!loading && actionError" class="config-action-error" role="alert">/)
  assert.match(source, /<strong>\{\{ actionErrorTitle \}\}<\/strong>/)
  assert.match(source, /<span>\{\{ actionError \}\}<\/span>/)
  assert.match(source, /@click="clearActionError"/)
  assert.match(source, /<div v-if="actionError" class="config-action-error config-action-error--dialog" role="alert">/)
  assert.match(source, /const openCreateDialog = \(\) => \{\s*clearActionError\(\)\s*showCreateDialog\.value = true\s*\}/)
  assert.match(source, /setActionError\('重新加载失败', cause, '重新加载失败'\)/)
  assert.match(source, /setActionError\('保存失败', cause, '保存失败'\)/)
  assert.match(source, /setActionError\('创建失败', cause, '创建失败'\)/)
  assert.match(source, /setActionError\('删除失败', cause, '删除失败'\)/)
  assert.doesNotMatch(source, /const actionError = ref\(''\)/)
  assert.doesNotMatch(source, /const actionErrorTitle = ref\('操作失败'\)/)
  assert.doesNotMatch(source, /function setActionError\(title: string, cause: unknown, fallback: string\): void/)
  assert.doesNotMatch(source, /function clearActionError\(\): void/)
  assert.doesNotMatch(source, /message\.error\(errorMessage\(cause\) \|\| '保存失败'\)/)
  assert.doesNotMatch(source, /message\.error\(errorMessage\(cause\) \|\| '创建失败'\)/)
  assert.doesNotMatch(source, /message\.error\(errorMessage\(cause\) \|\| '删除失败'\)/)
})

test('SettingsView keeps global settings state and merge helpers typed', () => {
  const source = readClientFile('./components/SettingsView.vue')

  assert.doesNotMatch(source, /ref<any>/)
  assert.doesNotMatch(source, /deepMerge = \(target: any, source: any\)/)
  assert.doesNotMatch(source, /JSON\.parse\(JSON\.stringify\(defaultSettings\)\)/)
  assert.match(source, /interface SettingsModel extends PlainRecord \{/)
  assert.match(source, /const settings = ref<SettingsModel>\(cloneDefaultSettings\(\)\)/)
  assert.match(source, /function deepMerge<T extends PlainRecord>\(target: T, source: unknown\): T/)
  assert.match(source, /function parseSettingsSnapshot\(value: string\): SettingsModel/)
})

test('SettingsView blocks default-form edits when global settings fail to load', () => {
  const source = readClientFile('./components/SettingsView.vue')

  assert.match(source, /const loadError = ref\(''\)/)
  assert.match(source, /const settingsLoaded = computed\(\(\) => Boolean\(originalSettings\.value\) && !loadError\.value\)/)
  assert.match(source, /<div v-else-if="loadError" class="settings-load-error" role="alert">/)
  assert.match(source, /<strong>加载全局设置失败<\/strong>/)
  assert.match(source, /<span>\{\{ loadError \}\}<\/span>/)
  assert.match(source, /@click="loadSettings"/)
  assert.match(source, /<div v-else-if="settingsLoaded" class="settings-content">/)
  assert.match(source, /<div class="save-bar" v-if="settingsLoaded && hasChanges">/)
  assert.match(source, /:disabled="!settingsLoaded \|\| saving"/)
  assert.match(source, /let loadRequestSeq = 0/)
  assert.match(source, /const loadSettings = async \(\): Promise<boolean> => \{/)
  assert.match(source, /const requestSeq = \+\+loadRequestSeq/)
  assert.match(source, /loadError\.value = ''\s*clearActionError\(\)\s*try \{/)
  assert.match(source, /const data = await settingsApi\.get\(\)\s*if \(requestSeq !== loadRequestSeq\) return false/)
  assert.match(source, /const details = errorMessage\(cause, '加载设置失败'\)/)
  assert.match(source, /catch \(cause\) \{\s*if \(requestSeq !== loadRequestSeq\) return false/)
  assert.match(source, /loadError\.value = details/)
  assert.match(source, /originalSettings\.value = ''/)
  assert.match(source, /return false\s*\} finally \{\s*if \(requestSeq === loadRequestSeq\) \{\s*loading\.value = false\s*\}/)
  assert.match(source, /if \(!settingsLoaded\.value\) \{\s*setActionError\('保存失败', '设置尚未加载，不能保存默认表单', '保存失败'\)/)
  assert.match(source, /import \{ useActionError \} from '\.\.\/composables\/use-action-error'/)
  assert.doesNotMatch(source, /function errorMessage\(cause: unknown\): string/)
})

test('SettingsView keeps save failures visible and prevents duplicate submissions', () => {
  const source = readClientFile('./components/SettingsView.vue')

  assert.match(source, /import \{ useActionError \} from '\.\.\/composables\/use-action-error'/)
  assert.match(source, /actionError,[\s\S]*actionErrorTitle,[\s\S]*setActionError,[\s\S]*clearActionError,[\s\S]*errorMessage,[\s\S]*\} = useActionError\(\{/)
  assert.match(source, /onError: \(_title, details\) => message\.error\(details\)/)
  assert.match(source, /<div v-if="actionError" class="settings-action-error" role="alert">/)
  assert.match(source, /<strong>\{\{ actionErrorTitle \}\}<\/strong>/)
  assert.match(source, /<span>\{\{ actionError \}\}<\/span>/)
  assert.match(source, /@click="clearActionError"/)
  assert.match(source, /:disabled="!settingsLoaded \|\| saving"/)
  assert.match(source, /<button class="save-bar-btn secondary" :disabled="saving" @click="resetChanges">放弃更改<\/button>/)
  assert.match(source, /<button class="save-bar-btn primary" :disabled="saving" @click="saveSettings">/)
  assert.match(source, /\{\{ saving \? '保存中\.\.\.' : '保存更改' \}\}/)
  assert.match(source, /const saveSettings = async \(\) => \{\s*if \(saving\.value\) return/)
  assert.ok(
    source.indexOf('if (saving.value) return') < source.indexOf('saving.value = true'),
  )
  assert.match(source, /saving\.value = true\s*clearActionError\(\)\s*try \{/)
  assert.match(source, /setActionError\('保存失败', cause, '保存设置失败'\)/)
  assert.doesNotMatch(source, /const actionError = ref\(''\)/)
  assert.doesNotMatch(source, /const actionErrorTitle = ref\('操作失败'\)/)
  assert.doesNotMatch(source, /function setActionError\(title: string, cause: unknown, fallback: string\): void/)
  assert.doesNotMatch(source, /function clearActionError\(\): void/)
  assert.doesNotMatch(source, /message\.error\(errorMessage\(cause\) \|\| '保存设置失败'\)/)
})

test('SettingsView uses shared confirmation before discarding local settings edits', () => {
  const source = readClientFile('./components/SettingsView.vue')

  assert.match(source, /import \{ useConfirm \} from '\.\.\/composables\/use-confirm'/)
  assert.match(source, /import ConfirmDialog from '\.\/primitives\/ConfirmDialog\.vue'/)
  assert.match(source, /<ConfirmDialog\b/)
  assert.match(source, /:open="confirmDialog\.open"/)
  assert.match(source, /:tone="confirmDialog\.tone"/)
  assert.match(source, /@confirm="acceptConfirm"/)
  assert.match(source, /@cancel="cancelConfirm"/)
  assert.match(source, /state: confirmDialog,[\s\S]*confirm,[\s\S]*accept: acceptConfirm,[\s\S]*cancel: cancelConfirm,[\s\S]*\} = useConfirm\(\)/)
  assert.match(source, /const confirmed = await confirm\(\{\s*title: '放弃更改'[\s\S]*tone: 'normal'/)
  assert.match(source, /const confirmed = await confirm\(\{\s*title: '恢复默认设置'[\s\S]*tone: 'danger'/)
  assert.doesNotMatch(source, /const confirmDialog = ref\(\{/)
  assert.doesNotMatch(source, /const showConfirm =/)
  assert.doesNotMatch(source, /const doConfirm =/)
  assert.doesNotMatch(source, /confirmDialog\.show/)
  assert.doesNotMatch(source, /modal-overlay|modal-dialog|fade-enter-active/)
})

test('ChatView delegates message rendering to a safe component instead of v-html', () => {
  const source = readClientFile('./components/ChatView.vue')

  assert.match(source, /<ChatMessageContent\b/)
  assert.doesNotMatch(source, /v-html="renderMessage\(msg\)"/)
  assert.doesNotMatch(source, /onclick="window\.open\('/)
})

test('ChatView uses a composite session key to avoid cross-platform collisions', () => {
  const source = readClientFile('./components/ChatView.vue')

  assert.match(source, /const buildSessionKey = \(params: \{/)
  assert.match(source, /return `\$\{params\.platform\}:\$\{params\.type\}:\$\{guildPart\}:\$\{params\.channelId\}`/)
  assert.match(source, /findSessionByKey\(sessionKey\)/)
})

test('ChatView keeps member and chat operation failures visible', () => {
  const source = readClientFile('./components/ChatView.vue')

  assert.match(source, /import \{ useActionFeedback \} from '\.\.\/composables\/use-action-feedback'/)
  assert.match(source, /import NoticeStack from '\.\/primitives\/NoticeStack\.vue'/)
  assert.match(source, /const \{\s*actionError,\s*actionErrorTitle,\s*notices,\s*pushError,\s*setActionError,\s*clearActionError,\s*dismissNotice,\s*errorMessage,\s*\} = useActionFeedback\(\)/)
  assert.match(source, /const membersLoadError = ref\(''\)/)
  assert.match(source, /let membersLoadRequestSeq = 0/)
  assert.match(source, /<div v-if="actionError" class="chat-action-error" role="alert">/)
  assert.match(source, /<div v-if="membersLoadError" class="members-error" role="alert">/)
  assert.match(source, /<strong>加载群成员失败<\/strong>/)
  assert.match(source, /@click="retryLoadMembers"/)
  assert.match(source, /<NoticeStack :items="notices" @dismiss="dismissNotice" \/>/)
  assert.match(source, /const loadGuildMembers = async \(guildId: string\) => \{\s*const requestSeq = \+\+membersLoadRequestSeq/)
  assert.match(source, /const result = await chatApi\.getGuildMembers\(guildId\)\s*if \(requestSeq !== membersLoadRequestSeq\) return\s*members\.value = result\.members \|\| \[\]/)
  assert.match(source, /catch \(e\) \{\s*if \(requestSeq !== membersLoadRequestSeq\) return/)
  assert.match(source, /membersLoadError\.value = details/)
  assert.match(source, /finally \{\s*if \(requestSeq === membersLoadRequestSeq\) \{\s*loadingMembers\.value = false/)
  assert.match(source, /membersLoadRequestSeq \+= 1\s*loadingMembers\.value = false\s*members\.value = \[\]/)
  assert.match(source, /pushError\('加载群成员失败', details\)/)
  assert.match(source, /setActionError\('撤回失败', cause, '撤回失败'\)/)
  assert.match(source, /setActionError\('获取会话信息失败', e, '获取会话信息失败'\)/)
  assert.match(source, /setActionError\('发送失败', cause, '发送失败'\)/)
  assert.match(source, /errorMessage\(e, '加载群成员失败'\)/)
  assert.match(source, /errorMessage\(e, '获取群资料失败'\)/)
  assert.match(source, /errorMessage\(e, '获取用户资料失败'\)/)
  assert.match(source, /const sendMessage = async \(\) => \{\s*if \(sending\.value\) return/)
  assert.ok(
    source.indexOf('if (sending.value) return') < source.indexOf('sending.value = true'),
  )
  assert.doesNotMatch(source, /const actionError = ref\(''\)/)
  assert.doesNotMatch(source, /const actionErrorTitle = ref\('操作失败'\)/)
  assert.doesNotMatch(source, /const notices = ref<NoticeItem\[\]>\(\[\]\)/)
  assert.doesNotMatch(source, /function setActionError\(/)
  assert.doesNotMatch(source, /function clearActionError\(/)
  assert.doesNotMatch(source, /function pushError\(/)
  assert.doesNotMatch(source, /function dismissNotice\(/)
  assert.doesNotMatch(source, /function scheduleDismiss\(/)
  assert.doesNotMatch(source, /function noticeId\(/)
  assert.doesNotMatch(source, /import \{ errorMessage \} from '\.\.\/utils\/error-message'/)
  assert.doesNotMatch(source, /const errorMessage = \(cause: unknown\)/)
  assert.doesNotMatch(source, /message\.error\('加载群成员失败: ' \+ errorMessage/)
})

test('dangerous moderation actions require shared confirmation before API mutation', () => {
  const reviewSource = readClientFile('./components/ReviewView.vue')
  const blacklistSource = readClientFile('./components/BlacklistView.vue')
  const warnsSource = readClientFile('./components/WarnsView.vue')

  assert.match(reviewSource, /<ConfirmDialog\b/)
  assert.ok(
    reviewSource.indexOf('const confirmed = await confirm(') < reviewSource.indexOf('await consolePageApi.workItemAction('),
  )
  assert.match(reviewSource, /if \(!confirmed\) return/)

  assert.match(blacklistSource, /<ConfirmDialog\b/)
  assert.ok(
    blacklistSource.indexOf('const confirmed = await confirm(') < blacklistSource.indexOf('await blacklistApi.remove({'),
  )
  assert.match(blacklistSource, /if \(!confirmed\) return/)

  assert.match(warnsSource, /<ConfirmDialog\b/)
  assert.ok(
    warnsSource.indexOf('const confirmed = await confirm(') < warnsSource.indexOf('await warnsApi.update(key, next)'),
  )
  assert.match(warnsSource, /if \(!confirmed\) \{\s*await refresh\(\)\s*return\s*\}/)
})

test('ReviewView keeps action failures inside the action panel instead of replacing page data', () => {
  const source = readClientFile('./components/ReviewView.vue')

  assert.match(source, /title="加载处置中心数据失败"/)
  assert.match(source, /import \{ useActionError \} from '\.\.\/composables\/use-action-error'/)
  assert.match(source, /actionError,[\s\S]*setActionError,[\s\S]*clearActionError,[\s\S]*\} = useActionError\(\{[\s\S]*onError: \(_title, details\) => message\.error\(details\)/)
  assert.match(source, /<p v-if="actionError" class="sh-review__action-error" role="alert">/)
  assert.match(source, /error\.value = errorMessage\(cause, '加载处置中心数据失败'\)/)
  assert.match(source, /data\.value = next\s*clearActionError\(\)/)
  assert.match(source, /selectedItemId\.value = itemId\s*clearActionError\(\)/)
  assert.match(source, /actionLoading\.value = true\s*clearActionError\(\)/)
  assert.match(source, /setActionError\('处置失败', cause, '处置失败'\)/)
  assert.match(source, /import \{ errorMessage \} from '\.\.\/utils\/error-message'/)
  assert.doesNotMatch(source, /function errorMessage\(cause: unknown, fallback: string\): string/)
  assert.ok(
    source.indexOf("setActionError('处置失败', cause, '处置失败')") >
      source.indexOf('await consolePageApi.workItemAction('),
  )
  assert.ok(
    source.indexOf("error.value = errorMessage(cause, '加载处置中心数据失败')") <
      source.indexOf('async function submitAction'),
  )
  assert.doesNotMatch(source, /const actionError = ref\(''\)/)
  assert.doesNotMatch(source, /actionError\.value =/)
  assert.doesNotMatch(source, /message\.error\(actionError\.value\)/)
})

test('AdmissionView keeps runtime action failures inside their operation sections', () => {
  const source = readClientFile('./components/AdmissionView.vue')

  assert.match(source, /title="加载入群认证数据失败"/)
  assert.match(source, /import \{ useActionError \} from '\.\.\/composables\/use-action-error'/)
  assert.match(source, /actionError,[\s\S]*setActionError,[\s\S]*clearActionError,[\s\S]*\} = useActionError\(\{[\s\S]*onError: \(_title, details\) => message\.error\(details\)/)
  assert.match(source, /actionError: settingsError,[\s\S]*setActionError: setSettingsError,[\s\S]*clearActionError: clearSettingsError,[\s\S]*\} = useActionError\(\{[\s\S]*defaultTitle: '保存运行开关失败'/)
  assert.match(source, /<p v-if="actionError" class="sh-admission__error" role="alert">/)
  assert.match(source, /<p v-if="settingsError" class="sh-admission__error" role="alert">/)
  assert.match(source, /error\.value = errorMessage\(cause, '加载入群认证数据失败'\)/)
  assert.match(source, /data\.value = next\s*clearActionError\(\)\s*clearSettingsError\(\)/)
  assert.match(source, /clearActionError\(\)\s*notice\.value = ''/)
  assert.match(source, /clearSettingsError\(\)\s*settingsNotice\.value = ''/)
  assert.match(source, /setActionError\('入群认证操作失败', cause, '入群认证操作失败'\)/)
  assert.match(source, /setSettingsError\('保存入群认证运行开关失败', cause, '保存入群认证运行开关失败'\)/)
  assert.match(source, /import \{ errorMessage \} from '\.\.\/utils\/error-message'/)
  assert.doesNotMatch(source, /function errorMessage\(cause: unknown, fallback: string\): string/)
  assert.ok(
    source.indexOf("setActionError('入群认证操作失败', cause, '入群认证操作失败')") >
      source.indexOf('await consolePageApi.admissionAction({'),
  )
  assert.ok(
    source.indexOf("setSettingsError('保存入群认证运行开关失败', cause, '保存入群认证运行开关失败')") >
      source.indexOf('await consolePageApi.saveAdmissionRuntimeSettings(patch)'),
  )

  const memberActionSource = source.slice(
    source.indexOf('async function submitMemberAction'),
    source.indexOf('async function submitRuntimeSetting'),
  )
  const runtimeSettingSource = source.slice(
    source.indexOf('async function submitRuntimeSetting'),
    source.indexOf('function actionLabel'),
  )
  assert.doesNotMatch(memberActionSource, /error\.value = cause instanceof Error/)
  assert.doesNotMatch(runtimeSettingSource, /error\.value = cause instanceof Error/)
  assert.doesNotMatch(source, /const actionError = ref\(''\)/)
  assert.doesNotMatch(source, /const settingsError = ref\(''\)/)
  assert.doesNotMatch(source, /actionError\.value =/)
  assert.doesNotMatch(source, /settingsError\.value =/)
  assert.doesNotMatch(source, /message\.error\(actionError\.value\)/)
  assert.doesNotMatch(source, /message\.error\(settingsError\.value\)/)
  assert.doesNotMatch(memberActionSource, /message\.error\(actionError\.value \|\|/)
  assert.doesNotMatch(runtimeSettingSource, /message\.error\(settingsError\.value \|\|/)
})

test('ConfigCenterView uses shared confirmation before discarding dirty governance forms', () => {
  const viewSource = readClientFile('./components/ConfigCenterView.vue')
  const composableSource = readClientFile('./composables/use-config-governance.ts')

  assert.match(viewSource, /<ConfirmDialog\b/)
  assert.match(viewSource, /title: '放弃未保存更改'/)
  assert.match(viewSource, /confirmText: '放弃更改'/)
  assert.doesNotMatch(composableSource, /window\.confirm/)
  assert.match(composableSource, /confirmDiscardChanges: ConfirmDiscardChanges/)
  assert.ok(
    composableSource.indexOf('await requestDiscardConfirmation()') < composableSource.indexOf('discardCurrentChanges()'),
  )
})

test('ConfigCenterView keeps governance save failures separate from load failures', () => {
  const viewSource = readClientFile('./components/ConfigCenterView.vue')
  const composableSource = readClientFile('./composables/use-config-governance.ts')

  assert.match(composableSource, /const error = ref\(''\)/)
  assert.match(composableSource, /import \{ useActionError \} from '\.\/use-action-error'/)
  assert.match(composableSource, /actionError,[\s\S]*actionErrorTitle,[\s\S]*setActionError,[\s\S]*clearActionError,[\s\S]*\} = useActionError\(\)/)
  assert.match(viewSource, /<div v-if="actionError" class="sh-load-error sh-config-governance__action-error" role="alert">/)
  assert.match(viewSource, /<strong>\{\{ actionErrorTitle \}\}<\/strong>/)
  assert.match(viewSource, /<span>\{\{ actionError \}\}<\/span>/)
  assert.match(viewSource, /@click="clearActionError"/)
  assert.match(composableSource, /validateTemplateForm/)
  assert.match(composableSource, /validateBindingForm/)
  assert.match(composableSource, /validatePolicyForm/)
  assert.match(composableSource, /if \(!validateBeforeSubmit\('保存模板失败', validateTemplateForm\(templateForm\)\)\) return/)
  assert.match(composableSource, /if \(!validateBeforeSubmit\('保存绑定失败', validateBindingForm\(bindingForm\)\)\) return/)
  assert.match(composableSource, /if \(!validateBeforeSubmit\('保存策略失败', validatePolicyForm\(policyForm\)\)\) return/)
  assert.match(composableSource, /await submit\('保存模板失败', async \(\) => \{/)
  assert.match(composableSource, /await submit\('保存绑定失败', async \(\) => \{/)
  assert.match(composableSource, /await submit\('保存策略失败', async \(\) => \{/)
  assert.match(composableSource, /setActionError\(title, cause, title\)/)
  assert.match(composableSource, /function validateBeforeSubmit\(title: string, validationError: string\) \{/)
  assert.doesNotMatch(composableSource, /const actionError = ref\(''\)/)
  assert.doesNotMatch(composableSource, /const actionErrorTitle = ref\('操作失败'\)/)
  assert.doesNotMatch(composableSource, /function clearActionError\(\) \{/)

  const submitSource = composableSource.slice(composableSource.indexOf('async function submit('))
  assert.doesNotMatch(submitSource, /error\.value = cause instanceof Error/)
})

test('SettingsView sidebar icon binding uses explicit string icon names', () => {
  const source = readClientFile('./components/SettingsView.vue')

  assert.doesNotMatch(source, /<k-icon :name="section"/)
  assert.match(source, /<k-icon :name="section\.icon"/)
  assert.match(source, /icon: 'stuhelperGroupCenter:octicons\./)
})

test('RolesView has no debug console output in browser code', () => {
  const source = readClientFile('./components/RolesView.vue')

  assert.doesNotMatch(source, /console\.(log|error|warn|info)\(/)
})

test('RolesView keeps role permission load failures visible and retryable', () => {
  const source = readClientFile('./components/RolesView.vue')

  assert.match(source, /const loadError = ref\(''\)/)
  assert.match(source, /<div v-if="loadError" class="roles-load-banner" role="alert">/)
  assert.match(source, /<main v-else-if="loadError" class="main-content roles-load-error" role="alert">/)
  assert.match(source, /<strong>刷新角色权限数据失败<\/strong>/)
  assert.match(source, /<strong>加载角色权限数据失败<\/strong>/)
  assert.match(source, /<button class="secondary-btn" @click="fetchData">重试<\/button>/)
  assert.match(source, /:disabled="loading \|\| Boolean\(loadError\) \|\| Boolean\(roleOperation\)"/)
  assert.match(source, /message\.warning\('角色权限数据尚未加载，无法创建角色'\)/)
  assert.match(source, /let loadRequestSeq = 0/)
  assert.match(source, /const requestSeq = \+\+loadRequestSeq\s*loading\.value = true/)
  assert.match(source, /const \[nextRoles, nextPermissions\] = await Promise\.all\(\[/)
  assert.match(source, /if \(requestSeq !== loadRequestSeq\) return\s*roles\.value = nextRoles/)
  assert.match(source, /roles\.value = nextRoles/)
  assert.match(source, /permissions\.value = nextPermissions/)
  assert.match(source, /catch \(e\) \{\s*if \(requestSeq !== loadRequestSeq\) return/)
  assert.match(source, /finally \{\s*if \(requestSeq === loadRequestSeq\) \{\s*loading\.value = false/)
  assert.match(source, /import \{ errorMessage \} from '\.\.\/utils\/error-message'/)
  assert.match(source, /function setLoadError\(title: string, cause: unknown, fallback: string\): void/)
  assert.match(source, /const details = errorMessage\(cause, fallback\)\s*loadError\.value = details\s*message\.error\(`\$\{title\}: \$\{details\}`\)/)
  assert.match(source, /currentRole\.value \? '刷新角色权限数据失败' : '加载角色权限数据失败'/)
  assert.doesNotMatch(source, /loadError\.value = '加载数据失败: ' \+ roleErrorMessage\(e\)/)
  assert.doesNotMatch(source, /message\.error\(loadError\.value\)/)
  assert.doesNotMatch(source, /function roleErrorMessage\(cause: unknown\): string/)
})

test('RolesView ignores stale role member loads when switching roles', () => {
  const source = readClientFile('./components/RolesView.vue')

  assert.match(source, /const memberLoading = ref\(false\)/)
  assert.match(source, /let membersLoadRequestSeq = 0/)
  assert.match(source, /const fetchRoleMembers = async \(roleId: string\) => \{\s*const requestSeq = \+\+membersLoadRequestSeq\s*memberLoading\.value = true/)
  assert.match(source, /const nextMembers = await authApi\.getRoleMembers\(roleId, true\)\s*if \(requestSeq !== membersLoadRequestSeq \|\| currentRole\.value\?\.id !== roleId\) return\s*currentRoleMembers\.value = nextMembers/)
  assert.match(source, /catch \(e\) \{\s*if \(requestSeq !== membersLoadRequestSeq \|\| currentRole\.value\?\.id !== roleId\) return\s*currentRoleMembers\.value = \[\]/)
  assert.match(source, /finally \{\s*if \(requestSeq === membersLoadRequestSeq && currentRole\.value\?\.id === roleId\) \{\s*memberLoading\.value = false/)
  assert.match(source, /membersLoadRequestSeq \+= 1\s*importPreviewRequestSeq \+= 1\s*memberLoading\.value = false\s*importLoading\.value = false\s*currentRole\.value = role\s*currentRoleMembers\.value = \[\]\s*importPreviewMembers\.value = \[\]\s*selectedImportIds\.value = new Set\(\)/)
  assert.match(source, /<div v-if="memberLoading" class="member-loading">加载角色成员中\.\.\.<\/div>/)
  assert.match(source, /:disabled="memberLoading \|\| addingMember"/)
  assert.match(source, /message\.warning\('角色成员正在加载，请稍后再导入'\)/)
  assert.match(source, /const roleId = currentRole\.value\.id[\s\S]*?await authApi\.importMembers\(roleId, userIds\)[\s\S]*?if \(currentRole\.value\?\.id === roleId\) \{\s*await fetchRoleMembers\(roleId\)/)
})

test('RolesView ignores stale member import preview loads', () => {
  const source = readClientFile('./components/RolesView.vue')

  assert.match(source, /let importPreviewRequestSeq = 0/)
  assert.match(source, /watch\(importSource, \(newSource\) => \{[\s\S]*?importPreviewRequestSeq \+= 1\s*importLoading\.value = false/)
  assert.match(source, /const closeImportDialog = \(\) => \{\s*importPreviewRequestSeq \+= 1\s*showImportDialog\.value = false/)
  assert.match(source, /const sourceRoleId = importSourceRoleId\.value\s*const targetRoleId = currentRole\.value\?\.id \?\? ''\s*const requestSeq = \+\+importPreviewRequestSeq/)
  assert.match(source, /const members = await authApi\.getRoleMembers\(sourceRoleId, true\)\s*if \(\s*requestSeq !== importPreviewRequestSeq \|\|\s*!showImportDialog\.value \|\|\s*importSource\.value !== 'role' \|\|\s*importSourceRoleId\.value !== sourceRoleId \|\|\s*currentRole\.value\?\.id !== targetRoleId\s*\) return/)
  assert.match(source, /catch \(e\) \{\s*if \(\s*requestSeq !== importPreviewRequestSeq \|\|\s*!showImportDialog\.value \|\|\s*importSource\.value !== 'role' \|\|\s*importSourceRoleId\.value !== sourceRoleId \|\|\s*currentRole\.value\?\.id !== targetRoleId\s*\) return/)
  assert.match(source, /const authorityLevel = Number\(importAuthorityLevel\.value\)\s*const targetRoleId = currentRole\.value\?\.id \?\? ''\s*const requestSeq = \+\+importPreviewRequestSeq/)
  assert.match(source, /const members = await authApi\.getUsersByAuthority\(authorityLevel\)\s*if \(\s*requestSeq !== importPreviewRequestSeq \|\|\s*!showImportDialog\.value \|\|\s*importSource\.value !== 'authority' \|\|\s*Number\(importAuthorityLevel\.value\) !== authorityLevel \|\|\s*currentRole\.value\?\.id !== targetRoleId\s*\) return/)
  assert.match(source, /const guildId = importGuildId\.value\.trim\(\)\s*const targetRoleId = currentRole\.value\?\.id \?\? ''\s*const requestSeq = \+\+importPreviewRequestSeq/)
  assert.match(source, /const members = await authApi\.getGuildAdmins\(guildId\)\s*if \(\s*requestSeq !== importPreviewRequestSeq \|\|\s*!showImportDialog\.value \|\|\s*importSource\.value !== 'guild-admin' \|\|\s*importGuildId\.value\.trim\(\) !== guildId \|\|\s*currentRole\.value\?\.id !== targetRoleId\s*\) return/)
  assert.match(source, /finally \{\s*if \(requestSeq === importPreviewRequestSeq\) \{\s*importLoading\.value = false/)
})

test('RolesView keeps role operation failures visible', () => {
  const source = readClientFile('./components/RolesView.vue')

  assert.match(source, /import \{ useActionError \} from '\.\.\/composables\/use-action-error'/)
  assert.match(source, /actionError,[\s\S]*actionErrorTitle,[\s\S]*setActionError,[\s\S]*clearActionError,[\s\S]*\} = useActionError\(\{[\s\S]*onError: \(title, details\) => message\.error\(`\$\{title\}: \$\{details\}`\)/)
  assert.match(source, /actionError: importError,[\s\S]*actionErrorTitle: importErrorTitle,[\s\S]*setActionError: setImportError,[\s\S]*clearActionError: clearImportError,[\s\S]*\} = useActionError\(\{[\s\S]*defaultTitle: '导入失败'/)
  assert.match(source, /<div v-if="actionError" class="roles-action-banner" role="alert">/)
  assert.match(source, /<div v-if="importError" class="roles-action-banner roles-action-banner--modal" role="alert">/)
  assert.match(source, /<strong>\{\{ actionErrorTitle \}\}<\/strong>/)
  assert.match(source, /<span>\{\{ actionError \}\}<\/span>/)
  assert.match(source, /<strong>\{\{ importErrorTitle \}\}<\/strong>/)
  assert.match(source, /<span>\{\{ importError \}\}<\/span>/)
  assert.match(source, /@click\.stop="openImportDialog"/)
  assert.match(source, /setActionError\('创建角色失败', e, '创建角色失败'\)/)
  assert.match(source, /setActionError\('保存失败', e, '保存失败'\)/)
  assert.match(source, /setActionError\('克隆失败', e, '克隆失败'\)/)
  assert.match(source, /setActionError\('删除失败', e, '删除失败'\)/)
  assert.match(source, /setActionError\('添加成员失败', e, '添加成员失败'\)/)
  assert.match(source, /setActionError\('移除成员失败', e, '移除成员失败'\)/)
  assert.match(source, /setActionError\('角色排序失败', error, '角色排序失败'\)/)
  assert.match(source, /setImportError\('导入失败', e, '导入失败'\)/)
  assert.doesNotMatch(source, /const actionError = ref\(''\)/)
  assert.doesNotMatch(source, /const actionErrorTitle = ref\('操作失败'\)/)
  assert.doesNotMatch(source, /const importError = ref\(''\)/)
  assert.doesNotMatch(source, /const importErrorTitle = ref\('导入失败'\)/)
  assert.doesNotMatch(source, /function setActionError\(title: string, cause: unknown, fallback: string\): void/)
  assert.doesNotMatch(source, /function setImportError\(title: string, cause: unknown, fallback: string\): void/)
  assert.doesNotMatch(source, /function clearActionError\(\): void/)
  assert.doesNotMatch(source, /function clearImportError\(\): void/)
  assert.doesNotMatch(source, /message\.error\('保存失败: ' \+ \(e instanceof Error \? e\.message : String\(e\)\)\)/)
  assert.doesNotMatch(source, /message\.error\('导入失败: ' \+ \(e instanceof Error \? e\.message : String\(e\)\)\)/)
})

test('RolesView uses shared confirmation for role mutations', () => {
  const source = readClientFile('./components/RolesView.vue')

  assert.match(source, /import \{ useConfirm \} from '\.\.\/composables\/use-confirm'/)
  assert.match(source, /import ConfirmDialog from '\.\/primitives\/ConfirmDialog\.vue'/)
  assert.match(source, /<ConfirmDialog\b/)
  assert.match(source, /:open="confirmDialog\.open"/)
  assert.match(source, /:tone="confirmDialog\.tone"/)
  assert.match(source, /@confirm="acceptConfirm"/)
  assert.match(source, /@cancel="cancelConfirm"/)
  assert.match(source, /state: confirmDialog,[\s\S]*confirm,[\s\S]*accept: acceptConfirm,[\s\S]*cancel: cancelConfirm,[\s\S]*\} = useConfirm\(\)/)
  assert.match(source, /const confirmed = await confirm\(\{\s*title: '未保存的修改'[\s\S]*tone: 'danger'/)
  assert.match(source, /const confirmed = await confirm\(\{\s*title: '重置更改'[\s\S]*tone: 'normal'/)
  assert.match(source, /const confirmed = await confirm\(\{\s*title: '克隆角色'[\s\S]*tone: 'normal'/)
  assert.match(source, /const confirmed = await confirm\(\{\s*title: '删除角色'[\s\S]*tone: 'danger'/)
  assert.doesNotMatch(source, /const confirmDialog = ref\(\{/)
  assert.doesNotMatch(source, /const showConfirm =/)
  assert.doesNotMatch(source, /const doConfirm =/)
  assert.doesNotMatch(source, /confirmDialog\.show/)
})

test('RolesView prevents duplicate role lifecycle mutations', () => {
  const source = readClientFile('./components/RolesView.vue')

  assert.match(source, /type RoleOperation = '' \| 'create' \| 'clone' \| 'delete'/)
  assert.match(source, /const roleOperation = ref<RoleOperation>\(''\)/)
  assert.match(source, /:disabled="loading \|\| Boolean\(loadError\) \|\| Boolean\(roleOperation\)"/)
  assert.match(source, /:title="roleOperation === 'create' \? '创建中' : '创建角色'"/)
  assert.match(source, /\{\{ roleOperation === 'create' \? '…' : '＋' \}\}/)
  assert.match(source, /<button class="clone-btn" :disabled="Boolean\(roleOperation\)" @click="cloneRole">/)
  assert.match(source, /\{\{ roleOperation === 'clone' \? '克隆中…' : '克隆角色' \}\}/)
  assert.match(source, /<button class="danger-btn" :disabled="Boolean\(roleOperation\)" @click="deleteRole">/)
  assert.match(source, /\{\{ roleOperation === 'delete' \? '删除中…' : '删除角色' \}\}/)
  assert.match(source, /const createRole = async \(\) => \{\s*if \(roleOperation\.value\) return/)
  assert.match(source, /roleOperation\.value = 'create'\s*const newRole: Role =/)
  assert.match(source, /const sourceRole = currentRole\.value\s*roleOperation\.value = 'clone'\s*try \{/)
  assert.match(source, /message: `确定要克隆角色"\$\{sourceRole\.name\}"吗？`/)
  assert.match(source, /permissions: Array\.isArray\(sourceRole\.permissions\) \? \[\.\.\.sourceRole\.permissions\] : \[\]/)
  assert.match(source, /const deleteRole = async \(\) => \{\s*if \(!currentRole\.value \|\| roleOperation\.value\) return\s*const roleToDelete = currentRole\.value\s*roleOperation\.value = 'delete'/)
  assert.match(source, /message: `确定要删除角色"\$\{roleToDelete\.name\}"吗？此操作不可撤销。`/)
  assert.match(source, /await authApi\.deleteRole\(roleToDelete\.id\)/)
  assert.match(source, /if \(currentRole\.value\?\.id === roleToDelete\.id\) \{/)
  assert.match(source, /finally \{\s*roleOperation\.value = ''\s*\}/)
  assert.match(source, /\.clone-btn:disabled,\s*\.danger-btn:disabled \{/)
})

test('RolesView prevents duplicate role save submissions', () => {
  const source = readClientFile('./components/RolesView.vue')

  assert.match(source, /const savingChanges = ref\(false\)/)
  assert.match(source, /<button class="reset-btn" :disabled="savingChanges" @click="resetChanges">重置<\/button>/)
  assert.match(source, /<button class="save-btn" :disabled="savingChanges" @click="saveChanges">/)
  assert.match(source, /\{\{ savingChanges \? '保存中…' : '保存更改' \}\}/)
  assert.match(source, /if \(!currentRole\.value \|\| savingChanges\.value\) return/)
  assert.match(source, /clearActionError\(\)\s*savingChanges\.value = true\s*try \{/)
  assert.match(source, /finally \{\s*savingChanges\.value = false\s*\}/)
  assert.match(source, /\.save-btn:disabled,\s*\.save-btn:disabled:hover \{/)
  assert.match(source, /\.reset-btn:disabled,\s*\.reset-btn:disabled:hover \{/)
})

test('RolesView prevents duplicate role member mutations', () => {
  const source = readClientFile('./components/RolesView.vue')

  assert.match(source, /const addingMember = ref\(false\)/)
  assert.match(source, /const removingMemberIds = ref\(new Set<string>\(\)\)/)
  assert.match(source, /:disabled="addingMember \|\| memberLoading"\s*@keyup\.enter="addMember"/)
  assert.match(source, /<button class="primary-btn" :disabled="memberLoading \|\| addingMember \|\| !newMemberId\.trim\(\)" @click\.stop="handleAddMember">/)
  assert.match(source, /\{\{ addingMember \? '添加中…' : '添加成员' \}\}/)
  assert.match(source, /<button class="secondary-btn" :disabled="memberLoading \|\| addingMember" @click\.stop="openImportDialog">导入成员<\/button>/)
  assert.match(source, /:disabled="removingMemberIds\.has\(member\.id\)"/)
  assert.match(source, /\{\{ removingMemberIds\.has\(member\.id\) \? '移除中…' : '移除' \}\}/)
  assert.match(source, /if \(addingMember\.value\) return/)
  assert.match(source, /if \(memberLoading\.value\) \{\s*message\.warning\('角色成员正在加载，请稍后再操作'\)/)
  assert.match(source, /addingMember\.value = true\s*try \{/)
  assert.match(source, /finally \{\s*addingMember\.value = false\s*\}/)
  assert.match(source, /if \(!currentRole\.value \|\| removingMemberIds\.value\.has\(userId\)\) return/)
  assert.match(source, /removingMemberIds\.value = new Set\(\[\.\.\.removingMemberIds\.value, userId\]\)/)
  assert.match(source, /const nextRemoving = new Set\(removingMemberIds\.value\)\s*nextRemoving\.delete\(userId\)\s*removingMemberIds\.value = nextRemoving/)
  assert.match(source, /\.danger-btn:disabled \{/)
})

test('RolesView reports role id copy failures after clipboard fallback', () => {
  const source = readClientFile('./components/RolesView.vue')

  assert.match(source, /import \{ copyTextToClipboard \} from '\.\.\/utils\/clipboard'/)
  assert.match(source, /if \(await copyTextToClipboard\(roleId\)\) \{/)
  assert.match(source, /message\.error\('复制角色 ID 失败，请手动复制'\)/)
  assert.doesNotMatch(source, /copyTextWithFallback/)
  assert.doesNotMatch(source, /navigator\.clipboard\.writeText/)
  assert.doesNotMatch(source, /document\.execCommand\('copy'\)/)
})

test('RolesView keeps backend error details for member import preview failures', () => {
  const source = readClientFile('./components/RolesView.vue')

  assert.match(source, /setActionError: setImportError/)
  assert.match(source, /setImportError\('加载成员列表失败', e, '加载成员列表失败'\)/)
  assert.match(source, /setImportError\('加载用户列表失败', e, '加载用户列表失败'\)/)
  assert.match(source, /setImportError\('获取群管理员失败', e, '获取群管理员失败'\)/)
  assert.doesNotMatch(source, /message\.error\('加载成员列表失败'\)/)
  assert.doesNotMatch(source, /message\.error\('加载用户列表失败'\)/)
  assert.doesNotMatch(source, /message\.error\('获取群管理员失败'\)/)
})

test('ChatView uses console API avatars instead of hard-coded QQ avatar URLs', () => {
  const source = readClientFile('./components/ChatView.vue')

  assert.doesNotMatch(source, /p\.qlogo\.cn/)
  assert.doesNotMatch(source, /q1\.qlogo\.cn/)
  assert.match(source, /displayAvatar = info\.avatar/)
})

test('ChatView reports message copy failures after clipboard fallback', () => {
  const source = readClientFile('./components/ChatView.vue')

  assert.match(source, /import \{ copyTextToClipboard \} from '\.\.\/utils\/clipboard'/)
  assert.match(source, /if \(await copyTextToClipboard\(plainText\)\) \{/)
  assert.match(source, /message\.error\('复制失败，请手动复制'\)/)
  assert.doesNotMatch(source, /copyTextWithFallback/)
  assert.doesNotMatch(source, /navigator\.clipboard\.writeText/)
  assert.doesNotMatch(source, /document\.execCommand\('copy'\)/)
  assert.doesNotMatch(source, /textarea\.parentNode\?\.removeChild\(textarea\)/)
})

test('ConfigView reports guild id copy failures after clipboard fallback', () => {
  const source = readClientFile('./components/ConfigView.vue')

  assert.match(source, /const copyGuildId = async/)
  assert.match(source, /import \{ copyTextToClipboard \} from '\.\.\/utils\/clipboard'/)
  assert.match(source, /if \(await copyTextToClipboard\(id\)\) \{/)
  assert.match(source, /message\.error\('复制失败，请手动复制'\)/)
  assert.doesNotMatch(source, /copyTextWithFallback/)
  assert.doesNotMatch(source, /navigator\.clipboard\.writeText/)
  assert.doesNotMatch(source, /document\.execCommand\('copy'\)/)
  assert.doesNotMatch(source, /textarea\.parentNode\?\.removeChild\(textarea\)/)
})

test('clipboard copy fallback is shared by console views', () => {
  const source = readClientFile('./utils/clipboard.ts')

  assert.match(source, /export async function copyTextToClipboard/)
  assert.match(source, /await clipboard\.writeText\(text\)/)
  assert.match(source, /return copyTextWithTextarea\(text, environment\)/)
  assert.match(source, /export function copyTextWithTextarea/)
  assert.match(source, /document\.execCommand\('copy'\)/)
  assert.match(source, /target\?\.remove\(\)/)
})

test('shell portals keep StuHelper design tokens when mounted under body', () => {
  const searchSource = readClientFile('./components/shell/SearchPanel.vue')
  const overlaySource = readClientFile('./components/shell/EntityOverlay.vue')
  const dockSource = readClientFile('./components/shell/ChatDock.vue')
  const confirmSource = readClientFile('./components/primitives/ConfirmDialog.vue')
  const drawerSource = readClientFile('./components/primitives/Drawer.vue')

  assert.match(searchSource, /class="stuhelperGroupCenter-portal sh-search"/)
  assert.match(overlaySource, /class="stuhelperGroupCenter-portal"/)
  assert.match(overlaySource, /<strong>加载实体资料失败<\/strong>/)
  assert.match(overlaySource, /state\.error = errorMessage\(cause, '加载实体资料失败'\)/)
  assert.match(overlaySource, /import \{ errorMessage \} from '\.\.\/\.\.\/utils\/error-message'/)
  assert.doesNotMatch(overlaySource, /function errorMessage\(cause: unknown, fallback: string\): string/)
  assert.match(overlaySource, /<button type="button" class="sh-overlay__retry" @click="reload">重试<\/button>/)
  assert.match(dockSource, /class="stuhelperGroupCenter-portal sh-dock"/)
  assert.match(confirmSource, /modal-class="stuhelperGroupCenter-portal"/)
  assert.match(drawerSource, /modal-class="stuhelperGroupCenter-portal"/)
})

test('EntityChip owns click propagation so row-level handlers do not also fire', () => {
  const source = readClientFile('./components/primitives/EntityChip.vue')

  assert.match(source, /:is="tag"/)
  assert.match(source, /@click\.stop="handleClick"/)
  assert.match(source, /@keydown\.enter\.stop\.prevent="handleClick"/)
  assert.match(source, /const tag = computed\(\(\) => props\.inline \? 'span' : 'button'\)/)
  assert.match(source, /event\.stopPropagation\(\)/)
})

test('QueueTable keeps wide data sets horizontally scrollable', () => {
  const source = readClientFile('./components/primitives/QueueTable.vue')

  assert.match(source, /:style="tableStyle"/)
  assert.match(source, /const DEFAULT_COLUMN_WIDTH = 160/)
  assert.match(source, /const ACTIONS_COLUMN_WIDTH = '120'/)
  assert.match(source, /minWidth: `\$\{calculateTableMinWidth\(\)\}px`/)
  assert.match(source, /:width="ACTIONS_COLUMN_WIDTH"/)
})

test('SearchPanel escape does not bubble into the AppShell escape chain', () => {
  const searchSource = readClientFile('./components/shell/SearchPanel.vue')
  const shellSource = readClientFile('./components/shell/AppShell.vue')

  assert.match(searchSource, /@keydown\.escape\.stop\.prevent="close"/)
  assert.match(shellSource, /if \(event\.defaultPrevented\) return/)
})

test('mobile shell exposes a CommandBar rail toggle outside the collapsed rail', () => {
  const commandSource = readClientFile('./components/shell/CommandBar.vue')
  const styleSource = readClientFile('./styles/shell.css')

  assert.match(commandSource, /class="sh-cmd__menu"/)
  assert.match(commandSource, /@click="shell\.toggleRail\(\)"/)
  assert.match(styleSource, /\.sh-shell__rail \{\s*position: fixed;/)
  assert.match(styleSource, /\.sh-shell\[data-rail-expanded='true'\] \.sh-shell__rail/)
})
