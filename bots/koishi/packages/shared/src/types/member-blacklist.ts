import type { components, operations } from './api.gen'

type Schemas = components['schemas']

// 成员黑名单 API 契约类型：全部别名自 api.gen.ts，禁止在此手写字段。

export type MemberBlacklistSubjectType = Schemas['MemberBlacklistSubjectType']
export type MemberBlacklistScopeType = Schemas['MemberBlacklistScopeType']
export type MemberBlacklistSource = Schemas['MemberBlacklistSource']
export type MemberBlacklistReasonCode = Schemas['MemberBlacklistReasonCode']
export type MemberBlacklistActorType = Schemas['MemberBlacklistActorType']
export type MemberBlacklistCreatedFrom = Schemas['MemberBlacklistCreatedFrom']
export type MemberBlacklistStatus = Schemas['MemberBlacklistStatus']
export type MemberBlacklistReleaseReasonCode = Schemas['MemberBlacklistReleaseReasonCode']

export type MemberBlacklistEntry = Schemas['MemberBlacklistEntry']

export type MemberBlacklistCreateRequest = Schemas['MemberBlacklistCreateRequest']

export type MemberBlacklistReleaseRequest = Schemas['MemberBlacklistReleaseRequest']

export type MemberBlacklistReleaseBySubjectRequest =
  Schemas['MemberBlacklistReleaseBySubjectRequest']

export type MemberBlacklistAccessRequest =
  operations['getBotMemberBlacklistAccess']['parameters']['query']

export type MemberBlacklistListRequest = NonNullable<
  operations['listBotMemberBlacklist']['parameters']['query']
>

export type MemberBlacklistListResult =
  operations['listBotMemberBlacklist']['responses'][200]['content']['application/json']['data']

export type MemberBlacklistAccessDecision = Schemas['MemberBlacklistAccessDecision']

export interface PlatformRequestOptions {
  readonly timeoutMs?: number
}
