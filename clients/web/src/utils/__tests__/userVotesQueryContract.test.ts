import { describe, expect, it, vi } from 'vitest'
import { createUserApi } from '../../../../shared/src/api/user'

describe('createUserApi.getMyVotes', () => {
  it('sends voteType as the query parameter name', async () => {
    const GET = vi.fn(async () => ({ data: undefined }))
    const api = createUserApi({ GET } as never)

    await api.getMyVotes(2, 10, 'dislike')

    expect(GET).toHaveBeenCalledWith(
      '/api/v1/course/review/user/votes',
      {
        params: {
          query: {
            page: 2,
            pageSize: 10,
            voteType: 'dislike',
          },
        },
      },
    )
  })
})
