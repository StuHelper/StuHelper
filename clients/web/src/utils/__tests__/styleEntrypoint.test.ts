import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('style entrypoint', () => {
  it('loads main.css from the application entrypoint', () => {
    const mainSource = readFileSync(resolve(__dirname, '../../main.ts'), 'utf-8')
    expect(mainSource).toContain("./styles/main.css")
  })

  it('does not load tailwind.css directly from App.vue anymore', () => {
    const appSource = readFileSync(resolve(__dirname, '../../App.vue'), 'utf-8')
    expect(appSource).not.toContain("@/styles/tailwind.css")
  })

  it('hydrates sessions after mount only when a local session hint exists', () => {
    const mainSource = readFileSync(resolve(__dirname, '../../main.ts'), 'utf-8')

    expect(mainSource).toContain('const hasSessionHint = hasStoredSessionHint()')
    expect(mainSource).toMatch(
      /if \(!hasSessionHint && authStore\.isAuthenticated\) \{\s*authStore\.clearSession\(\)\s*\}/,
    )
    expect(mainSource).toMatch(
      /app\.mount\('#app'\)\s*if \(hasSessionHint\) \{\s*void authStore\.bootstrapSession\(\)\s*\}/,
    )
  })

  it('does not probe guest routes unless local auth state exists', () => {
    const routerSource = readFileSync(resolve(__dirname, '../../router/index.ts'), 'utf-8')

    expect(routerSource).toContain('(Boolean(to.meta.guest) && authStore.isAuthenticated)')
  })

  it('uses the home route as the default SSO return target from the login page', () => {
    const loginSource = readFileSync(
      resolve(__dirname, '../../modules/auth/views/LoginPage.vue'),
      'utf-8',
    )

    expect(loginSource).toContain('function defaultAuthenticatedRoute()')
    expect(loginSource).toContain('return new URL("/", window.location.origin).toString()')
    expect(loginSource).toContain('return defaultAuthenticatedRoute()')
  })

  it('keeps the login page inside the shared shell without global background side effects', () => {
    const loginSource = readFileSync(
      resolve(__dirname, '../../modules/auth/views/LoginPage.vue'),
      'utf-8',
    )
    const routerSource = readFileSync(resolve(__dirname, '../../router/index.ts'), 'utf-8')

    expect(routerSource).toMatch(/name:\s*"login"[\s\S]*meta:\s*\{\s*titleKey:\s*"routes\.login",\s*guest:\s*true\s*\}/)
    expect(loginSource).not.toContain('ParticleBackground')
    expect(loginSource).not.toContain(':global([data-theme="dark"])')
    expect(loginSource).not.toMatch(/\[data-theme="dark"\]\s*\{[\s\S]*opacity/)
  })

  it('keeps anonymous favorite actions clickable without eager session bootstrap', () => {
    const favoriteSource = readFileSync(
      resolve(__dirname, '../../components/business/review/FavoriteButton.vue'),
      'utf-8',
    )

    expect(favoriteSource).toContain('if (loading.value || authStore.bootstrapPending) return true')
    expect(favoriteSource).toContain('if (authStore.isAuthenticated && !authStore.bootstrapCompleted) return true')
  })

  it('keeps course and teacher lists as semantic links', () => {
    const courseListSource = readFileSync(
      resolve(__dirname, '../../modules/course/views/CourseListPage.vue'),
      'utf-8',
    )
    const teacherHubSource = readFileSync(
      resolve(__dirname, '../../modules/course/views/TeacherHubPage.vue'),
      'utf-8',
    )

    expect(courseListSource).toContain('<router-link')
    expect(courseListSource).toContain(':to="`/courses/${course.id}/reviews`"')
    expect(teacherHubSource).toContain('<router-link')
    expect(teacherHubSource).toContain(':to="`/teachers/${teacher.teacherID}`"')
  })

  it('routes header write-review actions to the page form instead of the deleted modal flow', () => {
    const shellSource = readFileSync(resolve(__dirname, '../../components/layout/AppShell.vue'), 'utf-8')
    const reviewPageSource = readFileSync(
      resolve(__dirname, '../../modules/review/views/ReviewPage.vue'),
      'utf-8',
    )
    const headerSource = readFileSync(resolve(__dirname, '../../components/layout/AppHeader.vue'), 'utf-8')
    const routerSource = readFileSync(resolve(__dirname, '../../router/index.ts'), 'utf-8')

    expect(shellSource).not.toContain('ReviewDialog')
    expect(shellSource).not.toContain('showPostModal')
    expect(headerSource).toContain("router.push({ name: 'course-review-post' })")
    expect(headerSource).not.toContain('openPostModal')
    expect(routerSource).toContain('path: "/courses/reviews/post"')
    expect(routerSource).toContain('path: "/courses/:id/reviews/post"')
    expect(routerSource).toContain('rememberReviewPostCourse(courseID)')
    expect(reviewPageSource).not.toContain('<ReviewDialog')
  })

  it('keeps review drafts user-scoped and recoverable from the post page', () => {
    const draftApiSource = readFileSync(resolve(__dirname, '../../../../shared/src/api/draft.ts'), 'utf-8')
    const draftStoreSource = readFileSync(resolve(__dirname, '../../stores/draft.ts'), 'utf-8')
    const postPageSource = readFileSync(
      resolve(__dirname, '../../modules/review/views/PostReviewPage.vue'),
      'utf-8',
    )

    expect(draftApiSource).toContain("client.GET('/api/v1/course/review/drafts')")
    expect(draftApiSource).toContain("client.DELETE('/api/v1/course/review/drafts')")
    expect(draftApiSource).not.toContain('/drafts/${')
    expect(draftStoreSource).toContain('const draft = ref<Draft | null>(null)')
    expect(draftStoreSource).not.toContain('Record<number')
    expect(postPageSource).toContain('DraftPromptDialog')
    expect(postPageSource).toContain('await draftStore.loadDraft(true)')
    expect(postPageSource).toContain('await draftStore.saveDraft(buildDraftPayload())')
    expect(postPageSource).toContain('onBeforeRouteLeave')
    expect(postPageSource).toContain('await draftStore.deleteDraft()')
  })

  it('keeps clickable notification rows keyboard reachable', () => {
    const notificationItemSource = readFileSync(
      resolve(__dirname, '../../components/common/NotificationItem.vue'),
      'utf-8',
    )

    expect(notificationItemSource).toContain('<button')
    expect(notificationItemSource).toContain('type="button"')
    expect(notificationItemSource).not.toContain('<div\\n    class="flex items-start gap-3 p-3 cursor-pointer')
  })

  it('labels advanced search controls for direct browser interaction', () => {
    const searchSource = readFileSync(
      resolve(__dirname, '../../modules/review/views/SearchPage.vue'),
      'utf-8',
    )

    expect(searchSource).toContain('for="advanced-course-name"')
    expect(searchSource).toContain('id="advanced-course-name"')
    expect(searchSource).toContain('for="advanced-teacher-name"')
    expect(searchSource).toContain('id="advanced-teacher-name"')
  })

  it('shows explicit phone binding configuration errors', () => {
    const phoneBindingSource = readFileSync(
      resolve(__dirname, '../../modules/user/views/PhoneBindingPage.vue'),
      'utf-8',
    )
    const zhUserSource = readFileSync(resolve(__dirname, '../../i18n/locales/zh-CN/user.ts'), 'utf-8')
    const enUserSource = readFileSync(resolve(__dirname, '../../i18n/locales/en-US/user.ts'), 'utf-8')

    expect(phoneBindingSource).toContain("status === 503")
    expect(phoneBindingSource).toContain("user.verification.phone.serviceUnavailable")
    expect(zhUserSource).toContain("serviceUnavailable")
    expect(enUserSource).toContain("serviceUnavailable")
  })
})
