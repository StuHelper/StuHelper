import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const styleSource = readFileSync(resolve(__dirname, '../../styles/tailwind.css'), 'utf-8')

describe('theme style fallback', () => {
  it('does not apply dark tokens unconditionally when data-theme is absent', () => {
    expect(styleSource).not.toContain('[data-theme="dark"],\n  :root:where(:not([data-theme="light"]):not([data-theme="dark"])) {')
    expect(styleSource).not.toContain('[data-theme="dark"] body::before,\n  :root:where(:not([data-theme="light"]):not([data-theme="dark"])) body::before {')
  })

  it('scopes system dark fallback under prefers-color-scheme', () => {
    expect(styleSource).toMatch(/@media \(prefers-color-scheme: dark\) \{\s+:root:not\(\[data-theme="light"\]\):not\(\[data-theme="dark"\]\) \{/)
    expect(styleSource).toMatch(/@media \(prefers-color-scheme: dark\) \{[\s\S]*:root:not\(\[data-theme="light"\]\):not\(\[data-theme="dark"\]\) body::before \{/)
  })
})
