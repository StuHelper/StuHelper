import { describe, expect, it } from 'vitest'
import {
  isJoinAdmissionHost,
  isJoinAdmissionPath,
  shouldBlockJoinHostRoute,
} from '../join-domain'

describe('join admission domain routing', () => {
  it('recognizes production and local join admission hosts', () => {
    expect(isJoinAdmissionHost('join.stuhelper.com')).toBe(true)
    expect(isJoinAdmissionHost('JOIN.LOCALHOST')).toBe(true)
    expect(isJoinAdmissionHost('stuhelper.com')).toBe(false)
    expect(isJoinAdmissionHost('localhost')).toBe(false)
  })

  it('only treats admission verification and mobile camera paths as join paths', () => {
    expect(isJoinAdmissionPath('/verify/ABCD1234')).toBe(true)
    expect(isJoinAdmissionPath('/admission/freshman/camera/mobile-token')).toBe(true)
    expect(isJoinAdmissionPath('/verify')).toBe(false)
    expect(isJoinAdmissionPath('/courses')).toBe(false)
    expect(isJoinAdmissionPath('/')).toBe(false)
  })

  it('blocks non-admission pages on join hosts', () => {
    expect(shouldBlockJoinHostRoute('join.stuhelper.com', '/courses')).toBe(true)
    expect(shouldBlockJoinHostRoute('join.localhost', '/verify/ABCD1234')).toBe(false)
    expect(shouldBlockJoinHostRoute('localhost', '/courses')).toBe(false)
  })
})
