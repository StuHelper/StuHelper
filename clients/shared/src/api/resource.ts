import type { ApiClient } from './client'
import type { components, operations } from '../types/api.gen'

export type ResourceListParams = operations['listResources']['parameters']['query']
export type ResourceMineListParams = operations['listMyResources']['parameters']['query']
export type ResourceItem = components['schemas']['ResourceItem']
export type CreateResourceRequest = components['schemas']['CreateResourceRequest']
export type UpdateResourceRequest = components['schemas']['UpdateResourceRequest']
export type ResourceDownloadURL = components['schemas']['ResourceDownloadURL']
export type ResourceID = string

type GeneratedResourcePath = operations['getResource']['parameters']['path']

function resourcePath(resourceID: ResourceID): GeneratedResourcePath {
  // openapi-typescript maps int64 path params to number, but URL ids must keep
  // their original string representation to avoid losing precision in browsers.
  return { resourceID } as unknown as GeneratedResourcePath
}

export const createResourceApi = (client: ApiClient) => ({
  listResources: (params?: ResourceListParams) =>
    client.GET('/api/v1/resources', { params: { query: params } }),

  listMyResources: (params?: ResourceMineListParams) =>
    client.GET('/api/v1/resources/mine', { params: { query: params } }),

  getResource: (resourceID: ResourceID) =>
    client.GET('/api/v1/resources/{resourceID}', {
      params: { path: resourcePath(resourceID) },
    }),

  createResource: (data: CreateResourceRequest) =>
    client.POST('/api/v1/resources', { body: data }),

  updateResource: (resourceID: ResourceID, data: UpdateResourceRequest) =>
    client.PATCH('/api/v1/resources/{resourceID}', {
      params: { path: resourcePath(resourceID) },
      body: data,
    }),

  deleteResource: (resourceID: ResourceID) =>
    client.DELETE('/api/v1/resources/{resourceID}', {
      params: { path: resourcePath(resourceID) },
    }),

  getDownloadURL: (resourceID: ResourceID) =>
    client.GET('/api/v1/resources/{resourceID}/download-url', {
      params: { path: resourcePath(resourceID) },
    }),
})
