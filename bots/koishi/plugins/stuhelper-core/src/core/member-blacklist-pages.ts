import type {
  MemberBlacklistEntry,
  MemberBlacklistListRequest,
  MemberBlacklistListResult,
  PlatformClient,
} from '@stuhelper/koishi-shared'

const MEMBER_BLACKLIST_PAGE_SIZE = 200

type MemberBlacklistListBackend = Pick<PlatformClient, 'listMemberBlacklist'>

export async function listAllMemberBlacklistPages(
  backend: MemberBlacklistListBackend,
  query: MemberBlacklistListRequest,
): Promise<MemberBlacklistListResult> {
  const list: MemberBlacklistEntry[] = []
  let total = 0

  for (let page = 1; ; page++) {
    const result = await backend.listMemberBlacklist({ ...query, page, pageSize: MEMBER_BLACKLIST_PAGE_SIZE })
    list.push(...result.list)
    total = result.total
    if (list.length >= total) {
      return { list, total }
    }
    if (result.list.length === 0) {
      throw new Error(`member blacklist pagination ended before total was reached: ${list.length}/${total}`)
    }
  }
}
