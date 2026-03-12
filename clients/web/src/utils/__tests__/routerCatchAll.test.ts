import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('router catch-all 404', () => {
  it('registers a catch-all pathMatch route', () => {
    const source = readFileSync(resolve(__dirname, '../../router/index.ts'), 'utf-8')
    expect(source).toContain('/:pathMatch(.*)*')
  })

  it('uses NotFoundPage as the catch-all component', () => {
    const source = readFileSync(resolve(__dirname, '../../router/index.ts'), 'utf-8')
    expect(source).toContain("NotFoundPage.vue")
  })
})
