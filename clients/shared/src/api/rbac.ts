import type { ApiClient } from './client'
import type { components } from '../types/api.gen'

type CreateRoleRequest = components['schemas']['CreateRoleRequest']
type UpdateRoleRequest = components['schemas']['UpdateRoleRequest']
type AssignRolePermissionsRequest = components['schemas']['AssignRolePermissionsRequest']
type AssignUserRolesRequest = components['schemas']['AssignUserRolesRequest']
type SetUserPermissionRequest = components['schemas']['SetUserPermissionRequest']
type CreateUserGroupRequest = components['schemas']['CreateUserGroupRequest']
type UpdateUserGroupRequest = components['schemas']['UpdateUserGroupRequest']
type AssignGroupMembersRequest = components['schemas']['AssignGroupMembersRequest']
type AssignGroupPermissionsRequest = components['schemas']['AssignGroupPermissionsRequest']

export const createRbacApi = (client: ApiClient) => ({
  // Roles
  listRoles: () =>
    client.GET('/api/v1/admin/roles'),

  createRole: (data: CreateRoleRequest) =>
    client.POST('/api/v1/admin/roles', { body: data }),

  updateRole: (roleID: number, data: UpdateRoleRequest) =>
    client.PUT('/api/v1/admin/roles/{roleID}', { params: { path: { roleID } }, body: data }),

  deleteRole: (roleID: number) =>
    client.DELETE('/api/v1/admin/roles/{roleID}', { params: { path: { roleID } } }),

  assignRolePermissions: (roleID: number, data: AssignRolePermissionsRequest) =>
    client.PUT('/api/v1/admin/roles/{roleID}/permissions', { params: { path: { roleID } }, body: data }),

  // Permissions
  listPermissions: (params?: { module?: string }) =>
    client.GET('/api/v1/admin/permissions', { params: { query: params } }),

  // User roles & permissions
  getUserRoles: (userID: number) =>
    client.GET('/api/v1/admin/users/{userID}/roles', { params: { path: { userID } } }),

  assignUserRoles: (userID: number, data: AssignUserRolesRequest) =>
    client.PUT('/api/v1/admin/users/{userID}/roles', { params: { path: { userID } }, body: data }),

  getUserEffectivePermissions: (userID: number) =>
    client.GET('/api/v1/admin/users/{userID}/permissions', { params: { path: { userID } } }),

  setUserPermissionOverride: (userID: number, data: SetUserPermissionRequest) =>
    client.PUT('/api/v1/admin/users/{userID}/permissions', { params: { path: { userID } }, body: data }),

  // Groups
  listGroups: () =>
    client.GET('/api/v1/admin/groups'),

  createGroup: (data: CreateUserGroupRequest) =>
    client.POST('/api/v1/admin/groups', { body: data }),

  updateGroup: (groupID: number, data: UpdateUserGroupRequest) =>
    client.PUT('/api/v1/admin/groups/{groupID}', { params: { path: { groupID } }, body: data }),

  deleteGroup: (groupID: number) =>
    client.DELETE('/api/v1/admin/groups/{groupID}', { params: { path: { groupID } } }),

  listGroupMembers: (groupID: number) =>
    client.GET('/api/v1/admin/groups/{groupID}/members', { params: { path: { groupID } } }),

  assignGroupMembers: (groupID: number, data: AssignGroupMembersRequest) =>
    client.PUT('/api/v1/admin/groups/{groupID}/members', { params: { path: { groupID } }, body: data }),

  assignGroupPermissions: (groupID: number, data: AssignGroupPermissionsRequest) =>
    client.PUT('/api/v1/admin/groups/{groupID}/permissions', { params: { path: { groupID } }, body: data }),
})
