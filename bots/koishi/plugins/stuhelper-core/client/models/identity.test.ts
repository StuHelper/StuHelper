import assert from 'node:assert/strict'
import test from 'node:test'

import type { IdentityPageData } from '../page-types'
import { buildIdentityModel } from './identity'

test('buildIdentityModel groups members by guild-first scope and resolves selected member detail', () => {
  const model = buildIdentityModel(createIdentityFixture(), {
    guildId: '1001',
    itemId: 'gm-2',
    keyword: '',
  })

  assert.equal(model.selectedGuildId, '1001')
  assert.equal(model.selectedMember?.memberId, '2002')
  assert.equal(model.detailCards[0].label, '认证状态')
})

function createIdentityFixture(): IdentityPageData {
  return {
    generatedAt: '2026-04-21T08:00:00.000Z',
    summary: {
      pendingMembers: 2,
      verifiedMembers: 1,
      boundUnverifiedMembers: 1,
      unboundMembers: 0,
      releasedMembers: 0,
    },
    groups: [
      { guildId: '1001', memberCount: 2, pendingCount: 2, releasedCount: 0 },
    ],
    members: [
      {
        id: 'gm-1',
        platform: 'onebot',
        botSelfId: 'bot',
        guildId: '1001',
        channelId: '1001',
        memberId: '2001',
        memberName: 'Alice',
        verificationState: 'bound_unverified',
        joinedAt: '2026-04-21T07:50:00.000Z',
        deadlineAt: '2026-04-21T08:20:00.000Z',
        mutedAt: '2026-04-21T07:51:00.000Z',
        reminderSentAt: null,
        releasedAt: null,
        kickedAt: null,
        lastError: null,
        createdAt: '2026-04-21T07:50:00.000Z',
        updatedAt: '2026-04-21T07:52:00.000Z',
        profile: {
          qqID: '2001',
          userID: 1,
          boundAt: '2026-04-20T07:00:00.000Z',
          verificationState: 'bound_unverified',
          studentVerified: false,
        },
      },
      {
        id: 'gm-2',
        platform: 'onebot',
        botSelfId: 'bot',
        guildId: '1001',
        channelId: '1001',
        memberId: '2002',
        memberName: 'Bob',
        verificationState: 'verified',
        joinedAt: '2026-04-21T07:40:00.000Z',
        deadlineAt: '2026-04-21T08:10:00.000Z',
        mutedAt: '2026-04-21T07:41:00.000Z',
        reminderSentAt: null,
        releasedAt: null,
        kickedAt: null,
        lastError: null,
        createdAt: '2026-04-21T07:40:00.000Z',
        updatedAt: '2026-04-21T07:42:00.000Z',
        profile: {
          qqID: '2002',
          userID: 2,
          boundAt: '2026-04-20T08:00:00.000Z',
          verificationState: 'verified',
          studentVerified: true,
        },
      },
    ],
    recentReleases: [],
    lookupErrors: [],
  }
}
