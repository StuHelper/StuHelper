import type { StuhelperPlatformConfig } from '../types'

const HEALTH_PATH = '/healthz'
const AUTH_SCHEME = 'Bearer'

export interface PlatformClient {
  getHealth(): Promise<void>
}

export function createPlatformClient(config: StuhelperPlatformConfig): PlatformClient {
  return {
    async getHealth() {
      const endpoint = new URL(HEALTH_PATH, config.baseUrl)
      const response = await fetch(endpoint, {
        headers: {
          Authorization: `${AUTH_SCHEME} ${config.serviceToken}`,
        },
      })

      if (!response.ok) {
        throw new Error(`platform request failed: ${response.status}`)
      }
    },
  }
}
