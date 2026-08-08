// @vitest-environment jsdom

import { describe, expect, it } from 'vitest'

import {
  isCrossOriginAccountFlowRedirect,
  readAccountFlowRedirect,
  readInternalRedirect,
} from '../internalRedirect'

describe('readInternalRedirect', () => {
  it('accepts only application-internal absolute paths', () => {
    expect(readInternalRedirect('/start?from=verification')).toBe('/start?from=verification')
    expect(readInternalRedirect('https://example.com')).toBe('/identity')
    expect(readInternalRedirect('//example.com/path')).toBe('/identity')
    expect(readInternalRedirect('/\\example.com')).toBe('/identity')
  })

  it('uses the caller fallback for missing values', () => {
    expect(readInternalRedirect(undefined, '/')).toBe('/')
  })

  it('allows only current join-domain business routes as cross-origin account returns', () => {
    expect(readAccountFlowRedirect('https://join.stuhelper.com/verify/ABCD?from=account'))
      .toBe('https://join.stuhelper.com/verify/ABCD?from=account')
    expect(readAccountFlowRedirect('http://join.localhost:3000/start'))
      .toBe('http://join.localhost:3000/start')
    expect(readAccountFlowRedirect('http://join.stuhelper.com/verify/ABCD')).toBe('/identity')
    expect(readAccountFlowRedirect('https://join.stuhelper.com/courses')).toBe('/identity')
    expect(readAccountFlowRedirect('https://evil.example/verify/ABCD')).toBe('/identity')
    expect(readAccountFlowRedirect('https://user@join.stuhelper.com/verify/ABCD')).toBe('/identity')
  })

  it('distinguishes an allowlisted cross-origin return from an internal route', () => {
    expect(isCrossOriginAccountFlowRedirect('/identity')).toBe(false)
    expect(isCrossOriginAccountFlowRedirect('https://join.stuhelper.com/start')).toBe(true)
  })
})
