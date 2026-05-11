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

  it('hydrates server cookie sessions after the app is mounted', () => {
    const mainSource = readFileSync(resolve(__dirname, '../../main.ts'), 'utf-8')

    expect(mainSource).toMatch(/app\.mount\('#app'\)\s*void authStore\.bootstrapSession\(\)/)
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
