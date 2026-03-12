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
})
