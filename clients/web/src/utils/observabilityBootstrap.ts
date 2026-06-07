import { isJoinAdmissionHost } from '@/router/join-domain'

interface ObservabilityBootstrapInput {
  apiBaseUrl?: string
  e2eApiStub?: string
  hostname: string
}

export function shouldInitObservability(input: ObservabilityBootstrapInput): boolean {
  if (input.e2eApiStub === '1') {
    return false
  }

  if (isJoinAdmissionHost(input.hostname) && (input.apiBaseUrl ?? '').trim() === '') {
    return false
  }

  return true
}
