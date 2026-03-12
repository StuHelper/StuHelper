import { describe, expect, it } from 'vitest'
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const repoRoot = resolve(__dirname, '../../../../../')

describe('stale frontend artifacts', () => {
  it('removes stale unused web pages that were replaced by active routes', () => {
    expect(existsSync(resolve(repoRoot, 'clients/web/src/modules/teacher/views/TeacherPage.vue'))).toBe(false)
    expect(existsSync(resolve(repoRoot, 'clients/web/src/modules/course/views/CourseDetailPage.vue'))).toBe(false)
  })

  it('removes duplicate generated api types from clients/web/src/types', () => {
    expect(existsSync(resolve(repoRoot, 'clients/web/src/types/api.gen.ts'))).toBe(false)
  })

  it('updates root readme away from deleted frontend path and old port', () => {
    const source = readFileSync(resolve(repoRoot, 'README.md'), 'utf-8')
    expect(source).not.toContain('clients/web/course')
    expect(source).not.toContain('http://localhost:5173')
  })

  it('updates clients readme away from old uni-app path and old ports', () => {
    const source = readFileSync(resolve(repoRoot, 'clients/README.md'), 'utf-8')
    expect(source).not.toContain('clients/uni-app')
    expect(source).not.toContain('http://localhost:5173')
    expect(source).not.toContain('http://localhost:5174')
  })

  it('updates project rules away from old frontend path and generated type path', () => {
    const source = readFileSync(resolve(repoRoot, '.project_rule/project_rules.md'), 'utf-8')
    expect(source).not.toContain('clients/web/course/src/')
    expect(source).not.toContain('clients/web/course/src/types/api.gen.ts')
  })

  it('updates auth and quick-start docs away from old local frontend port', () => {
    const quickStart = readFileSync(resolve(repoRoot, 'docs/tutorials/quick-start.md'), 'utf-8')
    const authGuide = readFileSync(resolve(repoRoot, 'docs/modules/auth/01-casdoor-sso.md'), 'utf-8')
    expect(quickStart).not.toContain('http://localhost:5173')
    expect(authGuide).not.toContain('http://localhost:5173')
  })
})
