import { describe, expect, it, vi } from 'vitest'

import { createIdentityApi } from '../identity'
import type { ApiClient } from '../client'

function createMockClient(): ApiClient {
  return {
    DELETE: vi.fn(),
    GET: vi.fn(),
    PATCH: vi.fn(),
    POST: vi.fn(),
    PUT: vi.fn(),
  } as unknown as ApiClient
}

describe('createIdentityApi', () => {
  it('posts academic match requests to the user profile endpoint', () => {
    const client = createMockClient()
    const api = createIdentityApi(client)

    api.matchStudentEmailAcademicStudent({
      schoolCode: '4111010006',
      studentID: '20250001',
      studentName: '张三',
    })

    expect(client.POST).toHaveBeenCalledWith(
      '/api/v1/user/profile/school-email/academic-match',
      {
        body: {
          schoolCode: '4111010006',
          studentID: '20250001',
          studentName: '张三',
        },
      },
    )
  })
})
