import assert from 'node:assert/strict'
import test from 'node:test'

import { buildIdentityPageData } from './identity-page.service'

test('buildIdentityPageData groups records, summaries and releases', () => {
  const data = buildIdentityPageData({
    generatedAt: '2026-04-21T08:00:00.000Z',
    guardRecords: [
      createGuardMember({
        id: 'gm-active',
        guildId: '1001',
        memberId: '2001',
        releasedAt: null,
      }),
      createGuardMember({
        id: 'gm-release',
        guildId: '1001',
        memberId: '2002',
        verificationState: 'verified',
        releasedAt: new Date('2026-04-21T07:59:00.000Z'),
      }),
    ],
    verificationProfiles: [
      {
        qqID: '2001',
        userID: 1,
        boundAt: '2026-04-20T10:00:00.000Z',
        verificationState: 'bound_unverified',
        studentVerified: false,
      },
      {
        qqID: '2002',
        userID: 2,
        boundAt: '2026-04-20T09:00:00.000Z',
        verificationState: 'verified',
        studentVerified: true,
      },
    ],
    lookupErrors: [{ memberId: '2003', message: 'platform unavailable' }],
  })

  assert.equal(data.summary.pendingMembers, 1)
  assert.equal(data.summary.verifiedMembers, 1)
  assert.equal(data.groups[0].guildId, '1001')
  assert.equal(data.members[0].profile?.qqID, '2001')
  assert.equal(data.recentReleases[0].memberId, '2002')
  assert.equal(data.lookupErrors[0].memberId, '2003')
})

function createGuardMember(overrides: Record<string, unknown>) {
  return {
    id: 'gm-1',
    platform: 'onebot',
    botSelfId: 'bot',
    guildId: '1001',
    channelId: '1001',
    memberId: '2001',
    memberName: 'Alice',
    verificationState: 'bound_unverified',
    joinedAt: new Date('2026-04-21T07:50:00.000Z'),
    deadlineAt: new Date('2026-04-21T08:20:00.000Z'),
    mutedAt: new Date('2026-04-21T07:51:00.000Z'),
    reminderSentAt: null,
    releasedAt: null,
    kickedAt: null,
    lastError: null,
    createdAt: new Date('2026-04-21T07:50:00.000Z'),
    updatedAt: new Date('2026-04-21T07:52:00.000Z'),
    ...overrides,
  }
}
